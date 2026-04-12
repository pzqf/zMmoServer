package gateway

import (
	"fmt"
	"time"

	"go.etcd.io/etcd/client/v3"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zMmoServer/GatewayServer/client"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/gameserver"
	"github.com/pzqf/zMmoServer/GatewayServer/metrics"
	"github.com/pzqf/zMmoServer/GatewayServer/report"
	"github.com/pzqf/zMmoServer/GatewayServer/version"
	"go.uber.org/zap"
)

// ServerType 网关服类型
const ServerTypeGateway zServer.ServerType = "gateway"

// BaseServer 网关服基础服务
type BaseServer struct {
	*zServer.BaseServer
	Config                *config.Config
	NetServer             *zNet.TcpServer
	EtcdClient            *clientv3.Client
	ClientService         *client.Service
	GameServerConnService *gameserver.ConnectionService
	ReportService         *report.Service
	MetricsService        *metrics.Metrics
}

// NewBaseServer 创建网关服基础服务
func NewBaseServer() *BaseServer {
	gs := &BaseServer{}

	baseServer := zServer.NewBaseServer(
		ServerTypeGateway,
		"gateway",
		"Gateway Server",
		version.Version,
		gs,
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

	// 解析ServerID
	serverID, err := id.ParseServerIDInt(int32(cfg.Server.ServerID))
	if err != nil {
		return fmt.Errorf("invalid gateway ServerID %d: %w", cfg.Server.ServerID, err)
	}
	serverIDStr := id.ServerIDString(serverID)
	s.SetId(fmt.Sprintf("gateway-%s", serverIDStr))

	// 初始化网络服务器
	tcpConfig := &zNet.TcpConfig{
		ListenAddress:       cfg.Server.ListenAddr,
		MaxClientCount:      cfg.Server.MaxConnections,
		HeartbeatDuration:   cfg.Server.HeartbeatInterval,
		ChanSize:            cfg.Server.ChanSize,
		MaxPacketDataSize:   int32(cfg.Server.MaxPacketDataSize),
		UseWorkerPool:       cfg.Server.UseWorkerPool,
		WorkerPoolSize:      cfg.Server.WorkerPoolSize,
		WorkerQueueSize:     cfg.Server.WorkerQueueSize,
		DisableEncryption:   cfg.Server.DisableEncryption,
		EnableKeyRotation:   cfg.Server.EnableKeyRotation,
		KeyRotationInterval: time.Duration(cfg.Server.KeyRotationInterval) * time.Second,
		MaxHistoryKeys:      cfg.Server.MaxHistoryKeys,
		EnableSequenceCheck: cfg.Server.EnableSequenceCheck,
		SequenceWindowSize:  cfg.Server.SequenceWindowSize,
		TimestampTolerance:  cfg.Server.TimestampTolerance,
	}

	s.NetServer = zNet.NewTcpServer(tcpConfig,
		zNet.WithDDoSConfig(&cfg.DDoS),
		zNet.WithCompressionConfig(&cfg.Compression),
		zNet.WithLogger(s.GetLogger()))

	// 初始化etcd客户端
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.Etcd.Endpoints},
		Username:    cfg.Etcd.Username,
		Password:    cfg.Etcd.Password,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		zLog.Error("Failed to create etcd client", zap.Error(err))
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	s.EtcdClient = etcdClient

	// 初始化客户端服务
	s.ClientService = client.NewService(cfg, s.NetServer, etcdClient)
	s.ClientService.Init()

	// 初始化GameServer连接服务
	s.GameServerConnService = gameserver.NewConnectionService(cfg, s.ClientService)
	if err := s.GameServerConnService.Init(); err != nil {
		return fmt.Errorf("failed to initialize GameServer connection service: %w", err)
	}

	// 设置GameServer代理到客户端服务
	s.ClientService.SetGameServerProxy(s.GameServerConnService.GetGameServerProxy())

	// 初始化监控服务
	s.MetricsService = metrics.NewMetrics(cfg, s.ClientService.GetConnMgr())

	// 初始化服务上报服务
	s.ReportService = report.NewService(s.ClientService.GetConnMgr(), s.GameServerConnService)

	// 设置状态为初始化中
	s.SetState(zServer.StateInitializing, "server initializing")

	// 注册组件
	s.RegisterComponent("Config", s.Config)
	s.RegisterComponent("NetServer", s.NetServer)
	s.RegisterComponent("ClientService", s.ClientService)
	s.RegisterComponent("GameServerConnService", s.GameServerConnService)
	s.RegisterComponent("ReportService", s.ReportService)
	s.RegisterComponent("MetricsService", s.MetricsService)

	zLog.Info("Gateway server initialized successfully")
	return nil
}

// OnAfterStart 启动后的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnAfterStart() error {
	// 启动监控服务
	if s.MetricsService != nil {
		if err := s.MetricsService.Start(); err != nil {
			return err
		}
	}

	// 启动GameServer连接服务
	if s.GameServerConnService != nil {
		if err := s.GameServerConnService.Start(s.GetContext()); err != nil {
			return err
		}
	}

	// 启动客户端服务
	if s.ClientService != nil {
		if err := s.ClientService.Start(); err != nil {
			return err
		}
	}

	// 启动服务上报服务
	if s.ReportService != nil {
		s.ReportService.Start(s.GetContext())
	}

	// 更新服务状态为就绪
	s.SetState(zServer.StateReady, "server ready")

	// 更新服务状态为健康
	s.SetState(zServer.StateHealthy, "server healthy")
	zLog.Info("Gateway server is healthy")

	return nil
}

// OnBeforeStop 停止前的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnBeforeStop() {
	// 设置状态为流量排空
	s.SetState(zServer.StateDraining, "server stopping")
	zLog.Info("Gateway server entering draining state")

	// 停止客户端服务
	if s.ClientService != nil {
		s.ClientService.Stop()
	}

	// 注销服务
	if s.GameServerConnService != nil {
		serviceDiscovery := s.GameServerConnService.GetServiceDiscovery()
		if serviceDiscovery != nil {
			if err := serviceDiscovery.Unregister(); err != nil {
				zLog.Warn("Failed to unregister service", zap.Error(err))
			}
			if err := serviceDiscovery.Close(); err != nil {
				zLog.Warn("Failed to close service discovery", zap.Error(err))
			}
		}
	}

	// 关闭etcd客户端
	if s.EtcdClient != nil {
		if err := s.EtcdClient.Close(); err != nil {
			zLog.Warn("Failed to close etcd client", zap.Error(err))
		}
	}
}

// OnAfterStop 停止后的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnAfterStop() {
	// 设置状态为已停止
	s.SetState(zServer.StateStopped, "server stopped")
	zLog.Info("Gateway server stopped completely")
}
