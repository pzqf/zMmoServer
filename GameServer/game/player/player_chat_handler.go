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

// handleTeamUpdate 收到一条队伍花名册变更（已 marshal 好的字节）→ 直接推客户端。
func (p *Player) handleTeamUpdate(msg *PlayerMessage) {
	data, ok := msg.Data.(*TeamUpdateData)
	if !ok || data.Data == nil {
		return
	}
	p.pushToClient(int32(protocol.TeamMsgId_MSG_TEAM_UPDATE_NOTIFY), data.Data)
}

// handleTradeUpdate 收到一条交易状态变更（已 marshal 好的字节）→ 直接推客户端。
func (p *Player) handleTradeUpdate(msg *PlayerMessage) {
	data, ok := msg.Data.(*TradeUpdateData)
	if !ok || data.Data == nil {
		return
	}
	p.pushToClient(int32(protocol.TradeMsgId_MSG_TRADE_UPDATE_NOTIFY), data.Data)
}
