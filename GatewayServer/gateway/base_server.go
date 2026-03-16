package gateway

import (
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zMmoServer/GatewayServer/auth"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/monitor"
	"github.com/pzqf/zMmoServer/GatewayServer/protocol"
	"github.com/pzqf/zMmoServer/GatewayServer/proxy"
	"github.com/pzqf/zMmoServer/GatewayServer/service"
	"github.com/pzqf/zMmoServer/GatewayServer/token"
	"github.com/pzqf/zMmoServer/GatewayServer/version"
	"github.com/pzqf/zMmoShared/net/protolayer"
	"go.uber.org/zap"
)

// ServerType 网关服类型
const ServerTypeGateway zServer.ServerType = "gateway"

// BaseServer 网关服基础服务器
type BaseServer struct {
	*zServer.BaseServer
	Config            *config.Config
	TCPService        *service.TCPService
	ConnectionManager *connection.ConnectionManager
	SecurityManager   *auth.SecurityManager
	AntiCheatManager  *auth.AntiCheatManager
	AuthHandler       *auth.AuthHandler
	GameServerProxy   proxy.GameServerProxy
	HeartbeatReporter *monitor.HeartbeatReporter
	TokenManager      *token.TokenManager
	ProtocolHandler   *protocol.ProtocolHandler
	NetServer         *zNet.TcpServer
	CompressionCfg    *protolayer.CompressionConfig
}

// NewBaseServer 创建网关服基础服务器
func NewBaseServer() *BaseServer {
	// 先创建子类实例
	gs := &BaseServer{}

	// 创建基础服务器，传入子类作为 hooks
	baseServer := zServer.NewBaseServer(
		ServerTypeGateway,
		"gateway-1",
		"Gateway Server",
		version.Version,
		gs, // 传入自身作为 LifecycleHooks 实现
	)

	gs.BaseServer = baseServer
	return gs
}

// OnBeforeStart 启动前的准备工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnBeforeStart() error {
	cfg := s.Config
	if cfg == nil {
		return nil
	}

	// 初始化连接管理器
	connManager := connection.NewConnectionManager()
	s.ConnectionManager = connManager

	// 初始化安全管理器
	securityManager := auth.NewSecurityManager(cfg)
	s.SecurityManager = securityManager

	// 初始化防作弊管理器
	antiCheatManager := auth.NewAntiCheatManager(cfg)
	s.AntiCheatManager = antiCheatManager

	// 初始化令牌管理器
	tokenManager := token.NewTokenManager(cfg)
	s.TokenManager = tokenManager

	// 初始化认证处理器
	authHandler := auth.NewAuthHandler(cfg, connManager, tokenManager)
	s.AuthHandler = authHandler

	// 初始化网络服务器
	tcpConfig := &zNet.TcpConfig{
		ListenAddress:     cfg.Server.ListenAddr,
		MaxClientCount:    cfg.Server.MaxConnections,
		HeartbeatDuration: cfg.Server.HeartbeatInterval,
		ChanSize:          cfg.Server.ChanSize,
		MaxPacketDataSize: int32(cfg.Server.MaxPacketDataSize),
		UseWorkerPool:     cfg.Server.UseWorkerPool,
		WorkerPoolSize:    cfg.Server.WorkerPoolSize,
		WorkerQueueSize:   cfg.Server.WorkerQueueSize,
	}
	// 使用函数式选项设置回调函数
	netServer := zNet.NewTcpServer(tcpConfig,
		zNet.WithAddSessionCallBack(s.onClientConnected),
		zNet.WithRemoveSessionCallBack(s.onClientDisconnected),
	)
	s.NetServer = netServer

	// 初始化协议处理器
	protocolHandler := protocol.NewProtocolHandler()
	s.ProtocolHandler = protocolHandler

	// 初始化压缩配置
	compressionCfg := &protolayer.CompressionConfig{
		Enabled:   true,
		Threshold: 1024,
		Level:     5,
	}
	s.CompressionCfg = compressionCfg

	// 初始化TCP服务
	tcpService := service.NewTCPService(cfg, netServer, connManager, protocolHandler, compressionCfg, securityManager)
	s.TCPService = tcpService

	// 初始化游戏服务器代理
	gameServerProxy := proxy.NewGameServerProxy(cfg, tcpService, connManager)
	s.GameServerProxy = gameServerProxy

	// 初始化心跳上报器
	globalServerAddr := config.GetEnv("GLOBAL_SERVER_ADDR", "127.0.0.1:8888")
	heartbeatReporter := monitor.NewHeartbeatReporter(cfg, connManager, globalServerAddr)
	s.HeartbeatReporter = heartbeatReporter

	// 注册组件
	s.RegisterComponent("Config", cfg)
	s.RegisterComponent("TCPService", tcpService)
	s.RegisterComponent("ConnectionManager", connManager)
	s.RegisterComponent("SecurityManager", securityManager)
	s.RegisterComponent("AntiCheatManager", antiCheatManager)
	s.RegisterComponent("AuthHandler", authHandler)
	s.RegisterComponent("GameServerProxy", gameServerProxy)
	s.RegisterComponent("HeartbeatReporter", heartbeatReporter)
	s.RegisterComponent("TokenManager", tokenManager)
	s.RegisterComponent("ProtocolHandler", protocolHandler)
	s.RegisterComponent("NetServer", netServer)
	s.RegisterComponent("CompressionCfg", compressionCfg)

	return nil
}

// OnAfterStart 启动后的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnAfterStart() error {
	// 注册消息分发处理器
	s.NetServer.RegisterDispatcher(s.handleClientMessage)

	// 启动TCP服务
	if err := s.TCPService.Start(); err != nil {
		return err
	}

	// 启动游戏服务器代理
	if err := s.GameServerProxy.Start(s.GetContext()); err != nil {
		return err
	}

	// 启动心跳上报器
	s.HeartbeatReporter.Start()

	// 启动防作弊管理器清理任务
	s.AntiCheatManager.StartCleanupTask()

	return nil
}

// OnBeforeStop 停止前的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnBeforeStop() {
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

// onClientConnected 客户端连接回调
func (s *BaseServer) onClientConnected(sessionID zNet.SessionIdType) {
	session := s.NetServer.GetSession(sessionID)
	if session == nil {
		zLog.Warn("Session not found on connect", zap.Uint64("session_id", uint64(sessionID)))
		return
	}

	s.ConnectionManager.AddSession(sessionID, "")

	zLog.Info("Client connected", zap.Uint64("session_id", uint64(sessionID)))
}

// onClientDisconnected 客户端断开连接回调
func (s *BaseServer) onClientDisconnected(sessionID zNet.SessionIdType) {
	sessionInfo, exists := s.ConnectionManager.GetSessionInfo(sessionID)
	if exists {
		s.ConnectionManager.RemoveSession(sessionID)
		s.GameServerProxy.UnregisterClient(sessionID)

		zLog.Info("Client disconnected",
			zap.Uint64("session_id", uint64(sessionID)),
			zap.Int64("account_id", int64(sessionInfo.AccountID)),
			zap.String("account_name", sessionInfo.AccountName))
	} else {
		zLog.Warn("Session info not found on disconnect", zap.Uint64("session_id", uint64(sessionID)))
	}
}

// handleClientMessage 处理客户端消息
func (s *BaseServer) handleClientMessage(session zNet.Session, packet *zNet.NetPacket) error {
	sessionID := session.GetSid()

	zLog.Info("Received client message",
		zap.Uint64("session_id", uint64(sessionID)),
		zap.Int32("proto_id", packet.ProtoId),
		zap.Int("data_size", len(packet.Data)))

	// 令牌验证消息（消息ID为2），不需要检查会话是否已验证token
	if packet.ProtoId == 2 {
		token := string(packet.Data)
		if err := s.AuthHandler.HandleTokenVerify(session, token); err != nil {
			zLog.Error("Failed to handle token verify",
				zap.Uint64("session_id", uint64(sessionID)),
				zap.Error(err))
			return err
		}
		return nil
	}

	// 检查会话是否已验证token
	sessionInfo, exists := s.ConnectionManager.GetSessionInfo(sessionID)
	if !exists || sessionInfo.AccountID == 0 {
		zLog.Warn("Session not authenticated", zap.Uint64("session_id", uint64(sessionID)))
		return fmt.Errorf("session not authenticated")
	}

	// 转发消息到GameServer
	if err := s.GameServerProxy.SendToClient(sessionID, packet.ProtoId, packet.Data); err != nil {
		zLog.Error("Failed to forward message to GameServer",
			zap.Uint64("session_id", uint64(sessionID)),
			zap.Error(err))
		return err
	}

	return nil
}
