package gameserver

import (
	"context"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/common"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/discovery"
	"github.com/pzqf/zMmoServer/GatewayServer/proxy"
	"go.uber.org/zap"
)

// ConnectionService GameServer连接服务
type ConnectionService struct {
	config          *config.Config
	clientService   common.ClientServiceInterface
	discovery       discovery.ServiceDiscovery
	gameServerProxy proxy.GameServerProxy
}

// NewConnectionService 创建GameServer连接服务
func NewConnectionService(cfg *config.Config, clientService common.ClientServiceInterface) *ConnectionService {
	return &ConnectionService{
		config:        cfg,
		clientService: clientService,
	}
}

// Init 初始化GameServer连接服务
func (s *ConnectionService) Init() error {
	// 初始化服务发现
	serviceDiscovery, err := discovery.NewServiceDiscovery(s.config)
	if err != nil {
		zLog.Error("Failed to create service discovery", zap.Error(err))
		return err
	}
	s.discovery = serviceDiscovery

	// 注册服务
	if err := s.discovery.Register(); err != nil {
		zLog.Error("Failed to register service", zap.Error(err))
		return err
	}

	// 初始化GameServer代理
	s.gameServerProxy = proxy.NewGameServerProxy(s.config, s.clientService)

	zLog.Info("GameServer connection service initialized successfully")
	return nil
}

// Start 启动GameServer连接服务
func (s *ConnectionService) Start(ctx context.Context) error {
	// 启动GameServer代理
	if err := s.gameServerProxy.Start(ctx); err != nil {
		zLog.Error("Failed to start GameServer proxy", zap.Error(err))
		return err
	}

	zLog.Info("GameServer connection service started successfully")
	return nil
}

// GetGameServerProxy 获取GameServer代理
func (s *ConnectionService) GetGameServerProxy() proxy.GameServerProxy {
	return s.gameServerProxy
}

// GetServiceDiscovery 获取服务发现
func (s *ConnectionService) GetServiceDiscovery() discovery.ServiceDiscovery {
	return s.discovery
}

// UpdateHeartbeat 更新心跳
func (s *ConnectionService) UpdateHeartbeat(status interface{}, players int) error {
	if s.discovery != nil {
		return s.discovery.UpdateHeartbeat(status.(string), players)
	}
	return nil
}
