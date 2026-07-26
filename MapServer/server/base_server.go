package server

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/tables"
	zcont "github.com/pzqf/zCommon/container"
	zdisc "github.com/pzqf/zCommon/discovery"
	zreq "github.com/pzqf/zCommon/request"
	util "github.com/pzqf/zCommon/util"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/health"
	"github.com/pzqf/zMmoServer/MapServer/maps"
	"github.com/pzqf/zMmoServer/MapServer/metrics"
	"github.com/pzqf/zMmoServer/MapServer/net/service"
	"github.com/pzqf/zMmoServer/MapServer/version"
	"go.uber.org/zap"
)

const ServerTypeMap zServer.ServerType = "map"

type BaseServer struct {
	*zServer.BaseServer
	isRunning        bool
	container        *zcont.Container
	metricsService   *metrics.Metrics
	healthChecker    *health.Checker
	dedupManager     *zreq.DedupStore
	timeoutManager   *zreq.TimeoutManager
	config           *config.Config
	mapManager       *maps.MapManager
	tcpService       *service.TCPService
	serviceDiscovery *zdisc.ServerServiceDiscovery
}

func NewBaseServer(cfg *config.Config) *BaseServer {
	bs := &BaseServer{
		container:      zcont.NewContainer(),
		metricsService: metrics.NewMetrics(cfg),
		healthChecker:  health.NewChecker(),
		dedupManager:   zreq.NewDedupStore(5 * time.Minute),
		timeoutManager: zreq.NewTimeoutManager(),
		config:         cfg,
	}
	bs.BaseServer = zServer.NewBaseServer(ServerTypeMap, "", "Map Server", version.Version, bs)

	serverID, err := id.ParseServerIDInt(int32(cfg.Server.ServerID))
	if err != nil {
		zLog.Fatal("Invalid map ServerID", zap.Error(err))
	}
	bs.SetId(fmt.Sprintf("map-%s", id.ServerIDString(serverID)))

	bs.initComponents()
	return bs
}

func (bs *BaseServer) initComponents() {
	bs.container.Register("config", bs.config)
	bs.container.Register("metricsService", bs.metricsService)
	bs.container.Register("healthChecker", bs.healthChecker)
	bs.container.Register("dedupManager", bs.dedupManager)
	bs.container.Register("timeoutManager", bs.timeoutManager)

	bs.registerMetrics()

	if err := bs.metricsService.Start(); err != nil {
		zLog.Error("Failed to start metrics service", zap.Error(err))
	}

	bs.mapManager = maps.NewMapManager()
	bs.container.Register("mapManager", bs.mapManager)

	bs.tcpService = service.NewTCPService(bs.config, bs.mapManager)
	bs.container.Register("tcpService", bs.tcpService)

	// Phase 3.5：把 zMetrics 的网络指标接入 zNet——连接/流量/解码错误随之进入
	// Prometheus（经既有 metricsService 暴露），trace_id 已在跨服 Envelope 全链路贯穿。
	if bs.metricsService != nil {
		bs.tcpService.SetMetricsRecorder(bs.metricsService.GetMetricsManager().GetNetworkMetrics())
	}

	// AOI 回程接线（Phase 2.3）：① 创建并注入 playerID→GameServer 映射（PlayerGameServerManager
	// 此前定义了却从未接线，导致映射从未填充）；② 把网络层作为 AOI 通知器注入地图管理器，
	// 使 Map.handleAOIEvent 能把视野事件回传给拥有 watcher 的 GameServer。
	bs.tcpService.SetPlayerGameServerManager(maps.NewPlayerGameServerManager())
	bs.mapManager.SetAOINotifier(bs.tcpService)

	sd, err := zdisc.NewServerServiceDiscovery(&zdisc.ServerServiceDiscoveryConfig{
		ServiceType: "map",
		ServerID:    int32(bs.config.Server.ServerID),
		ListenAddr:  bs.config.Server.ListenAddr,
		Etcd:        &bs.config.Etcd,
	})
	if err != nil {
		zLog.Error("Failed to create service discovery", zap.Error(err))
		return
	}
	bs.serviceDiscovery = sd
	bs.container.Register("serviceDiscovery", sd)
}

func (bs *BaseServer) OnBeforeStart() error {
	bs.isRunning = true
	bs.SetState(zServer.StateInitializing, "server initializing")

	bs.healthChecker.UpdateComponentStatus(health.ComponentConfig, health.StatusStarting, "Loading configuration")
	bs.healthChecker.UpdateComponentStatus(health.ComponentContainer, health.StatusStarting, "Initializing container")

	if err := tables.GetTableManager().LoadAllTables(); err != nil {
		bs.healthChecker.UpdateComponentStatus(health.ComponentConfig, health.StatusUnhealthy, err.Error())
		return util.WrapError(err, "failed to load excel tables")
	}
	bs.healthChecker.UpdateComponentStatus(health.ComponentConfig, health.StatusHealthy, "Configuration loaded successfully")

	mapType, err := bs.loadMapsFromExcelTables()
	if err != nil {
		bs.healthChecker.UpdateComponentStatus(health.ComponentMap, health.StatusUnhealthy, err.Error())
		return util.WrapError(err, "failed to load maps from map.xlsx")
	}
	bs.healthChecker.UpdateComponentStatus(health.ComponentMap, health.StatusHealthy, "Maps loaded successfully")

	zLog.Info("Map configuration validated",
		zap.String("maps_mode", bs.config.Maps.Mode),
		zap.String("configured_map_ids", intSliceToCSV(bs.config.Maps.MapIDs)),
		zap.String("loaded_map_type", mapType),
		zap.Int("loaded_map_count", bs.mapManager.GetMapCount()),
	)

	bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusStarting, "Starting TCP service")
	if err := bs.tcpService.Start(bs.GetContext()); err != nil {
		bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusUnhealthy, err.Error())
		zLog.Error("Failed to start TCP service", zap.Error(err))
	} else {
		bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusHealthy, "TCP service started successfully")
	}

	if err := bs.registerServiceDiscovery(); err != nil {
		return err
	}

	go bs.startServiceDiscoveryMonitor()
	go bs.startHeartbeat()
	go bs.startGameLoop()

	bs.healthChecker.LogStatus()
	return nil
}

func (bs *BaseServer) registerServiceDiscovery() error {
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
		return util.WrapError(err, "failed to register service")
	}
	bs.healthChecker.UpdateComponentStatus(health.ComponentDiscovery, health.StatusHealthy, "Service registered successfully")
	zLog.Info("Service registered successfully", zap.String("service_id", bs.serviceDiscovery.GetServerID()))
	return nil
}

func (bs *BaseServer) OnAfterStart() error {
	bs.SetState(zServer.StateReady, "server ready")
	bs.SetState(zServer.StateHealthy, "server healthy")
	zLog.Info("Map server is healthy")
	bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusHealthy, "Server is healthy")
	return nil
}

func (bs *BaseServer) OnBeforeStop() {
	bs.SetState(zServer.StateDraining, "server stopping")
	zLog.Info("Map server entering draining state")
	bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusStopping, "Stopping TCP service")

	if bs.tcpService != nil {
		bs.tcpService.Stop(bs.GetContext())
	}
	if bs.metricsService != nil {
		bs.metricsService.GetMetricsManager().ResetAll()
	}
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
	bs.SetState(zServer.StateStopped, "server stopped")
	bs.isRunning = false
	bs.healthChecker.UpdateComponentStatus(health.ComponentTCP, health.StatusUnhealthy, "Server stopped")
	bs.healthChecker.UpdateComponentStatus(health.ComponentContainer, health.StatusUnhealthy, "Container stopped")
	zLog.Info("Map server stopped completely")
}

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

func (bs *BaseServer) startGameLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lastTime := time.Now()
	var reapAccum time.Duration // 实例地图空置回收节流：累计到 reapInterval 才扫一次

	const reapInterval = 5 * time.Second   // 每 5s 扫一次空置实例
	const reapGrace = 30 * time.Second     // 实例空置满 30s 才回收

	for {
		select {
		case <-bs.GetContext().Done():
			return
		case <-ticker.C:
			now := time.Now()
			deltaTime := now.Sub(lastTime)
			lastTime = now

			// MAP-2 单写者模型：不再在本主循环 goroutine 上直接跑各地图的帧更新
			// （那样会与网络分发 goroutine 并发改动同一批对象 → 数据竞争），改为向每张
			// 地图各自的 goroutine 投递一帧,由该 goroutine 串行执行 AI/Buff/玩家/技能/事件更新。
			bs.mapManager.PostTickAll(deltaTime)

			// 空置实例地图回收（分线/战场/跨服）：与 PostTickAll 同 goroutine 串行，避免销毁地图
			// 与 tick 投递并发。副本不在此列（由显式销毁/自身生命周期管）。建议③。
			reapAccum += deltaTime
			if reapAccum >= reapInterval {
				reapAccum = 0
				bs.mapManager.ReapEmpty(reapGrace)
			}
		}
	}
}

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

			for _, gs := range gameServers {
				if gs.Status == "healthy" || gs.Status == "ready" {
					zLog.Info("Found healthy GameServer", zap.String("address", gs.Address))
				}
			}
		}
	}
}

func (bs *BaseServer) loadMapsFromExcelTables() (string, error) {
	tm := tables.GetTableManager()
	mapLoader := tm.GetMapLoader()
	allMaps := mapLoader.GetAllMaps()

	if len(allMaps) == 0 {
		zLog.Warn("No maps found in map.xlsx, skipping map creation")
		return "excel", nil
	}

	configuredMapIDs := bs.config.Maps.MapIDs
	createdCount := 0

	for _, mapCfg := range allMaps {
		if len(configuredMapIDs) > 0 {
			found := false
			for _, id := range configuredMapIDs {
				if int32(id) == mapCfg.MapID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		mapID := id.MapIdType(mapCfg.MapID)
		newMap := bs.mapManager.CreateMap(
			mapID,
			mapCfg.MapID,
			mapCfg.Name,
			float32(mapCfg.Width),
			float32(mapCfg.Height),
		)

		if newMap != nil {
			newMap.SetMaxPlayers(mapCfg.MaxPlayers)
			newMap.SetDescription(mapCfg.Description)
			newMap.SetWeatherType(mapCfg.WeatherType)
			newMap.SetMinLevel(mapCfg.MinLevel)
			newMap.SetMaxLevel(mapCfg.MaxLevel)
			createdCount++

			// 分线测试夹具（env ZMMO_TEST_LAYER=<mapID>）：把该逻辑图标为可分线、softCap=1，
			// 供"两客户端进同图分到不同层"的 E2E。生产应由 mapconfig 决定 layerable + soft/hardCap。
			if os.Getenv("ZMMO_TEST_LAYER") == strconv.Itoa(int(mapCfg.MapID)) {
				bs.mapManager.EnableLayering().RegisterLayerable(mapID, maps.LayerConfig{
					MapConfigID: mapCfg.MapID,
					Name:        mapCfg.Name,
					Width:       float32(mapCfg.Width),
					Height:      float32(mapCfg.Height),
					SoftCap:     1,
					HardCap:     2,
				})
				zLog.Info("Test layerable map registered (ZMMO_TEST_LAYER)", zap.Int32("map_id", mapCfg.MapID))
			}
		}
	}

	zLog.Info("Maps created from excel tables", zap.Int("count", createdCount))
	return "excel", nil
}

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

func (bs *BaseServer) registerMetrics() {
	zLog.Info("Map metrics registered via MetricsService")
}
