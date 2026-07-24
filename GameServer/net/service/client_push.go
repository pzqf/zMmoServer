package service

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zNet"
	"google.golang.org/protobuf/proto"
)

// SendToClient 实现 game/common.ClientSender：把服务端主动消息（如 AOI 视野推送）发往客户端。
//
// 经已接受的 Gateway 会话（与请求-响应同一 zNet 通道/分帧）发送，BaseMessage.SessionId 携带
// 客户端会话 ID，供 Gateway 路由到正确客户端。此前 PlayerClientSender→gatewayProxy→
// cm.SendToGateway(cm.gatewayConn) 的路径因连接拓扑相反（实际是 Gateway 连 GameServer，
// 而非 GameServer 连 Gateway）、cm.gatewayConn 从未被设置，导致所有服务端主动推送恒失败
// "gateway not connected"；改走此路径后与请求-响应一致，服务端推送可达客户端。
func (ts *TCPService) SendToClient(sessionID interface{}, protoId int32, data []byte) error {
	v := ts.gatewaySession.Load()
	if v == nil {
		return fmt.Errorf("gateway session not established")
	}
	gwSession, ok := v.(zNet.Session)
	if !ok || gwSession == nil {
		return fmt.Errorf("invalid gateway session")
	}

	// Player.sessionID 存的是字符串（EnterGame(sessionID string)），也兼容数值型
	var clientSessionID zNet.SessionIdType
	switch v := sessionID.(type) {
	case uint64: // zNet.SessionIdType 即 uint64（类型别名）
		clientSessionID = zNet.SessionIdType(v)
	case string:
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			clientSessionID = zNet.SessionIdType(n)
		}
	}
	if clientSessionID == 0 {
		return fmt.Errorf("invalid client session id: %v", sessionID)
	}

	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: uint64(clientSessionID),
		ServerId:  uint32(ts.config.Server.ServerID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}
	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(ts.config.Server.ServerID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeGateway),
		Message:      baseMsg,
	}
	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		return fmt.Errorf("marshal cross server message: %w", err)
	}
	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ts.config.Server.ServerID))
	wrapped := crossserver.Wrap(meta, crossMsgData)
	return gwSession.Send(zNet.ProtoIdType(protoId), wrapped)
}
