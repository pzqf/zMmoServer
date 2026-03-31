package gateway

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/net/protolayer"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zMmoServer/GatewayServer/auth"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/proxy"
	gwService "github.com/pzqf/zMmoServer/GatewayServer/service"
	"github.com/pzqf/zMmoServer/GatewayServer/token"
	"github.com/pzqf/zMmoServer/GatewayServer/version"
	"go.uber.org/zap"
)

// ServerType 网关服类型
const ServerTypeGateway zServer.ServerType = "gateway"

// BaseServer 网关服基础服务
type BaseServer struct {
	*zServer.BaseServer
	Config            *config.Config
	TCPService        *gwService.TCPService
	ConnectionManager *connection.ConnectionManager
	SecurityManager   *auth.SecurityManager
	AntiCheatManager  *auth.AntiCheatManager
	AuthHandler       *auth.AuthHandler
	GameServerProxy   proxy.GameServerProxy
	TokenManager      *token.TokenManager
	NetServer         *zNet.TcpServer
	CompressionCfg    *protolayer.CompressionConfig
	ServiceDiscovery  *discovery.ServiceDiscovery
}

// NewBaseServer 创建网关服基础服务
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

	// 初始化压缩配置
	compressionCfg := &protolayer.CompressionConfig{
		Enabled:   true,
		Threshold: 1024,
		Level:     5,
	}
	s.CompressionCfg = compressionCfg

	// 初始化TCP服务
	tcpService := gwService.NewTCPService(cfg, netServer, connManager, compressionCfg, securityManager)
	s.TCPService = tcpService

	// 初始化游戏服务器代理
	gameServerProxy := proxy.NewGameServerProxy(cfg, tcpService, connManager)
	s.GameServerProxy = gameServerProxy

	// 初始化服务发现（基于 etcd，适配 k8s）
	etcdEndpoints := strings.Split(cfg.Etcd.Endpoints, ",")
	etcdConfig := &discovery.EtcdConfig{
		Endpoints:      cfg.Etcd.Endpoints,
		Username:       cfg.Etcd.Username,
		Password:       cfg.Etcd.Password,
		CACertPath:     cfg.Etcd.CACertPath,
		ClientCertPath: cfg.Etcd.ClientCertPath,
		ClientKeyPath:  cfg.Etcd.ClientKeyPath,
	}
	serviceDiscovery, err := discovery.NewServiceDiscoveryWithConfig(etcdEndpoints, etcdConfig)
	if err != nil {
		zLog.Error("Failed to create service discovery", zap.Error(err))
		return fmt.Errorf("failed to create service discovery: %w", err)
	}
	s.ServiceDiscovery = serviceDiscovery
	zLog.Info("Using etcd service discovery", zap.Strings("endpoints", etcdEndpoints))
	// 注册服务发现组件
	s.RegisterComponent("ServiceDiscovery", serviceDiscovery)

	// 注册当前服务：严格使用 6 位 GameServerID（GroupID(4)+ServerIndex(2)）
	serverID, err := id.ParseServerIDInt(int32(cfg.Server.ServerID))
	if err != nil {
		return fmt.Errorf("invalid gateway ServerID %d: %w", cfg.Server.ServerID, err)
	}
	serverIDStr := id.ServerIDString(serverID)
	groupID := id.GroupIDStringFromServerID(serverID)
	s.SetId(fmt.Sprintf("gateway-%s", serverIDStr))
	serviceInfo := &discovery.ServerInfo{
		ID:        fmt.Sprintf("gateway-%s", serverIDStr),
		GroupID:   groupID,
		Status:    discovery.ServerStatusHealthy,
		Address:   cfg.Server.ExternalAddr,
		Port:      0, // 网关服端口已在Address中包含
		Load:      0,
		Players:   0,
		ReadyTime: time.Now().Unix(),
	}

	if err := serviceDiscovery.Register(serviceInfo); err != nil {
		zLog.Warn("Failed to register service", zap.Error(err))
	} else {
		zLog.Info("Service registered successfully",
			zap.String("service_id", serviceInfo.ID),
			zap.String("address", serviceInfo.Address))

		// 启动心跳保持
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-s.GetContext().Done():
					return
				case <-ticker.C:
					// 重新创建serviceInfo并更新字段
					updatedInfo := &discovery.ServerInfo{
						ID:        serviceInfo.ID,
						GroupID:   serviceInfo.GroupID,
						Status:    discovery.ServerStatusHealthy,
						Address:   serviceInfo.Address,
						Port:      serviceInfo.Port,
						Load:      0,
						Players:   s.ConnectionManager.GetConnectionCount(),
						ReadyTime: serviceInfo.ReadyTime,
					}
					if err := serviceDiscovery.Register(updatedInfo); err != nil {
						zLog.Warn("Failed to keep service alive", zap.Error(err))
					}
				}
			}
		}()
	}

	// 注册组件
	s.RegisterComponent("Config", cfg)
	s.RegisterComponent("TCPService", tcpService)
	s.RegisterComponent("ConnectionManager", connManager)
	s.RegisterComponent("SecurityManager", securityManager)
	s.RegisterComponent("AntiCheatManager", antiCheatManager)
	s.RegisterComponent("AuthHandler", authHandler)
	s.RegisterComponent("GameServerProxy", gameServerProxy)

	s.RegisterComponent("TokenManager", tokenManager)
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

	// 启动防作弊管理器清理任务
	s.AntiCheatManager.StartCleanupTask()
	return nil
}

// OnBeforeStop 停止前的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnBeforeStop() {

	// 停止游戏服务器代理
	if s.GameServerProxy != nil {
		s.GameServerProxy.Stop(s.GetContext())
	}

	// 停止TCP服务
	if s.TCPService != nil {
		s.TCPService.Stop()
	}

	// 注销服务发现
	if s.ServiceDiscovery != nil {
		if s.Config != nil {
			serverID, err := id.ParseServerIDInt(int32(s.Config.Server.ServerID))
			if err != nil {
				zLog.Warn("Failed to parse gateway ServerID on unregister", zap.Error(err))
				return
			}
			serviceID := id.ServerIDString(serverID)
			// 使用从服务器ID中提取的groupID
			groupID := id.GroupIDStringFromServerID(id.ServerIdType(s.Config.Server.GroupID))
			if err := s.ServiceDiscovery.Unregister("gateway", groupID, serviceID); err != nil {
				zLog.Warn("Failed to unregister service", zap.Error(err))
			} else {
				zLog.Info("Service unregistered successfully", zap.String("service_id", serviceID))
			}
		}
		if err := s.ServiceDiscovery.Close(); err != nil {
			zLog.Warn("Failed to close service discovery", zap.Error(err))
		}
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

	// enter game：当前客户端实现用 protoId=6，payload 前4字节是 selected server_id
	// Gateway 必须强制校验选服是否与本 Gateway 绑定的 GameServer 一致
	if packet.ProtoId == 6 {
		if len(packet.Data) < 4 {
			return fmt.Errorf("enter game: invalid payload length %d", len(packet.Data))
		}

		selectedServerID := int32(binary.BigEndian.Uint32(packet.Data[:4]))

		targetGameServerID, err := id.ParseServerIDInt(int32(s.Config.GameServer.GameServerID))
		if err != nil {
			return fmt.Errorf("enter game: invalid GameServerID %d: %w", s.Config.GameServer.GameServerID, err)
		}
		targetFullIDInt := int32(targetGameServerID)

		match := selectedServerID == targetFullIDInt
		if !match {
			zLog.Warn("enter game: server_id mismatch",
				zap.Int32("selected_server_id", selectedServerID),
				zap.Int32("target_server_id", targetFullIDInt))
			return fmt.Errorf("enter game: server_id mismatch")
		}
	}

	// 转发消息到GameServer
	if err := s.GameServerProxy.SendToGameServer(sessionID, packet.ProtoId, packet.Data); err != nil {
		zLog.Error("Failed to forward message to GameServer",
			zap.Uint64("session_id", uint64(sessionID)),
			zap.Error(err))
		return err
	}

	return nil
}
