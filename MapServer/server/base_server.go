package server

import (
	"context"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zInject"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zService"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoServer/MapServer/maps"
	"github.com/pzqf/zMmoServer/MapServer/net/service"
	"github.com/pzqf/zMmoShared/metrics"
	"go.uber.org/zap"
)

type BaseServer struct {
	*zService.ServiceManager
	wg             sync.WaitGroup
	isRunning      bool
	startCalled    bool
	metricsManager *metrics.MetricsManager
	config         *config.Config
	connManager    *connection.ConnectionManager
	mapManager     *maps.MapManager
	tcpService     *service.TCPService
}

func NewBaseServer() *BaseServer {
	// 加载配置
	cfg, err := config.LoadConfig("config.ini")
	if err != nil {
		zLog.Fatal("Failed to load config", zap.Error(err))
	}

	bs := &BaseServer{
		ServiceManager: zService.NewServiceManager(),
		metricsManager: metrics.NewMetricsManager(),
		config:         cfg,
	}

	bs.RegisterSingleton("config", cfg)
	bs.RegisterSingleton("metricsManager", bs.metricsManager)
	bs.registerMetrics()

	// 初始化连接管理器
	bs.connManager = connection.NewConnectionManager(cfg)
	bs.RegisterSingleton("connManager", bs.connManager)

	// 初始化地图管理器
	bs.mapManager = maps.NewMapManager()
	bs.RegisterSingleton("mapManager", bs.mapManager)

	// 初始化TCP服务
	bs.tcpService = service.NewTCPService(cfg, bs.connManager)
	bs.RegisterSingleton("tcpService", bs.tcpService)

	// 创建默认地图
	bs.createDefaultMaps()

	return bs
}

func (bs *BaseServer) Start() error {
	if bs.startCalled {
		return nil
	}

	bs.wg.Add(1)
	bs.startCalled = true

	zLog.Info("Starting server services...")

	// 启动TCP服务
	if err := bs.tcpService.Start(context.Background()); err != nil {
		zLog.Error("Failed to start TCP service", zap.Error(err))
	}

	// 连接到 GameServer（示例）
	// 实际应用中，应该从配置或服务发现获取 GameServer 列表
	if err := bs.connManager.ConnectToGameServer("game1", bs.config.GameServer.GameServerAddr); err != nil {
		zLog.Warn("Failed to connect to GameServer, will retry later", zap.Error(err))
	} else {
		// 注册地图到 GameServer
		bs.connManager.RegisterMapToGameServer(1, "game1")
		bs.connManager.RegisterMapToGameServer(2, "game1")
		bs.connManager.RegisterMapToGameServer(3, "game1")
	}

	// 示例：连接到第二个 GameServer（跨服场景）
	// if err := bs.connManager.ConnectToGameServer("game2", "127.0.0.1:9003"); err != nil {
	// 	zLog.Warn("Failed to connect to GameServer 2, will retry later", zap.Error(err))
	// } else {
	// 	// 注册跨服地图
	// 	bs.connManager.RegisterMapToGameServer(100, "game2")
	// }

	bs.ServeServices()

	bs.isRunning = true

	// 启动事件更新循环
	go bs.eventUpdateLoop()

	return nil
}

func (bs *BaseServer) Stop() {
	if !bs.isRunning {
		return
	}

	zLog.Info("Stopping server...")
	bs.CloseServices()

	// 断开与 GameServer 的连接
	if bs.connManager != nil {
		bs.connManager.DisconnectFromGameServer("game1")
		// bs.connManager.DisconnectFromGameServer("game2") // 断开第二个 GameServer 连接
	}

	bs.isRunning = false
	if bs.startCalled {
		bs.wg.Done()
		bs.startCalled = false
	}
}

func (bs *BaseServer) Wait() {
	bs.wg.Wait()
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
		select {
		case <-ticker.C:
			// 更新所有地图的事件
			bs.mapManager.UpdateAllMapsEvents()
			// 更新所有地图的技能
			bs.mapManager.UpdateAllMapsSkills()
		}
	}

	zLog.Info("Event update loop stopped")
}

// createDefaultMaps 创建默认地图
func (bs *BaseServer) createDefaultMaps() {
	// 创建默认地图 1
	map1 := bs.mapManager.CreateMap(1, 1, "新手村", 1000.0, 1000.0, bs.connManager)
	zLog.Info("Created default map", zap.Int32("map_id", 1), zap.String("name", map1.GetName()))

	// 创建默认地图 2
	map2 := bs.mapManager.CreateMap(2, 2, "主城", 2000.0, 2000.0, bs.connManager)
	zLog.Info("Created default map", zap.Int32("map_id", 2), zap.String("name", map2.GetName()))

	// 创建默认地图 3
	map3 := bs.mapManager.CreateMap(3, 3, "野外", 3000.0, 3000.0, bs.connManager)
	zLog.Info("Created default map", zap.Int32("map_id", 3), zap.String("name", map3.GetName()))

	// 创建跨服地图 100
	map100 := bs.mapManager.CreateMap(100, 100, "跨服战场", 5000.0, 5000.0, bs.connManager)
	zLog.Info("Created cross-server map", zap.Int32("map_id", 100), zap.String("name", map100.GetName()))

	// 地图配置通过zMmoShared的表格系统加载
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
