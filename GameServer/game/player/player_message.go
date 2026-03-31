package player

import (
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// handleMessage 处理消息
func (p *Player) handleMessage(msg *PlayerMessage) {
	zLog.Debug("Handling player message",
		zap.Int64("player_id", int64(p.GetPlayerID())),
		zap.Int("source", int(msg.Source)),
		zap.Uint32("type", uint32(msg.Type)))

	switch msg.Type {
	case MsgAddGold:
		p.handleAddGold(msg)
	case MsgDeductGold:
		p.handleDeductGold(msg)
	case MsgAddDiamond:
		p.handleAddDiamond(msg)
	case MsgDeductDiamond:
		p.handleDeductDiamond(msg)
	case MsgAddItem:
		p.handleAddItem(msg)
	case MsgRemoveItem:
		p.handleRemoveItem(msg)
	case MsgUseItem:
		p.handleUseItem(msg)
	default:
		zLog.Warn("Unknown message type",
			zap.Int64("player_id", int64(p.GetPlayerID())),
			zap.Uint32("message_type", uint32(msg.Type)))
	}
}

// handleAddGold 处理增加金币消息
func (p *Player) handleAddGold(msg *PlayerMessage) {
	req, ok := msg.Data.(*GoldRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	p.AddGold(req.Amount)

	if msg.Callback != nil {
		response := &GoldResponse{
			BaseResponse: BaseResponse{Success: true},
			CurrentGold:  p.GetGold(),
		}
		msg.Callback <- response
	}
}

// handleDeductGold 处理扣除金币消息
func (p *Player) handleDeductGold(msg *PlayerMessage) {
	req, ok := msg.Data.(*GoldRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	success := p.ReduceGold(req.Amount)

	if msg.Callback != nil {
		response := &GoldResponse{
			BaseResponse: BaseResponse{Success: success, Error: ""},
			CurrentGold:  p.GetGold(),
		}
		if !success {
			response.Error = "insufficient gold"
		}
		msg.Callback <- response
	}
}

// handleAddDiamond 处理增加钻石消息
func (p *Player) handleAddDiamond(msg *PlayerMessage) {
	req, ok := msg.Data.(*DiamondRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	p.AddDiamond(req.Amount)

	if msg.Callback != nil {
		response := &DiamondResponse{
			BaseResponse:   BaseResponse{Success: true},
			CurrentDiamond: p.GetDiamond(),
		}
		msg.Callback <- response
	}
}

// handleDeductDiamond 处理扣除钻石消息
func (p *Player) handleDeductDiamond(msg *PlayerMessage) {
	req, ok := msg.Data.(*DiamondRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	success := p.ReduceDiamond(req.Amount)

	if msg.Callback != nil {
		response := &DiamondResponse{
			BaseResponse:   BaseResponse{Success: success, Error: ""},
			CurrentDiamond: p.GetDiamond(),
		}
		if !success {
			response.Error = "insufficient diamond"
		}
		msg.Callback <- response
	}
}

// handleAddItem 处理增加物品消息
func (p *Player) handleAddItem(msg *PlayerMessage) {
	req, ok := msg.Data.(*AddItemRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	// TODO: 实现物品添加逻辑

	if msg.Callback != nil {
		response := &ItemResponse{
			BaseResponse: BaseResponse{Success: true},
			ItemID:       req.ItemID,
			ItemCount:    req.ItemCount,
		}
		msg.Callback <- response
	}
}

// handleRemoveItem 处理移除物品消息
func (p *Player) handleRemoveItem(msg *PlayerMessage) {
	req, ok := msg.Data.(*RemoveItemRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	// TODO: 实现物品移除逻辑

	if msg.Callback != nil {
		response := &ItemResponse{
			BaseResponse: BaseResponse{Success: true},
			ItemID:       req.ItemID,
			ItemCount:    0,
		}
		msg.Callback <- response
	}
}

// handleUseItem 处理使用物品消息
func (p *Player) handleUseItem(msg *PlayerMessage) {
	req, ok := msg.Data.(*UseItemRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	// TODO: 实现物品使用逻辑

	if msg.Callback != nil {
		response := &ItemResponse{
			BaseResponse: BaseResponse{Success: true},
			ItemID:       req.ItemID,
		}
		msg.Callback <- response
	}
}

// sendErrorResponse 发送错误响应
func (p *Player) sendErrorResponse(msg *PlayerMessage, errMsg string) {
	if msg.Callback != nil {
		msg.Callback <- &BaseResponse{Success: false, Error: errMsg}
	}
	zLog.Error("Player message handler error",
		zap.Int64("player_id", int64(p.GetPlayerID())),
		zap.String("error", errMsg))
}

// SendMessage 发送消息到玩家Actor
func (p *Player) SendMessage(msg *PlayerMessage) error {
	p.BaseActor.SendMessage(msg)
	return nil
}

// SendMessageToPlayer 发送消息给指定玩家（便捷方法）
func SendMessageToPlayer(playerID int64, source MessageSource, msgType MessageType, data interface{}) error {
	return fmt.Errorf("use PlayerManager.RouteMessage instead")
}
