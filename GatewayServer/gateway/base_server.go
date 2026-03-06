package gateway

import (
	"github.com/pzqf/zMmoServer/GatewayServer/auth"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/monitor"
	"github.com/pzqf/zMmoServer/GatewayServer/proxy"
	"github.com/pzqf/zMmoServer/GatewayServer/service"
	"github.com/pzqf/zMmoShared/server"
)

// BaseServer 网关服基础服务器
type BaseServer struct {
	*server.BaseServer
	Config            *config.Config
	TCPService        *service.TCPService
	ConnectionManager *connection.ConnectionManager
	SecurityManager   *auth.SecurityManager
	AntiCheatManager  *auth.AntiCheatManager
	AuthHandler       *auth.AuthHandler
	GameServerProxy   proxy.GameServerProxy
	HeartbeatReporter *monitor.HeartbeatReporter
}

// NewBaseServer 创建网关服基础服务器
func NewBaseServer() *BaseServer {
	baseServer := server.NewBaseServer(
		server.ServerTypeGateway,
		"gateway-1",
		"Gateway Server",
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

	// 初始化安全管理器
	securityManager := auth.NewSecurityManager(cfg)
	s.SecurityManager = securityManager

	// 初始化防作弊管理器
	antiCheatManager := auth.NewAntiCheatManager()
	s.AntiCheatManager = antiCheatManager

	// 初始化认证处理器
	authHandler := auth.NewAuthHandler(securityManager, antiCheatManager)
	s.AuthHandler = authHandler

	// 初始化游戏服务器代理
	gameServerProxy, err := proxy.NewGameServerProxy(cfg, connManager)
	if err != nil {
		return err
	}
	s.GameServerProxy = gameServerProxy

	// 初始化心跳上报器
	heartbeatReporter := monitor.NewHeartbeatReporter(cfg, connManager)
	s.HeartbeatReporter = heartbeatReporter

	// 初始化TCP服务
	tcpService, err := service.NewTCPService(cfg, connManager, authHandler, gameServerProxy)
	if err != nil {
		return err
	}
	s.TCPService = tcpService

	// 注册组件
	s.RegisterComponent("Config", cfg)
	s.RegisterComponent("TCPService", tcpService)
	s.RegisterComponent("ConnectionManager", connManager)
	s.RegisterComponent("SecurityManager", securityManager)
	s.RegisterComponent("AntiCheatManager", antiCheatManager)
	s.RegisterComponent("AuthHandler", authHandler)
	s.RegisterComponent("GameServerProxy", gameServerProxy)
	s.RegisterComponent("HeartbeatReporter", heartbeatReporter)

	return nil
}

// onAfterStart 启动后的工作
func (s *BaseServer) onAfterStart() error {
	// 启动TCP服务
	if err := s.TCPService.Start(); err != nil {
		return err
	}

	// 启动游戏服务器代理
	if err := s.GameServerProxy.Start(s.GetContext()); err != nil {
		return err
	}

	// 启动心跳上报器
	if err := s.HeartbeatReporter.Start(); err != nil {
		return err
	}

	return nil
}

// onBeforeStop 停止前的工作
func (s *BaseServer) onBeforeStop() {
	// 停止心跳上报器
	if s.HeartbeatReporter != nil {
		s.HeartbeatReporter.Stop()
	}

	// 停止游戏服务器代理
	if s.GameServerProxy != nil {
		s.GameServerProxy.Stop(s.GetContext())
	}

	// 停止TCP服务
	if s.TCPService != nil {
		s.TCPService.Stop()
	}
}
