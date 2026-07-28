package crossserver

import "github.com/pzqf/zCommon/protocol"

// 跨服迁移消息 ID —— 取自 internal.proto 的 InternalMsgId 610 段，与其它跨服消息同一号段，
// 满足「外层 zNet ProtoId == 内层 BaseMessage.MsgId == InternalMsgId」的三重一致（见 docs/协议契约.md）。
// 响应不单独分号：CrossTransport 的响应复用请求 msgID，方向由 Envelope.MessageType 区分。
//
// 历史：曾是 30001-30008 的私有号段，且经裸 Wrap(meta, json) 发送、msgID 根本没上线，
// 收侧无法路由，控制面从没跑通。2026-07-28 归并到 InternalMsgId 并走共用 codec。
const (
	MsgMigrationRequest     = uint32(protocol.InternalMsgId_MSG_INTERNAL_MIGRATION_REQUEST)
	MsgMigrationData        = uint32(protocol.InternalMsgId_MSG_INTERNAL_MIGRATION_DATA)
	MsgMigrationRollback    = uint32(protocol.InternalMsgId_MSG_INTERNAL_MIGRATION_ROLLBACK)
	MsgMigrationComplete    = uint32(protocol.InternalMsgId_MSG_INTERNAL_MIGRATION_COMPLETE)
	MsgMigrationHeartbeat   = uint32(protocol.InternalMsgId_MSG_INTERNAL_MIGRATION_HEARTBEAT)
	MsgMigrationQueryStatus = uint32(protocol.InternalMsgId_MSG_INTERNAL_MIGRATION_QUERY_STATUS)
)

type MigrationState uint8

const (
	MigrationStateNone        MigrationState = iota
	MigrationStateRequested
	MigrationStatePreparing
	MigrationStatePrepared
	MigrationStateTransferring
	MigrationStateTransferred
	MigrationStateCommitting
	MigrationStateCommitted
	MigrationStateCompleting
	MigrationStateCompleted
	MigrationStateRollingBack
	MigrationStateRolledBack
	MigrationStateFailed
	MigrationStateTimedOut
)

func (s MigrationState) String() string {
	switch s {
	case MigrationStateNone:
		return "none"
	case MigrationStateRequested:
		return "requested"
	case MigrationStatePreparing:
		return "preparing"
	case MigrationStatePrepared:
		return "prepared"
	case MigrationStateTransferring:
		return "transferring"
	case MigrationStateTransferred:
		return "transferred"
	case MigrationStateCommitting:
		return "committing"
	case MigrationStateCommitted:
		return "committed"
	case MigrationStateCompleting:
		return "completing"
	case MigrationStateCompleted:
		return "completed"
	case MigrationStateRollingBack:
		return "rolling_back"
	case MigrationStateRolledBack:
		return "rolled_back"
	case MigrationStateFailed:
		return "failed"
	case MigrationStateTimedOut:
		return "timed_out"
	default:
		return "unknown"
	}
}

func (s MigrationState) IsTerminal() bool {
	return s == MigrationStateCompleted || s == MigrationStateRolledBack || s == MigrationStateFailed || s == MigrationStateTimedOut
}

type MigrationType uint8

const (
	MigrationTypeGameToGame    MigrationType = iota + 1
	MigrationTypeGameToMap
	MigrationTypeMapToGame
	MigrationTypeMapToMap
)

func (t MigrationType) String() string {
	switch t {
	case MigrationTypeGameToGame:
		return "game_to_game"
	case MigrationTypeGameToMap:
		return "game_to_map"
	case MigrationTypeMapToGame:
		return "map_to_game"
	case MigrationTypeMapToMap:
		return "map_to_map"
	default:
		return "unknown"
	}
}

// 载荷类型统一用 internal.proto 生成的 protobuf 消息（protocol.MigrationRequest /
// MigrationPrepareAck / MigrationDataTransfer / MigrationCommitAck / MigrationRollbackNotify /
// MigrationCompleteNotify / MigrationHeartbeat / MigrationStatusQuery / MigrationStatus），
// 不再有一份平行的 JSON 结构体——契约「全程 protobuf、禁止 JSON」见 docs/协议契约.md。
