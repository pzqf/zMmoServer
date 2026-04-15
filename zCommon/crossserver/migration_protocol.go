package crossserver

const (
	MsgMigrationRequest       uint32 = 30001
	MsgMigrationPrepare       uint32 = 30002
	MsgMigrationData          uint32 = 30003
	MsgMigrationCommit        uint32 = 30004
	MsgMigrationRollback      uint32 = 30005
	MsgMigrationComplete      uint32 = 30006
	MsgMigrationHeartbeat     uint32 = 30007
	MsgMigrationQueryStatus   uint32 = 30008
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

type MigrationRequestPayload struct {
	MigrationID   uint64
	PlayerID      int64
	AccountID     int64
	PlayerName    string
	SourceServer  int32
	SourceService uint8
	TargetServer  int32
	TargetService uint8
	TargetMapID   int32
	MigrationType MigrationType
	Reason        string
}

type MigrationPreparePayload struct {
	MigrationID uint64
	PlayerID    int64
	Accepted    bool
	Reason      string
}

type MigrationDataPayload struct {
	MigrationID  uint64
	PlayerID     int64
	PlayerData   []byte
	MapData      []byte
	Checksum     uint32
	DataVersion  int64
}

type MigrationCommitPayload struct {
	MigrationID uint64
	PlayerID    int64
	Success     bool
	Reason      string
}

type MigrationRollbackPayload struct {
	MigrationID uint64
	PlayerID    int64
	Reason      string
}

type MigrationCompletePayload struct {
	MigrationID uint64
	PlayerID    int64
	NewServerID int32
}

type MigrationHeartbeatPayload struct {
	MigrationID uint64
	State       MigrationState
	Timestamp   int64
}

type MigrationStatusPayload struct {
	MigrationID   uint64
	PlayerID      int64
	State         MigrationState
	SourceServer  int32
	TargetServer  int32
	MigrationType MigrationType
	CreatedAt     int64
	UpdatedAt     int64
}
