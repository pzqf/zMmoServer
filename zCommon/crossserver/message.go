package crossserver

// BaseMessage 基础消息结构
type BaseMessage struct {
	MsgID       uint32 `json:"msg_id"`        // 消息ID（全局唯一）
	SessionID   uint64 `json:"session_id"`    // 会话ID（仅网关<->游戏服务器）
	PlayerID    uint64 `json:"player_id"`     // 玩家ID（全局唯一）
	ServerID    uint32 `json:"server_id"`     // 服务器ID（发送方）
	Timestamp   uint64 `json:"timestamp"`     // 时间戳
	Data        []byte `json:"data"`          // 消息数据
	MapID       uint32 `json:"map_id"`        // 地图ID（可选，用于路由）
	MapServerID uint32 `json:"map_server_id"` // 地图服务器ID（可选，用于路由）
}

// CrossServerMessage 跨服务器消息包装
type CrossServerMessage struct {
	TraceID      uint64      `json:"trace_id"`       // 跟踪ID
	FromService  uint8       `json:"from_service"`   // 来源服务类型
	ToService    uint8       `json:"to_service"`     // 目标服务类型
	FromServerID uint32      `json:"from_server_id"` // 来源服务器ID
	ToServerID   uint32      `json:"to_server_id"`   // 目标服务器ID
	Message      BaseMessage `json:"message"`        // 原始消息
}


