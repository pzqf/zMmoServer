package client

import (
	"context"
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

// TcpSessionWrapper 会话包装器，用于在会话移除时提供会话信息
type TcpSessionWrapper struct {
	sessionID zNet.SessionIdType
	server    *zNet.TcpServer
}

// GetSid 获取会话ID
func (w *TcpSessionWrapper) GetSid() zNet.SessionIdType {
	return w.sessionID
}

// GetObj 获取附加对象
func (w *TcpSessionWrapper) GetObj() interface{} {
	return nil
}

// SetObj 设置附加对象
func (w *TcpSessionWrapper) SetObj(obj interface{}) {
}

// GetClientIP 获取客户端IP地址
func (w *TcpSessionWrapper) GetClientIP() string {
	// 由于会话已经被移除，无法获取客户端IP
	// 这里返回空字符串，实际使用中可能需要从其他地方获取
	return ""
}

// Send 发送数据
func (w *TcpSessionWrapper) Send(protoId zNet.ProtoIdType, data []byte) error {
	// 会话已经被移除，无法发送数据
	return nil
}

// Close 关闭会话
func (w *TcpSessionWrapper) Close() {
	// 会话已经被移除，无需操作
}

// Start 启动会话
func (w *TcpSessionWrapper) Start() {
	// 会话已经被移除，无需操作
}

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

	// 注册连接事件处理
	netServer.SetOnAddSession(func(sessionID zNet.SessionIdType) {
		if session := netServer.GetSession(sessionID); session != nil {
			s.clientHandler.OnConnect(session)
		}
	})

	// 注册会话移除回调
	netServer.SetOnRemoveSession(func(sessionID zNet.SessionIdType) {
		// 当会话被移除时，处理客户端断开连接
		s.clientHandler.OnClose(&TcpSessionWrapper{sessionID: sessionID, server: netServer})
	})

	// 注意：TcpServer的OnRemoveSession在会话关闭时调用

	return s
}

// Init 初始化客户端服务
func (s *Service) Init(ctx context.Context) {
	// 启动安全管理器的清理任务
	s.ipManager.StartCleanupTask()

	// 启动防作弊管理器的清理任务
	s.antiCheatManager.StartCleanupTask(ctx)

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
	s.messageHandler = NewMessageHandler(s.ipManager, s.antiCheatManager, gameServerProxy, s.authHandler)
	s.netServer.RegisterDispatcher(s.messageHandler.HandleMessage)

	if s.authHandler != nil {
		s.authHandler.SetGameServerProxy(gameServerProxy)
	}

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
func (s *Service) SendToClient(sessionID zNet.SessionIdType, msgID uint32, data []byte) error {
	session := s.netServer.GetSession(sessionID)
	if session == nil {
		zLog.Warn("Session not found", zap.Uint64("session_id", uint64(sessionID)))
		return fmt.Errorf("session not found")
	}

	return session.Send(zNet.ProtoIdType(msgID), data)
}

// GetSessionCount 获取会话数量
func (s *Service) GetSessionCount() int {
	return len(s.netServer.GetAllSession())
}
