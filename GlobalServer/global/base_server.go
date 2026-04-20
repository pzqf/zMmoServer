package global

import (
	"fmt"
	"strings"
	"time"

	"github.com/pzqf/zCommon/common/id"
	sharedDB "github.com/pzqf/zCommon/db"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/health"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zMmoServer/GlobalServer/config"
	"github.com/pzqf/zMmoServer/GlobalServer/db"
	"github.com/pzqf/zMmoServer/GlobalServer/gameserverlist"
	"github.com/pzqf/zMmoServer/GlobalServer/handler"
	"github.com/pzqf/zMmoServer/GlobalServer/http"
	"github.com/pzqf/zMmoServer/GlobalServer/metrics"
	"github.com/pzqf/zMmoServer/GlobalServer/version"
	"go.uber.org/zap"
)

const ServerTypeGlobal zServer.ServerType = "global"

type BaseServer struct {
	*zServer.BaseServer
	Config           *config.Config
	HTTPService      *http.HttpService
	DBService        *db.DBService
	Metrics          *metrics.Metrics
	HealthManager    *health.HealthManager
	ServiceDiscovery *discovery.ServiceDiscovery
	serverIDStr      string
	serviceInfo      *discovery.ServerInfo
}

func NewBaseServer(cfg *config.Config) *BaseServer {
	gs := &BaseServer{Config: cfg}
	gs.BaseServer = zServer.NewBaseServer(ServerTypeGlobal, "", "Global Server", version.Version, gs)
	return gs
}

func (s *BaseServer) OnBeforeStart() error {
	if s.Config == nil {
		return nil
	}

	s.serverIDStr = fmt.Sprintf("global-%d", s.Config.Server.ServerID)
	s.SetId(s.serverIDStr)

	s.initHealthManager()
	s.SetState(zServer.StateInitializing, "server initializing")

	if err := s.initDatabase(); err != nil {
		return err
	}

	if err := id.InitIDGenerator(s.Config.Server.WorkerID, s.Config.Server.DatacenterID); err != nil {
		return err
	}

	s.initMetrics()
	handler.InitJWTSecret(s.Config.Server.JWTSecret, s.Config.Server.TokenExpiryHours)

	if err := s.initHTTPService(); err != nil {
		return err
	}

	s.DBService = db.NewDBService(s.Config)

	if err := s.initServiceDiscovery(); err != nil {
		return err
	}

	s.registerComponents()
	return nil
}

func (s *BaseServer) initHealthManager() {
	hm := health.NewHealthManager(s.serverIDStr, "global")
	hm.RegisterCheck(health.NewMemoryChecker())
	hm.RegisterCheck(health.NewGoroutineChecker())
	hm.RegisterCheck(health.NewTimeChecker())
	s.HealthManager = hm
}

func (s *BaseServer) initDatabase() error {
	dbConfigs := map[string]sharedDB.DBConfig{
		"global": s.Config.Database,
	}
	if err := sharedDB.InitDBManagerWithRepos(dbConfigs, sharedDB.RepoTypeGlobalServer); err != nil {
		return err
	}

	dbMgr := sharedDB.GetMgr()
	if dbMgr == nil {
		return nil
	}

	conn := dbMgr.GetConnector("global")
	if conn == nil {
		return nil
	}

	if err := sharedDB.InitTables(conn, sharedDB.RepoTypeGlobal); err != nil {
		return fmt.Errorf("init database tables: %w", err)
	}
	if err := sharedDB.InitDefaultData(conn); err != nil {
		return fmt.Errorf("init default data: %w", err)
	}
	return nil
}

func (s *BaseServer) initMetrics() {
	s.Metrics = metrics.NewMetrics(&s.Config.Metrics)
}

func (s *BaseServer) initHTTPService() error {
	httpService := http.NewService()
	httpService.SetConfig(&s.Config.HTTP)
	httpService.SetShutdownFunc(func() { s.Shutdown() })
	httpService.SetMetrics(s.Metrics)
	if err := httpService.Init(); err != nil {
		return err
	}
	s.HTTPService = httpService
	return nil
}

func (s *BaseServer) initServiceDiscovery() error {
	etcdEndpoints := strings.Split(s.Config.Etcd.Endpoints, ",")
	sd, err := discovery.NewServiceDiscoveryWithConfig(etcdEndpoints, &s.Config.Etcd)
	if err != nil {
		return fmt.Errorf("create service discovery: %w", err)
	}
	s.ServiceDiscovery = sd
	zLog.Info("Using etcd service discovery", zap.Strings("endpoints", etcdEndpoints))

	if err := gameserverlist.InitServerListManager(sd); err != nil {
		return fmt.Errorf("init server list manager: %w", err)
	}

	s.serviceInfo = &discovery.ServerInfo{
		ID:            s.serverIDStr,
		ServiceType:   "global",
		GroupID:       s.Config.Server.GroupID,
		Status:        zServer.StateInitializing,
		Address:       s.Config.HTTP.ListenAddress,
		ReadyTime:     time.Now().Unix(),
		LastHeartbeat: time.Now().Unix(),
	}

	if err := s.registerWithRetry(); err != nil {
		return err
	}

	go s.heartbeatLoop()
	return nil
}

func (s *BaseServer) registerWithRetry() error {
	const maxRetries = 3
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := s.ServiceDiscovery.Register(s.serviceInfo); err != nil {
			zLog.Warn("Failed to register service", zap.Error(err), zap.Int("retry", i+1))
			lastErr = err
			if s.Metrics != nil {
				s.Metrics.IncrementServiceDiscoveryFailures()
			}
			time.Sleep(time.Duration(i+1) * time.Second)
		} else {
			zLog.Debug("Service registered successfully",
				zap.String("service_id", s.serviceInfo.ID),
				zap.String("address", s.serviceInfo.Address))
			if s.Metrics != nil {
				s.Metrics.IncrementServiceRegister()
			}
			return nil
		}
	}
	return fmt.Errorf("register service after %d retries: %w", maxRetries, lastErr)
}

func (s *BaseServer) heartbeatLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.GetContext().Done():
			return
		case <-ticker.C:
			updatedInfo := &discovery.ServerInfo{
				ID:            s.serviceInfo.ID,
				ServiceType:   "global",
				GroupID:       s.Config.Server.GroupID,
				Status:        s.GetState(),
				Address:       s.serviceInfo.Address,
				Port:          s.serviceInfo.Port,
				ReadyTime:     s.serviceInfo.ReadyTime,
				LastHeartbeat: time.Now().Unix(),
			}

			const maxRetries = 2
			var lastErr error
			for i := 0; i < maxRetries; i++ {
				if err := s.ServiceDiscovery.Register(updatedInfo); err != nil {
					lastErr = err
					if s.Metrics != nil {
						s.Metrics.IncrementServiceDiscoveryFailures()
					}
					time.Sleep(time.Duration(i+1) * time.Second)
				} else {
					if s.Metrics != nil {
						s.Metrics.IncrementServiceHeartbeat()
					}
					lastErr = nil
					break
				}
			}
			if lastErr != nil {
				zLog.Error("Failed to keep service alive after retries", zap.Error(lastErr))
			}
		}
	}
}

func (s *BaseServer) registerComponents() {
	s.RegisterComponent("Config", s.Config)
	s.RegisterComponent("HTTPService", s.HTTPService)
	s.RegisterComponent("DBService", s.DBService)
	s.RegisterComponent("Metrics", s.Metrics)
	s.RegisterComponent("HealthManager", s.HealthManager)
	s.RegisterComponent("ServiceDiscovery", s.ServiceDiscovery)
	s.RegisterComponent("ServerListManager", gameserverlist.GetServerListManager())
}

func (s *BaseServer) OnAfterStart() error {
	if s.DBService != nil {
		if err := s.DBService.Start(); err != nil {
			return err
		}
	}

	if err := s.loadStaticServers(); err != nil {
		zLog.Warn("Failed to load static servers", zap.Error(err))
	}
	if s.Metrics != nil {
		if err := s.Metrics.Start(); err != nil {
			return err
		}
	}
	if s.HTTPService != nil {
		if err := s.HTTPService.Start(); err != nil {
			return err
		}
	}

	s.SetState(zServer.StateReady, "server ready")

	if s.HealthManager != nil && s.HealthManager.IsHealthy() {
		s.SetState(zServer.StateHealthy, "server healthy")
		zLog.Info("Global server is healthy")
	} else {
		zLog.Warn("Global server health check failed")
	}
	return nil
}

func (s *BaseServer) OnBeforeStop() {
	s.SetState(zServer.StateDraining, "server draining")
	zLog.Info("Global server entering draining state")

	if s.HTTPService != nil {
		s.HTTPService.Stop()
	}
	if s.DBService != nil {
		s.DBService.Stop()
	}
	if s.ServiceDiscovery != nil {
		s.unregisterWithRetry()
		if err := s.ServiceDiscovery.Close(); err != nil {
			zLog.Warn("Failed to close service discovery", zap.Error(err))
		}
	}
	if manager := gameserverlist.GetServerListManager(); manager != nil {
		manager.Close()
	}
}

func (s *BaseServer) unregisterWithRetry() {
	if s.Config == nil {
		return
	}
	serverID, _ := id.ParseServerIDInt(int32(s.Config.Server.ServerID))
	serviceID := id.ServerIDString(serverID)

	const maxRetries = 3
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := s.ServiceDiscovery.Unregister("global", s.Config.Server.GroupID, serviceID); err != nil {
			zLog.Warn("Failed to unregister service", zap.Error(err), zap.Int("retry", i+1))
			lastErr = err
			if s.Metrics != nil {
				s.Metrics.IncrementServiceDiscoveryFailures()
			}
			time.Sleep(time.Duration(i+1) * time.Second)
		} else {
			zLog.Info("Service unregistered successfully", zap.String("service_id", serviceID))
			if s.Metrics != nil {
				s.Metrics.IncrementServiceUnregister()
			}
			return
		}
	}
	zLog.Error("Failed to unregister service after retries", zap.Error(lastErr))
}

func (s *BaseServer) OnAfterStop() {
	s.SetState(zServer.StateStopped, "server stopped")
	zLog.Info("Global server stopped completely")
}

func (s *BaseServer) loadStaticServers() error {
	if s.DBService == nil {
		return nil
	}
	servers, err := s.DBService.GetGameServers()
	if err != nil {
		return fmt.Errorf("get game servers: %w", err)
	}
	if manager := gameserverlist.GetServerListManager(); manager != nil {
		manager.LoadStaticServers(servers)
		zLog.Info("Static servers loaded from MySQL", zap.Int("count", len(servers)))
	}
	return nil
}
