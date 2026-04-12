package crossserver

import (
	"encoding/json"
	"time"

	"google.golang.org/protobuf/proto"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/message"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"go.uber.org/zap"
)

// MessageProcessor 跨服务器消息处理器
type MessageProcessor struct {
	serverID int32
}

// NewMessageProcessor 创建跨服务器消息处理器
func NewMessageProcessor(serverID int32) *MessageProcessor {
	return &MessageProcessor{
		serverID: serverID,
	}
}

// WrapMessage 封装跨服务器消息
func (mp *MessageProcessor) WrapMessage(sessionID zNet.SessionIdType, protoId int32, playerID id.PlayerIdType, data []byte) ([]byte, error) {
	// 创建基础消息
	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: uint64(sessionID),
		PlayerId:  uint64(playerID),
		ServerId:  uint32(mp.serverID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}

	// 创建跨服务器消息
	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(mp.serverID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeGateway),
		Message:      baseMsg,
	}

	// 使用Protocol Buffers序列化消息
	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		zLog.Error("Failed to marshal cross server message", zap.Error(err))
		return nil, err
	}

	// 编码消息
	encodedMsg, err := message.Encode(uint32(protoId), crossMsgData)
	if err != nil {
		zLog.Error("Failed to encode message", zap.Error(err))
		return nil, err
	}

	// 包装消息
	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, mp.serverID)
	encodedMsg = crossserver.Wrap(meta, encodedMsg)

	zLog.Debug("Wrapped cross-server message",
		zap.Uint64("trace_id", meta.TraceID),
		zap.Uint64("request_id", meta.RequestID),
		zap.Int32("proto_id", protoId),
		zap.Int32("server_id", mp.serverID))

	return encodedMsg, nil
}

// UnwrapMessage 解析跨服务器消息
func (mp *MessageProcessor) UnwrapMessage(data []byte) (int32, uint64, uint64, []byte, error) {
	// 解包装
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

	// 解码消息
	msg, err := message.Decode(data)
	if err != nil {
		zLog.Error("Failed to decode message", zap.Error(err))
		return 0, 0, 0, nil, err
	}

	// 解析跨服务器消息
	var crossMsg crossserver.CrossServerMessage
	if err := json.Unmarshal(msg.Data, &crossMsg); err != nil {
		zLog.Error("Failed to unmarshal cross server message", zap.Error(err))
		return 0, 0, 0, nil, err
	}

	// 提取基础消息
	baseMsg := crossMsg.Message

	zLog.Debug("Unwrapped cross-server message",
		zap.Uint32("msg_id", msg.Header.MsgID),
		zap.Uint64("session_id", baseMsg.SessionID),
		zap.Uint64("player_id", baseMsg.PlayerID))

	return int32(msg.Header.MsgID), baseMsg.SessionID, baseMsg.PlayerID, baseMsg.Data, nil
}

// SendToGateway 发送消息到Gateway
func (mp *MessageProcessor) SendToGateway(session zNet.Session, sessionID zNet.SessionIdType, protoId int32, playerID id.PlayerIdType, data []byte) error {
	// 封装消息
	encodedMsg, err := mp.WrapMessage(sessionID, protoId, playerID, data)
	if err != nil {
		return err
	}

	// 发送消息
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
