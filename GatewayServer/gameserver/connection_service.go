package gameserver

import (
	"context"

	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/common"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/proxy"
	"go.uber.org/zap"
)

type ConnectionService struct {
	config          *config.Config
	clientService   common.ClientServiceInterface
	discovery       *discovery.ServerServiceDiscovery
	gameServerProxy proxy.GameServerProxy
}

func NewConnectionService(cfg *config.Config, clientService common.ClientServiceInterface) *ConnectionService {
	return &ConnectionService{
		config:        cfg,
		clientService: clientService,
	}
}

func (s *ConnectionService) Init() error {
	serviceDiscovery, err := discovery.NewServerServiceDiscovery(&discovery.ServerServiceDiscoveryConfig{
		ServiceType: "gateway",
		ServerID:    int32(s.config.Server.ServerID),
		ListenAddr:  s.config.Server.ListenAddr,
		Etcd:        &s.config.Etcd,
	})
	if err != nil {
		zLog.Error("Failed to create service discovery", zap.Error(err))
		return err
	}
	s.discovery = serviceDiscovery

	if err := serviceDiscovery.Register(); err != nil {
		zLog.Error("Failed to register service", zap.Error(err))
		return err
	}

	s.gameServerProxy = proxy.NewGameServerProxy(s.config, s.clientService)

	zLog.Info("GameServer connection service initialized successfully")
	return nil
}

func (s *ConnectionService) Start(ctx context.Context) error {
	if err := s.gameServerProxy.Start(ctx); err != nil {
		zLog.Error("Failed to start GameServer proxy", zap.Error(err))
		return err
	}

	zLog.Info("GameServer connection service started successfully")
	return nil
}

func (s *ConnectionService) GetGameServerProxy() proxy.GameServerProxy {
	return s.gameServerProxy
}

func (s *ConnectionService) GetServiceDiscovery() *discovery.ServerServiceDiscovery {
	return s.discovery
}

func (s *ConnectionService) UpdateHeartbeat(status interface{}, players int) error {
	if s.discovery != nil {
		return s.discovery.UpdateHeartbeat(status.(string), players)
	}
	return nil
}

func (s *ConnectionService) IsGameServerConnected() bool {
	if s.gameServerProxy == nil {
		return false
	}
	return s.gameServerProxy.IsConnected()
}
