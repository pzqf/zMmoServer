package trade

import (
	"sync"
	"testing"

	"github.com/pzqf/zCommon/common/id"
)

// mockAccount 简单金币账户（非并发场景够用）。
type mockAccount struct{ gold int64 }

func (m *mockAccount) ReduceGold(amount int64) bool {
	if m.gold < amount {
		return false
	}
	m.gold -= amount
	return true
}
func (m *mockAccount) AddGold(amount int64) { m.gold += amount }

func TestExecuteSwap_Success(t *testing.T) {
	a := &mockAccount{gold: 1000}
	b := &mockAccount{gold: 1000}
	ok, msg := ExecuteSwap(a, b, 100, 30)
	if !ok {
		t.Fatalf("swap 应成功, got %q", msg)
	}
	// A: -100 +30 = 930；B: -30 +100 = 1070
	if a.gold != 930 || b.gold != 1070 {
		t.Fatalf("交换后金币错误: A=%d(期望930) B=%d(期望1070)", a.gold, b.gold)
	}
}

func TestExecuteSwap_RollbackOnInsufficient(t *testing.T) {
	a := &mockAccount{gold: 1000}
	b := &mockAccount{gold: 10} // B 不足付 30
	ok, _ := ExecuteSwap(a, b, 100, 30)
	if ok {
		t.Fatalf("B 金币不足, swap 应失败")
	}
	// A 应被回滚到原值,双方均不变
	if a.gold != 1000 || b.gold != 10 {
		t.Fatalf("回滚后金币应不变: A=%d(期望1000) B=%d(期望10)", a.gold, b.gold)
	}
}

func TestTradeManager_SessionFlow(t *testing.T) {
	tm := NewTradeManager()
	a, b := id.PlayerIdType(1), id.PlayerIdType(2)

	if _, err := tm.Start(a, a); err == nil {
		t.Fatalf("与自己交易应失败")
	}
	s, err := tm.Start(a, b)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if s.A != a || s.B != b {
		t.Fatalf("会话双方错误")
	}
	if _, err := tm.Start(a, id.PlayerIdType(3)); err == nil {
		t.Fatalf("已在交易中应拒绝再次发起")
	}

	if _, err := tm.SetGold(a, 100); err != nil {
		t.Fatalf("SetGold a: %v", err)
	}
	if _, err := tm.SetGold(b, 30); err != nil {
		t.Fatalf("SetGold b: %v", err)
	}

	_, both, _ := tm.Confirm(a)
	if both {
		t.Fatalf("仅 a 确认不应双确认")
	}
	s2, both, _ := tm.Confirm(b)
	if !both {
		t.Fatalf("双方确认后应 both=true")
	}
	if s2.GoldA != 100 || s2.GoldB != 30 {
		t.Fatalf("报价快照错误: %d/%d", s2.GoldA, s2.GoldB)
	}
	tm.Finish(a, b)
	if _, err := tm.SetGold(a, 1); err == nil {
		t.Fatalf("Finish 后会话应已清理")
	}
}

// TestTradeManager_DoubleConfirmSingleFire 防重复 CONFIRM 导致 ExecuteSwap 跑两次(金币翻倍)。
// 双方确认后,首次 Confirm 返回 ready=true,之后任何重复 Confirm 都必须 ready=false。
func TestTradeManager_DoubleConfirmSingleFire(t *testing.T) {
	tm := NewTradeManager()
	a, b := id.PlayerIdType(1), id.PlayerIdType(2)
	if _, err := tm.Start(a, b); err != nil {
		t.Fatalf("Start: %v", err)
	}
	_, _ = tm.SetGold(a, 100)
	_, _ = tm.SetGold(b, 30)

	if _, r, _ := tm.Confirm(a); r {
		t.Fatalf("仅 a 确认不应 ready")
	}
	if _, r, _ := tm.Confirm(b); !r {
		t.Fatalf("双方确认首次应 ready")
	}
	// 重复 CONFIRM（模拟客户端重发 / worker pool 并发）——绝不能再 ready
	if _, r, _ := tm.Confirm(b); r {
		t.Fatalf("重复确认不应再 ready(否则金币翻倍)")
	}
	if _, r, _ := tm.Confirm(a); r {
		t.Fatalf("已 completing,任何再确认都不应 ready")
	}
}

// TestTradeManager_SetGoldResetsConfirm 改价重置确认（T2）：一方确认后另一方改价，双方确认必须清零，
// 防"确认后偷改价"在旧值上成交。同时验证重复设同值不打回确认（避免应价震荡）。
func TestTradeManager_SetGoldResetsConfirm(t *testing.T) {
	tm := NewTradeManager()
	a, b := id.PlayerIdType(1), id.PlayerIdType(2)
	if _, err := tm.Start(a, b); err != nil {
		t.Fatalf("Start: %v", err)
	}
	_, _ = tm.SetGold(a, 100)
	_, _ = tm.SetGold(b, 30)

	// a 先确认；此时 b 改价 → 必须把 a 的确认也清零（防在旧值上成交）
	_, _, _ = tm.Confirm(a)
	s, _ := tm.SetGold(b, 50)
	if s.ConfirmA || s.ConfirmB {
		t.Fatalf("改价后双方确认应清零, got A=%v B=%v", s.ConfirmA, s.ConfirmB)
	}

	// a 重新确认；重复设同值不应打回已有确认（避免应价震荡）
	_, _, _ = tm.Confirm(a)
	s2, _ := tm.SetGold(b, 50)
	if !s2.ConfirmA {
		t.Fatalf("重复设同值不应清零已有确认")
	}

	// 值已稳定，双方确认 → 成交
	if _, both, _ := tm.Confirm(b); !both {
		t.Fatalf("值稳定后双方确认应成交")
	}
}

// TestTradeManager_ConcurrentNoRace 并发发起/报价/确认/取消,-race 下应零竞争。
func TestTradeManager_ConcurrentNoRace(t *testing.T) {
	tm := NewTradeManager()
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		a := id.PlayerIdType(2 * i)
		b := id.PlayerIdType(2*i + 1)
		go func() {
			defer wg.Done()
			if _, err := tm.Start(a, b); err == nil {
				_, _ = tm.SetGold(a, 10)
				_, _, _ = tm.Confirm(a)
				_, _ = tm.Cancel(b)
			}
		}()
	}
	wg.Wait()
}
