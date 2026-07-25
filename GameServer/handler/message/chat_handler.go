package message

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// ChatHandler 网关侧聊天处理器（业务层建设 2026-07-25）。
// 与 ItemHandler 的「per-player 请求→回调」不同：聊天发送把消息**扇出给全服在线玩家**（世界频道），
// 测「一个玩家发起 → 广播 → 多客户端接收」这条此前业务未覆盖的架构路径。
type ChatHandler struct {
	playerManager *player.PlayerManager
	serverID      int32
}

func NewChatHandler(pm *player.PlayerManager, serverID int32) *ChatHandler {
	return &ChatHandler{playerManager: pm, serverID: serverID}
}

func (h *ChatHandler) Handle(session zNet.Session, protoId int32, data []byte) error {
	if protoId != int32(protocol.ChatMsgId_MSG_CHAT_SEND) {
		return nil
	}
	var req protocol.ClientChatSendRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal chat send", zap.Error(err))
		return err
	}
	if req.Text == "" {
		return nil
	}

	fromName := ""
	if p, err := h.playerManager.GetPlayer(id.PlayerIdType(req.PlayerId)); err == nil && p != nil {
		fromName = p.GetName()
	}
	zLog.Info("Chat send",
		zap.Int64("from", req.PlayerId), zap.Int32("channel", req.Channel), zap.String("text", req.Text))

	notify := &player.ChatNotifyData{
		Channel:      req.Channel,
		FromPlayerID: req.PlayerId,
		FromName:     fromName,
		Text:         req.Text,
	}
	// 世界频道：扇出给全服在线玩家（含发送者自己）。附近频道(CHAT_AREA)预留，可换 BroadcastMessageToPlayers。
	h.playerManager.BroadcastMessage(player.NewPlayerMessage(0, player.SourceGateway, player.MsgChatNotify, notify))
	return nil
}
