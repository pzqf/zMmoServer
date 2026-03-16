package proxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"

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
	SendToClient(sessionID zNet.SessionIdType, protoId int32, data []byte) error
	RegisterClient(connID zNet.SessionIdType, accountID int64)
	UnregisterClient(connID zNet.SessionIdType)
}

type gameServerProxy struct {
	config      *config.Config
	tcpService  *service.TCPService
	connManager *connection.ConnectionManager
	connMap     map[zNet.SessionIdType]int64 // connID -> accountID
	accountMap  map[int64]zNet.SessionIdType // accountID -> connID
	mutex       sync.RWMutex
	tcpClient   *zNet.TcpClient
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

	if gsp.tcpClient != nil {
		gsp.tcpClient.Close()
	}

	return nil
}

func (gsp *gameServerProxy) connectToGameServer(ctx context.Context) {
	// 解析GameServer地址
	tcpAddr, err := net.ResolveTCPAddr("tcp", gsp.config.GameServer.GameServerAddr)
	if err != nil {
		zLog.Error("Failed to resolve GameServer address", zap.Error(err))
		return
	}

	// 创建TcpClient配置
	clientConfig := &zNet.TcpClientConfig{
		ServerAddr:        tcpAddr.IP.String(),
		ServerPort:        tcpAddr.Port,
		HeartbeatDuration: 30,
		MaxPacketDataSize: 1024 * 1024,
		AutoReconnect:     true,
		ReconnectDelay:    5,
	}

	// 创建TcpClient
	gsp.tcpClient = zNet.NewTcpClient(clientConfig,
		zNet.WithClientLogger(zLog.GetStandardLogger()),
		zNet.WithClientStateCallback(func(state zNet.ClientState) {
			switch state {
			case zNet.ClientStateConnected:
				zLog.Info("Connected to GameServer", zap.String("addr", gsp.config.GameServer.GameServerAddr))
			case zNet.ClientStateDisconnected:
				zLog.Warn("Disconnected from GameServer")
			case zNet.ClientStateReconnecting:
				zLog.Info("Reconnecting to GameServer...")
			}
		}),
	)

	// 注册消息处理器
	gsp.tcpClient.RegisterDispatcher(gsp.handleGameServerMessage)

	// 连接到GameServer
	err = gsp.tcpClient.Connect()
	if err != nil {
		zLog.Error("Failed to connect to GameServer", zap.Error(err))
		return
	}
}

func (gsp *gameServerProxy) handleGameServerMessage(session zNet.Session, packet *zNet.NetPacket) error {
	// 处理GameServer消息
	gsp.processGameServerMessage(packet.Data)
	return nil
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

	// 处理内部消息
	if msgID == uint32(protocol.InternalMsgId_MSG_INTERNAL_SERVICE_HEARTBEAT) {
		// 处理心跳消息
		var heartbeatReq protocol.ServiceHeartbeatRequest
		if err := proto.Unmarshal(data[8:length], &heartbeatReq); err != nil {
			zLog.Error("Failed to unmarshal heartbeat request", zap.Error(err))
			return
		}
		zLog.Debug("Received heartbeat from GameServer", zap.Int32("server_id", heartbeatReq.ServerId), zap.Int32("online_count", heartbeatReq.OnlineCount))
		return
	}

	// 解析消息内容，提取sessionID
	var msg protocol.ClientMessage
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

func (gsp *gameServerProxy) SendToClient(sessionID zNet.SessionIdType, protoId int32, data []byte) error {
	if gsp.tcpClient == nil || !gsp.tcpClient.IsConnected() {
		return fmt.Errorf("not connected to GameServer")
	}

	// 使用Send方法发送消息
	err := gsp.tcpClient.Send(protoId, data)
	if err != nil {
		zLog.Error("Failed to send message to GameServer", zap.Error(err))
		return err
	}

	zLog.Info("Message sent to GameServer", zap.Uint64("session_id", uint64(sessionID)), zap.Int32("proto_id", protoId), zap.Int("data_len", len(data)))
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
