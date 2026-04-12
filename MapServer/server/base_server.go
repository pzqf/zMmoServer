package server

import (
	"fmt"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zEngine/zService"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoServer/MapServer/container"
	"github.com/pzqf/zMmoServer/MapServer/discovery"
	"github.com/pzqf/zMmoServer/MapServer/health"
	"github.com/pzqf/zMmoServer/MapServer/maps"
	"github.com/pzqf/zMmoServer/MapServer/metrics"
	"github.com/pzqf/zMmoServer/MapServer/net/service"
	"github.com/pzqf/zMmoServer/MapServer/request"
	"github.com/pzqf/zMmoServer/MapServer/utils"
	"github.com/pzqf/zMmoServer/MapServer/version"
	"go.uber.org/zap"
)

const ServerTypeMap zServer.ServerType = "map"

type BaseServer struct {
	*zServer.BaseServer
	*zService.ServiceManager
	isRunning        bool
	container        *container.Container
	metricsManager   *metrics.MetricsManager
	healthChecker    *health.Checker
	dedupManager     *request.DedupManager
	timeoutManager   *request.TimeoutManager
	config           *config.Config
	connManager      *connection.ConnectionManager
	mapManager       *maps.MapManager
	tcpService       *service.TCPService
	serviceDiscovery *discovery.ServiceDiscovery
}

func NewBaseServer(configPath string) *BaseServer {
	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		zLog.Fatal("Failed to load config", zap.Error(err))
	}

	// 创建依赖注入容器
	container := container.NewContainer()

	bs := &BaseServer{
		ServiceManager: zService.NewServiceManager(),
		container:      container,
		metricsManager: metrics.NewMetricsManager(),
		healthChecker:  health.NewChecker(),
		dedupManager:   request.NewDedupManager(5 * time.Minute),
		timeoutManager: request.NewTimeoutManager(),
		config:         cfg,
	}
	bs.BaseServer = zServer.NewBaseServer(ServerTypeMap, "", "Map Server", version.Version, bs)

	// 注册单例到容器
	container.Register("config", cfg)
	container.Register("metricsManager", bs.metricsManager)
	container.Register("healthChecker", bs.healthChecker)
	container.Register("dedupManager", bs.dedupManager)
	container.Register("timeoutManager", bs.timeoutManager)

	bs.registerMetrics()

	// 严格使用 6 位 ServerID（GroupID(4)+ServerIndex(2)）
	serverID, err := id.ParseServerIDInt(int32(cfg.Server.ServerID))
	if err != nil {
		zLog.Fatal("Invalid map ServerID", zap.Error(err))
	}
	serverIDStr := id.ServerIDString(serverID)
	bs.SetId(fmt.Sprintf("map-%s", serverIDStr))

	// 初始化连接管理器
	bs.connManager = connection.NewConnectionManager(cfg)
	container.Register("connManager", bs.connManager)

	// 初始化地图管理器
	bs.mapManager = maps.NewMapManager()
	container.Register("mapManager", bs.mapManager)
	bs.connManager.SetMapHandler(bs.mapManager)

	// 初始化TCP服务
	bs.tcpService = service.NewTCPService(cfg, bs.connManager, bs.mapManager)
	container.Register("tcpService", bs.tcpService)

	// 初始化服务发现（基于 etcd，适配 k8s）
	serviceDiscovery, err := discovery.NewServiceDiscovery(cfg)
	if err != nil {
		zLog.Error("Failed to create service discovery", zap.Error(err))
		return nil
	}
	bs.serviceDiscovery = serviceDiscovery
	container.Register("serviceDiscovery", serviceDiscovery)
	zLog.Info("Using etcd service discovery", zap.String("endpoints", cfg.Etcd.Endpoints))

	return bs
}

func (bs *BaseServer) OnBeforeStart() error {
	zLog.Info("Starting server services...")
	bs.isRunning = true

	// 设置状态为初始化中
	bs.SetState(zServer.StateInitializing, "server initializing")

	// 更新健康检查状态
	bs.healthChecker.UpdateComponentStatus(health.ComponentConfig, health.StatusStarting, "Loading configuration")
	bs.healthChecker.UpdateComponentStatus(health.ComponentContainer, health.StatusStarting, "Initializing container")

	// 严格加载配置表，缺失配置直接启动失败
	if err := tables.GetTableManager().LoadAllTables(); err != nil {
		bs.healthChecker.UpdateComponentStatus(health.ComponentConfig, health.StatusUnhealthy, err.Error())
		return utils.WrapError(err, "failed to load excel tables")
	}
	bs.healthChecker.UpdateComponentStatus(health.ComponentConfig, health.StatusHealthy, "Configuration loaded successfully")

	// 启动监控指标服务
	zLog.Info("Metrics manager initialized")

	mapType, err := bs.loadMapsFromExcelTables()
	if err != nil {
		bs.healthChecker.UpdateComponentStatus(health.ComponentMap, health.StatusUnhealthy, err.Error())
		return utils.WrapError(err, "failed to load maps from map.xlsx")
	}
	bs.healthChecker.UpdateComponentStatus(health.ComponentMap, health.StatusHealthy, "Maps loaded successfully")

	zLog.Info(
		"Map configuration validated",
		zap.String("maps_mode", bs.config.Maps.Mode),
		zap.String("configured_map_ids", intSliceToCSV(bs.config.Maps.MapIDs)),
		zap.String("loaded_map_type", mapType),
		zap.Int("loaded_map_count", bs.mapManager.GetMapCount()),
	)

	// 启动TCP服务
	bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusStarting, "Starting TCP service")
	if err := bs.tcpService.Start(bs.GetContext()); err != nil {
		bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusUnhealthy, err.Error())
		zLog.Error("Failed to start TCP service", zap.Error(err))
	} else {
		bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusHealthy, "TCP service started successfully")
	}

	// 注册当前服务到服务发现
	bs.healthChecker.UpdateComponentStatus(health.ComponentDiscovery, health.StatusStarting, "Registering service")

	mapIDs := bs.mapManager.GetAllMapIDs()
	if len(mapIDs) == 0 && len(bs.config.Maps.MapIDs) > 0 {
		for _, id := range bs.config.Maps.MapIDs {
			mapIDs = append(mapIDs, int32(id))
		}
	}
	bs.serviceDiscovery.UpdateMapIDs(mapIDs)

	if err := bs.serviceDiscovery.Register(); err != nil {
		bs.healthChecker.UpdateComponentStatus(health.ComponentDiscovery, health.StatusUnhealthy, err.Error())
		return utils.WrapError(err, "failed to register service")
	}
	bs.healthChecker.UpdateComponentStatus(health.ComponentDiscovery, health.StatusHealthy, "Service registered successfully")
	zLog.Info("Service registered successfully", zap.String("service_id", bs.serviceDiscovery.GetServerID()))

	go bs.startServiceDiscoveryMonitor()
	go bs.startHeartbeat()

	bs.healthChecker.LogStatus()

	return nil
}

func (bs *BaseServer) OnAfterStart() error {
	// 更新服务状态为就绪
	bs.SetState(zServer.StateReady, "server ready")

	// 更新服务状态为健康
	bs.SetState(zServer.StateHealthy, "server healthy")
	zLog.Info("Map server is healthy")

	// 更新健康检查状态
	bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusHealthy, "Server is healthy")

	return nil
}

func (bs *BaseServer) OnBeforeStop() {
	// 设置状态为流量排空
	bs.SetState(zServer.StateDraining, "server stopping")
	zLog.Info("Map server entering draining state")

	// 更新健康检查状态
	bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusStopping, "Stopping TCP service")

	// 停止TCP服务
	if bs.tcpService != nil {
		bs.tcpService.Stop(bs.GetContext())
	}

	// 停止监控指标服务
	if bs.metricsManager != nil {
		bs.metricsManager.ResetAll()
	}

	// 注销服务
	if bs.serviceDiscovery != nil {
		bs.healthChecker.UpdateComponentStatus(health.ComponentDiscovery, health.StatusStopping, "Unregistering service")
		if err := bs.serviceDiscovery.Unregister(); err != nil {
			zLog.Warn("Failed to unregister service", zap.Error(err))
		} else {
			zLog.Info("Service unregistered successfully", zap.String("service_id", bs.serviceDiscovery.GetServerID()))
		}
		if err := bs.serviceDiscovery.Close(); err != nil {
			zLog.Warn("Failed to close service discovery", zap.Error(err))
		}
		bs.healthChecker.UpdateComponentStatus(health.ComponentDiscovery, health.StatusUnhealthy, "Service unregistered")
	}
}

func (bs *BaseServer) OnAfterStop() {
	// 设置状态为已停止
	bs.SetState(zServer.StateStopped, "server stopped")
	bs.isRunning = false

	// 更新健康检查状态
	bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusUnhealthy, "Server stopped")
	bs.healthChecker.UpdateComponentStatus(health.ComponentContainer, health.StatusUnhealthy, "Container stopped")

	zLog.Info("Map server stopped completely")
}

// startHeartbeat 启动心跳保持
func (bs *BaseServer) startHeartbeat() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bs.GetContext().Done():
			return
		case <-ticker.C:
			currentMapIDs := bs.mapManager.GetAllMapIDs()
			bs.serviceDiscovery.UpdateMapIDs(currentMapIDs)

			if err := bs.serviceDiscovery.UpdateHeartbeat(string(bs.GetState()), 0); err != nil {
				zLog.Warn("Failed to send heartbeat", zap.Error(err))
				bs.healthChecker.UpdateComponentStatus(health.ComponentDiscovery, health.StatusUnhealthy, "Failed to send heartbeat")
			} else {
				bs.healthChecker.UpdateComponentStatus(health.ComponentDiscovery, health.StatusHealthy, "Heartbeat sent successfully")
			}
		}
	}
}

// startServiceDiscoveryMonitor 启动服务发现监听
func (bs *BaseServer) startServiceDiscoveryMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bs.GetContext().Done():
			return
		case <-ticker.C:
			gameServers, err := bs.serviceDiscovery.DiscoverInGroup("game", bs.serviceDiscovery.GetGroupID())
			if err != nil {
				zLog.Warn("Failed to discover game servers", zap.Error(err))
				bs.healthChecker.UpdateComponentStatus(health.ComponentDiscovery, health.StatusUnhealthy, "Failed to discover game servers")
				continue
			}

			zLog.Info("Discovered game servers", zap.Int("count", len(gameServers)))
			bs.healthChecker.UpdateComponentStatus(health.ComponentDiscovery, health.StatusHealthy, "Game servers discovered successfully")

			for _, gameServer := range gameServers {
				if gameServer.Status == "healthy" || gameServer.Status == "ready" {
					zLog.Info("Found healthy GameServer", zap.String("address", gameServer.Address))
				}
			}
		}
	}
}

// loadMapsFromExcelTables 从配置表加载地图
func (bs *BaseServer) loadMapsFromExcelTables() (string, error) {
	// 实现从配置表加载地图的逻辑
	// 这里只是示例，实际实现需要根据具体的配置表结构
	return "excel", nil
}

// intSliceToCSV 将 int 切片转换为 CSV 字符串
func intSliceToCSV(slice []int) string {
	if len(slice) == 0 {
		return ""
	}

	result := fmt.Sprintf("%d", slice[0])
	for _, v := range slice[1:] {
		result += fmt.Sprintf(",%d", v)
	}
	return result
}

// registerMetrics 注册指标
func (bs *BaseServer) registerMetrics() {
	// 注册服务器基本指标
	bs.metricsManager.RegisterCounter("mapserver_requests_total", "Total number of requests", nil)
	bs.metricsManager.RegisterGauge("mapserver_connections", "Current number of connections", nil)
	bs.metricsManager.RegisterGauge("mapserver_players", "Current number of players", nil)
	bs.metricsManager.RegisterGauge("mapserver_maps", "Current number of maps", nil)
	bs.metricsManager.RegisterHistogram("mapserver_request_duration_seconds", "Request duration in seconds", []float64{0.001, 0.01, 0.1, 1, 5, 10}, nil)

	// 注册服务发现指标
	bs.metricsManager.RegisterCounter("mapserver_service_discovery_register_total", "Total number of service discovery register attempts", nil)
	bs.metricsManager.RegisterCounter("mapserver_service_discovery_discover_total", "Total number of service discovery discover attempts", nil)

	// 注册地图指标
	bs.metricsManager.RegisterCounter("mapserver_map_enter_total", "Total number of map enter requests", nil)
	bs.metricsManager.RegisterCounter("mapserver_map_leave_total", "Total number of map leave requests", nil)
	bs.metricsManager.RegisterCounter("mapserver_map_move_total", "Total number of map move requests", nil)
	bs.metricsManager.RegisterCounter("mapserver_map_attack_total", "Total number of map attack requests", nil)
}
