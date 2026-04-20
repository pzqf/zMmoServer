package server

import (
	"fmt"
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
	"github.com/pzqf/zMmoServer/MapServer/connection"
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
	connManager      *connection.ConnectionManager
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

	bs.connManager = connection.NewConnectionManager(bs.config)
	bs.container.Register("connManager", bs.connManager)

	bs.mapManager = maps.NewMapManager()
	bs.container.Register("mapManager", bs.mapManager)
	bs.connManager.SetMapHandler(bs.mapManager)

	bs.tcpService = service.NewTCPService(bs.config, bs.connManager, bs.mapManager)
	bs.container.Register("tcpService", bs.tcpService)

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

	for {
		select {
		case <-bs.GetContext().Done():
			return
		case <-ticker.C:
			now := time.Now()
			deltaTime := now.Sub(lastTime)
			lastTime = now

			bs.mapManager.UpdateAllMapsAI(deltaTime)
			bs.mapManager.UpdateAllMapsBuffs(deltaTime)
			bs.mapManager.UpdateAllMapsPlayers()
			bs.mapManager.UpdateAllMapsSkills()
			bs.mapManager.UpdateAllMapsEvents()
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
			bs.connManager,
		)

		if newMap != nil {
			newMap.SetMaxPlayers(mapCfg.MaxPlayers)
			newMap.SetDescription(mapCfg.Description)
			newMap.SetWeatherType(mapCfg.WeatherType)
			newMap.SetMinLevel(mapCfg.MinLevel)
			newMap.SetMaxLevel(mapCfg.MaxLevel)
			createdCount++
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
