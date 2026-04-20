package gateway

import (
	"context"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	"github.com/pzqf/zMmoServer/GameServer/gateway/proxy"
	"go.uber.org/zap"
)

type ConnectionService struct {
	config       *config.Config
	connMgr      *connection.ConnectionManager
	gatewayProxy proxy.GatewayProxy
}

func NewConnectionService(cfg *config.Config, connMgr *connection.ConnectionManager) *ConnectionService {
	return &ConnectionService{
		config:  cfg,
		connMgr: connMgr,
	}
}

func (s *ConnectionService) Init() error {
	s.gatewayProxy = proxy.NewGatewayProxy(s.config, s.connMgr)

	zLog.Info("Gateway connection service initialized successfully")
	return nil
}

func (s *ConnectionService) Start(ctx context.Context) error {
	if err := s.gatewayProxy.Start(ctx); err != nil {
		zLog.Error("Failed to start Gateway proxy", zap.Error(err))
		return err
	}

	zLog.Info("Gateway connection service started successfully")
	return nil
}

func (s *ConnectionService) GetGatewayProxy() proxy.GatewayProxy {
	return s.gatewayProxy
}
