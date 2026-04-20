package player

import (
	"fmt"

	"github.com/pzqf/zCommon/game"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

func (p *Player) handleMessage(msg *PlayerMessage) {
	zLog.Debug("Handling player message",
		zap.Int64("player_id", int64(p.GetPlayerID())),
		zap.Int("source", int(msg.Source)),
		zap.Uint32("type", uint32(msg.Type)))

	switch msg.Type {
	case MsgNetEnterGame:
		p.handleNetEnterGame(msg)
	case MsgNetLeaveGame:
		p.handleNetLeaveGame(msg)
	case MsgNetMapEnter:
		p.handleNetMapEnter(msg)
	case MsgNetMapLeave:
		p.handleNetMapLeave(msg)
	case MsgNetMapMove:
		p.handleNetMapMove(msg)
	case MsgNetMapAttack:
		p.handleNetMapAttack(msg)
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
	case MsgLearnSkill:
		p.handleLearnSkill(msg)
	case MsgUseSkill:
		p.handleUseSkill(msg)
	case MsgUpgradeSkill:
		p.handleUpgradeSkill(msg)
	case MsgAddBuff:
		p.handleAddBuff(msg)
	case MsgRemoveBuff:
		p.handleRemoveBuff(msg)
	case MsgAcceptQuest:
		p.handleAcceptQuest(msg)
	case MsgCompleteQuest:
		p.handleCompleteQuest(msg)
	case MsgEquipItem:
		p.handleEquipItem(msg)
	case MsgUnequipItem:
		p.handleUnequipItem(msg)
	case MsgAOIEnterView:
		p.handleAOIEnterView(msg)
	case MsgAOILeaveView:
		p.handleAOILeaveView(msg)
	case MsgAOIMove:
		p.handleAOIMove(msg)
	default:
		zLog.Warn("Unknown message type",
			zap.Int64("player_id", int64(p.GetPlayerID())),
			zap.Uint32("message_type", uint32(msg.Type)))
	}
}

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

func (p *Player) handleAddItem(msg *PlayerMessage) {
	req, ok := msg.Data.(*AddItemRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	item := game.NewItem(int64(req.ItemID), int32(req.ItemID), "", game.ItemTypeConsumable, req.ItemCount)
	slot, err := p.inventory.AddItem(item)
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}

	if msg.Callback != nil {
		response := &ItemResponse{
			BaseResponse: BaseResponse{Success: true},
			ItemID:       req.ItemID,
			ItemCount:    req.ItemCount,
			Slot:         slot,
		}
		msg.Callback <- response
	}
}

func (p *Player) handleRemoveItem(msg *PlayerMessage) {
	req, ok := msg.Data.(*RemoveItemRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	_, err := p.inventory.RemoveItem(req.Slot, req.ItemCount)
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}

	if msg.Callback != nil {
		response := &ItemResponse{
			BaseResponse: BaseResponse{Success: true},
			ItemID:       req.ItemID,
			ItemCount:    0,
		}
		msg.Callback <- response
	}
}

func (p *Player) handleUseItem(msg *PlayerMessage) {
	req, ok := msg.Data.(*UseItemRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	_, err := p.inventory.RemoveItem(req.Slot, 1)
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}

	if msg.Callback != nil {
		response := &ItemResponse{
			BaseResponse: BaseResponse{Success: true},
			ItemID:       req.ItemID,
		}
		msg.Callback <- response
	}
}

func (p *Player) handleLearnSkill(msg *PlayerMessage) {
	req, ok := msg.Data.(*SkillRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	skill := game.NewSkill(req.SkillID, "", game.SkillTypeActive)
	err := p.skillMgr.LearnSkill(skill)
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &SkillResponse{
			BaseResponse: BaseResponse{Success: true},
			SkillID:      req.SkillID,
		}
	}
}

func (p *Player) handleUseSkill(msg *PlayerMessage) {
	req, ok := msg.Data.(*SkillRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	err := p.skillMgr.UseSkill(req.SkillID)
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &SkillResponse{
			BaseResponse: BaseResponse{Success: true},
			SkillID:      req.SkillID,
		}
	}
}

func (p *Player) handleUpgradeSkill(msg *PlayerMessage) {
	req, ok := msg.Data.(*SkillRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	err := p.skillMgr.UpgradeSkill(req.SkillID)
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &SkillResponse{
			BaseResponse: BaseResponse{Success: true},
			SkillID:      req.SkillID,
		}
	}
}

func (p *Player) handleAddBuff(msg *PlayerMessage) {
	req, ok := msg.Data.(*BuffRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	buff := game.NewBuff(req.BuffID, "", game.BuffTypePositive, req.Duration)
	p.buffMgr.AddBuff(buff)

	if msg.Callback != nil {
		msg.Callback <- &BuffResponse{
			BaseResponse: BaseResponse{Success: true},
			BuffID:       req.BuffID,
		}
	}
}

func (p *Player) handleRemoveBuff(msg *PlayerMessage) {
	req, ok := msg.Data.(*BuffRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	_, err := p.buffMgr.RemoveBuff(req.BuffID)
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &BuffResponse{
			BaseResponse: BaseResponse{Success: true},
			BuffID:       req.BuffID,
		}
	}
}

func (p *Player) handleAcceptQuest(msg *PlayerMessage) {
	req, ok := msg.Data.(*QuestRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	task := game.NewTask(req.QuestID, int32(req.QuestID), "", game.TaskTypeMain)
	err := p.taskMgr.AcceptTask(task)
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &QuestResponse{
			BaseResponse: BaseResponse{Success: true},
			QuestID:      req.QuestID,
		}
	}
}

func (p *Player) handleCompleteQuest(msg *PlayerMessage) {
	req, ok := msg.Data.(*QuestRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	_, err := p.taskMgr.CompleteTask(req.QuestID)
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &QuestResponse{
			BaseResponse: BaseResponse{Success: true},
			QuestID:      req.QuestID,
		}
	}
}

func (p *Player) handleEquipItem(msg *PlayerMessage) {
	req, ok := msg.Data.(*EquipRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	item, ok := p.inventory.GetItem(req.Slot)
	if !ok {
		p.sendErrorResponse(msg, fmt.Sprintf("item not found in slot %d", req.Slot))
		return
	}

	old, _ := p.equipment.Equip(game.EquipSlot(req.EquipSlot), item)
	if old != nil {
		p.inventory.AddItem(old)
	}

	p.inventory.RemoveItem(req.Slot, item.GetCount())

	p.syncEquipStats()

	if msg.Callback != nil {
		msg.Callback <- &EquipResponse{
			BaseResponse: BaseResponse{Success: true},
			EquipSlot:    req.EquipSlot,
		}
	}
}

func (p *Player) handleUnequipItem(msg *PlayerMessage) {
	req, ok := msg.Data.(*EquipRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	item, err := p.equipment.Unequip(game.EquipSlot(req.EquipSlot))
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}

	p.inventory.AddItem(item)

	p.syncEquipStats()

	if msg.Callback != nil {
		msg.Callback <- &EquipResponse{
			BaseResponse: BaseResponse{Success: true},
			EquipSlot:    req.EquipSlot,
		}
	}
}

func (p *Player) syncEquipStats() {
	if p.equipment == nil {
		return
	}
	attack := p.equipment.CalculateAttackBonus()
	defense := p.equipment.CalculateDefenseBonus()
	hp := p.equipment.CalculateHPBonus()
	mp := p.equipment.CalculateMPBonus()

	zLog.Debug("Equip stats synced",
		zap.Int32("attack", attack),
		zap.Int32("defense", defense),
		zap.Int32("hp", hp),
		zap.Int32("mp", mp))
}

func (p *Player) sendErrorResponse(msg *PlayerMessage, errMsg string) {
	if msg.Callback != nil {
		msg.Callback <- &BaseResponse{Success: false, Error: errMsg}
	}
	zLog.Error("Player message handler error",
		zap.Int64("player_id", int64(p.GetPlayerID())),
		zap.String("error", errMsg))
}


