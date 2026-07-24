package crossserver

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/pzqf/zCommon/protocol"
)

// 本文件是 GameServer <-> MapServer 跨服消息的【单一编解码真相源】。
//
// 统一契约（成熟化改造 Phase 0.3 定稿，见 docs/协议契约.md）：
//   - 线格式 = crossserver.Wrap(meta, payload) 得到的 40 字节 Envelope + payload
//   - payload = proto.Marshal(protocol.CrossServerMessage)  —— 全程 protobuf，禁止 JSON
//   - 内层 protocol.BaseMessage.Data = proto.Marshal(具体业务消息)
//   - 外层 zNet ProtoId == 内层 BaseMessage.MsgId == InternalMsgId（400/402/404/406 请求，
//     401/403/405/407 响应），三者一致，便于收发两端按同一 ID 路由。
//
// 收发两端（GameServer 的 map_service / connection，MapServer 的 net/service）
// 必须只经由本文件的 PackMessage / UnpackMessage / BuildBaseMessage 进行编解码，
// 不得各自手写 proto/json 拼装，以免再次出现编码或类型不一致导致链路断裂。

// BuildBaseMessage 构造内层 BaseMessage：把已 marshal 的业务消息字节包进带路由字段的 BaseMessage。
// msgID 应为 InternalMsgId（如 MSG_INTERNAL_MAP_ENTER_REQUEST=400）。
func BuildBaseMessage(msgID uint32, playerID uint64, serverID uint32, mapID uint32, payload []byte) *protocol.BaseMessage {
	return &protocol.BaseMessage{
		MsgId:    msgID,
		PlayerId: playerID,
		ServerId: serverID,
		MapId:    mapID,
		Data:     payload,
	}
}

// PackMessage 将内层 BaseMessage 封装为「Envelope + protobuf CrossServerMessage」线格式字节。
// meta 决定 Envelope 头（TraceID/RequestID/MessageType 等），fromService/toService/服务器 ID
// 写入 CrossServerMessage 供路由。返回可直接经 zNet Send 发送的字节。
func PackMessage(meta Meta, fromService, toService uint8, fromServerID, toServerID uint32, base *protocol.BaseMessage) ([]byte, error) {
	cross := &protocol.CrossServerMessage{
		TraceId:      meta.TraceID,
		FromServerId: fromServerID,
		ToServerId:   toServerID,
		FromService:  uint32(fromService),
		ToService:    uint32(toService),
		Message:      base,
	}
	body, err := proto.Marshal(cross)
	if err != nil {
		return nil, fmt.Errorf("marshal cross server message: %w", err)
	}
	return Wrap(meta, body), nil
}

// UnpackMessage 解析 PackMessage 产出的线格式字节，返回 Envelope meta、外层 CrossServerMessage
// 及内层 BaseMessage（= cross.Message，可能为 nil）。要求数据带合法 Envelope 头；
// 非本协议流量（Magic 不符）视为错误，避免静默吞掉。
func UnpackMessage(data []byte) (Meta, *protocol.CrossServerMessage, *protocol.BaseMessage, error) {
	meta, payload, wrapped, err := Unwrap(data)
	if err != nil {
		return Meta{}, nil, nil, fmt.Errorf("unwrap envelope: %w", err)
	}
	if !wrapped {
		return Meta{}, nil, nil, fmt.Errorf("missing or invalid cross-server envelope")
	}

	cross := &protocol.CrossServerMessage{}
	if err := proto.Unmarshal(payload, cross); err != nil {
		return Meta{}, nil, nil, fmt.Errorf("unmarshal cross server message: %w", err)
	}
	return meta, cross, cross.Message, nil
}
