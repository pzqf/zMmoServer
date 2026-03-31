package discovery

// ServerStatus 服务器状态
type ServerStatus string

const (
	ServerStatusStarting     ServerStatus = "starting"
	ServerStatusInitializing ServerStatus = "initializing"
	ServerStatusReady        ServerStatus = "ready"
	ServerStatusHealthy      ServerStatus = "healthy"
	ServerStatusDraining     ServerStatus = "draining"
	ServerStatusMaintenance  ServerStatus = "maintenance"
	ServerStatusStopped      ServerStatus = "stopped"
)

// ServerInfo 服务器信息
type ServerInfo struct {
	ID            string
	ServiceType   string
	GroupID       string
	Status        ServerStatus
	Address       string
	Port          int
	Load          float64
	Players       int
	ReadyTime     int64
	LastHeartbeat int64
}

// ServerEvent 服务器事件
type ServerEvent struct {
	GroupID   string
	ServerID  string
	EventType string
	Status    ServerStatus
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
	SetStatus(serverID string, status ServerStatus) error
	GetStatus(serverID string) (ServerStatus, error)
	WatchStatus(serverID string) (<-chan *ServerStatus, error)
}

// ConnectionManager 连接管理接口
type ConnectionManager interface {
	Connect(address string, port int) error
	IsConnected(id string) bool
	GetConnection(id string) interface{}
	Close(id string) error
}
