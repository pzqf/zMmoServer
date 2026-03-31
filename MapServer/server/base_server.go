package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zEngine/zInject"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zEngine/zService"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoServer/MapServer/maps"
	"github.com/pzqf/zMmoServer/MapServer/net/service"
	"github.com/pzqf/zMmoServer/MapServer/version"
	"go.uber.org/zap"
)

const ServerTypeMap zServer.ServerType = "map"

type BaseServer struct {
	*zServer.BaseServer
	*zService.ServiceManager
	isRunning        bool
	metricsManager   *metrics.MetricsManager
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

	bs := &BaseServer{
		ServiceManager: zService.NewServiceManager(),
		metricsManager: metrics.NewMetricsManager(),
		config:         cfg,
	}
	bs.BaseServer = zServer.NewBaseServer(ServerTypeMap, "", "Map Server", version.Version, bs)

	bs.RegisterSingleton("config", cfg)
	bs.RegisterSingleton("metricsManager", bs.metricsManager)
	bs.registerMetrics()

	// 初始化连接管理器
	bs.connManager = connection.NewConnectionManager(cfg)
	bs.RegisterSingleton("connManager", bs.connManager)

	// 初始化地图管理器
	bs.mapManager = maps.NewMapManager()
	bs.RegisterSingleton("mapManager", bs.mapManager)
	bs.connManager.SetMapHandler(bs.mapManager)

	// 初始化TCP服务
	bs.tcpService = service.NewTCPService(cfg, bs.connManager, bs.mapManager)
	bs.RegisterSingleton("tcpService", bs.tcpService)

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
		return nil
	}
	bs.serviceDiscovery = serviceDiscovery
	bs.RegisterSingleton("serviceDiscovery", serviceDiscovery)
	zLog.Info("Using etcd service discovery", zap.Strings("endpoints", etcdEndpoints))

	return bs
}

func (bs *BaseServer) OnBeforeStart() error {
	zLog.Info("Starting server services...")
	bs.isRunning = true

	// 严格加载配置表，缺失配置直接启动失败
	if err := tables.GetTableManager().LoadAllTables(); err != nil {
		return fmt.Errorf("failed to load excel tables: %w", err)
	}

	mapType, err := bs.loadMapsFromExcelTables()
	if err != nil {
		return fmt.Errorf("failed to load maps from map.xlsx: %w", err)
	}
	zLog.Info(
		"Map configuration validated",
		zap.String("maps_mode", bs.config.Maps.Mode),
		zap.String("configured_map_ids", intSliceToCSV(bs.config.Maps.MapIDs)),
		zap.String("loaded_map_type", mapType),
		zap.Int("loaded_map_count", bs.mapManager.GetMapCount()),
	)

	// 启动TCP服务
	if err := bs.tcpService.Start(bs.GetContext()); err != nil {
		zLog.Error("Failed to start TCP service", zap.Error(err))
	}

	// 连接到GameServer（示例）
	// 实际应用中，应该从配置或服务发现获取 GameServer 列表
	// 注册当前服务到服务发现（严格使用6位ServerID）
	serverID, err := id.ParseServerIDInt(int32(bs.config.Server.ServerID))
	if err != nil {
		return fmt.Errorf("invalid map ServerID %d: %w", bs.config.Server.ServerID, err)
	}
	serverIDStr := id.ServerIDString(serverID)
	groupID := id.GroupIDStringFromServerID(serverID)
	bs.SetId(fmt.Sprintf("map-%s", serverIDStr))

	// 获取地图信息
	mapList := bs.mapManager.GetAllMaps()
	mapIDs := make([]string, 0, len(mapList))
	for _, m := range mapList {
		mapIDs = append(mapIDs, fmt.Sprintf("%d", m.GetID()))
	}

	serviceInfo := &discovery.ServerInfo{
		ID:        fmt.Sprintf("map-%s", serverIDStr),
		GroupID:   groupID,
		Status:    discovery.ServerStatusHealthy,
		Address:   bs.config.Server.ListenAddr,
		Port:      0, // 地图服端口已在Address中包含
		Load:      0,
		Players:   0,
		ReadyTime: time.Now().Unix(),
	}

	if err := bs.serviceDiscovery.Register(serviceInfo); err != nil {
		zLog.Warn("Failed to register service", zap.Error(err))
	} else {
		zLog.Info("Service registered successfully",
			zap.String("service_id", serviceInfo.ID),
			zap.String("address", serviceInfo.Address))

		// 启动心跳保持
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for bs.isRunning {
				<-ticker.C
				// 重新创建serviceInfo并更新字段
				updatedInfo := &discovery.ServerInfo{
					ID:        serviceInfo.ID,
					GroupID:   serviceInfo.GroupID,
					Status:    discovery.ServerStatusHealthy,
					Address:   serviceInfo.Address,
					Port:      serviceInfo.Port,
					Load:      0,
					Players:   0,
					ReadyTime: serviceInfo.ReadyTime,
				}
				if err := bs.serviceDiscovery.Register(updatedInfo); err != nil {
					zLog.Warn("Failed to keep service alive", zap.Error(err))
				}
			}
		}()
	}

	// 从服务发现获取GameServer列表
	gameServers, err := bs.serviceDiscovery.Discover("game", groupID)
	ownedMapIDs := bs.getOwnedMapIDs()
	if err != nil {
		zLog.Warn("Failed to discover game servers, using config address", zap.Error(err))
		// 如果服务发现失败，使用配置的地址
		gameID := serviceIDFromAddress("game", bs.config.GameServer.GameServerAddr)
		if err := bs.connManager.ConnectToGameServer(gameID, bs.config.GameServer.GameServerAddr); err != nil {
			zLog.Warn("Failed to connect to GameServer, will retry later", zap.Error(err))
		} else {
			for _, mapID := range ownedMapIDs {
				bs.connManager.RegisterMapToGameServer(mapID, gameID)
			}
		}
	} else {
		// 使用服务发现获取的GameServer地址
		for _, gameServer := range gameServers {
			gameID := gameServer.ID
			if gameID == "" {
				gameID = serviceIDFromAddress("game", gameServer.Address)
			}
			if err := bs.connManager.ConnectToGameServer(gameID, gameServer.Address); err != nil {
				zLog.Warn("Failed to connect to GameServer", zap.Error(err), zap.String("address", gameServer.Address))
			} else {
				for _, mapID := range ownedMapIDs {
					bs.connManager.RegisterMapToGameServer(mapID, gameID)
				}
				zLog.Info("Connected to GameServer", zap.String("address", gameServer.Address))
			}
		}
	}

	// 示例：连接到第二台GameServer（跨服场景）
	// if err := bs.connManager.ConnectToGameServer("game2", "127.0.0.1:9003"); err != nil {
	// 	zLog.Warn("Failed to connect to GameServer 2, will retry later", zap.Error(err))
	// } else {
	// 	// 注册跨服地图
	// 	bs.connManager.RegisterMapToGameServer(100, "game2")
	// }

	// 启动事件更新循环
	go bs.eventUpdateLoop()

	return nil
}

func (bs *BaseServer) OnAfterStart() error {
	return nil
}

func (bs *BaseServer) OnBeforeStop() {
	bs.stopInternal()
}

func (bs *BaseServer) Start() error {
	return bs.BaseServer.Start()
}

func (bs *BaseServer) Run() error {
	return bs.BaseServer.Run()
}

func (bs *BaseServer) Stop() {
	bs.BaseServer.Stop()
}

func (bs *BaseServer) stopInternal() {
	if !bs.isRunning {
		return
	}

	zLog.Info("Stopping server...")

	// 断开与GameServer 的连接
	if bs.connManager != nil {
		for _, gameServerID := range bs.connManager.GetConnectedGameServerIDs() {
			bs.connManager.DisconnectFromGameServer(gameServerID)
		}
	}

	// 注销服务发现
	if bs.serviceDiscovery != nil {
		if bs.config != nil {
			serverID, parseErr := id.ParseServerIDInt(int32(bs.config.Server.ServerID))
			if parseErr != nil {
				zLog.Warn("Skip unregister due to invalid map ServerID", zap.Error(parseErr))
			} else {
				serviceID := id.ServerIDString(serverID)
				// 使用从服务器ID中提取的groupID
				groupID := id.GroupIDStringFromServerID(id.ServerIdType(bs.config.Server.GroupID))
				if err := bs.serviceDiscovery.Unregister("map", groupID, serviceID); err != nil {
					zLog.Warn("Failed to unregister service", zap.Error(err))
				} else {
					zLog.Info("Service unregistered successfully", zap.String("service_id", serviceID))
				}
			}
		}
		if err := bs.serviceDiscovery.Close(); err != nil {
			zLog.Warn("Failed to close service discovery", zap.Error(err))
		}
	}

	bs.isRunning = false
}

func (bs *BaseServer) getOwnedMapIDs() []int {
	mapList := bs.mapManager.GetAllMaps()
	mapIDs := make([]int, 0, len(mapList))
	for _, m := range mapList {
		mapIDs = append(mapIDs, int(m.GetID()))
	}
	return mapIDs
}

func (bs *BaseServer) loadMapsFromExcelTables() (string, error) {
	mapLoader := tables.GetTableManager().GetMapLoader()
	if mapLoader == nil {
		return "", fmt.Errorf("map table loader is nil")
	}

	allMaps := mapLoader.GetAllMaps()
	if len(allMaps) == 0 {
		return "", fmt.Errorf("no map configs found in map.xlsx")
	}

	allowed := make(map[int]struct{})
	if len(bs.config.Maps.MapIDs) > 0 {
		for _, mapID := range bs.config.Maps.MapIDs {
			allowed[mapID] = struct{}{}
		}
	}

	loaded := 0
	expectedMapType := expectedMapTypeFromMode(bs.config.Maps.Mode)
	mergedMapType := ""
	for mapID, mapCfg := range allMaps {
		if len(allowed) > 0 {
			if _, ok := allowed[int(mapID)]; !ok {
				continue
			}
		}

		currentType := mapTypeLabel(mapCfg.MapType)
		if expectedMapType != "" && currentType != expectedMapType {
			return "", fmt.Errorf(
				"map %d type mismatch: Maps.Mode=%s expects %s, but map.xlsx has %s",
				mapID,
				bs.config.Maps.Mode,
				expectedMapType,
				currentType,
			)
		}

		newMap := bs.mapManager.CreateMap(
			id.MapIdType(mapID),
			mapCfg.MapID,
			mapCfg.Name,
			float32(mapCfg.Width),
			float32(mapCfg.Height),
			bs.connManager,
		)
		newMap.SetMaxPlayers(mapCfg.MaxPlayers)
		newMap.SetDescription(mapCfg.Description)
		newMap.SetWeatherType(mapCfg.WeatherType)
		newMap.SetMinLevel(mapCfg.MinLevel)
		newMap.SetMaxLevel(mapCfg.MaxLevel)
		loaded++

		if mergedMapType == "" {
			mergedMapType = currentType
		} else if mergedMapType != currentType {
			mergedMapType = "mixed"
		}
	}

	if loaded == 0 {
		return "", fmt.Errorf("no maps matched config Maps.MapIDs")
	}
	if mergedMapType == "" {
		mergedMapType = "normal"
	}

	zLog.Info("Maps loaded from map.xlsx", zap.Int("count", loaded), zap.String("map_type", mergedMapType))
	return mergedMapType, nil
}

func mapTypeLabel(mapType int32) string {
	switch mapType {
	case 1:
		return "single_server"
	case 2:
		return "mirror"
	case 3:
		return "cross_group"
	default:
		return "normal"
	}
}

func expectedMapTypeFromMode(mode string) string {
	switch mode {
	case config.MapModeSingleServer:
		return "single_server"
	case config.MapModeMirror:
		return "mirror"
	case config.MapModeCrossGroup:
		return "cross_group"
	default:
		return ""
	}
}

func intSliceToCSV(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ids))
	for _, idVal := range ids {
		parts = append(parts, fmt.Sprintf("%d", idVal))
	}
	return strings.Join(parts, ",")
}

func serviceIDFromAddress(prefix, addr string) string {
	cleanAddr := strings.NewReplacer(":", "_", ".", "_", "/", "_", "\\", "_").Replace(addr)
	return fmt.Sprintf("%s-%s", prefix, cleanAddr)
}

func (bs *BaseServer) RegisterDependency(name string, factory interface{}) {
	bs.ServiceManager.RegisterDependency(name, factory)
}

func (bs *BaseServer) RegisterSingleton(name string, instance interface{}) {
	bs.ServiceManager.RegisterSingleton(name, instance)
}

func (bs *BaseServer) ResolveDependency(name string) (interface{}, error) {
	return bs.ServiceManager.ResolveDependency(name)
}

func (bs *BaseServer) GetContainer() zInject.Container {
	return bs.ServiceManager.GetContainer()
}

// eventUpdateLoop 事件更新循环
func (bs *BaseServer) eventUpdateLoop() {
	zLog.Info("Starting event update loop")

	ticker := time.NewTicker(100 * time.Millisecond) // 100ms 一次更新
	defer ticker.Stop()

	for bs.isRunning {
		<-ticker.C
		// 更新所有地图的事件
		bs.mapManager.UpdateAllMapsEvents()
		// 更新所有地图的技能
		bs.mapManager.UpdateAllMapsSkills()
	}

	zLog.Info("Event update loop stopped")
}

// registerMetrics 注册监控指标
func (bs *BaseServer) registerMetrics() {
	// 注册地图相关指标
	bs.metricsManager.RegisterGauge("map_server_active_maps", "Number of active maps", nil)
	bs.metricsManager.RegisterGauge("map_server_total_game_objects", "Total number of game objects", nil)
	bs.metricsManager.RegisterGauge("map_server_active_players", "Number of active players on maps", nil)

	// 注册系统指标
	bs.metricsManager.RegisterGauge("map_server_uptime_seconds", "Map server uptime in seconds", nil)
	bs.metricsManager.RegisterCounter("map_server_total_spawns", "Total number of spawns", nil)

	// 注册网络指标
	bs.metricsManager.RegisterGauge("map_server_active_connections", "Number of active network connections", nil)
	bs.metricsManager.RegisterCounter("map_server_total_packets", "Total number of network packets", nil)

	zLog.Info("Map server metrics registered successfully")
}
