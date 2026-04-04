package discovery

import "github.com/pzqf/zEngine/zServer"

// ServerInfo 服务器信息
type ServerInfo struct {
	ID            string
	ServiceType   string
	GroupID       string
	Status        zServer.ServerState
	Address       string
	Port          int
	Load          float64
	Players       int
	ReadyTime     int64
	LastHeartbeat int64
	MapIDs        []int32 // MapServer负责的地图ID列表
}

// ServerEvent 服务器事件
type ServerEvent struct {
	GroupID   string
	ServerID  string
	EventType string
	Status    zServer.ServerState
	Timestamp int64
	Data      interface{}
}

// Discovery 服务发现接口
type Discovery interface {
	Discover(serverType string, groupID string) ([]*ServerInfo, error)
	Watch(serverType string, groupID string) (<-chan *ServerEvent, error)
}

// StatusManager 状态管理接口
type StatusManager interface {
	SetStatus(serverID string, status zServer.ServerState) error
	GetStatus(serverID string) (zServer.ServerState, error)
	WatchStatus(serverID string) (<-chan zServer.ServerState, error)
}

// ConnectionManager 连接管理接口
type ConnectionManager interface {
	Connect(address string, port int) error
	IsConnected(id string) bool
	GetConnection(id string) interface{}
	Close(id string) error
}
