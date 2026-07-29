package player

// —— 资产存盘的脏标记（崩溃安全，2026-07-29）——
//
// **问题**：玩家的背包/技能/仓库/buff/任务全部只活在 actor 内存里，写库只发生在
// LeaveGame（登出）和 PersistAllOnline（优雅关服）两处。周期存盘 saveLoop 只写**属性**
// （等级/金币等 online 缓存），不碰这些资产。于是进程崩溃、被 kill、断电，
// 丢的不是"最近 30 秒"，而是**本次登录以来的全部资产变更**——玩家打了两小时装备全没。
//
// **修法**：按消息类型标脏 + 周期只存脏玩家。丢失窗口从"整场游戏"收敛到"一个存盘间隔"。
// 这是 MMO 的常规保证；要做到零丢失得上写前日志（每次变更先落盘再应答），
// 那是另一个量级的工程，不在本次范围。
//
// 为什么把标记集中在 handleMessage 一处：资产的写入口散落在 inventory/skillMgr/warehouse/
// buffMgr/taskMgr 各自的方法里，逐个去标既容易漏、也容易在后续重构中掉。
// 消息类型是这些写入的**唯一入口**（actor 模型下，改玩家状态必经消息），在这里拦一道最稳。

// assetMutatingMessages 会改动**需要存盘的资产**的消息类型。
//
// ⚠ 新增会改背包/技能/仓库/buff/任务的消息时，务必在此登记，否则该变更在崩溃时会丢。
// 只影响瞬时态/属性的消息（移动、攻击、AOI、查询类）不必登记——属性走 saveLoop 的 online 缓存。
var assetMutatingMessages = map[MessageType]bool{
	// 背包
	MsgNetItemUse:  true,
	MsgNetItemMove: true,
	MsgAddItem:     true,
	MsgRemoveItem:  true,
	MsgUseItem:     true,
	MsgEquipItem:   true,
	MsgUnequipItem: true,
	// 仓库
	MsgNetWarehouseStore:    true,
	MsgNetWarehouseRetrieve: true,
	// 技能
	MsgNetSkillLearn:   true,
	MsgNetSkillUpgrade: true,
	MsgLearnSkill:      true,
	MsgUpgradeSkill:    true,
	// buff
	MsgAddBuff:    true,
	MsgRemoveBuff: true,
	// 任务
	MsgAcceptQuest:   true,
	MsgCompleteQuest: true,
}

// MarkAssetsDirty 标记该玩家的资产有未落库的变更。
func (p *Player) MarkAssetsDirty() {
	p.assetsDirty.Store(true)
}

// TakeAssetsDirty 取走脏标记（测试并清除）：返回自上次取走以来是否有过变更。
//
// 先清后存是刻意的：存盘期间若又有新变更，会重新置脏、下一轮再存；
// 反过来（存完再清）会把存盘窗口内的变更连同标记一起清掉，那次变更就永久丢了。
func (p *Player) TakeAssetsDirty() bool {
	return p.assetsDirty.Swap(false)
}

// AssetsDirty 只读查询（测试/排查用，不清标记）。
func (p *Player) AssetsDirty() bool {
	return p.assetsDirty.Load()
}
