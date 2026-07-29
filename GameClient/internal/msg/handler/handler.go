package handler

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zNet"
)

// quiet 静音开关。压测时每秒有成千上万条响应/AOI 广播，逐条 fmt.Printf 到 stdout 会成为
// 客户端侧的主要开销，把测出来的服务端吞吐压得没法看（量的成了终端刷屏速度）。
// 故本包所有人类可读输出统一走 logf，压测前 SetQuiet(true) 关掉。
var quiet atomic.Bool

// SetQuiet 开/关本包的人类可读输出（压测用）。
func SetQuiet(q bool) { quiet.Store(q) }

func logf(format string, a ...interface{}) {
	if quiet.Load() {
		return
	}
	fmt.Printf(format, a...)
}

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
	signalChans map[uint32]chan signalItem

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
		signalChans: map[uint32]chan signalItem{
			uint32(protocol.SystemMsgId_MSG_SYSTEM_TOKEN_VERIFY_RESPONSE): make(chan signalItem, 1),
			uint32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE):   make(chan signalItem, 1),
			uint32(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE):              make(chan signalItem, 1),
			uint32(protocol.MapMsgId_MSG_MAP_CROSS_ENTER_RESPONSE):        make(chan signalItem, 1),
			// 移动/攻击的响应屏障：闭环压测要"发一条等一条"，没有这两个就只能盲发、量不出服务端吞吐。
			// 缓冲刻意开大：signal 是**非阻塞投递**，缓冲满就丢——压测下响应常常连着到（网络批量/GC 停顿后），
			// 缓冲为 1 时后到的那条直接被丢，等待方只能干等到 5 秒超时，几条就把整个测试窗口吃光
			// （实测每客户端固定丢约 3 条、超时占掉大半窗口，吞吐数字全废）。配合按到达时刻过滤，深缓冲无副作用。
			uint32(protocol.MapMsgId_MSG_MAP_MOVE_RESPONSE):   make(chan signalItem, 256),
			uint32(protocol.MapMsgId_MSG_MAP_ATTACK_RESPONSE): make(chan signalItem, 256),
		},
	}
}

// signalItem 一条响应信号：带**到达时刻**，供闭环压测严格关联"这条响应是不是我这次请求的"。
type signalItem struct {
	result int32
	at     time.Time
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
	case ch <- signalItem{result: result, at: time.Now()}:
	default:
	}
}

// Drain 丢弃某类响应上残留的信号。
//
// 闭环压测必须在发下一条请求前调它：信号 channel 缓冲为 1，若上一轮有**迟到或重复**的响应
// 落进缓冲，下一轮 WaitFor 会立刻拿到这条陈旧信号当成"新请求已完成"，于是请求与响应错位
// ——计数虚高，且真正那条响应最终无人认领、表现为莫名其妙的超时。
func (h *MessageHandler) Drain(protoId uint32) {
	h.mu.Lock()
	ch := h.signalChans[protoId]
	h.mu.Unlock()
	if ch == nil {
		return
	}
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// WaitFor 等待某类响应到达并返回其 Result；超时返回 ok=false。
// 单发一收：channel 缓冲 1，响应即便早于 WaitFor 到达也已缓冲，不会漏。
func (h *MessageHandler) WaitFor(protoId uint32, timeout time.Duration) (result int32, ok bool) {
	return h.WaitForAfter(protoId, time.Time{}, timeout)
}

// WaitForAfter 等待**到达时刻晚于 since** 的响应。
//
// 闭环压测必须用它而不是 WaitFor：只按"channel 里有东西"判定的话，上一轮迟到的响应会被
// 当成本轮的，于是这一轮"零耗时"就完成了——计数虚高、吞吐虚高，而真正那条响应最终无人认领、
// 表现为莫名其妙的 5 秒超时。曾实测出"单客户端 2990 次/秒 而 p50=515µs"这种自相矛盾的数字。
func (h *MessageHandler) WaitForAfter(protoId uint32, since time.Time, timeout time.Duration) (result int32, ok bool) {
	h.mu.Lock()
	ch := h.signalChans[protoId]
	h.mu.Unlock()
	if ch == nil {
		return 0, false
	}

	deadline := time.Now().Add(timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0, false
		}
		select {
		case item := <-ch:
			if item.at.Before(since) {
				continue // 陈旧响应，丢弃继续等
			}
			return item.result, true
		case <-time.After(remaining):
			return 0, false
		}
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
			logf("ItemListResponse: Result=%d, 背包物品数=%d\n", r.Result, len(r.Items))
			for _, s := range r.Items {
				logf("  背包[%d] item=%d x%d\n", s.Slot, s.ItemId, s.Count)
			}
		}
	case uint32(protocol.ItemMsgId_MSG_ITEM_USE_RESPONSE):
		var r protocol.ClientItemUseResponse
		if proto.Unmarshal(data, &r) == nil {
			logf("ItemUseResponse: Result=%d, Slot=%d, Remaining=%d, Err=%s\n", r.Result, r.Slot, r.Remaining, r.Error)
		}
	case uint32(protocol.ItemMsgId_MSG_ITEM_MOVE_RESPONSE):
		var r protocol.ClientItemMoveResponse
		if proto.Unmarshal(data, &r) == nil {
			logf("ItemMoveResponse: Result=%d, Err=%s\n", r.Result, r.Error)
		}
	case uint32(protocol.ItemMsgId_MSG_WAREHOUSE_LIST_RESPONSE):
		var r protocol.ClientWarehouseListResponse
		if proto.Unmarshal(data, &r) == nil {
			logf("WarehouseListResponse: Result=%d, 仓库物品数=%d\n", r.Result, len(r.Items))
			for _, s := range r.Items {
				logf("  仓库[%d] item=%d x%d\n", s.Slot, s.ItemId, s.Count)
			}
		}
	case uint32(protocol.ItemMsgId_MSG_WAREHOUSE_STORE_RESPONSE):
		var r protocol.ClientWarehouseStoreResponse
		if proto.Unmarshal(data, &r) == nil {
			logf("WarehouseStoreResponse: Result=%d, 仓库格=%d, Err=%s\n", r.Result, r.WarehouseSlot, r.Error)
		}
	case uint32(protocol.ItemMsgId_MSG_WAREHOUSE_RETRIEVE_RESPONSE):
		var r protocol.ClientWarehouseRetrieveResponse
		if proto.Unmarshal(data, &r) == nil {
			logf("WarehouseRetrieveResponse: Result=%d, 背包格=%d, Err=%s\n", r.Result, r.BagSlot, r.Error)
		}
	case uint32(protocol.SkillMsgId_MSG_SKILL_LIST_RESPONSE):
		var r protocol.ClientSkillListResponse
		if proto.Unmarshal(data, &r) == nil {
			logf("SkillListResponse: Result=%d, 技能数=%d\n", r.Result, len(r.Skills))
			for _, s := range r.Skills {
				logf("  技能 id=%d Lv%d/%d\n", s.SkillId, s.Level, s.MaxLevel)
			}
		}
	case uint32(protocol.SkillMsgId_MSG_SKILL_LEARN_RESPONSE):
		var r protocol.ClientSkillLearnResponse
		if proto.Unmarshal(data, &r) == nil {
			logf("SkillLearnResponse: Result=%d, SkillID=%d, Err=%s\n", r.Result, r.SkillId, r.Error)
		}
	case uint32(protocol.SkillMsgId_MSG_SKILL_UPGRADE_RESPONSE):
		var r protocol.ClientSkillUpgradeResponse
		if proto.Unmarshal(data, &r) == nil {
			logf("SkillUpgradeResponse: Result=%d, SkillID=%d, Level=%d, Err=%s\n", r.Result, r.SkillId, r.Level, r.Error)
		}
	case uint32(protocol.SkillMsgId_MSG_SKILL_CAST_RESPONSE):
		var r protocol.ClientSkillCastResponse
		if proto.Unmarshal(data, &r) == nil {
			logf("SkillCastResponse: Result=%d, SkillID=%d, Err=%s\n", r.Result, r.SkillId, r.Error)
		}
	case uint32(protocol.MapMsgId_MSG_MAP_ENTER_VIEW):
		var n protocol.EntityEnterViewNotify
		if proto.Unmarshal(data, &n) == nil {
			h.viewAdd(n.EntityId)
			var x, y, z float32
			if n.Pos != nil {
				x, y, z = n.Pos.X, n.Pos.Y, n.Pos.Z
			}
			logf("[AOI] 实体进入视野: entity=%d pos=(%.0f,%.0f,%.0f) 当前视野=%v\n", n.EntityId, x, y, z, h.viewList())
		}
	case uint32(protocol.MapMsgId_MSG_MAP_LEAVE_VIEW):
		var n protocol.EntityLeaveViewNotify
		if proto.Unmarshal(data, &n) == nil {
			h.viewDel(n.EntityId)
			logf("[AOI] 实体离开视野: entity=%d 当前视野=%v\n", n.EntityId, h.viewList())
		}
	case uint32(protocol.MapMsgId_MSG_MAP_ENTITY_MOVE):
		var n protocol.EntityMoveNotify
		if proto.Unmarshal(data, &n) == nil {
			var x, y, z float32
			if n.NewPos != nil {
				x, y, z = n.NewPos.X, n.NewPos.Y, n.NewPos.Z
			}
			logf("[AOI] 实体移动: entity=%d ->(%.0f,%.0f,%.0f)\n", n.EntityId, x, y, z)
		}
	case uint32(protocol.MapMsgId_MSG_MAP_ENTITY_ATTR):
		var n protocol.EntityAttrNotify
		if proto.Unmarshal(data, &n) == nil {
			logf("[AOI] 实体血量变更: entity=%d HP=%d/%d\n", n.EntityId, n.CurHp, n.MaxHp)
		}
	case uint32(protocol.MapMsgId_MSG_MAP_ENTITY_DEATH):
		var n protocol.EntityDeathNotify
		if proto.Unmarshal(data, &n) == nil {
			logf("[AOI] 实体死亡: entity=%d killer=%d\n", n.EntityId, n.KillerId)
		}
	case uint32(protocol.MapMsgId_MSG_MAP_ENTITY_BUFF):
		var n protocol.EntityBuffNotify
		if proto.Unmarshal(data, &n) == nil {
			op := "移除"
			if n.Added {
				op = "获得"
			}
			logf("[AOI] 实体buff%s: entity=%d buff=%d 剩余=%dms\n", op, n.EntityId, n.BuffId, n.RemainingMs)
		}
	case uint32(protocol.ItemMsgId_MSG_ITEM_PICKUP_NOTIFY):
		var n protocol.ClientItemPickupNotify
		if proto.Unmarshal(data, &n) == nil {
			if n.Result == 0 {
				logf("[拾取] 成功: item=%d x%d 已入背包\n", n.ItemId, n.Count)
			} else {
				logf("[拾取] 失败: %s\n", n.Error)
			}
		}
	case uint32(protocol.ChatMsgId_MSG_CHAT_NOTIFY):
		var n protocol.ClientChatNotify
		if proto.Unmarshal(data, &n) == nil {
			ch := "世界"
			if n.Channel == int32(protocol.ChatChannel_CHAT_AREA) {
				ch = "附近"
			}
			logf("[聊天][%s] %s(%d): %s\n", ch, n.FromName, n.FromPlayerId, n.Text)
		}
	case uint32(protocol.TeamMsgId_MSG_TEAM_UPDATE_NOTIFY):
		var n protocol.ClientTeamUpdateNotify
		if proto.Unmarshal(data, &n) == nil {
			if n.Disbanded {
				logf("[队伍] 已离队/解散: team=%d\n", n.TeamId)
			} else {
				names := ""
				for _, m := range n.Members {
					tag := ""
					if m.IsLeader {
						tag = "(队长)"
					}
					names += fmt.Sprintf("%s%s ", m.Name, tag)
				}
				logf("[队伍] 更新: team=%d 成员数=%d [ %s]\n", n.TeamId, len(n.Members), names)
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
			logf("[交易][%s] A(%d)出%d确认%v <-> B(%d)出%d确认%v %s\n",
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
			logf("[邮件] 列表: %d 封\n", len(r.Mails))
			h.mu.Lock()
			claim := h.mailAutoClaim
			h.mu.Unlock()
			for _, m := range r.Mails {
				logf("  邮件#%d 来自%s: %s (金币%d) 已领=%v\n", m.MailId, m.Sender, m.Title, m.Gold, m.IsClaimed)
				if claim != nil && !m.IsClaimed {
					claim(m.MailId)
				}
			}
		}
	case uint32(protocol.MailMsgId_MSG_MAIL_CLAIM_RESPONSE):
		var r protocol.ClientMailClaimResponse
		if proto.Unmarshal(data, &r) == nil {
			if r.Result == 0 {
				logf("[邮件] 领取成功: 邮件#%d 到账金币%d\n", r.MailId, r.Gold)
			} else {
				logf("[邮件] 领取失败: 邮件#%d %s\n", r.MailId, r.Error)
			}
		}
	default:
		logf("Received message: ProtoId=%d, DataSize=%d\n", protoId, len(data))
	}
}

func (h *MessageHandler) handleTokenVerifyResponse(data []byte) {
	var resp protocol.ServerMessage
	if err := proto.Unmarshal(data, &resp); err != nil {
		logf("Failed to unmarshal TokenVerifyResponse: %v\n", err)
		return
	}
	logf("TokenVerifyResponse: Result=%d, ErrorMsg=%s\n", resp.Result, resp.ErrorMsg)
	h.signal(uint32(protocol.SystemMsgId_MSG_SYSTEM_TOKEN_VERIFY_RESPONSE), resp.Result)
}

func (h *MessageHandler) handlePlayerLoginResponse(data []byte) {
	var resp protocol.PlayerLoginResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		logf("Failed to unmarshal PlayerLoginResponse: %v\n", err)
		return
	}
	logf("PlayerLoginResponse: Result=%d, ErrorMsg=%s\n", resp.Result, resp.ErrorMsg)
	h.signal(uint32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE), resp.Result)
	if resp.Result == 0 && resp.PlayerInfo != nil {
		logf("Player: ID=%d, Name=%s, Level=%d, Gold=%d\n",
			resp.PlayerInfo.PlayerId, resp.PlayerInfo.Name, resp.PlayerInfo.Level, resp.PlayerInfo.Gold)
	}
}

func (h *MessageHandler) handlePlayerCreateResponse(data []byte) {
	var resp protocol.PlayerCreateResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		logf("Failed to unmarshal PlayerCreateResponse: %v\n", err)
		return
	}
	logf("PlayerCreateResponse: Result=%d, ErrorMsg=%s\n", resp.Result, resp.ErrorMsg)
	if resp.Result == 0 && resp.PlayerInfo != nil {
		logf("Player: ID=%d, Name=%s, Level=%d\n",
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
		logf("Failed to unmarshal ClientMapEnterResponse: %v\n", err)
		return
	}
	logf("ClientMapEnterResponse: Result=%d, ErrorMsg=%s, MapID=%d\n", resp.Result, resp.ErrorMsg, resp.MapId)
	h.signal(uint32(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE), resp.Result)
}

// handleCrossEnterResponse 跨服进图应答：打印承载 MapServer 与实例 mapID——
// 跨 realm E2E 就靠这两个字段断言"两个 realm 的玩家确实落到同一台服务器的同一张实例图上"。
func (h *MessageHandler) handleCrossEnterResponse(data []byte) {
	var resp protocol.ClientCrossEnterResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		logf("Failed to unmarshal ClientCrossEnterResponse: %v\n", err)
		return
	}
	logf("[跨服] ClientCrossEnterResponse: Result=%d, ErrorMsg=%s, ActivityID=%d, MapServerID=%d, MapID=%d\n",
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
		logf("Failed to unmarshal ClientMapMoveResponse: %v\n", err)
		return
	}
	logf("ClientMapMoveResponse: Result=%d\n", resp.Result)
	h.signal(uint32(protocol.MapMsgId_MSG_MAP_MOVE_RESPONSE), resp.Result)
}

func (h *MessageHandler) handleMapAttackResponse(data []byte) {
	var resp protocol.ClientMapAttackResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		logf("Failed to unmarshal ClientMapAttackResponse: %v\n", err)
		return
	}
	logf("ClientMapAttackResponse: Result=%d, TargetID=%d, Damage=%d, TargetHp=%d\n",
		resp.Result, resp.TargetId, resp.Damage, resp.TargetHp)
	h.signal(uint32(protocol.MapMsgId_MSG_MAP_ATTACK_RESPONSE), resp.Result)
}

func (h *MessageHandler) GetDispatcher() zNet.HandlerFun {
	return func(session zNet.Session, netPacket *zNet.NetPacket) error {
		h.HandleMessage(uint32(netPacket.ProtoId), netPacket.Data)
		return nil
	}
}
