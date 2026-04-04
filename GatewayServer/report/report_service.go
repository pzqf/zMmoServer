package report

import (
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/client/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/gameserver"
	"go.uber.org/zap"
)

// Service 服务上报服务
type Service struct {
	connMgr             *connection.ClientConnMgr
	gameServerConnSvc   *gameserver.ConnectionService
}

// NewService 创建服务上报服务
func NewService(connMgr *connection.ClientConnMgr, gameServerConnSvc *gameserver.ConnectionService) *Service {
	return &Service{
		connMgr:           connMgr,
		gameServerConnSvc: gameServerConnSvc,
	}
}

// Start 启动服务上报服务
func (s *Service) Start(ctx interface{}) {
	// 启动心跳保持
	go s.startHeartbeat(ctx)

	zLog.Info("Report service started successfully")
}

// startHeartbeat 启动心跳保持
func (s *Service) startHeartbeat(ctx interface{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.(interface{ Done() <-chan struct{} }).Done():
			return
		case <-ticker.C:
			s.sendHeartbeat()
		}
	}
}

// sendHeartbeat 发送心跳
func (s *Service) sendHeartbeat() {
	// 获取服务状态
	state := "healthy"
	players := s.connMgr.GetAccountCount()

	// 发送心跳
	if err := s.gameServerConnSvc.UpdateHeartbeat(state, players); err != nil {
		zLog.Warn("Failed to send heartbeat", zap.Error(err))
	}

	zLog.Debug("Heartbeat sent", zap.String("state", state), zap.Int("players", players))
}
