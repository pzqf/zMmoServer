package gateway

import (
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

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

const ServerTypeGateway zServer.ServerType = "gateway"

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

func NewBaseServer(cfg *config.Config) *BaseServer {
	gs := &BaseServer{Config: cfg}
	gs.BaseServer = zServer.NewBaseServer(ServerTypeGateway, "gateway", "Gateway Server", version.Version, gs)
	return gs
}

func (s *BaseServer) OnBeforeStart() error {
	if s.Config == nil {
		return nil
	}

	s.initServerID()
	s.SetState(zServer.StateInitializing, "server initializing")

	s.initNetServer()

	if err := s.initEtcdClient(); err != nil {
		return err
	}

	s.initServices()
	s.registerComponents()

	zLog.Info("Gateway server initialized successfully")
	return nil
}

func (s *BaseServer) initServerID() {
	serverID, err := id.ParseServerIDInt(int32(s.Config.Server.ServerID))
	if err != nil {
		zLog.Fatal("Invalid gateway ServerID", zap.Error(err))
	}
	s.SetId(fmt.Sprintf("gateway-%s", id.ServerIDString(serverID)))
}

func (s *BaseServer) initNetServer() {
	cfg := s.Config
	s.NetServer = zNet.NewTcpServer(&zNet.TcpConfig{
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
	},
		zNet.WithDDoSConfig(&cfg.DDoS),
		zNet.WithCompressionConfig(&cfg.Compression),
		zNet.WithLogger(s.GetLogger()),
	)
}

func (s *BaseServer) initEtcdClient() error {
	cfg := s.Config
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.Etcd.Endpoints},
		Username:    cfg.Etcd.Username,
		Password:    cfg.Etcd.Password,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("create etcd client: %w", err)
	}
	s.EtcdClient = etcdClient
	return nil
}

func (s *BaseServer) initServices() {
	s.ClientService = client.NewService(s.Config, s.NetServer, s.EtcdClient)
	s.ClientService.Init()

	s.GameServerConnService = gameserver.NewConnectionService(s.Config, s.ClientService)
	if err := s.GameServerConnService.Init(); err != nil {
		zLog.Fatal("Failed to initialize GameServer connection service", zap.Error(err))
	}

	s.ClientService.SetGameServerProxy(s.GameServerConnService.GetGameServerProxy())
	s.MetricsService = metrics.NewMetrics(s.Config, s.ClientService.GetConnMgr())
	s.ReportService = report.NewService(s.ClientService.GetConnMgr(), s.GameServerConnService)
}

func (s *BaseServer) registerComponents() {
	s.RegisterComponent("Config", s.Config)
	s.RegisterComponent("NetServer", s.NetServer)
	s.RegisterComponent("ClientService", s.ClientService)
	s.RegisterComponent("GameServerConnService", s.GameServerConnService)
	s.RegisterComponent("ReportService", s.ReportService)
	s.RegisterComponent("MetricsService", s.MetricsService)
}

func (s *BaseServer) OnAfterStart() error {
	if s.MetricsService != nil {
		if err := s.MetricsService.Start(); err != nil {
			return err
		}
	}
	if s.GameServerConnService != nil {
		if err := s.GameServerConnService.Start(s.GetContext()); err != nil {
			return err
		}
	}
	if s.ClientService != nil {
		if err := s.ClientService.Start(); err != nil {
			return err
		}
	}
	if s.ReportService != nil {
		s.ReportService.Start(s.GetContext())
	}

	s.SetState(zServer.StateReady, "server ready")
	s.SetState(zServer.StateHealthy, "server healthy")
	zLog.Info("Gateway server is healthy")
	return nil
}

func (s *BaseServer) OnBeforeStop() {
	s.SetState(zServer.StateDraining, "server stopping")
	zLog.Info("Gateway server entering draining state")

	if s.ClientService != nil {
		s.ClientService.Stop()
	}
	if s.GameServerConnService != nil {
		sd := s.GameServerConnService.GetServiceDiscovery()
		if sd != nil {
			if err := sd.Unregister(); err != nil {
				zLog.Warn("Failed to unregister service", zap.Error(err))
			}
			if err := sd.Close(); err != nil {
				zLog.Warn("Failed to close service discovery", zap.Error(err))
			}
		}
	}
	if s.EtcdClient != nil {
		if err := s.EtcdClient.Close(); err != nil {
			zLog.Warn("Failed to close etcd client", zap.Error(err))
		}
	}
}

func (s *BaseServer) OnAfterStop() {
	s.SetState(zServer.StateStopped, "server stopped")
	zLog.Info("Gateway server stopped completely")
}
