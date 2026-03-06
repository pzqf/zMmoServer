package maps

import (
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoServer/MapServer/server"
)

// BaseServer 地图服基础服务器
type BaseServer struct {
	*sharedserver.BaseServer
	Config            *config.Config
	MapServer         *server.MapServer
	ConnectionManager *connection.ConnectionManager
}

// NewBaseServer 创建地图服基础服务器
func NewBaseServer() *BaseServer {
	baseServer := sharedserver.NewBaseServer(
		sharedserver.ServerTypeMap,
		"map-1",
		"Map Server",
		"1.0.0",
	)

	server := &BaseServer{
		BaseServer: baseServer,
	}

	// 设置生命周期钩子
	server.BaseServer.onBeforeStartFunc = server.onBeforeStart
	server.BaseServer.onAfterStartFunc = server.onAfterStart
	server.BaseServer.onBeforeStopFunc = server.onBeforeStop

	return server
}

// onBeforeStart 启动前的准备工作
func (s *BaseServer) onBeforeStart() error {
	// 加载配置
	cfg, err := config.LoadConfig("config.ini")
	if err != nil {
		return err
	}
	s.Config = cfg

	// 初始化连接管理器
	connManager := connection.NewConnectionManager()
	s.ConnectionManager = connManager

	// 初始化地图服务器
	mapServer, err := server.NewMapServer(cfg, connManager)
	if err != nil {
		return err
	}
	s.MapServer = mapServer

	// 注册组件
	s.RegisterComponent("Config", cfg)
	s.RegisterComponent("MapServer", mapServer)
	s.RegisterComponent("ConnectionManager", connManager)

	return nil
}

// onAfterStart 启动后的工作
func (s *BaseServer) onAfterStart() error {
	// 启动地图服务器
	if err := s.MapServer.Start(); err != nil {
		return err
	}

	return nil
}

// onBeforeStop 停止前的工作
func (s *BaseServer) onBeforeStop() {
	// 停止地图服务器
	if s.MapServer != nil {
		s.MapServer.Stop()
	}
}
