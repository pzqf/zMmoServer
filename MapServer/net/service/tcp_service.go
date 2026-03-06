package service

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"go.uber.org/zap"
)

type TCPService struct {
	config      *config.Config
	connManager *connection.ConnectionManager
	listener    net.Listener
	isRunning   bool
	wg          sync.WaitGroup
}

func NewTCPService(cfg *config.Config, connManager *connection.ConnectionManager) *TCPService {
	return &TCPService{
		config:      cfg,
		connManager: connManager,
		isRunning:   false,
	}
}

func (ts *TCPService) Name() string {
	return "TCPService"
}

func (ts *TCPService) Start(ctx context.Context) error {
	if ts.isRunning {
		return nil
	}

	zLog.Info("Starting TCP service...", zap.String("addr", ts.config.Server.ListenAddr))

	listener, err := net.Listen("tcp", ts.config.Server.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", ts.config.Server.ListenAddr, err)
	}

	ts.listener = listener
	ts.isRunning = true

	ts.wg.Add(1)
	go ts.acceptConnections(ctx)

	zLog.Info("TCP service started successfully", zap.String("addr", ts.config.Server.ListenAddr))

	return nil
}

func (ts *TCPService) Stop(ctx context.Context) error {
	if !ts.isRunning {
		return nil
	}

	zLog.Info("Stopping TCP service...")

	if ts.listener != nil {
		ts.listener.Close()
	}

	ts.isRunning = false
	ts.wg.Wait()

	zLog.Info("TCP service stopped")

	return nil
}

func (ts *TCPService) acceptConnections(ctx context.Context) {
	defer ts.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := ts.listener.Accept()
			if err != nil {
				if ts.isRunning {
					zLog.Error("Failed to accept connection", zap.Error(err))
				}
				continue
			}

			ts.wg.Add(1)
			go ts.handleConnection(ctx, conn)
		}
	}
}

func (ts *TCPService) handleConnection(ctx context.Context, conn net.Conn) {
	defer ts.wg.Done()
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	connID := fmt.Sprintf("%d", time.Now().UnixNano())

	zLog.Info("New connection", zap.String("conn_id", connID), zap.String("client_addr", clientAddr))

	ts.connManager.AddConnection(connID, conn)

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	buffer := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := conn.Read(buffer)
			if err != nil {
				if ts.isRunning {
					zLog.Info("Connection closed", zap.String("conn_id", connID), zap.Error(err))
				}
				ts.connManager.RemoveConnection(connID)
				return
			}

			if n > 0 {
				conn.SetReadDeadline(time.Now().Add(60 * time.Second))

				// 处理消息
				// 这里可以添加消息处理逻辑
				zLog.Info("Received data", zap.String("conn_id", connID), zap.Int("data_len", n))
			}
		}
	}
}
