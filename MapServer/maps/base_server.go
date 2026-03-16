package maps

import (
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/connection"
)

// BaseServer 地图服基础服务器
type BaseServer struct {
	Config            *config.Config
	ConnectionManager *connection.ConnectionManager
	MapManager        *MapManager
}

// NewBaseServer 创建地图服基础服务器
func NewBaseServer() *BaseServer {
	s := &BaseServer{}

	return s
}

// Start 启动服务器
func (s *BaseServer) Start() error {
	// 加载配置
	cfg, err := config.LoadConfig("config.ini")
	if err != nil {
		return err
	}
	s.Config = cfg

	// 初始化连接管理器
	connManager := connection.NewConnectionManager(cfg)
	s.ConnectionManager = connManager

	// 初始化地图管理器
	mapManager := NewMapManager()
	s.MapManager = mapManager

	// 启动连接管理器
	if err := s.ConnectionManager.ConnectToGameServer("game1", cfg.GameServer.GameServerAddr); err != nil {
		// 记录警告，继续启动
	}

	// 启动地图管理器
	if err := s.MapManager.Start(); err != nil {
		return err
	}

	return nil
}

// Stop 停止服务器
func (s *BaseServer) Stop() {
	// 停止连接管理器
	if s.ConnectionManager != nil {
		s.ConnectionManager.DisconnectFromGameServer("game1")
	}

	// 停止地图管理器
	if s.MapManager != nil {
		s.MapManager.Stop()
	}
}
