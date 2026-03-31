package game

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/db"
	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/protocol"
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
	StatusManager     *ServiceStatusManager
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

	// 初始化服务状态管理器
	statusManager := NewServiceStatusManager()
	s.StatusManager = statusManager

	// 初始化连接管理器
	connManager := connection.NewConnectionManager(cfg)
	s.ConnectionManager = connManager

	// 初始化会话管理器
	sessionManager := session.NewSessionManager()
	s.SessionManager = sessionManager

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

	// 初始化PlayerManager
	playerManager := player.NewPlayerManager()
	s.PlayerManager = playerManager

	// 初始化地图服务
	mapService := maps.NewMapService(cfg, protocol)
	// 设置连接管理器
	mapService.SetConnectionManager(connManager)
	s.MapService = mapService

	// 初始化TCP服务
	tcpService := tcpservice.NewTCPService(cfg, connManager, sessionManager, playerManager, playerService, playerHandler, mapService, protocol)
	s.TCPService = tcpService

	// 初始化监控系统
	metrics := metrics.NewMetrics(cfg, connManager, sessionManager, mapService)
	s.Metrics = metrics
	mapService.SetOnOutboxStatsChanged(func(stats maps.OutboxStats) {
		if s.Metrics != nil {
			s.Metrics.UpdateConsistencyOutbox(stats.Pending, stats.Dead)
		}
	})
	tcpService.SetOnGatewayDedupeHit(func(total uint64) {
		if s.Metrics != nil {
			s.Metrics.UpdateGatewayDedupeHits(total)
		}
	})

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
		return fmt.Errorf("failed to create service discovery: %w", err)
	}
	s.ServiceDiscovery = serviceDiscovery
	zLog.Info("Using etcd service discovery", zap.Strings("endpoints", etcdEndpoints))
	// 注册服务发现组件
	s.RegisterComponent("ServiceDiscovery", serviceDiscovery)

	// 注册当前服务：严格使用 6 位 GameServerID（GroupID(4)+ServerIndex(2)）
	serverID, err := id.ParseServerIDInt(int32(cfg.Server.ServerID))
	if err != nil {
		return fmt.Errorf("invalid game ServerID %d: %w", cfg.Server.ServerID, err)
	}
	serverIDStr := id.ServerIDString(serverID)
	groupID := id.GroupIDStringFromServerID(serverID)

	s.SetId(fmt.Sprintf("game-%s", serverIDStr))

	serviceInfo := &discovery.ServerInfo{
		ID:        fmt.Sprintf("game-%s", serverIDStr),
		GroupID:   groupID,
		Status:    discovery.ServerStatusHealthy,
		Address:   cfg.Server.ExternalAddr,
		Port:      0, // 游戏服端口已在Address中包含
		Load:      0,
		Players:   0,
		ReadyTime: time.Now().Unix(),
	}

	if err := serviceDiscovery.Register(serviceInfo); err != nil {
		zLog.Warn("Failed to register service", zap.Error(err))
	} else {
		zLog.Info("Service registered successfully",
			zap.String("service_id", serviceInfo.ID),
			zap.String("address", serviceInfo.Address))

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
						ID:        serviceInfo.ID,
						GroupID:   serviceInfo.GroupID,
						Status:    discovery.ServerStatusHealthy,
						Address:   serviceInfo.Address,
						Port:      serviceInfo.Port,
						Load:      0,
						Players:   int(s.PlayerManager.GetPlayerCount()),
						ReadyTime: serviceInfo.ReadyTime,
					}
					if err := serviceDiscovery.Register(updatedInfo); err != nil {
						zLog.Warn("Failed to keep service alive", zap.Error(err))
					}
				}
			}
		}()
	}

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
	s.RegisterComponent("Metrics", metrics)
	s.RegisterComponent("StatusManager", statusManager)

	return nil
}

// OnAfterStart 启动后的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnAfterStart() error {
	// 启动TCP服务
	if err := s.TCPService.Start(s.GetContext()); err != nil {
		return err
	}

	// 启动监控服务
	if err := s.Metrics.Start(); err != nil {
		return err
	}

	// 从服务发现获取MapServer列表并连接
	go s.discoverAndConnectMapServers()
	go s.consistencyMonitorLoop()

	// 等待Gateway连接
	zLog.Info("Waiting for Gateway connection...")
	select {
	case <-s.ConnectionManager.GatewayConnectedChan():
		zLog.Info("Gateway connected successfully")
	case <-time.After(30 * time.Second):
		zLog.Warn("Gateway connection timeout, continuing startup")
	}

	// 更新服务状态为运行中
	s.StatusManager.SetStatus(ServiceStatusRunning)

	// 通知Gateway服务状态变化
	s.notifyGatewayStatusChange()

	return nil
}

func (s *BaseServer) consistencyMonitorLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.GetContext().Done():
			return
		case <-ticker.C:
			if s.MapService == nil {
				continue
			}
			stats := s.MapService.GetOutboxStats()
			if stats.Pending > 0 || stats.Dead > 0 {
				zLog.Warn("Cross-server consistency status",
					zap.Int("outbox_pending", stats.Pending),
					zap.Int("outbox_dead", stats.Dead))
			} else {
				zLog.Debug("Cross-server consistency status",
					zap.Int("outbox_pending", stats.Pending),
					zap.Int("outbox_dead", stats.Dead))
			}
			if s.Metrics != nil {
				s.Metrics.UpdateConsistencyOutbox(stats.Pending, stats.Dead)
				if s.TCPService != nil {
					s.Metrics.UpdateGatewayDedupeHits(s.TCPService.GetGatewayDedupeHits())
				}
			}

			if stats.Dead > 0 {
				deadSamples := s.MapService.GetOutboxDeadLetters(3)
				for _, dead := range deadSamples {
					zLog.Warn("Outbox dead-letter snapshot",
						zap.Uint64("request_id", dead.RequestID),
						zap.String("topic", dead.Topic),
						zap.Int("attempts", dead.Attempts),
						zap.String("last_error", dead.LastError))
				}
			}

			// 自动清理超过24小时的死信，防止长期堆积。
			purged := s.MapService.PurgeOutboxDeadLetters(24 * time.Hour)
			if purged > 0 {
				zLog.Info("Purged outbox dead letters", zap.Int("count", purged))
			}
		}
	}
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
		if inst.Status == discovery.ServerStatusHealthy || inst.Status == discovery.ServerStatusReady {
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

// OnBeforeStop 停止前的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnBeforeStop() {
	// 更新服务状态为停止中
	s.StatusManager.SetStatus(ServiceStatusStopping)

	// 通知Gateway服务状态变化
	s.notifyGatewayStatusChange()

	// 停止TCP服务
	if s.TCPService != nil {
		s.TCPService.Stop(s.GetContext())
	}

	// 停止玩家服务
	if s.PlayerService != nil {
		s.PlayerService.Stop()
	}

	// 关闭数据库连接器
	if s.DBConnector != nil {
		s.DBConnector.Close()
	}

	// 注销服务发现
	if s.ServiceDiscovery != nil {
		if s.Config != nil {
			serverID := id.MustParseServerIDInt(int32(s.Config.Server.ServerID))
			serviceID := id.ServerIDString(serverID)
			// 使用从服务器ID中提取的groupID
			groupID := id.GroupIDStringFromServerID(id.ServerIdType(s.Config.Server.GroupID))
			if err := s.ServiceDiscovery.Unregister("game", groupID, serviceID); err != nil {
				zLog.Warn("Failed to unregister service", zap.Error(err))
			} else {
				zLog.Info("Service unregistered successfully", zap.String("service_id", serviceID))
			}
		}
		if err := s.ServiceDiscovery.Close(); err != nil {
			zLog.Warn("Failed to close service discovery", zap.Error(err))
		}
	}

	// 更新服务状态为已停止
	s.StatusManager.SetStatus(ServiceStatusStopped)

	// 通知Gateway服务状态变化
	s.notifyGatewayStatusChange()
}

// HandleShutdownCommand 处理停服指令
func (s *BaseServer) HandleShutdownCommand() {
	zLog.Info("Received shutdown command")

	// 更新服务状态为停止中
	s.StatusManager.SetStatus(ServiceStatusStopping)

	// 通知Gateway服务状态变化
	s.notifyGatewayStatusChange()

	// 优雅关闭服务�?	s.BaseServer.Stop()
}

// notifyGatewayStatusChange 通知Gateway服务状态变化
func (s *BaseServer) notifyGatewayStatusChange() {
	// 构建服务状态消息
	// 使用心跳消息格式发送服务状态
	status := s.StatusManager.GetStatus()
	statusStr := "Unknown"
	switch status {
	case ServiceStatusRunning:
		statusStr = "Running"
	case ServiceStatusMaintenance:
		statusStr = "Maintenance"
	case ServiceStatusStopping:
		statusStr = "Stopping"
	case ServiceStatusStopped:
		statusStr = "Stopped"
	}

	// 构建心跳消息
	heartbeatReq := &protocol.ServiceHeartbeatRequest{
		ServerId:    int32(s.Config.Server.ServerID),
		ServiceType: protocol.ServiceType_SERVICE_TYPE_GAME,
		OnlineCount: 0,             // 实际应该从PlayerManager获取
		Status:      int32(status), // 使用状态枚举值
		Load:        0.0,
	}

	// 序列化心跳消息
	heartbeatData, err := proto.Marshal(heartbeatReq)
	if err != nil {
		zLog.Error("Failed to marshal heartbeat request", zap.Error(err))
		return
	}

	// 构建数据包：长度前缀 + 消息ID + 数据
	msgID := uint32(protocol.InternalMsgId_MSG_INTERNAL_SERVICE_HEARTBEAT)
	length := 4 + len(heartbeatData)
	packet := make([]byte, 4+4+len(heartbeatData))
	binary.BigEndian.PutUint32(packet[:4], uint32(length))
	binary.BigEndian.PutUint32(packet[4:8], msgID)
	copy(packet[8:], heartbeatData)

	// 发送心跳消息
	if err := s.ConnectionManager.SendToGateway(packet); err != nil {
		zLog.Warn("Failed to send status change to Gateway", zap.Error(err))
	} else {
		zLog.Info("Sent status change to Gateway", zap.String("status", statusStr))
	}
}
