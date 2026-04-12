package gateway

import (
	"context"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/gateway/proxy"
)

// ConnectionService Gateway连接服务
type ConnectionService struct {
	config       *config.Config
	gatewayProxy proxy.GatewayProxy
}

// NewConnectionService 创建Gateway连接服务
func NewConnectionService(cfg *config.Config) *ConnectionService {
	return &ConnectionService{
		config: cfg,
	}
}

// Init 初始化Gateway连接服务
func (s *ConnectionService) Init() error {
	// 初始化Gateway代理
	s.gatewayProxy = proxy.NewGatewayProxy(s.config)

	zLog.Info("Gateway connection service initialized successfully")
	return nil
}

// Start 启动Gateway连接服务
func (s *ConnectionService) Start(ctx context.Context) error {
	// Gateway代理暂时不需要单独启动，因为TCPService已经处理了Gateway连接
	// if err := s.gatewayProxy.Start(ctx); err != nil {
	// 	zLog.Error("Failed to start Gateway proxy", zap.Error(err))
	// 	return err
	// }

	zLog.Info("Gateway connection service started successfully")
	return nil
}

// GetGatewayProxy 获取Gateway代理
func (s *ConnectionService) GetGatewayProxy() proxy.GatewayProxy {
	return s.gatewayProxy
}
