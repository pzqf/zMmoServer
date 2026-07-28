package handler

import (
	"fmt"
	"sort"
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

	// 交易自动应价器（事件驱动）：收到交易 OPEN 通知时，本方若报价未到位则设价、报价到位未确认则确认。
	tradeMyID    int64
	tradeGold    int64
	tradeSetGold func(gold int64)
	tradeConfirm func()

	// 邮件自动领取：收到邮件列表时，对每封未领邮件调用 claim。
	mailAutoClaim func(mailID int64)

	// 关键屏障响应信号：按 protoId 投递响应 Result（缓冲 1，单发一收）。
	// 复用 playerIDCh 的 channel 信号范式，让 main 侧的"发送→等响应且判 Result"
	// 取代盲等 sleep（token 验证 / 进游戏 / 进图三步）。
	signalChans map[uint32]chan int32

	// AOI 视野集：记当前视野内的实体ID（进入视野+1、离开-1），供观测"我视野里有谁"——
	// 分线隔离/无缝交接等场景可直接看客户端视野是否含某玩家。
	viewSet map[int64]bool

	// 跨服进图落点（realm §5.1）：服务端分配的承载 MapServer 与实例 mapID。
	crossMapServerID int32
	crossMapID       int32
}

func (h *MessageHandler) viewAdd(entityID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.viewSet[entityID] = true
}

func (h *MessageHandler) viewDel(entityID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.viewSet, entityID)
}

// viewList 返回当前视野内实体ID（升序，稳定输出）。
func (h *MessageHandler) viewList() []int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]int64, 0, len(h.viewSet))
	for id := range h.viewSet {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// EnableMailAutoClaim 开启邮件自动领取：收到邮件列表后逐封领取未领邮件。
func (h *MessageHandler) EnableMailAutoClaim(claim func(mailID int64)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.mailAutoClaim = claim
}

func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		playerIDCh: make(chan int64, 1),
		viewSet:    make(map[int64]bool),
		signalChans: map[uint32]chan int32{
			uint32(protocol.SystemMsgId_MSG_SYSTEM_TOKEN_VERIFY_RESPONSE): make(chan int32, 1),
			uint32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE):   make(chan int32, 1),
			uint32(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE):              make(chan int32, 1),
			uint32(protocol.MapMsgId_MSG_MAP_CROSS_ENTER_RESPONSE):        make(chan int32, 1),
		},
	}
}

// signal 向对应 protoId 的屏障投递响应 Result（非阻塞，缓冲已满则丢弃——单发一收场景无碍）。
func (h *MessageHandler) signal(protoId uint32, result int32) {
	h.mu.Lock()
	ch := h.signalChans[protoId]
	h.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- result:
	default:
	}
}

// WaitFor 等待某类响应到达并返回其 Result；超时返回 ok=false。
// 单发一收：channel 缓冲 1，响应即便早于 WaitFor 到达也已缓冲，不会漏。
func (h *MessageHandler) WaitFor(protoId uint32, timeout time.Duration) (result int32, ok bool) {
	h.mu.Lock()
	ch := h.signalChans[protoId]
	h.mu.Unlock()
	if ch == nil {
		return 0, false
	}
	select {
	case r := <-ch:
		return r, true
	case <-time.After(timeout):
		return 0, false
	}
}

// EnableTradeAuto 开启交易自动应价：本方 playerID、愿出金币、以及设价/确认两个发送闭包。
func (h *MessageHandler) EnableTradeAuto(myID, gold int64, setGold func(int64), confirm func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tradeMyID = myID
	h.tradeGold = gold
	h.tradeSetGold = setGold
	h.tradeConfirm = confirm
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
	case uint32(protocol.MapMsgId_MSG_MAP_CROSS_ENTER_RESPONSE):
		h.handleCrossEnterResponse(data)
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
	case uint32(protocol.MapMsgId_MSG_MAP_ENTER_VIEW):
		var n protocol.EntityEnterViewNotify
		if proto.Unmarshal(data, &n) == nil {
			h.viewAdd(n.EntityId)
			var x, y, z float32
			if n.Pos != nil {
				x, y, z = n.Pos.X, n.Pos.Y, n.Pos.Z
			}
			fmt.Printf("[AOI] 实体进入视野: entity=%d pos=(%.0f,%.0f,%.0f) 当前视野=%v\n", n.EntityId, x, y, z, h.viewList())
		}
	case uint32(protocol.MapMsgId_MSG_MAP_LEAVE_VIEW):
		var n protocol.EntityLeaveViewNotify
		if proto.Unmarshal(data, &n) == nil {
			h.viewDel(n.EntityId)
			fmt.Printf("[AOI] 实体离开视野: entity=%d 当前视野=%v\n", n.EntityId, h.viewList())
		}
	case uint32(protocol.MapMsgId_MSG_MAP_ENTITY_MOVE):
		var n protocol.EntityMoveNotify
		if proto.Unmarshal(data, &n) == nil {
			var x, y, z float32
			if n.NewPos != nil {
				x, y, z = n.NewPos.X, n.NewPos.Y, n.NewPos.Z
			}
			fmt.Printf("[AOI] 实体移动: entity=%d ->(%.0f,%.0f,%.0f)\n", n.EntityId, x, y, z)
		}
	case uint32(protocol.MapMsgId_MSG_MAP_ENTITY_ATTR):
		var n protocol.EntityAttrNotify
		if proto.Unmarshal(data, &n) == nil {
			fmt.Printf("[AOI] 实体血量变更: entity=%d HP=%d/%d\n", n.EntityId, n.CurHp, n.MaxHp)
		}
	case uint32(protocol.MapMsgId_MSG_MAP_ENTITY_DEATH):
		var n protocol.EntityDeathNotify
		if proto.Unmarshal(data, &n) == nil {
			fmt.Printf("[AOI] 实体死亡: entity=%d killer=%d\n", n.EntityId, n.KillerId)
		}
	case uint32(protocol.MapMsgId_MSG_MAP_ENTITY_BUFF):
		var n protocol.EntityBuffNotify
		if proto.Unmarshal(data, &n) == nil {
			op := "移除"
			if n.Added {
				op = "获得"
			}
			fmt.Printf("[AOI] 实体buff%s: entity=%d buff=%d 剩余=%dms\n", op, n.EntityId, n.BuffId, n.RemainingMs)
		}
	case uint32(protocol.ItemMsgId_MSG_ITEM_PICKUP_NOTIFY):
		var n protocol.ClientItemPickupNotify
		if proto.Unmarshal(data, &n) == nil {
			if n.Result == 0 {
				fmt.Printf("[拾取] 成功: item=%d x%d 已入背包\n", n.ItemId, n.Count)
			} else {
				fmt.Printf("[拾取] 失败: %s\n", n.Error)
			}
		}
	case uint32(protocol.ChatMsgId_MSG_CHAT_NOTIFY):
		var n protocol.ClientChatNotify
		if proto.Unmarshal(data, &n) == nil {
			ch := "世界"
			if n.Channel == int32(protocol.ChatChannel_CHAT_AREA) {
				ch = "附近"
			}
			fmt.Printf("[聊天][%s] %s(%d): %s\n", ch, n.FromName, n.FromPlayerId, n.Text)
		}
	case uint32(protocol.TeamMsgId_MSG_TEAM_UPDATE_NOTIFY):
		var n protocol.ClientTeamUpdateNotify
		if proto.Unmarshal(data, &n) == nil {
			if n.Disbanded {
				fmt.Printf("[队伍] 已离队/解散: team=%d\n", n.TeamId)
			} else {
				names := ""
				for _, m := range n.Members {
					tag := ""
					if m.IsLeader {
						tag = "(队长)"
					}
					names += fmt.Sprintf("%s%s ", m.Name, tag)
				}
				fmt.Printf("[队伍] 更新: team=%d 成员数=%d [ %s]\n", n.TeamId, len(n.Members), names)
			}
		}
	case uint32(protocol.TradeMsgId_MSG_TRADE_UPDATE_NOTIFY):
		var n protocol.ClientTradeUpdateNotify
		if proto.Unmarshal(data, &n) == nil {
			st := "进行中"
			if n.State == int32(protocol.TradeState_TRADE_DONE) {
				st = "成交"
			} else if n.State == int32(protocol.TradeState_TRADE_CANCELLED) {
				st = "取消/失败"
			}
			fmt.Printf("[交易][%s] A(%d)出%d确认%v <-> B(%d)出%d确认%v %s\n",
				st, n.APlayerId, n.AGold, n.AConfirmed, n.BPlayerId, n.BGold, n.BConfirmed, n.Message)
			// 事件驱动自动应价（仅 OPEN 态）：本方报价未到位则设价，已到位未确认则确认，直至双方确认成交。
			h.mu.Lock()
			myID, gold, setGold, confirm := h.tradeMyID, h.tradeGold, h.tradeSetGold, h.tradeConfirm
			h.mu.Unlock()
			if n.State == int32(protocol.TradeState_TRADE_OPEN) && setGold != nil && gold > 0 {
				var myGold int64
				var myConfirmed, isParty bool
				if n.APlayerId == myID {
					myGold, myConfirmed, isParty = n.AGold, n.AConfirmed, true
				} else if n.BPlayerId == myID {
					myGold, myConfirmed, isParty = n.BGold, n.BConfirmed, true
				}
				if isParty {
					if myGold != gold {
						setGold(gold)
					} else if !myConfirmed {
						confirm()
					}
				}
			}
		}
	case uint32(protocol.MailMsgId_MSG_MAIL_LIST_RESPONSE):
		var r protocol.ClientMailListResponse
		if proto.Unmarshal(data, &r) == nil {
			fmt.Printf("[邮件] 列表: %d 封\n", len(r.Mails))
			h.mu.Lock()
			claim := h.mailAutoClaim
			h.mu.Unlock()
			for _, m := range r.Mails {
				fmt.Printf("  邮件#%d 来自%s: %s (金币%d) 已领=%v\n", m.MailId, m.Sender, m.Title, m.Gold, m.IsClaimed)
				if claim != nil && !m.IsClaimed {
					claim(m.MailId)
				}
			}
		}
	case uint32(protocol.MailMsgId_MSG_MAIL_CLAIM_RESPONSE):
		var r protocol.ClientMailClaimResponse
		if proto.Unmarshal(data, &r) == nil {
			if r.Result == 0 {
				fmt.Printf("[邮件] 领取成功: 邮件#%d 到账金币%d\n", r.MailId, r.Gold)
			} else {
				fmt.Printf("[邮件] 领取失败: 邮件#%d %s\n", r.MailId, r.Error)
			}
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
	h.signal(uint32(protocol.SystemMsgId_MSG_SYSTEM_TOKEN_VERIFY_RESPONSE), resp.Result)
}

func (h *MessageHandler) handlePlayerLoginResponse(data []byte) {
	var resp protocol.PlayerLoginResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal PlayerLoginResponse: %v\n", err)
		return
	}
	fmt.Printf("PlayerLoginResponse: Result=%d, ErrorMsg=%s\n", resp.Result, resp.ErrorMsg)
	h.signal(uint32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE), resp.Result)
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
	h.signal(uint32(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE), resp.Result)
}

// handleCrossEnterResponse 跨服进图应答：打印承载 MapServer 与实例 mapID——
// 跨 realm E2E 就靠这两个字段断言"两个 realm 的玩家确实落到同一台服务器的同一张实例图上"。
func (h *MessageHandler) handleCrossEnterResponse(data []byte) {
	var resp protocol.ClientCrossEnterResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		fmt.Printf("Failed to unmarshal ClientCrossEnterResponse: %v\n", err)
		return
	}
	fmt.Printf("[跨服] ClientCrossEnterResponse: Result=%d, ErrorMsg=%s, ActivityID=%d, MapServerID=%d, MapID=%d\n",
		resp.Result, resp.ErrorMsg, resp.ActivityId, resp.MapServerId, resp.MapId)
	h.mu.Lock()
	h.crossMapID = resp.MapId
	h.crossMapServerID = resp.MapServerId
	h.mu.Unlock()
	h.signal(uint32(protocol.MapMsgId_MSG_MAP_CROSS_ENTER_RESPONSE), resp.Result)
}

// CrossEnterInfo 上次跨服进图的落点（承载 MapServer / 实例 mapID）。
func (h *MessageHandler) CrossEnterInfo() (mapServerID int32, mapID int32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.crossMapServerID, h.crossMapID
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
