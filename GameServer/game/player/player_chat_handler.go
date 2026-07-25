package player

import (
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// handleChatNotify 收到一条聊天广播（经 PlayerManager.BroadcastMessage 扇出到本玩家 Actor）→ 推客户端。
func (p *Player) handleChatNotify(msg *PlayerMessage) {
	data, ok := msg.Data.(*ChatNotifyData)
	if !ok {
		return
	}
	notify := &protocol.ClientChatNotify{
		Channel:      data.Channel,
		FromPlayerId: data.FromPlayerID,
		FromName:     data.FromName,
		Text:         data.Text,
	}
	out, err := proto.Marshal(notify)
	if err != nil {
		zLog.Error("Failed to marshal chat notify", zap.Error(err))
		return
	}
	p.pushToClient(int32(protocol.ChatMsgId_MSG_CHAT_NOTIFY), out)
}
