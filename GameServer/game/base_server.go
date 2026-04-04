package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/db"
	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	"github.com/pzqf/zMmoServer/GameServer/game/maps"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"github.com/pzqf/zMmoServer/GameServer/handler"
	"github.com/pzqf/zMmoServer/GameServer/metrics"
	"github.com/pzqf/zMmoServer/GameServer/net/protolayer"
	tcpservice "github.com/pzqf/zMmoServer/GameServer/net/service"
	playerservice "github.com/pzqf/zMmoServer/GameServer/service"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"go.uber.org/zap"
)

// ServerType 游戏服类型
const ServerTypeGame zServer.ServerType = "game"

// BaseServer 游戏服基础服务

type BaseServer struct {
	*zServer.BaseServer
	Config            *config.Config
	TCPService        *tcpservice.TCPService
	ConnectionManager *connection.ConnectionManager
	SessionManager    *session.SessionManager
	PlayerManager     *player.PlayerManager
	PlayerHandler     *handler.PlayerHandler
	MapService        *maps.MapService
	Protocol          protolayer.Protocol
	PlayerService     *playerservice.PlayerService
	DBConnector       connector.DBConnector
	PlayerDAO         *dao.PlayerDAO
	Metrics           *metrics.Metrics
	ServiceDiscovery  *discovery.ServiceDiscovery
}

// NewBaseServer 创建游戏服基础服务
func NewBaseServer() *BaseServer {
	// 先创建子类实例
	gs := &BaseServer{}

	// 创建基础服务器，传入子类作为 hooks
	baseServer := zServer.NewBaseServer(
		ServerTypeGame,
		"", // 在 OnBeforeStart 中根据配置设置服务ID
		"Game Server",
		"1.0.0",
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

	// 严格使用 6 位 ServerID（GroupID(4)+ServerIndex(2)）
	serverID, err := id.ParseServerIDInt(int32(cfg.Server.ServerID))
	if err != nil {
		return fmt.Errorf("invalid game ServerID %d: %w", cfg.Server.ServerID, err)
	}
	serverIDStr := id.ServerIDString(serverID)
	s.SetId(fmt.Sprintf("game-%s", serverIDStr))

	// 初始化连接管理器
	connManager := connection.NewConnectionManager(cfg)
	s.ConnectionManager = connManager

	// 初始化会话管理器
	sessionManager := session.NewSessionManager()
	s.SessionManager = sessionManager

	// 设置状态为初始化中
	s.SetState(zServer.StateInitializing, "server initializing")

	// 初始化数据库连接器
	dbConfig := connector.DBConfig{
		Host:     cfg.Database.DBHost,
		Port:     cfg.Database.DBPort,
		User:     cfg.Database.DBUser,
		Password: cfg.Database.DBPassword,
		DBName:   cfg.Database.DBName,
		Driver:   cfg.Database.DBType,
	}
	dbConnector := connector.NewDBConnector("game", dbConfig.Driver, 1000)
	if err := dbConnector.Init(dbConfig); err != nil {
		return err
	}
	if err := dbConnector.Start(); err != nil {
		return err
	}
	s.DBConnector = dbConnector

	// 初始化数据库表结构
	if err := db.InitTables(dbConnector, db.RepoTypeGameServer); err != nil {
		zLog.Error("Failed to initialize database tables", zap.Error(err))
		return err
	}

	// 初始化默认数据
	if err := db.InitDefaultData(dbConnector); err != nil {
		zLog.Error("Failed to initialize default data", zap.Error(err))
		return err
	}

	// 初始化玩家DAO
	playerDAO := dao.NewPlayerDAO(dbConnector)
	s.PlayerDAO = playerDAO

	// 初始化玩家服务
	playerService := playerservice.NewPlayerService(playerDAO)
	s.PlayerService = playerService

	// 初始化玩家处理器
	playerHandler := handler.NewPlayerHandler(sessionManager, playerService)
	s.PlayerHandler = playerHandler

	// 初始化协议处理器
	protocol := protolayer.NewProtobufProtocol()
	s.Protocol = protocol

	// 初始化玩家管理器
	playerManager := player.NewPlayerManager()
	s.PlayerManager = playerManager

	// 初始化地图服务
	mapService := maps.NewMapService(cfg, protocol)
	mapService.SetConnectionManager(connManager)
	s.MapService = mapService

	// 初始化Metrics服务
	metricsService := metrics.NewMetrics(cfg, connManager, sessionManager, mapService)
	s.Metrics = metricsService

	// 初始化TCP服务
	tcpService := tcpservice.NewTCPService(cfg, connManager, sessionManager, playerManager, playerService, playerHandler, mapService, protocol)
	s.TCPService = tcpService

	// 初始化服务发现
	etcdEndpoints := strings.Split(cfg.Etcd.Endpoints, ",")
	serviceDiscovery, err := discovery.NewServiceDiscoveryWithConfig(etcdEndpoints, &cfg.Etcd)
	if err != nil {
		zLog.Error("Failed to create service discovery", zap.Error(err))
		return fmt.Errorf("failed to create service discovery: %w", err)
	}
	s.ServiceDiscovery = serviceDiscovery

	// 注册服务
	groupID := id.GroupIDStringFromServerID(serverID)
	serviceInfo := &discovery.ServerInfo{
		ID:            serverIDStr,
		ServiceType:   "game",
		GroupID:       groupID,
		Status:        "initializing",
		Address:       cfg.Server.ListenAddr,
		Port:          0,
		Load:          0,
		Players:       0,
		ReadyTime:     time.Now().Unix(),
		LastHeartbeat: time.Now().Unix(),
	}

	// 注册服务，添加重试机制
	maxRetries := 3
	var registerErr error
	for i := 0; i < maxRetries; i++ {
		if err := serviceDiscovery.Register(serviceInfo); err != nil {
			zLog.Warn("Failed to register service", zap.Error(err), zap.Int("retry", i+1))
			registerErr = err
			time.Sleep(time.Duration(i+1) * time.Second)
		} else {
			zLog.Info("Service registered successfully", zap.String("service_id", serviceInfo.ID))
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
				// 更新服务发现中的状态
				state := s.GetState()
				updatedInfo := &discovery.ServerInfo{
					ID:            serverIDStr,
					ServiceType:   "game",
					GroupID:       groupID,
					Status:        state,
					Address:       serviceInfo.Address,
					Port:          serviceInfo.Port,
					Load:          0,
					Players:       0,
					ReadyTime:     serviceInfo.ReadyTime,
					LastHeartbeat: time.Now().Unix(),
				}
				if err := serviceDiscovery.Register(updatedInfo); err != nil {
					zLog.Warn("Failed to update service status", zap.Error(err))
				}
			}
		}
	}()

	// 启动服务发现监听
	go s.discoverAndConnectMapServers()

	// 注册组件
	s.RegisterComponent("Config", cfg)
	s.RegisterComponent("TCPService", tcpService)
	s.RegisterComponent("ConnectionManager", connManager)
	s.RegisterComponent("SessionManager", sessionManager)
	s.RegisterComponent("PlayerManager", playerManager)
	s.RegisterComponent("PlayerHandler", playerHandler)
	s.RegisterComponent("MapService", mapService)
	s.RegisterComponent("Protocol", protocol)
	s.RegisterComponent("PlayerService", playerService)
	s.RegisterComponent("DBConnector", dbConnector)
	s.RegisterComponent("PlayerDAO", playerDAO)
	s.RegisterComponent("Metrics", metricsService)
	s.RegisterComponent("ServiceDiscovery", serviceDiscovery)

	return nil
}

// OnAfterStart 启动后的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnAfterStart() error {
	// 启动TCP服务
	if s.TCPService != nil {
		if err := s.TCPService.Start(s.GetContext()); err != nil {
			return err
		}
	}

	// 启动Metrics服务
	if s.Metrics != nil {
		if err := s.Metrics.Start(); err != nil {
			return err
		}
	}

	// 启动地图服务
	if s.MapService != nil {
		if err := s.MapService.Start(s.GetContext()); err != nil {
			return err
		}
	}

	// 更新服务状态为就绪
	s.SetState(zServer.StateReady, "server ready")

	// 更新服务状态为健康
	s.SetState(zServer.StateHealthy, "server healthy")
	zLog.Info("Game server is healthy")

	// 通知Gateway服务状态变化
	s.notifyGatewayStatusChange()

	return nil
}

// OnBeforeStop 停止前的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnBeforeStop() {
	// 更新服务状态为流量排空（优雅下线）
	s.SetState(zServer.StateDraining, "server stopping")
	zLog.Info("Game server entering draining state")

	// 通知Gateway服务状态变化
	s.notifyGatewayStatusChange()

	// 停止TCP服务
	if s.TCPService != nil {
		s.TCPService.Stop(s.GetContext())
	}

	// 停止地图服务
	if s.MapService != nil {
		s.MapService.Stop(s.GetContext())
	}

	// 停止数据库连接
	if s.DBConnector != nil {
		// DBConnector 没有 Stop 方法，直接关闭
		// s.DBConnector.Stop()
	}

	// 注销服务
	if s.ServiceDiscovery != nil && s.Config != nil {
		serverID, _ := id.ParseServerIDInt(int32(s.Config.Server.ServerID))
		serverIDStr := id.ServerIDString(serverID)
		groupID := id.GroupIDStringFromServerID(serverID)
		if err := s.ServiceDiscovery.Unregister("game", groupID, serverIDStr); err != nil {
			zLog.Warn("Failed to unregister service", zap.Error(err))
		} else {
			zLog.Info("Service unregistered successfully", zap.String("service_id", serverIDStr))
		}
		if err := s.ServiceDiscovery.Close(); err != nil {
			zLog.Warn("Failed to close service discovery", zap.Error(err))
		}
	}
}

// OnAfterStop 停止后的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnAfterStop() {
	// 设置状态为已停止
	s.SetState(zServer.StateStopped, "server stopped")
	zLog.Info("Game server stopped completely")

	// 通知Gateway服务状态变化
	s.notifyGatewayStatusChange()
}

// discoverAndConnectMapServers 从服务发现获取MapServer列表并连接
func (s *BaseServer) discoverAndConnectMapServers() {
	serverID := id.MustParseServerIDInt(int32(s.Config.Server.ServerID))
	groupID := id.GroupIDStringFromServerID(serverID)

	// 定期从服务发现获取MapServer列表
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.GetContext().Done():
			return
		case <-ticker.C:
			// 从服务发现获取MapServer列表
			mapServers, err := s.ServiceDiscovery.Discover("map", groupID)
			if err != nil {
				zLog.Warn("Failed to discover map servers", zap.Error(err))
				continue
			}

			zLog.Info("Discovered map servers", zap.Int("count", len(mapServers)))

			// 选择健康的MapServer
			healthyMapServers := s.selectHealthyMapServers(mapServers)
			if len(healthyMapServers) == 0 {
				zLog.Warn("No healthy MapServer instances found")
				continue
			}

			// 连接到MapServer
			for _, mapServer := range healthyMapServers {
				// 暂时使用默认地图ID列表（实际应该从配置或其他方式获取）
				mapIDs := []int{1001, 1002, 2001, 2002, 3001, 3002, 4001, 4002, 5001}
				mapType := "default"

				// 连接到MapServer
				if err := s.ConnectionManager.ConnectToMapServer(mapServer.Address, mapIDs); err != nil {
					zLog.Warn("Failed to connect to MapServer", zap.Error(err), zap.String("address", mapServer.Address), zap.String("map_type", mapType), zap.Ints("map_ids", mapIDs))
				} else {
					zLog.Info("Connected to MapServer", zap.String("address", mapServer.Address), zap.String("map_type", mapType), zap.Ints("map_ids", mapIDs))
				}
			}
		}
	}
}

// selectHealthyMapServers 选择健康的MapServer
func (s *BaseServer) selectHealthyMapServers(instances []*discovery.ServerInfo) []*discovery.ServerInfo {
	if len(instances) == 0 {
		return nil
	}

	var healthyInstances []*discovery.ServerInfo
	var bestInstances []*discovery.ServerInfo
	var bestScore float64

	// 首先筛选健康的服务器
	for _, inst := range instances {
		if inst.Status == "healthy" || inst.Status == "ready" {
			healthyInstances = append(healthyInstances, inst)
		}
	}

	if len(healthyInstances) == 0 {
		// 如果没有健康的服务器，返回所有服务器
		return instances
	}

	// 选择负载最低的服务器
	for _, inst := range healthyInstances {
		// 计算服务器评分：负载越低，评分越高
		score := 100.0
		if inst.Load > 0 {
			score -= inst.Load * 10
		}

		// 选择评分最高的服务器
		if len(bestInstances) == 0 || score > bestScore {
			bestInstances = []*discovery.ServerInfo{inst}
			bestScore = score
		} else if score == bestScore {
			bestInstances = append(bestInstances, inst)
		}
	}

	zLog.Info("Selected healthy MapServer instances", zap.Int("count", len(bestInstances)))
	return bestInstances
}

// notifyGatewayStatusChange 通知Gateway服务状态变化
func (s *BaseServer) notifyGatewayStatusChange() {
	// 构建服务状态消息
	state := s.GetState()
	statusStr := "Unknown"
	switch state {
	case zServer.StateReady, zServer.StateHealthy:
		statusStr = "Running"
	case zServer.StateMaintenance:
		statusStr = "Maintenance"
	case zServer.StateDraining:
		statusStr = "Stopping"
	case zServer.StateStopped:
		statusStr = "Stopped"
	}

	zLog.Info("Gateway status change notification", zap.String("status", statusStr))
	// 暂时注释掉发送逻辑，后续实现
	// if err := s.ConnectionManager.SendToGateway(packet); err != nil {
	// 	zLog.Warn("Failed to send status change to Gateway", zap.Error(err))
	// } else {
	// 	zLog.Info("Sent status change to Gateway", zap.String("status", statusStr))
	// }
}
