package global

import (
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zMmoServer/GlobalServer/config"
	"github.com/pzqf/zMmoServer/GlobalServer/db"
	"github.com/pzqf/zMmoServer/GlobalServer/handler"
	"github.com/pzqf/zMmoServer/GlobalServer/http"
	"github.com/pzqf/zMmoServer/GlobalServer/metrics"
	"github.com/pzqf/zMmoServer/GlobalServer/serverstatus"
	"github.com/pzqf/zMmoServer/GlobalServer/version"
	"github.com/pzqf/zMmoShared/common/id"
	sharedDB "github.com/pzqf/zMmoShared/db"
	"go.uber.org/zap"
)

// ServerType 全局服类型
const ServerTypeGlobal zServer.ServerType = "global"

// BaseServer 全局服基础服务器
type BaseServer struct {
	*zServer.BaseServer
	Config      *config.Config
	HTTPService *http.HttpService
	DBService   *db.DBService
	Metrics     *metrics.Metrics
}

// NewBaseServer 创建全局服基础服务器
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

	// 设置服务器ID（格式：前缀+ID）
	serverId := fmt.Sprintf("global-%d", cfg.Server.ServerID)
	s.SetId(serverId)

	// 初始化 DBManager（只初始化 GlobalServer 需要的 Repository）
	dbConfigs := map[string]sharedDB.DBConfig{
		"global": cfg.Database,
	}

	if err := sharedDB.InitDBManagerWithRepos(dbConfigs, sharedDB.RepoTypeGlobalServer); err != nil {
		return err
	}

	// 初始化 Redis 和服务器状态管理器
	if err := serverstatus.InitManager(cfg.Redis.ToRedisConfig()); err != nil {
		return fmt.Errorf("failed to init server status manager: %w", err)
	}

	// 从 MySQL 加载静态服务器配置
	if err := s.loadStaticServers(); err != nil {
		zLog.Warn("Failed to load static servers", zap.Error(err))
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

	// 注册组件
	s.RegisterComponent("Config", cfg)
	s.RegisterComponent("HTTPService", httpService)
	s.RegisterComponent("DBService", dbService)
	s.RegisterComponent("Metrics", metricsService)
	s.RegisterComponent("ServerStatusManager", serverstatus.GetManager())

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

	return nil
}

// OnBeforeStop 停止前的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnBeforeStop() {
	// 停止 HTTP 服务
	if s.HTTPService != nil {
		s.HTTPService.Stop()
	}

	// 停止数据库服务
	if s.DBService != nil {
		s.DBService.Stop()
	}

	// 关闭服务器状态管理器
	if manager := serverstatus.GetManager(); manager != nil {
		manager.Close()
	}

	// 关闭 DBManager
	dbMgr := sharedDB.GetMgr()
	if dbMgr != nil {
		dbMgr.Close()
	}
}

// loadStaticServers 从MySQL加载静态服务器配置
func (s *BaseServer) loadStaticServers() error {
	dbMgr := sharedDB.GetMgr()
	if dbMgr == nil {
		return fmt.Errorf("db manager not initialized")
	}

	servers, err := dbMgr.GameServerRepository.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get game servers: %w", err)
	}

	manager := serverstatus.GetManager()
	if manager == nil {
		return fmt.Errorf("server status manager not initialized")
	}

	manager.LoadStaticServers(servers)
	return nil
}
