package handler

import (
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zNet"
)

type MessageHandler struct {
	mu              sync.Mutex
	createdPlayerID int64
	playerIDCh      chan int64
}

func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		playerIDCh: make(chan int64, 1),
	}
}

func (h *MessageHandler) GetCreatedPlayerID() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.createdPlayerID
}

func (h *MessageHandler) HandleMessage(protoId uint32, data []byte) {
	switch protoId {
	case uint32(protocol.SystemMsgId_MSG_SYSTEM_TOKEN_VERIFY_RESPONSE):
		h.handleTokenVerifyResponse(data)
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
	case uint32(protocol.ItemMsgId_MSG_ITEM_LIST_RESPONSE):
		var r protocol.ClientItemListResponse
		if proto.Unmarshal(data, &r) == nil {
			fmt.Printf("ItemListResponse: Result=%d, 背包物品数=%d\n", r.Result, len(r.Items))
			for _, s := range r.Items {
				fmt.Printf("  背包[%d] item=%d x%d\n", s.Slot, s.ItemId, s.Count)
			}
		}
	case uint32(protocol.ItemMsgId_MSG_ITEM_USE_RESPONSE):
		var r protocol.ClientItemUseResponse
		if proto.Unmarshal(data, &r) == nil {
			fmt.Printf("ItemUseResponse: Result=%d, Slot=%d, Remaining=%d, Err=%s\n", r.Result, r.Slot, r.Remaining, r.Error)
		}
	case uint32(protocol.ItemMsgId_MSG_ITEM_MOVE_RESPONSE):
		var r protocol.ClientItemMoveResponse
		if proto.Unmarshal(data, &r) == nil {
			fmt.Printf("ItemMoveResponse: Result=%d, Err=%s\n", r.Result, r.Error)
		}
	case uint32(protocol.ItemMsgId_MSG_WAREHOUSE_LIST_RESPONSE):
		var r protocol.ClientWarehouseListResponse
		if proto.Unmarshal(data, &r) == nil {
			fmt.Printf("WarehouseListResponse: Result=%d, 仓库物品数=%d\n", r.Result, len(r.Items))
			for _, s := range r.Items {
				fmt.Printf("  仓库[%d] item=%d x%d\n", s.Slot, s.ItemId, s.Count)
			}
		}
	case uint32(protocol.ItemMsgId_MSG_WAREHOUSE_STORE_RESPONSE):
		var r protocol.ClientWarehouseStoreResponse
		if proto.Unmarshal(data, &r) == nil {
			fmt.Printf("WarehouseStoreResponse: Result=%d, 仓库格=%d, Err=%s\n", r.Result, r.WarehouseSlot, r.Error)
		}
	case uint32(protocol.ItemMsgId_MSG_WAREHOUSE_RETRIEVE_RESPONSE):
		var r protocol.ClientWarehouseRetrieveResponse
		if proto.Unmarshal(data, &r) == nil {
			fmt.Printf("WarehouseRetrieveResponse: Result=%d, 背包格=%d, Err=%s\n", r.Result, r.BagSlot, r.Error)
		}
	case uint32(protocol.SkillMsgId_MSG_SKILL_LIST_RESPONSE):
		var r protocol.ClientSkillListResponse
		if proto.Unmarshal(data, &r) == nil {
			fmt.Printf("SkillListResponse: Result=%d, 技能数=%d\n", r.Result, len(r.Skills))
			for _, s := range r.Skills {
				fmt.Printf("  技能 id=%d Lv%d/%d\n", s.SkillId, s.Level, s.MaxLevel)
			}
		}
	case uint32(protocol.SkillMsgId_MSG_SKILL_LEARN_RESPONSE):
		var r protocol.ClientSkillLearnResponse
		if proto.Unmarshal(data, &r) == nil {
			fmt.Printf("SkillLearnResponse: Result=%d, SkillID=%d, Err=%s\n", r.Result, r.SkillId, r.Error)
		}
	case uint32(protocol.SkillMsgId_MSG_SKILL_UPGRADE_RESPONSE):
		var r protocol.ClientSkillUpgradeResponse
		if proto.Unmarshal(data, &r) == nil {
			fmt.Printf("SkillUpgradeResponse: Result=%d, SkillID=%d, Level=%d, Err=%s\n", r.Result, r.SkillId, r.Level, r.Error)
		}
	case uint32(protocol.SkillMsgId_MSG_SKILL_CAST_RESPONSE):
		var r protocol.ClientSkillCastResponse
		if proto.Unmarshal(data, &r) == nil {
			fmt.Printf("SkillCastResponse: Result=%d, SkillID=%d, Err=%s\n", r.Result, r.SkillId, r.Error)
		}
	default:
		fmt.Printf("Received message: ProtoId=%d, DataSize=%d\n", protoId, len(data))
	}
}

func (h *MessageHandler) handleTokenVerifyResponse(data []byte) {
	var resp protocol.ServerMessage
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal TokenVerifyResponse: %v\n", err)
		return
	}
	fmt.Printf("TokenVerifyResponse: Result=%d, ErrorMsg=%s\n", resp.Result, resp.ErrorMsg)
}

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

func (h *MessageHandler) handlePlayerCreateResponse(data []byte) {
	var resp protocol.PlayerCreateResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal PlayerCreateResponse: %v\n", err)
		return
	}
	fmt.Printf("PlayerCreateResponse: Result=%d, ErrorMsg=%s\n", resp.Result, resp.ErrorMsg)
	if resp.Result == 0 && resp.PlayerInfo != nil {
		fmt.Printf("Player: ID=%d, Name=%s, Level=%d\n",
			resp.PlayerInfo.PlayerId, resp.PlayerInfo.Name, resp.PlayerInfo.Level)
		h.mu.Lock()
		h.createdPlayerID = resp.PlayerInfo.PlayerId
		h.mu.Unlock()
		select {
		case h.playerIDCh <- resp.PlayerInfo.PlayerId:
		default:
		}
	}
}

func (h *MessageHandler) WaitForPlayerID() int64 {
	h.mu.Lock()
	if h.createdPlayerID != 0 {
		pid := h.createdPlayerID
		h.mu.Unlock()
		return pid
	}
	h.mu.Unlock()

	select {
	case id := <-h.playerIDCh:
		return id
	case <-time.After(5 * time.Second):
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.createdPlayerID
	}
}

func (h *MessageHandler) handleMapEnterResponse(data []byte) {
	var resp protocol.ClientMapEnterResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal ClientMapEnterResponse: %v\n", err)
		return
	}
	fmt.Printf("ClientMapEnterResponse: Result=%d, ErrorMsg=%s, MapID=%d\n", resp.Result, resp.ErrorMsg, resp.MapId)
}

func (h *MessageHandler) handleMapMoveResponse(data []byte) {
	var resp protocol.ClientMapMoveResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal ClientMapMoveResponse: %v\n", err)
		return
	}
	fmt.Printf("ClientMapMoveResponse: Result=%d\n", resp.Result)
}

func (h *MessageHandler) handleMapAttackResponse(data []byte) {
	var resp protocol.ClientMapAttackResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal ClientMapAttackResponse: %v\n", err)
		return
	}
	fmt.Printf("ClientMapAttackResponse: Result=%d, TargetID=%d, Damage=%d, TargetHp=%d\n",
		resp.Result, resp.TargetId, resp.Damage, resp.TargetHp)
}

func (h *MessageHandler) GetDispatcher() zNet.HandlerFun {
	return func(session zNet.Session, netPacket *zNet.NetPacket) error {
		h.HandleMessage(uint32(netPacket.ProtoId), netPacket.Data)
		return nil
	}
}
