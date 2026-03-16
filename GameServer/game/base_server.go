package game

import (
	"encoding/binary"
	"time"

	"github.com/gogo/protobuf/proto"
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
	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/dao"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
)

// ServerType 游戏服类型
const ServerTypeGame zServer.ServerType = "game"

// BaseServer 游戏服基础服务器
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
}

// NewBaseServer 创建游戏服基础服务器
func NewBaseServer() *BaseServer {
	// 先创建子类实例
	gs := &BaseServer{}

	// 创建基础服务器，传入子类作为 hooks
	baseServer := zServer.NewBaseServer(
		ServerTypeGame,
		"game-1",
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
	s.MapService = mapService

	// 初始化TCP服务
	tcpService := tcpservice.NewTCPService(cfg, connManager, sessionManager, playerManager, playerService, playerHandler, mapService, protocol)
	s.TCPService = tcpService

	// 初始化监控系统
	metrics := metrics.NewMetrics(cfg, connManager, sessionManager)
	s.Metrics = metrics

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

	// 通知Gateway服务状态变更
	s.notifyGatewayStatusChange()

	return nil
}

// OnBeforeStop 停止前的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnBeforeStop() {
	// 更新服务状态为停止中
	s.StatusManager.SetStatus(ServiceStatusStopping)

	// 通知Gateway服务状态变更
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

	// 更新服务状态为已停止
	s.StatusManager.SetStatus(ServiceStatusStopped)

	// 通知Gateway服务状态变更
	s.notifyGatewayStatusChange()
}

// HandleShutdownCommand 处理停服指令
func (s *BaseServer) HandleShutdownCommand() {
	zLog.Info("Received shutdown command")

	// 更新服务状态为停止中
	s.StatusManager.SetStatus(ServiceStatusStopping)

	// 通知Gateway服务状态变更
	s.notifyGatewayStatusChange()

	// 优雅关闭服务器
	s.BaseServer.Stop()
}

// notifyGatewayStatusChange 通知Gateway服务状态变更
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
