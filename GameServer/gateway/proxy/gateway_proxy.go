package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/message"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/config"
	msgHandler "github.com/pzqf/zMmoServer/GameServer/handler/message"
	"go.uber.org/zap"
)

type GatewayProxy interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SendToGateway(sessionID zNet.SessionIdType, protoId int32, data []byte) error
}

type gatewayProxy struct {
	config     *config.Config
	tcpServer  *zNet.TcpServer
	router     *msgHandler.Router
}

func NewGatewayProxy(cfg *config.Config) GatewayProxy {
	router := msgHandler.NewRouter()
	// 这里需要注册消息处理器
	// 暂时只创建路由器，后续会在初始化时注册具体的处理器
	return &gatewayProxy{
		config: cfg,
		router: router,
	}
}

func (gp *gatewayProxy) Start(ctx context.Context) error {
	zLog.Info("Starting Gateway proxy...")

	// 创建zNet.TcpServer配置（用于Gateway连接）
	tcpConfig := &zNet.TcpConfig{
		ListenAddress:       gp.config.Server.ListenAddr,
		MaxClientCount:      gp.config.Server.MaxConnections,
		HeartbeatDuration:   gp.config.Server.HeartbeatInterval,
		ChanSize:            gp.config.Server.ChanSize,
		MaxPacketDataSize:   int32(gp.config.Server.MaxPacketDataSize),
		UseWorkerPool:       gp.config.Server.UseWorkerPool,
		WorkerPoolSize:      gp.config.Server.WorkerPoolSize,
		WorkerQueueSize:     gp.config.Server.WorkerQueueSize,
		DisableEncryption:   gp.config.Server.DisableEncryption,
		EnableKeyRotation:   gp.config.Server.EnableKeyRotation,
		KeyRotationInterval: time.Duration(gp.config.Server.KeyRotationInterval) * time.Second,
		MaxHistoryKeys:      gp.config.Server.MaxHistoryKeys,
		EnableSequenceCheck: gp.config.Server.EnableSequenceCheck,
		SequenceWindowSize:  gp.config.Server.SequenceWindowSize,
		TimestampTolerance:  gp.config.Server.TimestampTolerance,
	}

	// 创建zNet.TcpServer（用于Gateway连接）
	gp.tcpServer = zNet.NewTcpServer(tcpConfig)

	// 注册消息处理器
	gp.tcpServer.RegisterDispatcher(gp.handleGatewayMessage)

	// 启动服务
	err := gp.tcpServer.Start()
	if err != nil {
		return fmt.Errorf("failed to start TCP service: %v", err)
	}

	zLog.Info("Gateway proxy started successfully", zap.String("addr", gp.config.Server.ListenAddr))

	return nil
}

func (gp *gatewayProxy) Stop(ctx context.Context) error {
	zLog.Info("Stopping Gateway proxy...")

	if gp.tcpServer != nil {
		gp.tcpServer.Close()
	}

	return nil
}

func (gp *gatewayProxy) handleGatewayMessage(session zNet.Session, packet *zNet.NetPacket) error {
	// 处理Gateway消息
	gp.processGatewayMessage(session, packet.ProtoId, packet.Data)
	return nil
}

func (gp *gatewayProxy) processGatewayMessage(session zNet.Session, protoId zNet.ProtoIdType, data []byte) {
	meta, payload, wrapped, unwrapErr := crossserver.Unwrap(data)
	if unwrapErr != nil {
		zLog.Error("Invalid cross-server envelope from Gateway", zap.Error(unwrapErr))
		return
	}
	if payload != nil {
		data = payload
	}
	if wrapped {
		zLog.Debug("Received cross-server envelope from Gateway",
			zap.Uint64("trace_id", meta.TraceID),
			zap.Uint64("request_id", meta.RequestID),
			zap.Int32("proto_id", int32(protoId)),
			zap.Int("server_id", gp.config.Server.ServerID))
	}

	// 使用消息解码
	msg, err := message.Decode(data)
	if err != nil {
		zLog.Error("Failed to decode message from Gateway", zap.Error(err))
		return
	}

	// 解析跨服务器消息
	var crossMsg crossserver.CrossServerMessage
	if err := json.Unmarshal(msg.Data, &crossMsg); err != nil {
		zLog.Error("Failed to unmarshal cross server message", zap.Error(err))
		return
	}

	// 提取基础消息
	baseMsg := crossMsg.Message

	// 使用消息路由器处理消息
	if gp.router != nil {
		err := gp.router.Handle(session, int32(msg.Header.MsgID), baseMsg.Data)
		if err != nil {
			zLog.Error("Failed to handle message", zap.Error(err), zap.Uint32("msg_id", msg.Header.MsgID))
		}
	}

	zLog.Info("Received message from Gateway",
		zap.Uint32("msg_id", msg.Header.MsgID),
		zap.Uint64("session_id", baseMsg.SessionID),
		zap.Uint64("player_id", baseMsg.PlayerID))
}

func (gp *gatewayProxy) SendToGateway(sessionID zNet.SessionIdType, protoId int32, data []byte) error {
	if gp.tcpServer == nil {
		return fmt.Errorf("TCP server not initialized")
	}

	// 创建基础消息
	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: uint64(sessionID),
		ServerId:  uint32(gp.config.Server.ServerID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}

	// 创建跨服务器消息
	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(gp.config.Server.ServerID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeGateway),
		Message:      baseMsg,
	}

	// 使用Protocol Buffers序列化消息
	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		zLog.Error("Failed to marshal cross server message", zap.Error(err))
		return err
	}

	// 编码消息
	encodedMsg, err := message.Encode(uint32(protoId), crossMsgData)
	if err != nil {
		zLog.Error("Failed to encode message", zap.Error(err))
		return err
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(gp.config.Server.ServerID))
	encodedMsg = crossserver.Wrap(meta, encodedMsg)
	zLog.Debug("Sending cross-server envelope to Gateway",
		zap.Uint64("trace_id", meta.TraceID),
		zap.Uint64("request_id", meta.RequestID),
		zap.Int32("proto_id", protoId),
		zap.Int("server_id", gp.config.Server.ServerID))

	// 这里需要找到对应的Gateway连接并发送消息
	// 暂时返回错误，后续实现
	return fmt.Errorf("SendToGateway not implemented yet")
}
