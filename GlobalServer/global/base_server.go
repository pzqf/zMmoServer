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
	"github.com/pzqf/zMmoServer/GlobalServer/handler"
	"github.com/pzqf/zMmoServer/GlobalServer/http"
	"github.com/pzqf/zMmoServer/GlobalServer/metrics"
	"github.com/pzqf/zMmoServer/GlobalServer/serverregistry"
	"github.com/pzqf/zMmoServer/GlobalServer/version"
	"go.uber.org/zap"
)

// ServerType 全局服类型
const ServerTypeGlobal zServer.ServerType = "global"

// global 没有分组概念，使用默认分组 "default"
const DefaultGroupID = "default"

// BaseServer 全局服基础服务
type BaseServer struct {
	*zServer.BaseServer
	Config           *config.Config
	HTTPService      *http.HttpService
	DBService        *db.DBService
	Metrics          *metrics.Metrics
	HealthManager    *health.HealthManager
	ServiceDiscovery *discovery.ServiceDiscovery
}

// NewBaseServer 创建全局服基础服务
func NewBaseServer() *BaseServer {
	// 先创建子类实例
	gs := &BaseServer{}

	// 创建基础服务器，传入子类作为 hooks
	// 注意：serverId 将在 OnBeforeStart 中根据配置设置
	baseServer := zServer.NewBaseServer(
		ServerTypeGlobal,
		"", // serverId 将在 OnBeforeStart 中设置
		"Global Server",
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

	serverIDStr := fmt.Sprintf("global-%d", cfg.Server.ServerID)
	s.SetId(serverIDStr)

	// 初始化健康管理器
	healthManager := health.NewHealthManager(serverIDStr, "global")
	// 注册健康检查
	healthManager.RegisterCheck(health.NewMemoryChecker())
	healthManager.RegisterCheck(health.NewGoroutineChecker())
	healthManager.RegisterCheck(health.NewTimeChecker())
	s.HealthManager = healthManager

	// 初始化 DBManager（只初始化 GlobalServer 需要的 Repository）
	dbConfigs := map[string]sharedDB.DBConfig{
		"global": cfg.Database,
	}

	// 设置状态为初始化中（使用 BaseServer 的状态管理）
	if err := s.SetState(zServer.StateInitializing, "server initializing"); err != nil {
		zLog.Warn("Failed to set initializing state", zap.Error(err))
	}

	if err := sharedDB.InitDBManagerWithRepos(dbConfigs, sharedDB.RepoTypeGlobalServer); err != nil {
		return err
	}

	// 初始化数据库表结构
	dbMgr := sharedDB.GetMgr()
	if dbMgr != nil {
		conn := dbMgr.GetConnector("global")
		if conn != nil {
			if err := sharedDB.InitTables(conn, sharedDB.RepoTypeGlobal); err != nil {
				zLog.Error("Failed to initialize database tables", zap.Error(err))
				return err
			}

			if err := sharedDB.InitDefaultData(conn); err != nil {
				zLog.Error("Failed to initialize default data", zap.Error(err))
				return err
			}
		}
	}

	// 初始化 ID 生成器
	if err := id.InitIDGenerator(cfg.Server.WorkerID, cfg.Server.DatacenterID); err != nil {
		return err
	}

	// 初始化 Metrics 服务
	metricsService := metrics.NewMetrics(&cfg.Metrics)
	s.Metrics = metricsService

	// 初始化 JWT 密钥
	handler.InitJWTSecret(cfg.Server.JWTSecret)

	// 初始化 HTTP 服务
	httpService := http.NewService()
	httpService.SetConfig(&cfg.HTTP)
	// 设置关闭回调，用于 HTTP 接口触发服务器关闭
	httpService.SetShutdownFunc(func() {
		s.Shutdown()
	})
	// 设置 metrics 实例
	httpService.SetMetrics(metricsService)
	// 初始化 HTTP 服务，注册路由
	if err := httpService.Init(); err != nil {
		return err
	}
	s.HTTPService = httpService

	// 初始化数据库服务
	dbService := db.NewDBService(cfg)
	s.DBService = dbService

	// 从 MySQL 加载静态服务器配置
	if err := s.loadStaticServers(); err != nil {
		zLog.Warn("Failed to load static servers", zap.Error(err))
	}

	// 初始化服务发现
	etcdEndpoints := strings.Split(cfg.Etcd.Endpoints, ",")
	serviceDiscovery, err := discovery.NewServiceDiscoveryWithConfig(etcdEndpoints, &cfg.Etcd)
	if err != nil {
		zLog.Error("Failed to create service discovery", zap.Error(err))
		return fmt.Errorf("failed to create service discovery: %w", err)
	}
	s.ServiceDiscovery = serviceDiscovery
	zLog.Info("Using etcd service discovery", zap.Strings("endpoints", etcdEndpoints))
	// 注册服务发现组件
	s.RegisterComponent("ServiceDiscovery", serviceDiscovery)

	// 初始化服务器列表管理器（使用服务发现）
	if err := serverregistry.InitServerListManager(serviceDiscovery); err != nil {
		return fmt.Errorf("failed to init server list manager: %w", err)
	}

	// 注册当前服务
	serviceInfo := &discovery.ServerInfo{
		ID:            serverIDStr,
		ServiceType:   "global",
		GroupID:       DefaultGroupID,
		Status:        zServer.StateInitializing,
		Address:       cfg.HTTP.ListenAddress,
		Port:          0, // 全局服端口已在Address中包含
		Load:          0,
		Players:       0,
		ReadyTime:     time.Now().Unix(),
		LastHeartbeat: time.Now().Unix(),
	}

	zLog.Info("Attempting to register service",
		zap.String("service_id", serviceInfo.ID),
		zap.String("address", serviceInfo.Address))

	// 服务注册，添加重试机制
	maxRetries := 3
	var registerErr error
	for i := 0; i < maxRetries; i++ {
		if err := serviceDiscovery.Register(serviceInfo); err != nil {
			zLog.Warn("Failed to register service", zap.Error(err), zap.Int("retry", i+1))
			registerErr = err
			if s.Metrics != nil {
				s.Metrics.IncrementServiceDiscoveryFailures()
			}
			time.Sleep(time.Duration(i+1) * time.Second) // 指数退避
		} else {
			zLog.Info("Service registered successfully",
				zap.String("service_id", serviceInfo.ID),
				zap.String("address", serviceInfo.Address))
			if s.Metrics != nil {
				s.Metrics.IncrementServiceRegister()
			}
			registerErr = nil
			break
		}
	}
	if registerErr != nil {
		return fmt.Errorf("failed to register service after %d retries: %w", maxRetries, registerErr)
	}

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
					ID:            serverIDStr,
					ServiceType:   "global",
					GroupID:       DefaultGroupID,
					Status:        s.GetState(),
					Address:       serviceInfo.Address,
					Port:          serviceInfo.Port,
					Load:          0,
					Players:       0,
					ReadyTime:     serviceInfo.ReadyTime,
					LastHeartbeat: time.Now().Unix(),
				}
				// 心跳发送，添加重试机制
				maxRetries := 2
				var heartbeatErr error
				for i := 0; i < maxRetries; i++ {
					if err := serviceDiscovery.Register(updatedInfo); err != nil {
						zLog.Warn("Failed to keep service alive", zap.Error(err), zap.Int("retry", i+1))
						heartbeatErr = err
						if s.Metrics != nil {
							s.Metrics.IncrementServiceDiscoveryFailures()
						}
						time.Sleep(time.Duration(i+1) * time.Second) // 指数退避
					} else {
						if s.Metrics != nil {
							s.Metrics.IncrementServiceHeartbeat()
						}
						zLog.Debug("Service heartbeat kept alive", zap.String("service_id", serviceInfo.ID))
						heartbeatErr = nil
						break
					}
				}
				if heartbeatErr != nil {
					zLog.Error("Failed to keep service alive after retries", zap.Error(heartbeatErr))
				}
			}
		}
	}()

	// 注册组件
	s.RegisterComponent("Config", cfg)
	s.RegisterComponent("HTTPService", httpService)
	s.RegisterComponent("DBService", dbService)
	s.RegisterComponent("Metrics", metricsService)
	s.RegisterComponent("HealthManager", healthManager)
	s.RegisterComponent("ServerListManager", serverregistry.GetServerListManager())

	return nil
}

// OnAfterStart 启动后的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnAfterStart() error {
	// 启动数据库服务
	if s.DBService != nil {
		if err := s.DBService.Start(); err != nil {
			return err
		}
	}

	// 启动 Metrics 服务
	if s.Metrics != nil {
		if err := s.Metrics.Start(); err != nil {
			return err
		}
	}

	// 启动 HTTP 服务
	if s.HTTPService != nil {
		if err := s.HTTPService.Start(); err != nil {
			return err
		}
	}

	// 更新服务状态为就绪
	if err := s.SetState(zServer.StateReady, "server ready"); err != nil {
		zLog.Warn("Failed to set ready state", zap.Error(err))
	}

	// 确认健康状态
	if s.HealthManager != nil && s.HealthManager.IsHealthy() {
		// 更新服务状态为健康
		if err := s.SetState(zServer.StateHealthy, "server healthy"); err != nil {
			zLog.Warn("Failed to set healthy state", zap.Error(err))
		}
		zLog.Info("Global server is healthy")
	} else {
		zLog.Warn("Global server health check failed")
	}

	return nil
}

// OnBeforeStop 停止前的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnBeforeStop() {
	// 设置状态为流量排空
	if err := s.SetState(zServer.StateDraining, "server draining"); err != nil {
		zLog.Warn("Failed to set draining state", zap.Error(err))
	}
	zLog.Info("Global server entering draining state")

	// 停止 HTTP 服务（不再接受新请求）
	if s.HTTPService != nil {
		s.HTTPService.Stop()
	}

	// 停止数据库服务
	if s.DBService != nil {
		s.DBService.Stop()
	}

	// 注销服务发现
	if s.ServiceDiscovery != nil {
		if s.Config != nil {
			serverID, _ := id.ParseServerIDInt(int32(s.Config.Server.ServerID))
			serviceID := id.ServerIDString(serverID)
			// 服务注销，添加重试机制
			maxRetries := 3
			var unregisterErr error
			for i := 0; i < maxRetries; i++ {
				if err := s.ServiceDiscovery.Unregister("global", DefaultGroupID, serviceID); err != nil {
					zLog.Warn("Failed to unregister service", zap.Error(err), zap.Int("retry", i+1))
					unregisterErr = err
					if s.Metrics != nil {
						s.Metrics.IncrementServiceDiscoveryFailures()
					}
					time.Sleep(time.Duration(i+1) * time.Second) // 指数退避
				} else {
					zLog.Info("Service unregistered successfully", zap.String("service_id", serviceID))
					if s.Metrics != nil {
						s.Metrics.IncrementServiceUnregister()
					}
					unregisterErr = nil
					break
				}
			}
			if unregisterErr != nil {
				zLog.Error("Failed to unregister service after retries", zap.Error(unregisterErr))
			}
		}
		if err := s.ServiceDiscovery.Close(); err != nil {
			zLog.Warn("Failed to close service discovery", zap.Error(err))
		}
	}

	// 关闭服务器列表管理器
	if manager := serverregistry.GetServerListManager(); manager != nil {
		manager.Close()
	}
}

// OnAfterStop 停止后的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnAfterStop() {
	// 设置状态为已停止
	if err := s.SetState(zServer.StateStopped, "server stopped"); err != nil {
		zLog.Warn("Failed to set stopped state", zap.Error(err))
	}
	zLog.Info("Global server stopped completely")
}

// loadStaticServers 从 MySQL 加载静态服务器配置
func (s *BaseServer) loadStaticServers() error {
	// 从数据库服务获取静态服务器配置
	if s.DBService != nil {
		servers, err := s.DBService.GetGameServers()
		if err != nil {
			return fmt.Errorf("failed to get game servers: %w", err)
		}

		// 加载到服务器列表管理器
		if manager := serverregistry.GetServerListManager(); manager != nil {
			manager.LoadStaticServers(servers)
			zLog.Info("Static servers loaded from MySQL", zap.Int("count", len(servers)))
		}
	}
	return nil
}
