package service

import (
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/auth"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/protocol"
	"github.com/pzqf/zMmoShared/net/protolayer"
	"go.uber.org/zap"
)

type TCPService struct {
	config         *config.Config
	netServer      *zNet.TcpServer
	connManager    *connection.ConnectionManager
	protocolParser *protocol.ProtocolHandler
	compressionCfg *protolayer.CompressionConfig
	securityManager *auth.SecurityManager
}

func NewTCPService(cfg *config.Config, netServer *zNet.TcpServer, connManager *connection.ConnectionManager, protocolParser *protocol.ProtocolHandler, compressionCfg *protolayer.CompressionConfig, securityManager *auth.SecurityManager) *TCPService {
	return &TCPService{
		config:         cfg,
		netServer:      netServer,
		connManager:    connManager,
		protocolParser: protocolParser,
		compressionCfg: compressionCfg,
		securityManager: securityManager,
	}
}

func (ts *TCPService) Start() error {
	zLog.Info("Starting TCP service...")

	return ts.netServer.Start()
}

func (ts *TCPService) Stop() error {
	zLog.Info("Stopping TCP service...")

	ts.netServer.Close()
	return nil
}

func (ts *TCPService) SendToClient(sessionID zNet.SessionIdType, data []byte) error {
	compressedData, _, err := protolayer.Compress(data, ts.compressionCfg)
	if err != nil {
		zLog.Error("Failed to compress data", zap.Error(err))
		return err
	}

	session := ts.netServer.GetSession(sessionID)
	if session == nil {
		zLog.Warn("Session not found", zap.Uint64("session_id", uint64(sessionID)))
		return fmt.Errorf("session not found")
	}

	return session.Send(0, compressedData)
}

func (ts *TCPService) GetSessionCount() int {
	return len(ts.netServer.GetAllSession())
}
