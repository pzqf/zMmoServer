package proxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/service"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type GameServerProxy interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SendToClient(sessionID zNet.SessionIdType, data []byte) error
	RegisterClient(connID zNet.SessionIdType, accountID int64)
	UnregisterClient(connID zNet.SessionIdType)
}

type gameServerProxy struct {
	config         *config.Config
	tcpService     *service.TCPService
	connManager    *connection.ConnectionManager
	connMap        map[zNet.SessionIdType]int64 // connID -> accountID
	accountMap     map[int64]zNet.SessionIdType // accountID -> connID
	mutex          sync.RWMutex
	gameConn       net.Conn
	isConnected    bool
	connecting     bool
	reconnectTimer *time.Timer
}

func NewGameServerProxy(cfg *config.Config, tcpService *service.TCPService, connManager *connection.ConnectionManager) GameServerProxy {
	return &gameServerProxy{
		config:      cfg,
		tcpService:  tcpService,
		connManager: connManager,
		connMap:     make(map[zNet.SessionIdType]int64),
		accountMap:  make(map[int64]zNet.SessionIdType),
	}
}

func (gsp *gameServerProxy) Start(ctx context.Context) error {
	zLog.Info("Starting GameServer proxy...")

	go gsp.connectToGameServer(ctx)

	return nil
}

func (gsp *gameServerProxy) Stop(ctx context.Context) error {
	zLog.Info("Stopping GameServer proxy...")

	if gsp.gameConn != nil {
		gsp.gameConn.Close()
	}

	if gsp.reconnectTimer != nil {
		gsp.reconnectTimer.Stop()
	}

	return nil
}

func (gsp *gameServerProxy) connectToGameServer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if gsp.isConnected || gsp.connecting {
				time.Sleep(1 * time.Second)
				continue
			}

			gsp.connecting = true
			conn, err := net.Dial("tcp", gsp.config.GameServer.GameServerAddr)
			if err != nil {
				zLog.Error("Failed to connect to GameServer", zap.Error(err))
				gsp.connecting = false
				gsp.scheduleReconnect()
				continue
			}

			zLog.Info("Connected to GameServer", zap.String("addr", gsp.config.GameServer.GameServerAddr))
			gsp.gameConn = conn
			gsp.isConnected = true
			gsp.connecting = false

			// 启动消息处理
			go gsp.handleGameServerMessages(ctx, conn)
		}
	}
}

func (gsp *gameServerProxy) scheduleReconnect() {
	if gsp.reconnectTimer != nil {
		gsp.reconnectTimer.Stop()
	}

	timeout := time.Duration(gsp.config.GameServer.GameServerConnectTimeout) * time.Second
	gsp.reconnectTimer = time.AfterFunc(timeout, func() {
		zLog.Info("Attempting to reconnect to GameServer...")
	})
}

func (gsp *gameServerProxy) handleGameServerMessages(ctx context.Context, conn net.Conn) {
	defer func() {
		conn.Close()
		gsp.isConnected = false
		zLog.Info("GameServer connection closed")
	}()

	buffer := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, err := conn.Read(buffer)
			if err != nil {
				if gsp.isConnected {
					zLog.Error("Failed to read from GameServer", zap.Error(err))
				}
				return
			}

			if n > 0 {
				gsp.processGameServerMessage(buffer[:n])
			}
		}
	}
}

func (gsp *gameServerProxy) processGameServerMessage(data []byte) {
	// 解析消息格式：长度前缀 + 消息ID + 数据
	if len(data) < 8 {
		zLog.Error("Invalid message format from GameServer", zap.Int("size", len(data)))
		return
	}

	// 解析长度
	length := binary.BigEndian.Uint32(data[:4])
	if length > MaxMsgLen {
		zLog.Error("Message too long from GameServer", zap.Uint32("length", length))
		return
	}

	if len(data) < int(length) {
		zLog.Error("Insufficient data from GameServer", zap.Int("actual", len(data)), zap.Uint32("expected", length))
		return
	}

	// 解析消息ID
	msgID := binary.BigEndian.Uint32(data[4:8])

	// 解析消息内容，提取sessionID
	var msg protocol.Message
	if err := proto.Unmarshal(data[8:length], &msg); err != nil {
		zLog.Error("Failed to unmarshal message", zap.Error(err))
		return
	}

	// 查找对应的客户端连接
	sessionID := zNet.SessionIdType(msg.SessionId)

	// 转发消息给客户端
	err := gsp.tcpService.SendToClient(sessionID, data)
	if err != nil {
		zLog.Error("Failed to send message to client", zap.Error(err), zap.Uint64("session_id", uint64(sessionID)))
		return
	}

	zLog.Debug("Message forwarded to client", zap.Uint32("msg_id", msgID), zap.Uint64("session_id", uint64(sessionID)))
}

const MaxMsgLen = 1024 * 1024

func (gsp *gameServerProxy) SendToClient(sessionID zNet.SessionIdType, data []byte) error {
	if !gsp.isConnected {
		return fmt.Errorf("not connected to GameServer")
	}

	// 直接转发客户端消息给GameServer
	_, err := gsp.gameConn.Write(data)
	if err != nil {
		gsp.isConnected = false
		zLog.Error("Failed to send message to GameServer", zap.Error(err))
		return err
	}

	zLog.Debug("Message sent to GameServer", zap.Uint64("session_id", uint64(sessionID)), zap.Int("data_len", len(data)))
	return nil
}

func (gsp *gameServerProxy) RegisterClient(connID zNet.SessionIdType, accountID int64) {
	gsp.mutex.Lock()
	defer gsp.mutex.Unlock()

	gsp.connMap[connID] = accountID
	gsp.accountMap[accountID] = connID

	zLog.Info("Client registered", zap.Uint64("conn_id", uint64(connID)), zap.Int64("account_id", accountID))
}

func (gsp *gameServerProxy) UnregisterClient(connID zNet.SessionIdType) {
	gsp.mutex.Lock()
	defer gsp.mutex.Unlock()

	accountID, exists := gsp.connMap[connID]
	if exists {
		delete(gsp.accountMap, accountID)
		delete(gsp.connMap, connID)
		zLog.Info("Client unregistered", zap.Uint64("conn_id", uint64(connID)), zap.Int64("account_id", accountID))
	}
}
