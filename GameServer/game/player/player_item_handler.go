package player

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/game"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// 物品 / 仓库 handler（业务层建设 2026-07-25）。均在 Player actor goroutine 上运行（经 RouteMessage），
// 操作 Player 自己的 inventory / warehouse——纯本地、无跨服往返，故不阻塞 actor。响应经 msg.Callback
// 以 protobuf 回给网关 handler，再下发客户端。

func (p *Player) replyItem(msg *PlayerMessage, protoId protocol.ItemMsgId, m proto.Message) {
	if msg.Callback == nil {
		return
	}
	data, err := proto.Marshal(m)
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}
	msg.Callback <- &NetResponse{ProtoId: int32(protoId), Data: data}
}

func itemsToSlots(items map[int32]*game.Item) []*protocol.ItemSlot {
	slots := make([]*protocol.ItemSlot, 0, len(items))
	for slot, it := range items {
		slots = append(slots, &protocol.ItemSlot{Slot: slot, ItemId: it.ConfigID, Count: it.GetCount()})
	}
	return slots
}

// handleNetItemPickup 客户端就近拾取请求（fire）：转发到 MapServer（物品权威）。拾到的物品由
// MapServer 判定后经 ItemGrant(506) 异步回推，由 handleItemGrant 落背包并推客户端 1307。
func (p *Player) handleNetItemPickup(msg *PlayerMessage) {
	req, ok := msg.Data.(*protocol.ClientItemPickupRequest)
	if !ok {
		return
	}
	mapOp := p.mapOp
	if mapOp == nil {
		return
	}
	playerID := id.PlayerIdType(req.PlayerId)
	mapID := id.MapIdType(req.MapId)
	go func() {
		if err := mapOp.Pickup(playerID, mapID); err != nil {
			zLog.Warn("Failed to forward pickup to MapServer", zap.Error(err))
		}
	}()
}

// handleItemGrant MapServer 拾取授权回推：把物品发进背包并推客户端拾取结果（1307）。
func (p *Player) handleItemGrant(msg *PlayerMessage) {
	req, ok := msg.Data.(*ItemGrantRequest)
	if !ok {
		return
	}
	notify := &protocol.ClientItemPickupNotify{ItemId: req.ItemID, Count: req.Count}
	if _, err := p.inventory.AddItem(game.NewItem(int64(req.ItemID), req.ItemID, "Loot", game.ItemTypeConsumable, req.Count)); err != nil {
		notify.Result = 1
		notify.Error = err.Error()
	}
	data, err := proto.Marshal(notify)
	if err != nil {
		return
	}
	p.pushToClient(int32(protocol.ItemMsgId_MSG_ITEM_PICKUP_NOTIFY), data)
}

// handleExpGrant 把 MapServer 战斗回写的经验加到持久化 actor（修 F-2）。在 actor 单写者 goroutine 内执行，
// 升级公式 expToNext=1000*level 与 MapServer 战斗对象一致（object.Player.LevelUp）。经验/等级随登出/周期存盘落库。
func (p *Player) handleExpGrant(msg *PlayerMessage) {
	req, ok := msg.Data.(*ExpGrantRequest)
	if !ok || req.Exp == 0 {
		return
	}
	a := p.GetAttrs()
	if a == nil {
		return
	}
	a.AddExp(req.Exp)
	for {
		level := a.GetLevel()
		need := int64(1000) * int64(level)
		if need <= 0 || a.GetExp() < need {
			break
		}
		a.SetExp(a.GetExp() - need)
		a.SetLevel(level + 1)
	}
}

func (p *Player) handleNetItemList(msg *PlayerMessage) {
	p.replyItem(msg, protocol.ItemMsgId_MSG_ITEM_LIST_RESPONSE, &protocol.ClientItemListResponse{
		Result: 0,
		Items:  itemsToSlots(p.inventory.GetAllItems()),
	})
}

func (p *Player) handleNetItemUse(msg *PlayerMessage) {
	req, ok := msg.Data.(*protocol.ClientItemUseRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}
	cnt := req.Count
	if cnt <= 0 {
		cnt = 1
	}
	resp := &protocol.ClientItemUseResponse{Slot: req.Slot}
	if _, err := p.inventory.RemoveItem(req.Slot, cnt); err != nil {
		resp.Result = 1
		resp.Error = err.Error()
	} else if it, ok := p.inventory.GetItem(req.Slot); ok {
		resp.Remaining = it.GetCount()
	}
	p.replyItem(msg, protocol.ItemMsgId_MSG_ITEM_USE_RESPONSE, resp)
}

func (p *Player) handleNetItemMove(msg *PlayerMessage) {
	req, ok := msg.Data.(*protocol.ClientItemMoveRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}
	resp := &protocol.ClientItemMoveResponse{}
	if err := p.inventory.MoveItem(req.FromSlot, req.ToSlot); err != nil {
		resp.Result = 1
		resp.Error = err.Error()
	}
	p.replyItem(msg, protocol.ItemMsgId_MSG_ITEM_MOVE_RESPONSE, resp)
}

func (p *Player) handleNetWarehouseList(msg *PlayerMessage) {
	p.replyItem(msg, protocol.ItemMsgId_MSG_WAREHOUSE_LIST_RESPONSE, &protocol.ClientWarehouseListResponse{
		Result: 0,
		Items:  itemsToSlots(p.warehouse.GetAllItems()),
	})
}

func (p *Player) handleNetWarehouseStore(msg *PlayerMessage) {
	req, ok := msg.Data.(*protocol.ClientWarehouseStoreRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}
	cnt := req.Count
	if cnt <= 0 {
		cnt = 1
	}
	resp := &protocol.ClientWarehouseStoreResponse{}
	src, ok := p.inventory.GetItem(req.BagSlot)
	if !ok {
		resp.Result = 1
		resp.Error = "no item in bag slot"
		p.replyItem(msg, protocol.ItemMsgId_MSG_WAREHOUSE_STORE_RESPONSE, resp)
		return
	}
	configID, name, itype := src.ConfigID, src.Name, src.Type
	if _, err := p.inventory.RemoveItem(req.BagSlot, cnt); err != nil {
		resp.Result = 1
		resp.Error = err.Error()
	} else {
		slot, err := p.warehouse.Store(game.NewItem(int64(configID), configID, name, itype, cnt))
		if err != nil {
			// 存仓失败：回滚回背包，避免物品凭空消失。
			_, _ = p.inventory.AddItem(game.NewItem(int64(configID), configID, name, itype, cnt))
			resp.Result = 1
			resp.Error = err.Error()
		} else {
			resp.WarehouseSlot = slot
		}
	}
	p.replyItem(msg, protocol.ItemMsgId_MSG_WAREHOUSE_STORE_RESPONSE, resp)
}

func (p *Player) handleNetWarehouseRetrieve(msg *PlayerMessage) {
	req, ok := msg.Data.(*protocol.ClientWarehouseRetrieveRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}
	cnt := req.Count
	if cnt <= 0 {
		cnt = 1
	}
	resp := &protocol.ClientWarehouseRetrieveResponse{}
	src, ok := p.warehouse.GetItem(req.WarehouseSlot)
	if !ok {
		resp.Result = 1
		resp.Error = "no item in warehouse slot"
		p.replyItem(msg, protocol.ItemMsgId_MSG_WAREHOUSE_RETRIEVE_RESPONSE, resp)
		return
	}
	configID, name, itype := src.ConfigID, src.Name, src.Type
	if _, err := p.warehouse.Retrieve(req.WarehouseSlot, cnt); err != nil {
		resp.Result = 1
		resp.Error = err.Error()
	} else {
		slot, err := p.inventory.AddItem(game.NewItem(int64(configID), configID, name, itype, cnt))
		if err != nil {
			// 取回失败（背包满）：回滚回仓库。
			_, _ = p.warehouse.Store(game.NewItem(int64(configID), configID, name, itype, cnt))
			resp.Result = 1
			resp.Error = err.Error()
		} else {
			resp.BagSlot = slot
		}
	}
	p.replyItem(msg, protocol.ItemMsgId_MSG_WAREHOUSE_RETRIEVE_RESPONSE, resp)
}
