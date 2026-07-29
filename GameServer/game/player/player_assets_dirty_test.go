package player

import "testing"

// 资产存盘崩溃安全的回归守护。
//
// 背景：背包/技能/仓库/buff/任务只活在 actor 内存里，原本只有登出与优雅关服才写库
// —— 进程崩溃/被 kill 丢的是**本次登录以来的全部资产变更**（打两小时装备全没），
// 而不是通常以为的"最近一次周期存盘之后"。现按消息类型标脏 + 周期只存脏玩家。

// TestAssetsDirty_MutatingMessageMarksDirty 会改资产的消息必须把玩家标脏，
// 否则周期存盘会跳过它，崩溃即丢。
func TestAssetsDirty_MutatingMessageMarksDirty(t *testing.T) {
	for msgType := range assetMutatingMessages {
		p := NewPlayer(1, 1, "T")
		if p.AssetsDirty() {
			t.Fatalf("新建玩家不应是脏的")
		}
		if !assetMutatingMessages[msgType] {
			continue
		}
		p.MarkAssetsDirty()
		if !p.AssetsDirty() {
			t.Fatalf("消息类型 %d 标脏后应为脏", msgType)
		}
	}
}

// TestAssetsDirty_NonMutatingMessagesNotListed 纯查询/瞬时态消息不应登记为改资产，
// 否则每次移动都标脏，周期存盘退化为全量重写。
func TestAssetsDirty_NonMutatingMessagesNotListed(t *testing.T) {
	shouldNotBeListed := []MessageType{
		MsgNetMapMove, MsgNetMapAttack, MsgNetMapEnter, MsgNetMapLeave,
		MsgNetItemList, MsgNetSkillList, MsgNetWarehouseList,
		MsgAOIEnterView, MsgAOILeaveView,
	}
	for _, m := range shouldNotBeListed {
		if assetMutatingMessages[m] {
			t.Fatalf("消息类型 %d 不改需存盘的资产，不应登记（会让周期存盘每帧全量重写）", m)
		}
	}
}

// TestAssetsDirty_TakeClearsAndReportsOnce 取走脏标记后变干净；再取返回 false。
func TestAssetsDirty_TakeClearsAndReportsOnce(t *testing.T) {
	p := NewPlayer(2, 1, "T")

	if p.TakeAssetsDirty() {
		t.Fatalf("干净玩家不应报脏")
	}

	p.MarkAssetsDirty()
	if !p.TakeAssetsDirty() {
		t.Fatalf("标脏后首次取应为 true")
	}
	if p.TakeAssetsDirty() {
		t.Fatalf("取走后应变干净")
	}
}

// TestAssetsDirty_ChangeDuringSaveIsNotLost ★关键不变量★：
// 存盘期间发生的新变更不能被吞掉。实现上先取走标记再存盘，
// 于是存盘窗口内的新变更会重新置脏、下一轮再存；反过来（存完再清）会把它连标记一起清掉、永久丢失。
func TestAssetsDirty_ChangeDuringSaveIsNotLost(t *testing.T) {
	p := NewPlayer(3, 1, "T")
	p.MarkAssetsDirty()

	// 模拟存盘：先取标记
	if !p.TakeAssetsDirty() {
		t.Fatalf("应取到脏标记")
	}
	// 存盘进行中，玩家又改了资产
	p.MarkAssetsDirty()

	// 下一轮必须还能取到——这次变更不能丢
	if !p.TakeAssetsDirty() {
		t.Fatalf("存盘窗口内的新变更被吞掉了（会造成真实丢档）")
	}
}

// TestAssetsDirty_CoversAllAssetKinds 五类需存盘的资产都要有代表消息在登记表里，
// 防止新增某类资产时漏登记。
func TestAssetsDirty_CoversAllAssetKinds(t *testing.T) {
	kinds := map[string][]MessageType{
		"背包":  {MsgAddItem, MsgRemoveItem, MsgNetItemUse, MsgEquipItem},
		"仓库":  {MsgNetWarehouseStore, MsgNetWarehouseRetrieve},
		"技能":  {MsgLearnSkill, MsgUpgradeSkill, MsgNetSkillLearn},
		"buff": {MsgAddBuff, MsgRemoveBuff},
		"任务":  {MsgAcceptQuest, MsgCompleteQuest},
	}
	for kind, msgs := range kinds {
		for _, m := range msgs {
			if !assetMutatingMessages[m] {
				t.Fatalf("%s 的消息类型 %d 未登记为改资产 → 崩溃时该类变更会丢", kind, m)
			}
		}
	}
}
