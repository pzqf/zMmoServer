package crossserver

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/pzqf/zCommon/protocol"
	consistency "github.com/pzqf/zEngine/zConsistency"
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
	ErrMigrationNoTransport   = errors.New("migration manager has no transport")
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

// MigrationManager 跨服玩家迁移的两阶段协调器。源服与目标服各跑一个实例，经 CrossTransport
// 收发同一组消息：源服调 ExecuteMigration 驱动整条链，目标服由注册的 handler 应答。
//
// 控制面时序（2026-07-28 修好传输层后第一次真正跑通）：
//
//	源服                                       目标服
//	 ExecuteMigration
//	   ├─ Call(MIGRATION_REQUEST) ───────────▶ handleMigrationRequest → OnMigrationPrepare
//	   ◀────────────── MigrationPrepareAck ───┘        （拒绝则源服 fail，玩家留在源服）
//	   ├─ SerializePlayer + crc32
//	   ├─ Call(MIGRATION_DATA) ──────────────▶ handleMigrationData → inbox 去重 + 校验和
//	   │                                              + OnMigrationDataReceived + DeserializePlayer
//	   ◀────────────── MigrationCommitAck ────┘        （失败则源服 Notify(ROLLBACK)）
//	   ├─ 本地提交（txManager + OnMigrationCommit）
//	   └─ Notify(MIGRATION_COMPLETE) ────────▶ handleMigrationComplete
//
// 铁律：目标落地成功之前，源服不得删玩家；目标侧按 migrationID 幂等（重投不重复落地）。
type MigrationManager struct {
	migrations  *zMap.TypedMap[uint64, *MigrationRecord]
	playerIndex *zMap.TypedMap[int64, uint64]
	config      MigrationConfig
	nextID      atomic.Uint64
	serializer  PlayerDataSerializer
	callback    MigrationCallback
	transport   *CrossTransport
	txManager   *consistency.TransactionManager
	outbox      consistency.OutboxStore
	inbox       consistency.InboxStore
	// mu 保护 MigrationRecord 的字段读写。记录会被三方并发触碰：驱动 goroutine（ExecuteMigration）、
	// 入站 handler goroutine、超时清理 goroutine（checkTimeouts）——无锁则是真 data race。
	// 锁只覆盖字段读写，不覆盖回调/网络调用。
	mu sync.RWMutex
}

func NewMigrationManager(
	config MigrationConfig,
	serializer PlayerDataSerializer,
	callback MigrationCallback,
	transport *CrossTransport,
	txManager *consistency.TransactionManager,
	outbox consistency.OutboxStore,
	inbox consistency.InboxStore,
) *MigrationManager {
	mm := &MigrationManager{
		migrations:  zMap.NewTypedMap[uint64, *MigrationRecord](),
		playerIndex: zMap.NewTypedMap[int64, uint64](),
		config:      config,
		serializer:  serializer,
		callback:    callback,
		transport:   transport,
		txManager:   txManager,
		outbox:      outbox,
		inbox:       inbox,
	}

	mm.registerHandlers()

	return mm
}

// registerHandlers 注册目标服侧的入站 handler。
// 用 RegisterHandlerAllServices：迁移可能由 Game/Map/Global 任一侧发起，逐个注册容易漏。
func (mm *MigrationManager) registerHandlers() {
	if mm.transport == nil {
		return
	}
	router := mm.transport.Router()

	router.RegisterHandlerAllServices(MsgMigrationRequest, mm.handleMigrationRequest)
	router.RegisterHandlerAllServices(MsgMigrationData, mm.handleMigrationData)
	router.RegisterHandlerAllServices(MsgMigrationRollback, mm.handleMigrationRollback)
	router.RegisterHandlerAllServices(MsgMigrationComplete, mm.handleMigrationComplete)
	router.RegisterHandlerAllServices(MsgMigrationHeartbeat, mm.handleMigrationHeartbeat)
	router.RegisterHandlerAllServices(MsgMigrationQueryStatus, mm.handleMigrationQueryStatus)
}

func (mm *MigrationManager) setState(record *MigrationRecord, state MigrationState) {
	mm.mu.Lock()
	record.State = state
	record.UpdatedAt = time.Now()
	mm.mu.Unlock()
}

// State 读取记录状态（加锁，供外部/测试安全观察）。
func (mm *MigrationManager) State(migrationID uint64) (MigrationState, bool) {
	record, exists := mm.migrations.Load(migrationID)
	if !exists {
		return MigrationStateNone, false
	}
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return record.State, true
}

// ExecuteMigration 源服侧驱动一次完整迁移，成功返回处于 Completed 的记录。
// 阻塞直到成功/被拒/失败/ctx 结束——调用方自行决定放哪个 goroutine 上跑（勿在地图 actor 上直接调）。
func (mm *MigrationManager) ExecuteMigration(
	ctx context.Context,
	playerID int64,
	accountID int64,
	playerName string,
	targetServer int32,
	targetService uint8,
	targetMapID int32,
	migrationType MigrationType,
	reason string,
) (*MigrationRecord, error) {
	if mm.transport == nil {
		return nil, ErrMigrationNoTransport
	}
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
		SourceServer:  mm.transport.LocalServerID(),
		SourceService: mm.transport.LocalService(),
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

	// 阶段一：问目标服接不接。
	if err := mm.requestAcceptance(ctx, record); err != nil {
		return nil, err
	}

	// 阶段二：序列化 + 投数据，等目标确认已落地。
	if err := mm.transferData(ctx, record); err != nil {
		return nil, err
	}

	// 阶段三：源服本地提交 + 通知目标完成。
	if err := mm.commitLocal(ctx, record); err != nil {
		return nil, err
	}

	zLog.Info("Migration completed",
		zap.Uint64("migration_id", migrationID),
		zap.Int64("player_id", record.PlayerID),
		zap.Int32("target_server", record.TargetServer))

	return record, nil
}

// requestAcceptance 阶段一：发 MIGRATION_REQUEST，等目标的 MigrationPrepareAck。
func (mm *MigrationManager) requestAcceptance(ctx context.Context, record *MigrationRecord) error {
	req := &protocol.MigrationRequest{
		MigrationId:   record.ID,
		PlayerId:      record.PlayerID,
		AccountId:     record.AccountID,
		PlayerName:    record.PlayerName,
		SourceServer:  record.SourceServer,
		SourceService: uint32(record.SourceService),
		TargetServer:  record.TargetServer,
		TargetService: uint32(record.TargetService),
		TargetMapId:   record.TargetMapID,
		MigrationType: uint32(record.Type),
		Reason:        record.Reason,
	}
	payload, err := proto.Marshal(req)
	if err != nil {
		mm.failMigration(record.ID, fmt.Sprintf("marshal request: %v", err))
		return fmt.Errorf("marshal migration request: %w", err)
	}

	respBytes, err := mm.transport.Call(ctx, record.TargetServer, record.TargetService,
		MsgMigrationRequest, record.PlayerID, payload)
	if err != nil {
		mm.failMigration(record.ID, fmt.Sprintf("send request failed: %v", err))
		return fmt.Errorf("send migration request: %w", err)
	}

	var ack protocol.MigrationPrepareAck
	if err := proto.Unmarshal(respBytes, &ack); err != nil {
		mm.failMigration(record.ID, fmt.Sprintf("unmarshal prepare ack: %v", err))
		return fmt.Errorf("unmarshal prepare ack: %w", err)
	}
	if !ack.Accepted {
		mm.failMigration(record.ID, fmt.Sprintf("target rejected: %s", ack.Reason))
		return fmt.Errorf("%w: %s", ErrMigrationRejected, ack.Reason)
	}
	return nil
}

// transferData 阶段二：序列化玩家态、经 outbox 记账后投给目标，等目标的 MigrationCommitAck。
// 目标未确认落地前绝不进入提交——玩家仍属源服。
func (mm *MigrationManager) transferData(ctx context.Context, record *MigrationRecord) error {
	mm.setState(record, MigrationStatePreparing)

	if mm.serializer != nil {
		playerData, mapData, err := mm.serializer.SerializePlayer(record.PlayerID)
		if err != nil {
			mm.failMigration(record.ID, fmt.Sprintf("serialize failed: %v", err))
			return fmt.Errorf("serialize player data: %w", err)
		}
		mm.mu.Lock()
		record.PlayerData = playerData
		record.MapData = mapData
		record.DataChecksum = crc32.ChecksumIEEE(playerData)
		record.RollbackData = playerData
		mm.mu.Unlock()
	}
	mm.setState(record, MigrationStatePrepared)

	transfer := &protocol.MigrationDataTransfer{
		MigrationId: record.ID,
		PlayerId:    record.PlayerID,
		PlayerData:  record.PlayerData,
		MapData:     record.MapData,
		Checksum:    record.DataChecksum,
		DataVersion: record.DataVersion,
	}
	payload, err := proto.Marshal(transfer)
	if err != nil {
		mm.failMigration(record.ID, fmt.Sprintf("marshal data: %v", err))
		return fmt.Errorf("marshal migration data: %w", err)
	}

	mm.setState(record, MigrationStateTransferring)

	if mm.outbox != nil {
		mm.outbox.Add(consistency.OutboxMessage{
			RequestID:      record.ID,
			TargetServerID: ServerKey(record.TargetServer),
			ProtoID:        int32(MsgMigrationData),
			Payload:        payload,
			CreatedAt:      time.Now(),
		})
	}

	respBytes, err := mm.transport.Call(ctx, record.TargetServer, record.TargetService,
		MsgMigrationData, record.PlayerID, payload)
	if err != nil {
		if mm.outbox != nil {
			mm.outbox.MarkAttempt(record.ID, err)
		}
		mm.notifyRollback(record, fmt.Sprintf("send data failed: %v", err))
		mm.failMigration(record.ID, fmt.Sprintf("send data failed: %v", err))
		return fmt.Errorf("send migration data: %w", err)
	}
	if mm.outbox != nil {
		mm.outbox.MarkSent(record.ID)
	}

	var ack protocol.MigrationCommitAck
	if err := proto.Unmarshal(respBytes, &ack); err != nil {
		mm.notifyRollback(record, "unmarshal commit ack failed")
		mm.failMigration(record.ID, fmt.Sprintf("unmarshal commit ack: %v", err))
		return fmt.Errorf("unmarshal commit ack: %w", err)
	}
	if !ack.Success {
		mm.notifyRollback(record, ack.Reason)
		mm.failMigration(record.ID, fmt.Sprintf("target refused data: %s", ack.Reason))
		return fmt.Errorf("%w: %s", ErrMigrationDataInvalid, ack.Reason)
	}

	mm.setState(record, MigrationStateTransferred)
	zLog.Info("Migration data accepted by target",
		zap.Uint64("migration_id", record.ID),
		zap.Int("data_size", len(record.PlayerData)))
	return nil
}

// commitLocal 阶段三：源服本地提交（分布式事务 + 业务回调），然后通知目标完成。
func (mm *MigrationManager) commitLocal(ctx context.Context, record *MigrationRecord) error {
	mm.mu.RLock()
	state := record.State
	mm.mu.RUnlock()
	if state != MigrationStateTransferred {
		return fmt.Errorf("%w: cannot commit from state %s", ErrMigrationInvalidState, state)
	}

	mm.setState(record, MigrationStateCommitting)

	if mm.txManager != nil {
		participants := []string{
			ServerKey(record.SourceServer),
			ServerKey(record.TargetServer),
		}
		if err := mm.txManager.Execute(ctx, fmt.Sprintf("migration_%d", record.ID), participants, record.Timeout, record); err != nil {
			mm.RollbackMigration(record.ID, fmt.Sprintf("transaction failed: %v", err))
			return fmt.Errorf("migration transaction: %w", err)
		}
	}

	if mm.callback != nil {
		if err := mm.callback.OnMigrationCommit(record); err != nil {
			mm.RollbackMigration(record.ID, fmt.Sprintf("commit callback failed: %v", err))
			return fmt.Errorf("commit callback: %w", err)
		}
	}

	mm.setState(record, MigrationStateCommitted)

	complete := &protocol.MigrationCompleteNotify{
		MigrationId: record.ID,
		PlayerId:    record.PlayerID,
		NewServerId: record.TargetServer,
	}
	if payload, err := proto.Marshal(complete); err == nil {
		if err := mm.transport.Notify(record.TargetServer, record.TargetService,
			MsgMigrationComplete, record.PlayerID, payload); err != nil {
			// 目标已落地并提交，完成通知只是收尾；投递失败不回滚，记日志由重试/对账兜底。
			zLog.Warn("Migration complete notify failed",
				zap.Uint64("migration_id", record.ID), zap.Error(err))
		}
	}

	mm.setState(record, MigrationStateCompleting)
	if mm.callback != nil {
		if err := mm.callback.OnMigrationComplete(record); err != nil {
			zLog.Warn("OnMigrationComplete failed",
				zap.Uint64("migration_id", record.ID), zap.Error(err))
		}
	}

	mm.setState(record, MigrationStateCompleted)
	mm.playerIndex.Delete(record.PlayerID)
	return nil
}

// notifyRollback 告诉目标服撤销本次迁移（通知，不等应答）。
func (mm *MigrationManager) notifyRollback(record *MigrationRecord, reason string) {
	if mm.transport == nil {
		return
	}
	payload, err := proto.Marshal(&protocol.MigrationRollbackNotify{
		MigrationId: record.ID,
		PlayerId:    record.PlayerID,
		Reason:      reason,
	})
	if err != nil {
		return
	}
	if err := mm.transport.Notify(record.TargetServer, record.TargetService,
		MsgMigrationRollback, record.PlayerID, payload); err != nil {
		zLog.Warn("Migration rollback notify failed",
			zap.Uint64("migration_id", record.ID), zap.Error(err))
	}
}

// RollbackMigration 撤销一次迁移：跑本地回滚回调、中止事务、通知对端。
func (mm *MigrationManager) RollbackMigration(migrationID uint64, reason string) {
	record, exists := mm.migrations.Load(migrationID)
	if !exists {
		return
	}

	mm.setState(record, MigrationStateRollingBack)

	if mm.callback != nil {
		if err := mm.callback.OnMigrationRollback(record); err != nil {
			zLog.Warn("OnMigrationRollback failed",
				zap.Uint64("migration_id", migrationID), zap.Error(err))
		}
	}

	if mm.txManager != nil {
		mm.txManager.Abort(migrationID)
	}

	mm.notifyRollback(record, reason)

	mm.setState(record, MigrationStateRolledBack)
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

	mm.setState(record, MigrationStateFailed)
	mm.playerIndex.Delete(record.PlayerID)

	zLog.Error("Migration failed",
		zap.Uint64("migration_id", migrationID),
		zap.Int64("player_id", record.PlayerID),
		zap.String("reason", reason))
}

// ---------------------------------------------------------------------------
// 目标服侧入站 handler。返回值即回投给源服的响应载荷（由 CrossTransport 打包回投）。
// ---------------------------------------------------------------------------

// handleMigrationRequest 目标服收到"请接纳这个玩家"，问业务层要不要接，应答 MigrationPrepareAck。
func (mm *MigrationManager) handleMigrationRequest(meta Meta, payload []byte) ([]byte, error) {
	var req protocol.MigrationRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("unmarshal migration request: %w", err)
	}

	zLog.Info("Migration request received",
		zap.Uint64("migration_id", req.MigrationId),
		zap.Int64("player_id", req.PlayerId),
		zap.Int32("source_server", req.SourceServer))

	accepted := true
	reason := ""

	if mm.callback != nil {
		record := &MigrationRecord{
			ID:            req.MigrationId,
			PlayerID:      req.PlayerId,
			AccountID:     req.AccountId,
			PlayerName:    req.PlayerName,
			SourceServer:  req.SourceServer,
			SourceService: uint8(req.SourceService),
			TargetServer:  req.TargetServer,
			TargetService: uint8(req.TargetService),
			TargetMapID:   req.TargetMapId,
			Type:          MigrationType(req.MigrationType),
			Reason:        req.Reason,
		}
		accepted, reason = mm.callback.OnMigrationPrepare(record)
	}

	return proto.Marshal(&protocol.MigrationPrepareAck{
		MigrationId: req.MigrationId,
		PlayerId:    req.PlayerId,
		Accepted:    accepted,
		Reason:      reason,
	})
}

// handleMigrationData 目标服收到玩家数据：按 migrationID 幂等去重 → 校验和 → 业务回调 → 反序列化落地，
// 应答 MigrationCommitAck。任何一步失败都必须 Release 掉 inbox 占用（INF-2），否则重投会命中
// "重复"分支误返成功 → 源服删玩家而目标未落地 → 玩家丢失。
func (mm *MigrationManager) handleMigrationData(meta Meta, payload []byte) ([]byte, error) {
	var transfer protocol.MigrationDataTransfer
	if err := proto.Unmarshal(payload, &transfer); err != nil {
		return nil, fmt.Errorf("unmarshal migration data: %w", err)
	}

	ack := func(success bool, reason string) ([]byte, error) {
		return proto.Marshal(&protocol.MigrationCommitAck{
			MigrationId: transfer.MigrationId,
			PlayerId:    transfer.PlayerId,
			Success:     success,
			Reason:      reason,
		})
	}
	// ackFail 失败应答：必须撤销 inbox 占用，否则重投会命中"重复"分支误返成功。
	ackFail := func(reason string) ([]byte, error) {
		if mm.inbox != nil {
			mm.inbox.Release(transfer.MigrationId)
		}
		return ack(false, reason)
	}

	if mm.inbox != nil {
		if !mm.inbox.TryAccept(transfer.MigrationId) {
			// 占用已被别人拿着：可能是「上一次已落地」，也可能是「另一次投递还在处理中」。
			// 只有确认落地过（记录已存）才能回成功——否则回失败让源服稍后重试。
			// 若在途也回成功，源服会据此提交并删玩家，而目标那次处理可能随后失败 → 玩家丢失。
			// 这里**不得** Release：那会撤销在途那次的占用。
			if _, applied := mm.migrations.Load(transfer.MigrationId); applied {
				zLog.Warn("Duplicate migration data received (already applied)",
					zap.Uint64("migration_id", transfer.MigrationId))
				return ack(true, "")
			}
			zLog.Warn("Migration data still being processed, asking source to retry",
				zap.Uint64("migration_id", transfer.MigrationId))
			return ack(false, "migration already in progress on target, retry later")
		}
	}

	checksum := crc32.ChecksumIEEE(transfer.PlayerData)
	if checksum != transfer.Checksum {
		return ackFail(fmt.Sprintf("checksum mismatch: expected %d, got %d", transfer.Checksum, checksum))
	}

	record := &MigrationRecord{
		ID:           transfer.MigrationId,
		PlayerID:     transfer.PlayerId,
		PlayerData:   transfer.PlayerData,
		MapData:      transfer.MapData,
		DataChecksum: transfer.Checksum,
		DataVersion:  transfer.DataVersion,
		State:        MigrationStateTransferred,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Timeout:      mm.config.DefaultTimeout,
	}

	if mm.callback != nil {
		if err := mm.callback.OnMigrationDataReceived(record); err != nil {
			return ackFail(err.Error())
		}
	}

	if mm.serializer != nil {
		if err := mm.serializer.DeserializePlayer(transfer.PlayerData, transfer.MapData); err != nil {
			return ackFail(fmt.Sprintf("deserialize failed: %v", err))
		}
	}

	mm.migrations.Store(transfer.MigrationId, record)

	if mm.inbox != nil {
		mm.inbox.Ack(transfer.MigrationId)
	}

	zLog.Info("Migration data received and deserialized",
		zap.Uint64("migration_id", transfer.MigrationId),
		zap.Int64("player_id", transfer.PlayerId))

	return proto.Marshal(&protocol.MigrationCommitAck{
		MigrationId: transfer.MigrationId,
		PlayerId:    transfer.PlayerId,
		Success:     true,
	})
}

// handleMigrationRollback 目标服收到"撤销"：丢弃已落地的半成品（业务回调负责真正撤销）。
func (mm *MigrationManager) handleMigrationRollback(meta Meta, payload []byte) ([]byte, error) {
	var notify protocol.MigrationRollbackNotify
	if err := proto.Unmarshal(payload, &notify); err != nil {
		return nil, fmt.Errorf("unmarshal rollback: %w", err)
	}

	record, exists := mm.migrations.Load(notify.MigrationId)
	if !exists {
		// 目标可能压根没收到数据（源服在阶段一/二失败即撤销），无记录属正常。
		zLog.Info("Migration rollback for unknown migration (nothing to undo)",
			zap.Uint64("migration_id", notify.MigrationId), zap.String("reason", notify.Reason))
		return nil, nil
	}

	mm.setState(record, MigrationStateRollingBack)
	if mm.callback != nil {
		if err := mm.callback.OnMigrationRollback(record); err != nil {
			zLog.Warn("OnMigrationRollback failed",
				zap.Uint64("migration_id", notify.MigrationId), zap.Error(err))
		}
	}
	// 撤销后允许同 migrationID 重投（源服重试整条链时不被误判为重复）。
	if mm.inbox != nil {
		mm.inbox.Release(notify.MigrationId)
	}
	mm.setState(record, MigrationStateRolledBack)
	mm.playerIndex.Delete(record.PlayerID)

	zLog.Warn("Migration rolled back on target",
		zap.Uint64("migration_id", notify.MigrationId),
		zap.String("reason", notify.Reason))
	return nil, nil
}

// handleMigrationComplete 目标服收到"源服已提交"：迁移收尾，玩家从此归目标服。
func (mm *MigrationManager) handleMigrationComplete(meta Meta, payload []byte) ([]byte, error) {
	var notify protocol.MigrationCompleteNotify
	if err := proto.Unmarshal(payload, &notify); err != nil {
		return nil, fmt.Errorf("unmarshal complete: %w", err)
	}

	record, exists := mm.migrations.Load(notify.MigrationId)
	if !exists {
		return nil, ErrMigrationNotFound
	}

	if mm.callback != nil {
		if err := mm.callback.OnMigrationComplete(record); err != nil {
			zLog.Warn("OnMigrationComplete failed",
				zap.Uint64("migration_id", notify.MigrationId), zap.Error(err))
		}
	}

	mm.setState(record, MigrationStateCompleted)
	mm.playerIndex.Delete(record.PlayerID)

	zLog.Info("Migration complete notification received",
		zap.Uint64("migration_id", notify.MigrationId),
		zap.Int64("player_id", notify.PlayerId))

	return nil, nil
}

func (mm *MigrationManager) handleMigrationHeartbeat(meta Meta, payload []byte) ([]byte, error) {
	var hb protocol.MigrationHeartbeat
	if err := proto.Unmarshal(payload, &hb); err != nil {
		return nil, fmt.Errorf("unmarshal heartbeat: %w", err)
	}

	if _, exists := mm.migrations.Load(hb.MigrationId); !exists {
		return nil, ErrMigrationNotFound
	}

	zLog.Debug("Migration heartbeat received",
		zap.Uint64("migration_id", hb.MigrationId),
		zap.String("state", MigrationState(hb.State).String()))

	return nil, nil
}

func (mm *MigrationManager) handleMigrationQueryStatus(meta Meta, payload []byte) ([]byte, error) {
	var query protocol.MigrationStatusQuery
	if err := proto.Unmarshal(payload, &query); err != nil {
		return nil, fmt.Errorf("unmarshal query: %w", err)
	}

	record, exists := mm.migrations.Load(query.MigrationId)
	if !exists {
		return nil, ErrMigrationNotFound
	}

	mm.mu.RLock()
	status := &protocol.MigrationStatus{
		MigrationId:   record.ID,
		PlayerId:      record.PlayerID,
		State:         uint32(record.State),
		SourceServer:  record.SourceServer,
		TargetServer:  record.TargetServer,
		MigrationType: uint32(record.Type),
		CreatedAt:     record.CreatedAt.Unix(),
		UpdatedAt:     record.UpdatedAt.Unix(),
	}
	mm.mu.RUnlock()

	return proto.Marshal(status)
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
	mm.mu.RLock()
	defer mm.mu.RUnlock()
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

	mm.mu.RLock()
	mm.migrations.Range(func(id uint64, record *MigrationRecord) bool {
		if record.State.IsTerminal() && now.Sub(record.UpdatedAt) > maxAge {
			toDelete = append(toDelete, id)
		}
		return true
	})
	mm.mu.RUnlock()

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

	var timedOut []*MigrationRecord
	mm.mu.Lock()
	mm.migrations.Range(func(id uint64, record *MigrationRecord) bool {
		if record.State.IsTerminal() {
			return true
		}
		if now.Sub(record.UpdatedAt) > record.Timeout {
			record.State = MigrationStateTimedOut
			record.UpdatedAt = now
			timedOut = append(timedOut, record)
		}
		return true
	})
	mm.mu.Unlock()

	for _, record := range timedOut {
		mm.playerIndex.Delete(record.PlayerID)
		zLog.Warn("Migration timed out",
			zap.Uint64("migration_id", record.ID),
			zap.Int64("player_id", record.PlayerID),
			zap.Duration("timeout", record.Timeout))
		if mm.callback != nil {
			if err := mm.callback.OnMigrationRollback(record); err != nil {
				zLog.Warn("OnMigrationRollback failed on timeout",
					zap.Uint64("migration_id", record.ID), zap.Error(err))
			}
		}
	}
}

func (mm *MigrationManager) ActiveCount() int {
	count := 0
	mm.mu.RLock()
	defer mm.mu.RUnlock()
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
