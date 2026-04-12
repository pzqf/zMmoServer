package handler

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zNet"
)

// MessageHandler 消息处理器
type MessageHandler struct {}

// NewMessageHandler 创建新的消息处理器
func NewMessageHandler() *MessageHandler {
	return &MessageHandler{}
}

// HandleMessage 处理消息
func (h *MessageHandler) HandleMessage(protoId uint32, data []byte) {
	// 解析消息内容
	switch protoId {
	case uint32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE):
		h.handlePlayerLoginResponse(data)
	case uint32(protocol.PlayerMsgId_MSG_PLAYER_CREATE_RESPONSE):
		h.handlePlayerCreateResponse(data)
	case uint32(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE):
		h.handleMapEnterResponse(data)
	case uint32(protocol.MapMsgId_MSG_MAP_MOVE_RESPONSE):
		h.handleMapMoveResponse(data)
	case uint32(protocol.MapMsgId_MSG_MAP_ATTACK_RESPONSE):
		h.handleMapAttackResponse(data)
	default:
		fmt.Printf("Received message: ProtoId=%d, DataSize=%d\n", protoId, len(data))
	}
}

// handlePlayerLoginResponse 处理玩家登录响应
func (h *MessageHandler) handlePlayerLoginResponse(data []byte) {
	var resp protocol.PlayerLoginResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal PlayerLoginResponse: %v\n", err)
		return
	}
	fmt.Printf("PlayerLoginResponse: Result=%d, ErrorMsg=%s\n", resp.Result, resp.ErrorMsg)
	if resp.Result == 0 && resp.PlayerInfo != nil {
		fmt.Printf("Player: ID=%d, Name=%s, Level=%d, Gold=%d\n",
			resp.PlayerInfo.PlayerId, resp.PlayerInfo.Name, resp.PlayerInfo.Level, resp.PlayerInfo.Gold)
	}
}

// handlePlayerCreateResponse 处理角色创建响应
func (h *MessageHandler) handlePlayerCreateResponse(data []byte) {
	var resp protocol.PlayerCreateResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal PlayerCreateResponse: %v\n", err)
		return
	}
	fmt.Printf("PlayerCreateResponse: Result=%d, ErrorMsg=%s\n", resp.Result, resp.ErrorMsg)
	if resp.Result == 0 && resp.PlayerInfo != nil {
		fmt.Printf("Player: ID=%d, Name=%s, Level=%d, Sex=%d, Age=%d\n",
			resp.PlayerInfo.PlayerId, resp.PlayerInfo.Name, resp.PlayerInfo.Level, 0, 0)
	}
}

// handleMapEnterResponse 处理进入地图响应
func (h *MessageHandler) handleMapEnterResponse(data []byte) {
	var resp protocol.ClientMapEnterResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal ClientMapEnterResponse: %v\n", err)
		return
	}
	fmt.Printf("ClientMapEnterResponse: Result=%d, ErrorMsg=%s, MapID=%d\n", resp.Result, resp.ErrorMsg, resp.MapId)
}

// handleMapMoveResponse 处理移动响应
func (h *MessageHandler) handleMapMoveResponse(data []byte) {
	var resp protocol.ClientMapMoveResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal ClientMapMoveResponse: %v\n", err)
		return
	}
	fmt.Printf("ClientMapMoveResponse: Result=%d\n", resp.Result)
}

// handleMapAttackResponse 处理攻击响应
func (h *MessageHandler) handleMapAttackResponse(data []byte) {
	var resp protocol.ClientMapAttackResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal ClientMapAttackResponse: %v\n", err)
		return
	}
	fmt.Printf("ClientMapAttackResponse: Result=%d, TargetID=%d, Damage=%d, TargetHp=%d\n",
		resp.Result, resp.TargetId, resp.Damage, resp.TargetHp)
}

// GetDispatcher 获取消息分发器
func (h *MessageHandler) GetDispatcher() zNet.HandlerFun {
	return func(session zNet.Session, netPacket *zNet.NetPacket) error {
		h.HandleMessage(uint32(netPacket.ProtoId), netPacket.Data)
		return nil
	}
}
