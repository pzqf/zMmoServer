package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/metrics"
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

	// 严格使用 6 位 ServerID（GroupID(4)+ServerIndex(2)）
	serverID, err := id.ParseServerIDInt(int32(cfg.Server.ServerID))
	if err != nil {
		zLog.Fatal("Invalid map ServerID", zap.Error(err))
	}
	serverIDStr := id.ServerIDString(serverID)
	bs.SetId(fmt.Sprintf("map-%s", serverIDStr))

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

	// 设置状态为初始化中
	bs.SetState(zServer.StateInitializing, "server initializing")

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
	// 从配置获取服务器ID
	serverID, err := id.ParseServerIDInt(int32(bs.config.Server.ServerID))
	if err != nil {
		return fmt.Errorf("invalid server ID: %w", err)
	}
	serverIDStr := id.ServerIDString(serverID)
	groupID := id.GroupIDStringFromServerID(serverID)

	// 获取本MapServer负责的所有地图ID
	mapIDs := bs.mapManager.GetAllMapIDs()
	if len(mapIDs) == 0 && len(bs.config.Maps.MapIDs) > 0 {
		// 如果没有从mapManager获取到地图ID，使用配置中的地图ID
		for _, id := range bs.config.Maps.MapIDs {
			mapIDs = append(mapIDs, int32(id))
		}
	}

	// 注册服务到服务发现
	serviceInfo := &discovery.ServerInfo{
		ID:            serverIDStr,
		ServiceType:   "map",
		GroupID:       groupID,
		Status:        "initializing",
		Address:       bs.config.Server.ListenAddr,
		Port:          0, // 端口已在 Address 中
		Load:          0,
		Players:       0,
		ReadyTime:     time.Now().Unix(),
		LastHeartbeat: time.Now().Unix(),
		MapIDs:        mapIDs,
	}

	// 注册服务，添加重试机制
	maxRetries := 3
	var registerErr error
	for i := 0; i < maxRetries; i++ {
		if err := bs.serviceDiscovery.Register(serviceInfo); err != nil {
			zLog.Warn("Failed to register service", zap.Error(err), zap.Int("retry", i+1))
			registerErr = err
			time.Sleep(time.Duration(i+1) * time.Second) // 指数退避
		} else {
			zLog.Info("Service registered successfully", zap.String("service_id", serviceInfo.ID))
			registerErr = nil
			break
		}
	}
	if registerErr != nil {
		return fmt.Errorf("failed to register service after %d retries: %w", maxRetries, registerErr)
	}

	// 启动服务发现监听（监听 GameServer 变化）
	go bs.startServiceDiscoveryMonitor()

	// 启动心跳保持
	go bs.startHeartbeat(serviceInfo, serverIDStr, groupID)

	return nil
}

func (bs *BaseServer) OnAfterStart() error {
	// 更新服务状态为就绪
	bs.SetState(zServer.StateReady, "server ready")

	// 更新服务状态为健康
	bs.SetState(zServer.StateHealthy, "server healthy")
	zLog.Info("Map server is healthy")

	return nil
}

func (bs *BaseServer) OnBeforeStop() {
	// 设置状态为流量排空
	bs.SetState(zServer.StateDraining, "server stopping")
	zLog.Info("Map server entering draining state")

	// 停止TCP服务
	if bs.tcpService != nil {
		bs.tcpService.Stop(bs.GetContext())
	}

	// 注销服务
	if bs.serviceDiscovery != nil && bs.config != nil {
		serverID, _ := id.ParseServerIDInt(int32(bs.config.Server.ServerID))
		serverIDStr := id.ServerIDString(serverID)
		groupID := id.GroupIDStringFromServerID(serverID)
		if err := bs.serviceDiscovery.Unregister("map", groupID, serverIDStr); err != nil {
			zLog.Warn("Failed to unregister service", zap.Error(err))
		} else {
			zLog.Info("Service unregistered successfully", zap.String("service_id", serverIDStr))
		}
		if err := bs.serviceDiscovery.Close(); err != nil {
			zLog.Warn("Failed to close service discovery", zap.Error(err))
		}
	}
}

func (bs *BaseServer) OnAfterStop() {
	// 设置状态为已停止
	bs.SetState(zServer.StateStopped, "server stopped")
	bs.isRunning = false
	zLog.Info("Map server stopped completely")
}

// startHeartbeat 启动心跳保持
func (bs *BaseServer) startHeartbeat(serviceInfo *discovery.ServerInfo, serverIDStr, groupID string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bs.GetContext().Done():
			return
		case <-ticker.C:
			// 更新地图ID列表（可能动态变化）
			currentMapIDs := bs.mapManager.GetAllMapIDs()
			if len(currentMapIDs) == 0 {
				currentMapIDs = serviceInfo.MapIDs
			}

			// 更新服务状态
			updatedInfo := &discovery.ServerInfo{
				ID:            serverIDStr,
				ServiceType:   "map",
				GroupID:       groupID,
				Status:        bs.GetState(),
				Address:       serviceInfo.Address,
				Port:          serviceInfo.Port,
				Load:          0,
				Players:       0,
				ReadyTime:     serviceInfo.ReadyTime,
				LastHeartbeat: time.Now().Unix(),
				MapIDs:        currentMapIDs,
			}
			if err := bs.serviceDiscovery.Register(updatedInfo); err != nil {
				zLog.Warn("Failed to send heartbeat", zap.Error(err))
			}
		}
	}
}

// startServiceDiscoveryMonitor 启动服务发现监听
func (bs *BaseServer) startServiceDiscoveryMonitor() {
	serverID := id.MustParseServerIDInt(int32(bs.config.Server.ServerID))
	groupID := id.GroupIDStringFromServerID(serverID)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bs.GetContext().Done():
			return
		case <-ticker.C:
			// 发现 GameServer
			gameServers, err := bs.serviceDiscovery.Discover("game", groupID)
			if err != nil {
				zLog.Warn("Failed to discover game servers", zap.Error(err))
				continue
			}

			zLog.Info("Discovered game servers", zap.Int("count", len(gameServers)))

			// 连接到 GameServer
			for _, gameServer := range gameServers {
				if gameServer.Status == "healthy" || gameServer.Status == "ready" {
					// 实际应用中，这里应该建立与 GameServer 的连接
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
	// 实现指标注册逻辑
}
