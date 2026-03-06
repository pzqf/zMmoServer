package game

import (
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"github.com/pzqf/zMmoServer/GameServer/service"
	"github.com/pzqf/zMmoShared/server"
)

// BaseServer 游戏服基础服务器
type BaseServer struct {
	*server.BaseServer
	Config           *config.Config
	TCPService       *service.TCPService
	ConnectionManager *connection.ConnectionManager
	SessionManager   *session.SessionManager
}

// NewBaseServer 创建游戏服基础服务器
func NewBaseServer() *BaseServer {
	baseServer := server.NewBaseServer(
		server.ServerTypeGame,
		"game-1",
		"Game Server",
		"1.0.0",
	)

	server := &BaseServer{
		BaseServer: baseServer,
	}

	// 设置生命周期钩子
	server.onBeforeStartFunc = server.onBeforeStart
	server.onAfterStartFunc = server.onAfterStart
	server.onBeforeStopFunc = server.onBeforeStop

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

	// 初始化会话管理器
	sessionManager := session.NewSessionManager()
	s.SessionManager = sessionManager

	// 初始化TCP服务
	tcpService, err := service.NewTCPService(cfg, connManager, sessionManager)
	if err != nil {
		return err
	}
	s.TCPService = tcpService

	// 注册组件
	s.RegisterComponent("Config", cfg)
	s.RegisterComponent("TCPService", tcpService)
	s.RegisterComponent("ConnectionManager", connManager)
	s.RegisterComponent("SessionManager", sessionManager)

	return nil
}

// onAfterStart 启动后的工作
func (s *BaseServer) onAfterStart() error {
	// 启动TCP服务
	if err := s.TCPService.Start(); err != nil {
		return err
	}

	return nil
}

// onBeforeStop 停止前的工作
func (s *BaseServer) onBeforeStop() {
	// 停止TCP服务
	if s.TCPService != nil {
		s.TCPService.Stop()
	}
}
