package consistency

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrTransactionAborted  = errors.New("transaction aborted")
	ErrTransactionTimeout  = errors.New("transaction timeout")
)

type TransactionState int

const (
	TransactionStatePending  TransactionState = iota
	TransactionStatePrepared
	TransactionStateCommitted
	TransactionStateAborted
)

func (s TransactionState) String() string {
	switch s {
	case TransactionStatePending:
		return "pending"
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

type Transaction struct {
	ID        uint64
	State     TransactionState
	Participants []string
	Prepared  map[string]bool
	CreatedAt time.Time
	Timeout   time.Duration
	Data      interface{}
}

type PrepareFunc func(txID uint64, participant string, data interface{}) error
type CommitFunc func(txID uint64, participant string, data interface{}) error
type RollbackFunc func(txID uint64, participant string, data interface{}) error

type TransactionManager struct {
	mu          sync.Mutex
	transactions map[uint64]*Transaction
	prepareFn   PrepareFunc
	commitFn    CommitFunc
	rollbackFn  RollbackFunc
	nextID      uint64
}

func NewTransactionManager(prepareFn PrepareFunc, commitFn CommitFunc, rollbackFn RollbackFunc) *TransactionManager {
	return &TransactionManager{
		transactions: make(map[uint64]*Transaction),
		prepareFn:    prepareFn,
		commitFn:     commitFn,
		rollbackFn:   rollbackFn,
	}
}

func (tm *TransactionManager) Begin(participants []string, timeout time.Duration, data interface{}) *Transaction {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.nextID++
	tx := &Transaction{
		ID:           tm.nextID,
		State:        TransactionStatePending,
		Participants: participants,
		Prepared:     make(map[string]bool),
		CreatedAt:    time.Now(),
		Timeout:      timeout,
		Data:         data,
	}
	tm.transactions[tx.ID] = tx

	zLog.Info("Transaction started",
		zap.Uint64("tx_id", tx.ID),
		zap.Strings("participants", participants))

	return tx
}

func (tm *TransactionManager) Prepare(ctx context.Context, txID uint64) error {
	tm.mu.Lock()
	tx, ok := tm.transactions[txID]
	if !ok {
		tm.mu.Unlock()
		return ErrTransactionNotFound
	}
	tm.mu.Unlock()

	if tx.State != TransactionStatePending {
		return fmt.Errorf("transaction %d is not in pending state: %s", txID, tx.State)
	}

	for _, participant := range tx.Participants {
		select {
		case <-ctx.Done():
			tm.Abort(txID)
			return ErrTransactionTimeout
		default:
		}

		if err := tm.prepareFn(txID, participant, tx.Data); err != nil {
			zLog.Error("Transaction prepare failed",
				zap.Uint64("tx_id", txID),
				zap.String("participant", participant),
				zap.Error(err))
			tm.Abort(txID)
			return fmt.Errorf("prepare failed for %s: %w", participant, err)
		}

		tm.mu.Lock()
		tx.Prepared[participant] = true
		tm.mu.Unlock()

		zLog.Debug("Transaction participant prepared",
			zap.Uint64("tx_id", txID),
			zap.String("participant", participant))
	}

	tm.mu.Lock()
	tx.State = TransactionStatePrepared
	tm.mu.Unlock()

	zLog.Info("Transaction prepared", zap.Uint64("tx_id", txID))
	return nil
}

func (tm *TransactionManager) Commit(ctx context.Context, txID uint64) error {
	tm.mu.Lock()
	tx, ok := tm.transactions[txID]
	if !ok {
		tm.mu.Unlock()
		return ErrTransactionNotFound
	}

	if tx.State != TransactionStatePrepared {
		tm.mu.Unlock()
		return fmt.Errorf("transaction %d is not in prepared state: %s", txID, tx.State)
	}
	tm.mu.Unlock()

	var commitErrors []error
	for _, participant := range tx.Participants {
		select {
		case <-ctx.Done():
			break
		default:
		}

		if err := tm.commitFn(txID, participant, tx.Data); err != nil {
			commitErrors = append(commitErrors, fmt.Errorf("%s: %w", participant, err))
			zLog.Error("Transaction commit failed for participant",
				zap.Uint64("tx_id", txID),
				zap.String("participant", participant),
				zap.Error(err))
		}
	}

	tm.mu.Lock()
	if len(commitErrors) == 0 {
		tx.State = TransactionStateCommitted
	} else {
		tx.State = TransactionStateAborted
	}
	tm.mu.Unlock()

	if len(commitErrors) > 0 {
		return fmt.Errorf("commit errors: %v", commitErrors)
	}

	zLog.Info("Transaction committed", zap.Uint64("tx_id", txID))
	return nil
}

func (tm *TransactionManager) Abort(txID uint64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tx, ok := tm.transactions[txID]
	if !ok {
		return
	}

	tx.State = TransactionStateAborted

	for _, participant := range tx.Participants {
		if tx.Prepared[participant] {
			if err := tm.rollbackFn(txID, participant, tx.Data); err != nil {
				zLog.Error("Transaction rollback failed",
					zap.Uint64("tx_id", txID),
					zap.String("participant", participant),
					zap.Error(err))
			}
		}
	}

	zLog.Info("Transaction aborted", zap.Uint64("tx_id", txID))
}

func (tm *TransactionManager) Execute(ctx context.Context, participants []string, timeout time.Duration, data interface{}) error {
	tx := tm.Begin(participants, timeout, data)

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

func (tm *TransactionManager) Cleanup(maxAge time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	for id, tx := range tm.transactions {
		age := now.Sub(tx.CreatedAt)
		if (tx.State == TransactionStateCommitted || tx.State == TransactionStateAborted) && age > maxAge {
			delete(tm.transactions, id)
		}
	}
}

type OperationLog struct {
	mu      sync.Mutex
	entries map[uint64]LogEntry
}

type LogEntry struct {
	ID        uint64
	Operation string
	Data      interface{}
	Timestamp time.Time
	Completed bool
}

func NewOperationLog() *OperationLog {
	return &OperationLog{
		entries: make(map[uint64]LogEntry),
	}
}

func (ol *OperationLog) Record(id uint64, operation string, data interface{}) {
	ol.mu.Lock()
	defer ol.mu.Unlock()
	ol.entries[id] = LogEntry{
		ID:        id,
		Operation: operation,
		Data:      data,
		Timestamp: time.Now(),
		Completed: false,
	}
}

func (ol *OperationLog) Complete(id uint64) {
	ol.mu.Lock()
	defer ol.mu.Unlock()
	if entry, ok := ol.entries[id]; ok {
		entry.Completed = true
		ol.entries[id] = entry
	}
}

func (ol *OperationLog) GetIncomplete() []LogEntry {
	ol.mu.Lock()
	defer ol.mu.Unlock()
	var result []LogEntry
	for _, entry := range ol.entries {
		if !entry.Completed {
			result = append(result, entry)
		}
	}
	return result
}

func (ol *OperationLog) Cleanup(maxAge time.Duration) {
	ol.mu.Lock()
	defer ol.mu.Unlock()
	now := time.Now()
	for id, entry := range ol.entries {
		if entry.Completed && now.Sub(entry.Timestamp) > maxAge {
			delete(ol.entries, id)
		}
	}
}
