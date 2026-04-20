package proxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/message"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	msgHandler "github.com/pzqf/zMmoServer/GameServer/handler/message"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type GatewayProxy interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SendToGateway(sessionID zNet.SessionIdType, protoId int32, data []byte) error
	BroadcastToGateway(protoId int32, data []byte) error
	IsConnected() bool
}

type gatewayProxy struct {
	config  *config.Config
	connMgr *connection.ConnectionManager
	router  *msgHandler.Router
}

func NewGatewayProxy(cfg *config.Config, connMgr *connection.ConnectionManager) GatewayProxy {
	router := msgHandler.NewRouter()
	return &gatewayProxy{
		config:  cfg,
		connMgr: connMgr,
		router:  router,
	}
}

func (gp *gatewayProxy) Start(ctx context.Context) error {
	zLog.Info("Starting Gateway proxy...")

	go gp.waitForGatewayConnection(ctx)

	return nil
}

func (gp *gatewayProxy) waitForGatewayConnection(ctx context.Context) {
	if gp.connMgr.IsGatewayConnected() {
		zLog.Info("Gateway already connected")
		go gp.receiveFromGateway(ctx)
		return
	}

	select {
	case <-gp.connMgr.GatewayConnectedChan():
		zLog.Info("Gateway connection established")
		go gp.receiveFromGateway(ctx)
	case <-ctx.Done():
		zLog.Info("Gateway proxy stopped while waiting for connection")
	}
}

func (gp *gatewayProxy) Stop(ctx context.Context) error {
	zLog.Info("Stopping Gateway proxy...")
	return nil
}

func (gp *gatewayProxy) IsConnected() bool {
	return gp.connMgr != nil && gp.connMgr.IsGatewayConnected()
}

func (gp *gatewayProxy) SendToGateway(sessionID zNet.SessionIdType, protoId int32, data []byte) error {
	if !gp.IsConnected() {
		return fmt.Errorf("gateway not connected")
	}

	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: uint64(sessionID),
		ServerId:  uint32(gp.config.Server.ServerID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}

	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(gp.config.Server.ServerID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeGateway),
		Message:      baseMsg,
	}

	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		return fmt.Errorf("marshal cross server message: %w", err)
	}

	encodedMsg, err := message.Encode(uint32(protoId), crossMsgData)
	if err != nil {
		return fmt.Errorf("encode message: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(gp.config.Server.ServerID))
	encodedMsg = crossserver.Wrap(meta, encodedMsg)

	packet := gp.buildPacket(uint32(protoId), encodedMsg)

	return gp.connMgr.SendToGateway(packet)
}

func (gp *gatewayProxy) BroadcastToGateway(protoId int32, data []byte) error {
	if !gp.IsConnected() {
		return fmt.Errorf("gateway not connected")
	}

	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		ServerId:  uint32(gp.config.Server.ServerID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}

	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(gp.config.Server.ServerID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeGateway),
		Message:      baseMsg,
	}

	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		return fmt.Errorf("marshal cross server message: %w", err)
	}

	encodedMsg, err := message.Encode(uint32(protoId), crossMsgData)
	if err != nil {
		return fmt.Errorf("encode message: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(gp.config.Server.ServerID))
	encodedMsg = crossserver.Wrap(meta, encodedMsg)

	packet := gp.buildPacket(uint32(protoId), encodedMsg)

	return gp.connMgr.SendToGateway(packet)
}

func (gp *gatewayProxy) buildPacket(msgID uint32, data []byte) []byte {
	length := uint32(8 + len(data))
	buf := make([]byte, 4+4+len(data))
	binary.BigEndian.PutUint32(buf[0:4], length)
	binary.BigEndian.PutUint32(buf[4:8], msgID)
	copy(buf[8:], data)
	return buf
}

func (gp *gatewayProxy) receiveFromGateway(ctx context.Context) {
	conn := gp.connMgr.GetGatewayConn()
	if conn == nil {
		return
	}

	buffer := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		n, err := conn.Read(buffer)
		if err != nil {
			zLog.Error("Failed to read from Gateway", zap.Error(err))
			return
		}

		if n > 0 {
			gp.processGatewayData(buffer[:n])
		}
	}
}

func (gp *gatewayProxy) processGatewayData(data []byte) {
	if len(data) < 8 {
		return
	}

	length := binary.BigEndian.Uint32(data[:4])
	if len(data) < int(length) {
		return
	}

	msgID := binary.BigEndian.Uint32(data[4:8])
	payload := data[8:length]

	meta, unwrapped, wrapped, unwrapErr := crossserver.Unwrap(payload)
	if unwrapErr != nil {
		zLog.Error("Invalid cross-server envelope from Gateway", zap.Error(unwrapErr))
		return
	}
	if wrapped {
		payload = unwrapped
		zLog.Debug("Received cross-server envelope from Gateway",
			zap.Uint64("trace_id", meta.TraceID),
			zap.Uint64("request_id", meta.RequestID),
			zap.Uint32("msg_id", msgID))
	}

	var crossMsg protocol.CrossServerMessage
	if err := proto.Unmarshal(payload, &crossMsg); err != nil {
		zLog.Error("Failed to unmarshal cross server message", zap.Error(err))
		return
	}

	baseMsg := crossMsg.Message
	if baseMsg == nil {
		zLog.Error("Cross server message has no base message")
		return
	}

	if gp.router != nil {
		if err := gp.router.Handle(nil, int32(baseMsg.MsgId), baseMsg.Data); err != nil {
			zLog.Error("Failed to handle message",
				zap.Error(err),
				zap.Uint32("msg_id", baseMsg.MsgId))
		}
	}

	zLog.Debug("Received message from Gateway",
		zap.Uint32("msg_id", baseMsg.MsgId),
		zap.Uint64("session_id", baseMsg.SessionId))
}

type PlayerClientSender struct {
	gatewayProxy GatewayProxy
}

func NewPlayerClientSender(gp GatewayProxy) *PlayerClientSender {
	return &PlayerClientSender{gatewayProxy: gp}
}

func (s *PlayerClientSender) SendToClient(sessionID interface{}, protoId int32, data []byte) error {
	sid, ok := sessionID.(zNet.SessionIdType)
	if !ok {
		sid = zNet.SessionIdType(0)
	}
	return s.gatewayProxy.SendToGateway(sid, protoId, data)
}
