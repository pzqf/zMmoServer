package crossserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"sync/atomic"
	"time"

	"github.com/pzqf/zCommon/consistency"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

var (
	ErrMigrationNotFound      = errors.New("migration not found")
	ErrMigrationInvalidState  = errors.New("invalid migration state transition")
	ErrMigrationAlreadyExists = errors.New("migration already exists for player")
	ErrMigrationRejected      = errors.New("migration rejected by target server")
	ErrMigrationTimeout       = errors.New("migration timed out")
	ErrMigrationDataInvalid   = errors.New("migration data checksum mismatch")
)

type MigrationRecord struct {
	ID            uint64
	PlayerID      int64
	AccountID     int64
	PlayerName    string
	State         MigrationState
	Type          MigrationType
	SourceServer  int32
	SourceService uint8
	TargetServer  int32
	TargetService uint8
	TargetMapID   int32
	Reason        string
	PlayerData    []byte
	MapData       []byte
	DataChecksum  uint32
	DataVersion   int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Timeout       time.Duration
	RollbackData  []byte
}

type PlayerDataSerializer interface {
	SerializePlayer(playerID int64) ([]byte, []byte, error)
	DeserializePlayer(playerData []byte, mapData []byte) error
}

type MigrationCallback interface {
	OnMigrationPrepare(record *MigrationRecord) (bool, string)
	OnMigrationDataReceived(record *MigrationRecord) error
	OnMigrationCommit(record *MigrationRecord) error
	OnMigrationRollback(record *MigrationRecord) error
	OnMigrationComplete(record *MigrationRecord) error
}

type MigrationConfig struct {
	DefaultTimeout time.Duration
	MaxRetries     int
	RetryInterval  time.Duration
	CleanupAge     time.Duration
	CleanupTick    time.Duration
}

func DefaultMigrationConfig() MigrationConfig {
	return MigrationConfig{
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     3,
		RetryInterval:  2 * time.Second,
		CleanupAge:     10 * time.Minute,
		CleanupTick:    30 * time.Second,
	}
}

type MigrationManager struct {
	migrations   *zMap.TypedMap[uint64, *MigrationRecord]
	playerIndex  *zMap.TypedMap[int64, uint64]
	config       MigrationConfig
	nextID       atomic.Uint64
	serializer   PlayerDataSerializer
	callback     MigrationCallback
	router       *CrossRouter
	serverRouter *ServerRouter
	txManager    *consistency.TransactionManager
	outbox       consistency.OutboxStore
	inbox        consistency.InboxStore
}

func NewMigrationManager(
	config MigrationConfig,
	serializer PlayerDataSerializer,
	callback MigrationCallback,
	router *CrossRouter,
	serverRouter *ServerRouter,
	txManager *consistency.TransactionManager,
	outbox consistency.OutboxStore,
	inbox consistency.InboxStore,
) *MigrationManager {
	mm := &MigrationManager{
		migrations:   zMap.NewTypedMap[uint64, *MigrationRecord](),
		playerIndex:  zMap.NewTypedMap[int64, uint64](),
		config:       config,
		serializer:   serializer,
		callback:     callback,
		router:       router,
		serverRouter: serverRouter,
		txManager:    txManager,
		outbox:       outbox,
		inbox:        inbox,
	}

	mm.registerHandlers()

	return mm
}

func (mm *MigrationManager) registerHandlers() {
	if mm.router == nil {
		return
	}

	mm.router.RegisterHandler(ServiceTypeGame, MsgMigrationRequest, mm.handleMigrationRequest)
	mm.router.RegisterHandler(ServiceTypeGame, MsgMigrationPrepare, mm.handleMigrationPrepare)
	mm.router.RegisterHandler(ServiceTypeGame, MsgMigrationData, mm.handleMigrationData)
	mm.router.RegisterHandler(ServiceTypeGame, MsgMigrationCommit, mm.handleMigrationCommit)
	mm.router.RegisterHandler(ServiceTypeGame, MsgMigrationRollback, mm.handleMigrationRollback)
	mm.router.RegisterHandler(ServiceTypeGame, MsgMigrationComplete, mm.handleMigrationComplete)
	mm.router.RegisterHandler(ServiceTypeGame, MsgMigrationHeartbeat, mm.handleMigrationHeartbeat)
	mm.router.RegisterHandler(ServiceTypeGame, MsgMigrationQueryStatus, mm.handleMigrationQueryStatus)

	mm.router.RegisterHandler(ServiceTypeMap, MsgMigrationRequest, mm.handleMigrationRequest)
	mm.router.RegisterHandler(ServiceTypeMap, MsgMigrationPrepare, mm.handleMigrationPrepare)
	mm.router.RegisterHandler(ServiceTypeMap, MsgMigrationData, mm.handleMigrationData)
	mm.router.RegisterHandler(ServiceTypeMap, MsgMigrationCommit, mm.handleMigrationCommit)
	mm.router.RegisterHandler(ServiceTypeMap, MsgMigrationRollback, mm.handleMigrationRollback)
	mm.router.RegisterHandler(ServiceTypeMap, MsgMigrationComplete, mm.handleMigrationComplete)
	mm.router.RegisterHandler(ServiceTypeMap, MsgMigrationHeartbeat, mm.handleMigrationHeartbeat)
	mm.router.RegisterHandler(ServiceTypeMap, MsgMigrationQueryStatus, mm.handleMigrationQueryStatus)
}

func (mm *MigrationManager) RequestMigration(ctx context.Context, playerID int64, accountID int64, playerName string, targetServer int32, targetService uint8, targetMapID int32, migrationType MigrationType, reason string) (*MigrationRecord, error) {
	if _, exists := mm.playerIndex.Load(playerID); exists {
		return nil, ErrMigrationAlreadyExists
	}

	migrationID := mm.nextID.Add(1)
	now := time.Now()

	record := &MigrationRecord{
		ID:            migrationID,
		PlayerID:      playerID,
		AccountID:     accountID,
		PlayerName:    playerName,
		State:         MigrationStateRequested,
		Type:          migrationType,
		SourceServer:  0,
		SourceService: 0,
		TargetServer:  targetServer,
		TargetService: targetService,
		TargetMapID:   targetMapID,
		Reason:        reason,
		CreatedAt:     now,
		UpdatedAt:     now,
		Timeout:       mm.config.DefaultTimeout,
	}

	mm.migrations.Store(migrationID, record)
	mm.playerIndex.Store(playerID, migrationID)

	zLog.Info("Migration requested",
		zap.Uint64("migration_id", migrationID),
		zap.Int64("player_id", playerID),
		zap.Int32("target_server", targetServer),
		zap.Uint8("target_service", targetService),
		zap.String("type", migrationType.String()))

	if err := mm.sendMigrationRequest(ctx, record); err != nil {
		mm.failMigration(migrationID, fmt.Sprintf("send request failed: %v", err))
		return nil, fmt.Errorf("send migration request: %w", err)
	}

	return record, nil
}

func (mm *MigrationManager) sendMigrationRequest(ctx context.Context, record *MigrationRecord) error {
	payload := MigrationRequestPayload{
		MigrationID:   record.ID,
		PlayerID:      record.PlayerID,
		AccountID:     record.AccountID,
		PlayerName:    record.PlayerName,
		SourceServer:  record.SourceServer,
		SourceService: record.SourceService,
		TargetServer:  record.TargetServer,
		TargetService: record.TargetService,
		TargetMapID:   record.TargetMapID,
		MigrationType: record.Type,
		Reason:        record.Reason,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request payload: %w", err)
	}

	meta := NewRequestMeta(record.SourceService, record.SourceServer)
	envelope := Wrap(meta, data)

	if mm.serverRouter != nil {
		return mm.serverRouter.SendToServer(fmt.Sprintf("%d", record.TargetServer), envelope)
	}

	return fmt.Errorf("no server router available")
}

func (mm *MigrationManager) PrepareMigration(ctx context.Context, migrationID uint64) error {
	record, exists := mm.migrations.Load(migrationID)
	if !exists {
		return ErrMigrationNotFound
	}

	if record.State != MigrationStateRequested {
		return fmt.Errorf("%w: cannot prepare from state %s", ErrMigrationInvalidState, record.State)
	}

	record.State = MigrationStatePreparing
	record.UpdatedAt = time.Now()

	if mm.serializer != nil {
		playerData, mapData, err := mm.serializer.SerializePlayer(record.PlayerID)
		if err != nil {
			mm.failMigration(migrationID, fmt.Sprintf("serialize failed: %v", err))
			return fmt.Errorf("serialize player data: %w", err)
		}

		record.PlayerData = playerData
		record.MapData = mapData
		record.DataChecksum = crc32.ChecksumIEEE(playerData)
		record.RollbackData = playerData
	}

	record.State = MigrationStatePrepared
	record.UpdatedAt = time.Now()

	zLog.Info("Migration prepared",
		zap.Uint64("migration_id", migrationID),
		zap.Int64("player_id", record.PlayerID))

	return mm.sendMigrationData(ctx, record)
}

func (mm *MigrationManager) sendMigrationData(ctx context.Context, record *MigrationRecord) error {
	if record.State != MigrationStatePrepared {
		return fmt.Errorf("%w: cannot send data from state %s", ErrMigrationInvalidState, record.State)
	}

	record.State = MigrationStateTransferring
	record.UpdatedAt = time.Now()

	payload := MigrationDataPayload{
		MigrationID: record.ID,
		PlayerID:    record.PlayerID,
		PlayerData:  record.PlayerData,
		MapData:     record.MapData,
		Checksum:    record.DataChecksum,
		DataVersion: record.DataVersion,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal data payload: %w", err)
	}

	meta := NewRequestMeta(record.SourceService, record.SourceServer)
	envelope := Wrap(meta, data)

	if mm.outbox != nil {
		mm.outbox.Add(consistency.OutboxMessage{
			RequestID:      record.ID,
			TargetServerID: fmt.Sprintf("%d", record.TargetServer),
			ProtoID:        int32(MsgMigrationData),
			Payload:        envelope,
			CreatedAt:      time.Now(),
		})
	}

	if mm.serverRouter != nil {
		if err := mm.serverRouter.SendToServer(fmt.Sprintf("%d", record.TargetServer), envelope); err != nil {
			if mm.outbox != nil {
				mm.outbox.MarkAttempt(record.ID, err)
			}
			return fmt.Errorf("send migration data: %w", err)
		}
	}

	if mm.outbox != nil {
		mm.outbox.MarkSent(record.ID)
	}

	zLog.Info("Migration data sent",
		zap.Uint64("migration_id", record.ID),
		zap.Int("data_size", len(record.PlayerData)))

	return nil
}

func (mm *MigrationManager) CommitMigration(ctx context.Context, migrationID uint64) error {
	record, exists := mm.migrations.Load(migrationID)
	if !exists {
		return ErrMigrationNotFound
	}

	if record.State != MigrationStateTransferred {
		return fmt.Errorf("%w: cannot commit from state %s", ErrMigrationInvalidState, record.State)
	}

	record.State = MigrationStateCommitting
	record.UpdatedAt = time.Now()

	if mm.txManager != nil {
		participants := []string{
			fmt.Sprintf("%d", record.SourceServer),
			fmt.Sprintf("%d", record.TargetServer),
		}
		if err := mm.txManager.Execute(ctx, fmt.Sprintf("migration_%d", migrationID), participants, record.Timeout, record); err != nil {
			mm.RollbackMigration(migrationID, fmt.Sprintf("transaction failed: %v", err))
			return fmt.Errorf("migration transaction: %w", err)
		}
	}

	if mm.callback != nil {
		if err := mm.callback.OnMigrationCommit(record); err != nil {
			mm.RollbackMigration(migrationID, fmt.Sprintf("commit callback failed: %v", err))
			return fmt.Errorf("commit callback: %w", err)
		}
	}

	record.State = MigrationStateCommitted
	record.UpdatedAt = time.Now()

	payload := MigrationCompletePayload{
		MigrationID: record.ID,
		PlayerID:    record.PlayerID,
		NewServerID: record.TargetServer,
	}

	data, _ := json.Marshal(payload)
	meta := NewRequestMeta(record.SourceService, record.SourceServer)
	envelope := Wrap(meta, data)

	if mm.serverRouter != nil {
		mm.serverRouter.SendToServer(fmt.Sprintf("%d", record.TargetServer), envelope)
	}

	record.State = MigrationStateCompleting
	record.UpdatedAt = time.Now()

	if mm.callback != nil {
		mm.callback.OnMigrationComplete(record)
	}

	record.State = MigrationStateCompleted
	record.UpdatedAt = time.Now()
	mm.playerIndex.Delete(record.PlayerID)

	zLog.Info("Migration completed",
		zap.Uint64("migration_id", migrationID),
		zap.Int64("player_id", record.PlayerID),
		zap.Int32("target_server", record.TargetServer))

	return nil
}

func (mm *MigrationManager) RollbackMigration(migrationID uint64, reason string) {
	record, exists := mm.migrations.Load(migrationID)
	if !exists {
		return
	}

	record.State = MigrationStateRollingBack
	record.UpdatedAt = time.Now()

	if mm.callback != nil {
		mm.callback.OnMigrationRollback(record)
	}

	if mm.txManager != nil {
		mm.txManager.Abort(migrationID)
	}

	payload := MigrationRollbackPayload{
		MigrationID: migrationID,
		PlayerID:    record.PlayerID,
		Reason:      reason,
	}

	data, _ := json.Marshal(payload)
	meta := NewRequestMeta(record.SourceService, record.SourceServer)
	envelope := Wrap(meta, data)

	if mm.serverRouter != nil {
		mm.serverRouter.SendToServer(fmt.Sprintf("%d", record.TargetServer), envelope)
	}

	record.State = MigrationStateRolledBack
	record.UpdatedAt = time.Now()
	mm.playerIndex.Delete(record.PlayerID)

	zLog.Warn("Migration rolled back",
		zap.Uint64("migration_id", migrationID),
		zap.Int64("player_id", record.PlayerID),
		zap.String("reason", reason))
}

func (mm *MigrationManager) failMigration(migrationID uint64, reason string) {
	record, exists := mm.migrations.Load(migrationID)
	if !exists {
		return
	}

	record.State = MigrationStateFailed
	record.UpdatedAt = time.Now()
	mm.playerIndex.Delete(record.PlayerID)

	zLog.Error("Migration failed",
		zap.Uint64("migration_id", migrationID),
		zap.Int64("player_id", record.PlayerID),
		zap.String("reason", reason))
}

func (mm *MigrationManager) handleMigrationRequest(meta Meta, payload []byte) ([]byte, error) {
	var req MigrationRequestPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}

	zLog.Info("Migration request received",
		zap.Uint64("migration_id", req.MigrationID),
		zap.Int64("player_id", req.PlayerID),
		zap.Int32("source_server", req.SourceServer))

	accepted := true
	reason := ""

	if mm.callback != nil {
		record := &MigrationRecord{
			ID:            req.MigrationID,
			PlayerID:      req.PlayerID,
			AccountID:     req.AccountID,
			PlayerName:    req.PlayerName,
			SourceServer:  req.SourceServer,
			SourceService: req.SourceService,
			TargetServer:  req.TargetServer,
			TargetService: req.TargetService,
			TargetMapID:   req.TargetMapID,
			Type:          req.MigrationType,
			Reason:        req.Reason,
		}
		accepted, reason = mm.callback.OnMigrationPrepare(record)
	}

	resp := MigrationPreparePayload{
		MigrationID: req.MigrationID,
		PlayerID:    req.PlayerID,
		Accepted:    accepted,
		Reason:      reason,
	}

	data, _ := json.Marshal(resp)
	return data, nil
}

func (mm *MigrationManager) handleMigrationPrepare(meta Meta, payload []byte) ([]byte, error) {
	var resp MigrationPreparePayload
	if err := json.Unmarshal(payload, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal prepare response: %w", err)
	}

	record, exists := mm.migrations.Load(resp.MigrationID)
	if !exists {
		return nil, ErrMigrationNotFound
	}

	if !resp.Accepted {
		mm.failMigration(resp.MigrationID, fmt.Sprintf("target rejected: %s", resp.Reason))
		return nil, ErrMigrationRejected
	}

	record.State = MigrationStatePrepared
	record.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), record.Timeout)
	defer cancel()

	if err := mm.sendMigrationData(ctx, record); err != nil {
		mm.failMigration(resp.MigrationID, fmt.Sprintf("send data failed: %v", err))
		return nil, err
	}

	return nil, nil
}

func (mm *MigrationManager) handleMigrationData(meta Meta, payload []byte) ([]byte, error) {
	var dataPayload MigrationDataPayload
	if err := json.Unmarshal(payload, &dataPayload); err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}

	if mm.inbox != nil {
		if !mm.inbox.TryAccept(dataPayload.MigrationID) {
			zLog.Warn("Duplicate migration data received",
				zap.Uint64("migration_id", dataPayload.MigrationID))
			resp := MigrationCommitPayload{
				MigrationID: dataPayload.MigrationID,
				PlayerID:    dataPayload.PlayerID,
				Success:     true,
			}
			respData, _ := json.Marshal(resp)
			return respData, nil
		}
	}

	checksum := crc32.ChecksumIEEE(dataPayload.PlayerData)
	if checksum != dataPayload.Checksum {
		resp := MigrationCommitPayload{
			MigrationID: dataPayload.MigrationID,
			PlayerID:    dataPayload.PlayerID,
			Success:     false,
			Reason:      fmt.Sprintf("checksum mismatch: expected %d, got %d", dataPayload.Checksum, checksum),
		}
		respData, _ := json.Marshal(resp)
		return respData, nil
	}

	record := &MigrationRecord{
		ID:           dataPayload.MigrationID,
		PlayerID:     dataPayload.PlayerID,
		PlayerData:   dataPayload.PlayerData,
		MapData:      dataPayload.MapData,
		DataChecksum: dataPayload.Checksum,
		DataVersion:  dataPayload.DataVersion,
		State:        MigrationStateTransferred,
	}

	if mm.callback != nil {
		if err := mm.callback.OnMigrationDataReceived(record); err != nil {
			resp := MigrationCommitPayload{
				MigrationID: dataPayload.MigrationID,
				PlayerID:    dataPayload.PlayerID,
				Success:     false,
				Reason:      err.Error(),
			}
			respData, _ := json.Marshal(resp)
			return respData, nil
		}
	}

	if mm.serializer != nil {
		if err := mm.serializer.DeserializePlayer(dataPayload.PlayerData, dataPayload.MapData); err != nil {
			resp := MigrationCommitPayload{
				MigrationID: dataPayload.MigrationID,
				PlayerID:    dataPayload.PlayerID,
				Success:     false,
				Reason:      fmt.Sprintf("deserialize failed: %v", err),
			}
			respData, _ := json.Marshal(resp)
			return respData, nil
		}
	}

	mm.migrations.Store(dataPayload.MigrationID, record)

	resp := MigrationCommitPayload{
		MigrationID: dataPayload.MigrationID,
		PlayerID:    dataPayload.PlayerID,
		Success:     true,
	}
	respData, _ := json.Marshal(resp)

	if mm.inbox != nil {
		mm.inbox.Ack(dataPayload.MigrationID)
	}

	zLog.Info("Migration data received and deserialized",
		zap.Uint64("migration_id", dataPayload.MigrationID),
		zap.Int64("player_id", dataPayload.PlayerID))

	return respData, nil
}

func (mm *MigrationManager) handleMigrationCommit(meta Meta, payload []byte) ([]byte, error) {
	var commitPayload MigrationCommitPayload
	if err := json.Unmarshal(payload, &commitPayload); err != nil {
		return nil, fmt.Errorf("unmarshal commit: %w", err)
	}

	record, exists := mm.migrations.Load(commitPayload.MigrationID)
	if !exists {
		return nil, ErrMigrationNotFound
	}

	if !commitPayload.Success {
		mm.RollbackMigration(commitPayload.MigrationID, commitPayload.Reason)
		return nil, fmt.Errorf("target commit failed: %s", commitPayload.Reason)
	}

	ctx, cancel := context.WithTimeout(context.Background(), record.Timeout)
	defer cancel()

	if err := mm.CommitMigration(ctx, commitPayload.MigrationID); err != nil {
		return nil, err
	}

	return nil, nil
}

func (mm *MigrationManager) handleMigrationRollback(meta Meta, payload []byte) ([]byte, error) {
	var rbPayload MigrationRollbackPayload
	if err := json.Unmarshal(payload, &rbPayload); err != nil {
		return nil, fmt.Errorf("unmarshal rollback: %w", err)
	}

	mm.RollbackMigration(rbPayload.MigrationID, rbPayload.Reason)
	return nil, nil
}

func (mm *MigrationManager) handleMigrationComplete(meta Meta, payload []byte) ([]byte, error) {
	var completePayload MigrationCompletePayload
	if err := json.Unmarshal(payload, &completePayload); err != nil {
		return nil, fmt.Errorf("unmarshal complete: %w", err)
	}

	record, exists := mm.migrations.Load(completePayload.MigrationID)
	if !exists {
		return nil, ErrMigrationNotFound
	}

	if mm.callback != nil {
		mm.callback.OnMigrationComplete(record)
	}

	record.State = MigrationStateCompleted
	record.UpdatedAt = time.Now()
	mm.playerIndex.Delete(record.PlayerID)

	zLog.Info("Migration complete notification received",
		zap.Uint64("migration_id", completePayload.MigrationID),
		zap.Int64("player_id", completePayload.PlayerID))

	return nil, nil
}

func (mm *MigrationManager) handleMigrationHeartbeat(meta Meta, payload []byte) ([]byte, error) {
	var hbPayload MigrationHeartbeatPayload
	if err := json.Unmarshal(payload, &hbPayload); err != nil {
		return nil, fmt.Errorf("unmarshal heartbeat: %w", err)
	}

	_, exists := mm.migrations.Load(hbPayload.MigrationID)
	if !exists {
		return nil, ErrMigrationNotFound
	}

	zLog.Debug("Migration heartbeat received",
		zap.Uint64("migration_id", hbPayload.MigrationID),
		zap.String("state", hbPayload.State.String()))

	return nil, nil
}

func (mm *MigrationManager) handleMigrationQueryStatus(meta Meta, payload []byte) ([]byte, error) {
	var query MigrationStatusPayload
	if err := json.Unmarshal(payload, &query); err != nil {
		return nil, fmt.Errorf("unmarshal query: %w", err)
	}

	record, exists := mm.migrations.Load(query.MigrationID)
	if !exists {
		return nil, ErrMigrationNotFound
	}

	status := MigrationStatusPayload{
		MigrationID:   record.ID,
		PlayerID:      record.PlayerID,
		State:         record.State,
		SourceServer:  record.SourceServer,
		TargetServer:  record.TargetServer,
		MigrationType: record.Type,
		CreatedAt:     record.CreatedAt.Unix(),
		UpdatedAt:     record.UpdatedAt.Unix(),
	}

	data, _ := json.Marshal(status)
	return data, nil
}

func (mm *MigrationManager) GetMigration(migrationID uint64) (*MigrationRecord, bool) {
	return mm.migrations.Load(migrationID)
}

func (mm *MigrationManager) GetMigrationByPlayer(playerID int64) (*MigrationRecord, bool) {
	migrationID, exists := mm.playerIndex.Load(playerID)
	if !exists {
		return nil, false
	}
	return mm.migrations.Load(migrationID)
}

func (mm *MigrationManager) GetActiveMigrations() []*MigrationRecord {
	var result []*MigrationRecord
	mm.migrations.Range(func(id uint64, record *MigrationRecord) bool {
		if !record.State.IsTerminal() {
			result = append(result, record)
		}
		return true
	})
	return result
}

func (mm *MigrationManager) StartCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(mm.config.CleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mm.Cleanup(mm.config.CleanupAge)
			mm.checkTimeouts()
		}
	}
}

func (mm *MigrationManager) Cleanup(maxAge time.Duration) {
	now := time.Now()
	var toDelete []uint64

	mm.migrations.Range(func(id uint64, record *MigrationRecord) bool {
		if record.State.IsTerminal() && now.Sub(record.UpdatedAt) > maxAge {
			toDelete = append(toDelete, id)
		}
		return true
	})

	for _, id := range toDelete {
		if record, exists := mm.migrations.Load(id); exists {
			mm.playerIndex.Delete(record.PlayerID)
			mm.migrations.Delete(id)
		}
	}

	if len(toDelete) > 0 {
		zLog.Debug("Cleaned up migration records", zap.Int("count", len(toDelete)))
	}
}

func (mm *MigrationManager) checkTimeouts() {
	now := time.Now()

	mm.migrations.Range(func(id uint64, record *MigrationRecord) bool {
		if record.State.IsTerminal() {
			return true
		}

		if now.Sub(record.UpdatedAt) > record.Timeout {
			record.State = MigrationStateTimedOut
			record.UpdatedAt = now
			mm.playerIndex.Delete(record.PlayerID)

			zLog.Warn("Migration timed out",
				zap.Uint64("migration_id", id),
				zap.Int64("player_id", record.PlayerID),
				zap.Duration("timeout", record.Timeout))

			if mm.callback != nil {
				mm.callback.OnMigrationRollback(record)
			}
		}
		return true
	})
}

func (mm *MigrationManager) ActiveCount() int {
	count := 0
	mm.migrations.Range(func(id uint64, record *MigrationRecord) bool {
		if !record.State.IsTerminal() {
			count++
		}
		return true
	})
	return count
}

func (mm *MigrationManager) TotalCount() int {
	return int(mm.migrations.Len())
}
