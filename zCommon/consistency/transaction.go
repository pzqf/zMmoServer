package consistency

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrTransactionAborted  = errors.New("transaction aborted")
	ErrTransactionTimeout  = errors.New("transaction timeout")
	ErrVoteRejected        = errors.New("participant voted reject")
)

type TransactionState int

const (
	TransactionStatePending TransactionState = iota
	TransactionStateVoting
	TransactionStatePrepared
	TransactionStateCommitted
	TransactionStateAborted
)

func (s TransactionState) String() string {
	switch s {
	case TransactionStatePending:
		return "pending"
	case TransactionStateVoting:
		return "voting"
	case TransactionStatePrepared:
		return "prepared"
	case TransactionStateCommitted:
		return "committed"
	case TransactionStateAborted:
		return "aborted"
	default:
		return "unknown"
	}
}

type VoteResult int

const (
	VoteCommit VoteResult = iota
	VoteAbort
)

type ParticipantVote struct {
	Participant string
	Vote        VoteResult
	Reason      string
	Timestamp   time.Time
}

type Transaction struct {
	ID           uint64
	State        TransactionState
	Participants []string
	Votes        map[string]*ParticipantVote
	Prepared     map[string]bool
	CreatedAt    time.Time
	Timeout      time.Duration
	Data         interface{}
	Coordinator  string
}

type PrepareFunc func(ctx context.Context, txID uint64, participant string, data interface{}) error
type CommitFunc func(ctx context.Context, txID uint64, participant string, data interface{}) error
type RollbackFunc func(ctx context.Context, txID uint64, participant string, data interface{}) error
type VoteFunc func(ctx context.Context, txID uint64, participant string, data interface{}) (VoteResult, error)

type TransactionManager struct {
	transactions *zMap.TypedMap[uint64, *Transaction]
	prepareFn    PrepareFunc
	commitFn     CommitFunc
	rollbackFn   RollbackFunc
	voteFn       VoteFunc
	nextID       atomic.Uint64
	cleanupTick  time.Duration
}

type TransactionManagerOption func(*TransactionManager)

func WithVoteFunc(fn VoteFunc) TransactionManagerOption {
	return func(tm *TransactionManager) {
		tm.voteFn = fn
	}
}

func WithCleanupInterval(d time.Duration) TransactionManagerOption {
	return func(tm *TransactionManager) {
		tm.cleanupTick = d
	}
}

func NewTransactionManager(prepareFn PrepareFunc, commitFn CommitFunc, rollbackFn RollbackFunc, opts ...TransactionManagerOption) *TransactionManager {
	tm := &TransactionManager{
		transactions: zMap.NewTypedMap[uint64, *Transaction](),
		prepareFn:    prepareFn,
		commitFn:     commitFn,
		rollbackFn:   rollbackFn,
		cleanupTick:  30 * time.Second,
	}

	for _, opt := range opts {
		opt(tm)
	}

	return tm
}

func (tm *TransactionManager) Begin(coordinator string, participants []string, timeout time.Duration, data interface{}) *Transaction {
	txID := tm.nextID.Add(1)
	tx := &Transaction{
		ID:           txID,
		State:        TransactionStatePending,
		Participants: participants,
		Votes:        make(map[string]*ParticipantVote),
		Prepared:     make(map[string]bool),
		CreatedAt:    time.Now(),
		Timeout:      timeout,
		Data:         data,
		Coordinator:  coordinator,
	}
	tm.transactions.Store(txID, tx)

	zLog.Info("Transaction started",
		zap.Uint64("tx_id", txID),
		zap.String("coordinator", coordinator),
		zap.Strings("participants", participants),
		zap.Duration("timeout", timeout))

	return tx
}

func (tm *TransactionManager) Vote(ctx context.Context, txID uint64, participant string, vote VoteResult, reason string) error {
	tx, exists := tm.transactions.Load(txID)
	if !exists {
		return ErrTransactionNotFound
	}

	tx.Votes[participant] = &ParticipantVote{
		Participant: participant,
		Vote:        vote,
		Reason:      reason,
		Timestamp:   time.Now(),
	}

	zLog.Debug("Transaction vote received",
		zap.Uint64("tx_id", txID),
		zap.String("participant", participant),
		zap.Int("vote", int(vote)),
		zap.String("reason", reason))

	if vote == VoteAbort {
		tx.State = TransactionStateAborted
		zLog.Warn("Transaction aborted due to vote reject",
			zap.Uint64("tx_id", txID),
			zap.String("participant", participant),
			zap.String("reason", reason))
	}

	return nil
}

func (tm *TransactionManager) Prepare(ctx context.Context, txID uint64) error {
	tx, exists := tm.transactions.Load(txID)
	if !exists {
		return ErrTransactionNotFound
	}

	if tx.State != TransactionStatePending {
		return fmt.Errorf("transaction %d is not in pending state: %s", txID, tx.State)
	}

	tx.State = TransactionStateVoting

	if tm.voteFn != nil {
		for _, participant := range tx.Participants {
			select {
			case <-ctx.Done():
				tm.Abort(txID)
				return ErrTransactionTimeout
			default:
			}

			vote, err := tm.voteFn(ctx, txID, participant, tx.Data)
			if err != nil {
				zLog.Error("Transaction vote request failed",
					zap.Uint64("tx_id", txID),
					zap.String("participant", participant),
					zap.Error(err))
				tm.Abort(txID)
				return fmt.Errorf("vote request failed for %s: %w", participant, err)
			}

			if vote == VoteAbort {
				tm.Abort(txID)
				return fmt.Errorf("%s voted abort: %w", participant, ErrVoteRejected)
			}
		}
	}

	for _, participant := range tx.Participants {
		select {
		case <-ctx.Done():
			tm.Abort(txID)
			return ErrTransactionTimeout
		default:
		}

		if err := tm.prepareFn(ctx, txID, participant, tx.Data); err != nil {
			zLog.Error("Transaction prepare failed",
				zap.Uint64("tx_id", txID),
				zap.String("participant", participant),
				zap.Error(err))
			tm.Abort(txID)
			return fmt.Errorf("prepare failed for %s: %w", participant, err)
		}

		tx.Prepared[participant] = true

		zLog.Debug("Transaction participant prepared",
			zap.Uint64("tx_id", txID),
			zap.String("participant", participant))
	}

	tx.State = TransactionStatePrepared

	zLog.Info("Transaction prepared", zap.Uint64("tx_id", txID))
	return nil
}

func (tm *TransactionManager) Commit(ctx context.Context, txID uint64) error {
	tx, exists := tm.transactions.Load(txID)
	if !exists {
		return ErrTransactionNotFound
	}

	if tx.State != TransactionStatePrepared {
		return fmt.Errorf("transaction %d is not in prepared state: %s", txID, tx.State)
	}

	var commitErrors []error
	for _, participant := range tx.Participants {
		select {
		case <-ctx.Done():
			break
		default:
		}

		if err := tm.commitFn(ctx, txID, participant, tx.Data); err != nil {
			commitErrors = append(commitErrors, fmt.Errorf("%s: %w", participant, err))
			zLog.Error("Transaction commit failed for participant",
				zap.Uint64("tx_id", txID),
				zap.String("participant", participant),
				zap.Error(err))
		}
	}

	if len(commitErrors) == 0 {
		tx.State = TransactionStateCommitted
	} else {
		tx.State = TransactionStateAborted
	}

	if len(commitErrors) > 0 {
		return fmt.Errorf("commit errors: %v", commitErrors)
	}

	zLog.Info("Transaction committed", zap.Uint64("tx_id", txID))
	return nil
}

func (tm *TransactionManager) Abort(txID uint64) {
	tx, exists := tm.transactions.Load(txID)
	if !exists {
		return
	}

	tx.State = TransactionStateAborted

	for _, participant := range tx.Participants {
		if tx.Prepared[participant] {
			if err := tm.rollbackFn(context.Background(), txID, participant, tx.Data); err != nil {
				zLog.Error("Transaction rollback failed",
					zap.Uint64("tx_id", txID),
					zap.String("participant", participant),
					zap.Error(err))
			}
		}
	}

	zLog.Info("Transaction aborted", zap.Uint64("tx_id", txID))
}

func (tm *TransactionManager) Execute(ctx context.Context, coordinator string, participants []string, timeout time.Duration, data interface{}) error {
	tx := tm.Begin(coordinator, participants, timeout, data)

	txCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := tm.Prepare(txCtx, tx.ID); err != nil {
		return fmt.Errorf("prepare phase failed: %w", err)
	}

	if err := tm.Commit(txCtx, tx.ID); err != nil {
		return fmt.Errorf("commit phase failed: %w", err)
	}

	return nil
}

func (tm *TransactionManager) GetTransaction(txID uint64) (*Transaction, bool) {
	return tm.transactions.Load(txID)
}

func (tm *TransactionManager) GetTransactionState(txID uint64) (TransactionState, bool) {
	tx, exists := tm.transactions.Load(txID)
	if !exists {
		return TransactionStatePending, false
	}
	return tx.State, true
}

func (tm *TransactionManager) Cleanup(maxAge time.Duration) {
	now := time.Now()
	var toDelete []uint64

	tm.transactions.Range(func(id uint64, tx *Transaction) bool {
		age := now.Sub(tx.CreatedAt)
		if (tx.State == TransactionStateCommitted || tx.State == TransactionStateAborted) && age > maxAge {
			toDelete = append(toDelete, id)
		}
		if tx.State == TransactionStatePending && age > tx.Timeout {
			toDelete = append(toDelete, id)
			zLog.Warn("Cleaning up timed out transaction",
				zap.Uint64("tx_id", id),
				zap.Duration("age", age))
		}
		return true
	})

	for _, id := range toDelete {
		tm.transactions.Delete(id)
	}

	if len(toDelete) > 0 {
		zLog.Debug("Cleaned up transactions", zap.Int("count", len(toDelete)))
	}
}

func (tm *TransactionManager) StartCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(tm.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tm.Cleanup(5 * time.Minute)
		}
	}
}

func (tm *TransactionManager) ActiveCount() int {
	count := 0
	tm.transactions.Range(func(id uint64, tx *Transaction) bool {
		if tx.State == TransactionStatePending || tx.State == TransactionStateVoting || tx.State == TransactionStatePrepared {
			count++
		}
		return true
	})
	return count
}

func (tm *TransactionManager) TotalCount() int {
	return int(tm.transactions.Len())
}
