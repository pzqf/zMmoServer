package client

import (
	"fmt"
	"net"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/client/auth"
	"github.com/pzqf/zMmoServer/GatewayServer/client/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/client/security"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/proxy"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// Service 客户端服务
type Service struct {
	config           *config.Config
	netServer        *zNet.TcpServer
	etcdClient       *clientv3.Client
	connMgr          *connection.ClientConnMgr
	ipManager        *security.IPManager
	antiCheatManager *security.AntiCheatManager
	authHandler      *auth.AuthHandler
	tokenManager     *auth.TokenManager
	clientHandler    *connection.ClientHandler
	messageHandler   *MessageHandler
}

// NewService 创建客户端服务
func NewService(cfg *config.Config, netServer *zNet.TcpServer, etcdClient *clientv3.Client) *Service {
	connMgr := connection.NewClientConnMgr()
	ipManager := security.NewIPManager(&cfg.Security, etcdClient)
	antiCheatManager := security.NewAntiCheatManager(cfg)
	tokenManager := auth.NewTokenManager(cfg)
	authHandler := auth.NewAuthHandler(cfg, connMgr, tokenManager)

	s := &Service{
		config:           cfg,
		netServer:        netServer,
		etcdClient:       etcdClient,
		connMgr:          connMgr,
		ipManager:        ipManager,
		antiCheatManager: antiCheatManager,
		authHandler:      authHandler,
		tokenManager:     tokenManager,
	}

	// 创建客户端处理器
	s.clientHandler = connection.NewClientHandler(connMgr, ipManager)

	// 连接事件处理由TcpServer内部处理

	return s
}

// Init 初始化客户端服务
func (s *Service) Init() {
	// 启动安全管理器的清理任务
	s.ipManager.StartCleanupTask()

	// 启动防作弊管理器的清理任务
	s.antiCheatManager.StartCleanupTask()

	zLog.Info("Client service initialized successfully")
}

// Start 启动客户端服务
func (s *Service) Start() error {
	zLog.Info("Starting client service...")

	// 尝试监听端口，检查是否可用
	ln, err := net.Listen("tcp", s.config.Server.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Server.ListenAddr, err)
	}
	ln.Close()

	zLog.Info("Client service started successfully", zap.String("listen_addr", s.config.Server.ListenAddr))
	return s.netServer.Start()
}

// Stop 停止客户端服务
func (s *Service) Stop() error {
	zLog.Info("Stopping client service...")

	s.netServer.Close()
	return nil
}

// SetGameServerProxy 设置GameServer代理
func (s *Service) SetGameServerProxy(gameServerProxy proxy.GameServerProxy) {
	// 重新创建消息处理器，注入GameServer代理
	s.messageHandler = NewMessageHandler(s.ipManager, s.antiCheatManager, gameServerProxy)
	s.netServer.RegisterDispatcher(s.messageHandler.HandleMessage)

	zLog.Info("GameServer proxy set successfully")
}

// GetConnMgr 获取连接管理器
func (s *Service) GetConnMgr() *connection.ClientConnMgr {
	return s.connMgr
}

// GetIPManager 获取IP管理器
func (s *Service) GetIPManager() *security.IPManager {
	return s.ipManager
}

// GetAntiCheatManager 获取防作弊管理器
func (s *Service) GetAntiCheatManager() *security.AntiCheatManager {
	return s.antiCheatManager
}

// GetAuthHandler 获取认证处理器
func (s *Service) GetAuthHandler() *auth.AuthHandler {
	return s.authHandler
}

// GetTokenManager 获取Token管理器
func (s *Service) GetTokenManager() *auth.TokenManager {
	return s.tokenManager
}

// SendToClient 发送消息给客户端
func (s *Service) SendToClient(sessionID zNet.SessionIdType, data []byte) error {
	session := s.netServer.GetSession(sessionID)
	if session == nil {
		zLog.Warn("Session not found", zap.Uint64("session_id", uint64(sessionID)))
		return fmt.Errorf("session not found")
	}

	return session.Send(0, data)
}

// GetSessionCount 获取会话数量
func (s *Service) GetSessionCount() int {
	return len(s.netServer.GetAllSession())
}
