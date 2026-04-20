package game

import (
	"fmt"
	"time"

	"github.com/pzqf/zCommon/common/id"
	zcont "github.com/pzqf/zCommon/container"
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
	"github.com/pzqf/zMmoServer/GameServer/gateway"
	"github.com/pzqf/zMmoServer/GameServer/gateway/proxy"
	"github.com/pzqf/zMmoServer/GameServer/handler"
	"github.com/pzqf/zMmoServer/GameServer/health"
	"github.com/pzqf/zMmoServer/GameServer/metrics"
	"github.com/pzqf/zMmoServer/GameServer/net/protolayer"
	tcpservice "github.com/pzqf/zMmoServer/GameServer/net/service"
	playerservice "github.com/pzqf/zMmoServer/GameServer/services"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"go.uber.org/zap"
)

const ServerTypeGame zServer.ServerType = "game"

type BaseServer struct {
	*zServer.BaseServer
	Config            *config.Config
	Container         *zcont.Container
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
	ServiceDiscovery  *discovery.ServerServiceDiscovery
	GatewayService    *gateway.ConnectionService
	HealthChecker     *health.Checker
}

func NewBaseServer(cfg *config.Config) *BaseServer {
	gs := &BaseServer{
		Config:    cfg,
		Container: zcont.NewContainer(),
	}
	gs.BaseServer = zServer.NewBaseServer(ServerTypeGame, "", "Game Server", "1.0.0", gs)
	return gs
}

func (s *BaseServer) OnBeforeStart() error {
	if s.Config == nil {
		return nil
	}

	s.initServerID()
	s.SetState(zServer.StateInitializing, "server initializing")

	if err := s.initDatabase(); err != nil {
		return err
	}

	s.initPlayerComponents()
	s.initMapComponents()
	s.initNetworkComponents()

	if err := s.initServiceDiscovery(); err != nil {
		return err
	}

	s.HealthChecker = health.NewGameServerChecker(s.ConnectionManager, s.PlayerManager, s.GatewayService)
	s.registerComponents()
	return nil
}

func (s *BaseServer) initServerID() {
	serverID, err := id.ParseServerIDInt(int32(s.Config.Server.ServerID))
	if err != nil {
		zLog.Fatal("Invalid game ServerID", zap.Error(err))
	}
	s.SetId(fmt.Sprintf("game-%s", id.ServerIDString(serverID)))
}

func (s *BaseServer) initDatabase() error {
	cfg := s.Config
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

	if err := db.InitTables(dbConnector, db.RepoTypeGameServer); err != nil {
		return fmt.Errorf("init database tables: %w", err)
	}
	if err := db.InitDefaultData(dbConnector); err != nil {
		return fmt.Errorf("init default data: %w", err)
	}

	s.PlayerDAO = dao.NewPlayerDAO(dbConnector)
	s.PlayerService = playerservice.NewPlayerService(s.PlayerDAO)
	return nil
}

func (s *BaseServer) initPlayerComponents() {
	s.ConnectionManager = connection.NewConnectionManager(s.Config)
	s.SessionManager = session.NewSessionManager()
	s.PlayerManager = player.NewPlayerManager()
	s.PlayerHandler = handler.NewPlayerHandler(s.SessionManager, s.PlayerService)
	s.Protocol = protolayer.NewProtobufProtocol()

	loginService := player.NewLoginService(s.PlayerManager, s.PlayerService, s.SessionManager)
	s.PlayerManager.SetLoginService(loginService)
}

func (s *BaseServer) initMapComponents() {
	s.MapService = maps.NewMapService(s.Config, s.Protocol)
	s.MapService.SetConnectionManager(s.ConnectionManager)
	s.PlayerManager.SetMapOperator(s.MapService)
	s.Metrics = metrics.NewMetrics(s.Config, s.ConnectionManager, s.SessionManager, s.MapService)
}

func (s *BaseServer) initNetworkComponents() {
	loginService := s.PlayerManager.GetLoginService()

	s.TCPService = tcpservice.NewTCPService(
		s.Config, s.ConnectionManager, s.SessionManager,
		s.PlayerManager, s.PlayerService, s.PlayerHandler,
		s.MapService, loginService, s.Protocol,
	)

	gatewayService := gateway.NewConnectionService(s.Config, s.ConnectionManager)
	if err := gatewayService.Init(); err != nil {
		zLog.Error("Failed to initialize Gateway connection service", zap.Error(err))
		return
	}
	s.GatewayService = gatewayService

	clientSender := proxy.NewPlayerClientSender(gatewayService.GetGatewayProxy())
	s.PlayerManager.SetClientSender(clientSender)
}

func (s *BaseServer) initServiceDiscovery() error {
	sd, err := discovery.NewServerServiceDiscovery(&discovery.ServerServiceDiscoveryConfig{
		ServiceType: "game",
		ServerID:    int32(s.Config.Server.ServerID),
		ListenAddr:  s.Config.Server.ListenAddr,
		Etcd:        &s.Config.Etcd,
	})
	if err != nil {
		return fmt.Errorf("create service discovery: %w", err)
	}
	s.ServiceDiscovery = sd

	if err := sd.Register(); err != nil {
		return fmt.Errorf("register service: %w", err)
	}

	go s.heartbeatLoop()
	go s.discoverAndConnectMapServers()
	return nil
}

func (s *BaseServer) heartbeatLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.GetContext().Done():
			return
		case <-ticker.C:
			if s.ServiceDiscovery != nil {
				if err := s.ServiceDiscovery.UpdateHeartbeat(string(s.GetState()), 0); err != nil {
					zLog.Warn("Failed to update service status", zap.Error(err))
				}
			}
		}
	}
}

func (s *BaseServer) registerComponents() {
	s.Container.Register("Config", s.Config)
	s.Container.Register("TCPService", s.TCPService)
	s.Container.Register("ConnectionManager", s.ConnectionManager)
	s.Container.Register("SessionManager", s.SessionManager)
	s.Container.Register("PlayerManager", s.PlayerManager)
	s.Container.Register("PlayerHandler", s.PlayerHandler)
	s.Container.Register("MapService", s.MapService)
	s.Container.Register("Protocol", s.Protocol)
	s.Container.Register("PlayerService", s.PlayerService)
	s.Container.Register("DBConnector", s.DBConnector)
	s.Container.Register("PlayerDAO", s.PlayerDAO)
	s.Container.Register("Metrics", s.Metrics)
	s.Container.Register("ServiceDiscovery", s.ServiceDiscovery)
	s.Container.Register("GatewayService", s.GatewayService)
	s.Container.Register("HealthChecker", s.HealthChecker)
}

func (s *BaseServer) OnAfterStart() error {
	if s.TCPService != nil {
		if err := s.TCPService.Start(s.GetContext()); err != nil {
			return err
		}
	}
	if s.Metrics != nil {
		if err := s.Metrics.Start(); err != nil {
			return err
		}
	}
	if s.MapService != nil {
		if err := s.MapService.Start(s.GetContext()); err != nil {
			return err
		}
	}
	if s.GatewayService != nil {
		if err := s.GatewayService.Start(s.GetContext()); err != nil {
			return err
		}
	}
	if s.HealthChecker != nil {
		go s.HealthChecker.Start(s.GetContext())
	}

	s.SetState(zServer.StateReady, "server ready")
	s.SetState(zServer.StateHealthy, "server healthy")
	zLog.Info("Game server is healthy")
	s.notifyGatewayStatusChange()
	return nil
}

func (s *BaseServer) OnBeforeStop() {
	s.SetState(zServer.StateDraining, "server stopping")
	zLog.Info("Game server entering draining state")
	s.notifyGatewayStatusChange()

	if s.TCPService != nil {
		s.TCPService.Stop(s.GetContext())
	}
	if s.MapService != nil {
		s.MapService.Stop(s.GetContext())
	}
	if s.ServiceDiscovery != nil {
		if err := s.ServiceDiscovery.Unregister(); err != nil {
			zLog.Warn("Failed to unregister service", zap.Error(err))
		} else {
			zLog.Info("Service unregistered successfully")
		}
		if err := s.ServiceDiscovery.Close(); err != nil {
			zLog.Warn("Failed to close service discovery", zap.Error(err))
		}
	}
}

func (s *BaseServer) OnAfterStop() {
	s.SetState(zServer.StateStopped, "server stopped")
	zLog.Info("Game server stopped completely")
	s.notifyGatewayStatusChange()
}

func (s *BaseServer) discoverAndConnectMapServers() {
	serverID := id.MustParseServerIDInt(int32(s.Config.Server.ServerID))
	groupID := id.GroupIDStringFromServerID(serverID)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.GetContext().Done():
			return
		case <-ticker.C:
			mapServers, err := s.ServiceDiscovery.DiscoverInGroup("map", groupID)
			if err != nil {
				zLog.Warn("Failed to discover map servers", zap.Error(err))
				continue
			}

			zLog.Info("Discovered map servers", zap.Int("count", len(mapServers)))

			healthyMapServers := s.selectHealthyMapServers(mapServers)
			if len(healthyMapServers) == 0 {
				zLog.Warn("No healthy MapServer instances found")
				continue
			}

			mapIDs := []int{1001, 1002, 2001, 2002, 3001, 3002, 4001, 4002, 5001}
			for _, mapServer := range healthyMapServers {
				addr := mapServer.Address
				if addr == "0.0.0.0" {
					addr = "127.0.0.1"
				}
				mapServerAddr := fmt.Sprintf("%s:%d", addr, mapServer.Port)
				if err := s.ConnectionManager.ConnectToMapServer(mapServerAddr, mapIDs); err != nil {
					zLog.Warn("Failed to connect to MapServer", zap.Error(err), zap.String("address", mapServerAddr))
				} else {
					zLog.Info("Connected to MapServer", zap.String("address", mapServerAddr))
				}
			}
		}
	}
}

func (s *BaseServer) selectHealthyMapServers(instances []*discovery.ServerInfo) []*discovery.ServerInfo {
	if len(instances) == 0 {
		return nil
	}

	var healthy []*discovery.ServerInfo
	for _, inst := range instances {
		if inst.Status == "healthy" || inst.Status == "ready" {
			healthy = append(healthy, inst)
		}
	}
	if len(healthy) == 0 {
		return instances
	}

	var best []*discovery.ServerInfo
	var bestScore float64
	for _, inst := range healthy {
		score := 100.0 - inst.Load*10
		if len(best) == 0 || score > bestScore {
			best = []*discovery.ServerInfo{inst}
			bestScore = score
		} else if score == bestScore {
			best = append(best, inst)
		}
	}
	return best
}

func (s *BaseServer) notifyGatewayStatusChange() {
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
}
