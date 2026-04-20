package crossserver

import (
	"fmt"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type MessageProcessor struct {
	serverID int32
}

func NewMessageProcessor(serverID int32) *MessageProcessor {
	return &MessageProcessor{
		serverID: serverID,
	}
}

func (mp *MessageProcessor) WrapMessage(sessionID zNet.SessionIdType, protoId int32, playerID id.PlayerIdType, data []byte) ([]byte, error) {
	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: uint64(sessionID),
		PlayerId:  uint64(playerID),
		ServerId:  uint32(mp.serverID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}

	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(mp.serverID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeGateway),
		Message:      baseMsg,
	}

	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		zLog.Error("Failed to marshal cross server message", zap.Error(err))
		return nil, err
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, mp.serverID)
	wrappedData := crossserver.Wrap(meta, crossMsgData)

	zLog.Debug("Wrapped cross-server message",
		zap.Uint64("trace_id", meta.TraceID),
		zap.Uint64("request_id", meta.RequestID),
		zap.Int32("proto_id", protoId),
		zap.Int32("server_id", mp.serverID))

	return wrappedData, nil
}

func (mp *MessageProcessor) UnwrapMessage(data []byte) (int32, uint64, uint64, []byte, error) {
	meta, payload, wrapped, unwrapErr := crossserver.Unwrap(data)
	if unwrapErr != nil {
		zLog.Error("Invalid cross-server envelope", zap.Error(unwrapErr))
		return 0, 0, 0, nil, unwrapErr
	}
	if payload != nil {
		data = payload
	}
	if wrapped {
		zLog.Debug("Received cross-server envelope",
			zap.Uint64("trace_id", meta.TraceID),
			zap.Uint64("request_id", meta.RequestID),
			zap.Int32("server_id", mp.serverID))
	}

	var crossMsg protocol.CrossServerMessage
	if err := proto.Unmarshal(data, &crossMsg); err != nil {
		zLog.Error("Failed to unmarshal cross server message", zap.Error(err))
		return 0, 0, 0, nil, err
	}

	baseMsg := crossMsg.Message
	if baseMsg == nil {
		zLog.Error("Cross server message has no base message")
		return 0, 0, 0, nil, fmt.Errorf("no base message")
	}

	zLog.Debug("Unwrapped cross-server message",
		zap.Uint32("msg_id", baseMsg.MsgId),
		zap.Uint64("session_id", baseMsg.SessionId),
		zap.Uint64("player_id", baseMsg.PlayerId))

	return int32(baseMsg.MsgId), baseMsg.SessionId, baseMsg.PlayerId, baseMsg.Data, nil
}

func (mp *MessageProcessor) SendToGateway(session zNet.Session, sessionID zNet.SessionIdType, protoId int32, playerID id.PlayerIdType, data []byte) error {
	encodedMsg, err := mp.WrapMessage(sessionID, protoId, playerID, data)
	if err != nil {
		return err
	}

	err = session.Send(zNet.ProtoIdType(protoId), encodedMsg)
	if err != nil {
		zLog.Error("Failed to send message to Gateway", zap.Error(err))
		return err
	}

	zLog.Info("Message sent to Gateway",
		zap.Uint64("session_id", uint64(sessionID)),
		zap.Int32("proto_id", protoId),
		zap.Uint64("player_id", uint64(playerID)))

	return nil
}
