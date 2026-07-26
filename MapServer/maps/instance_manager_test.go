package maps

import (
	"runtime"
	"testing"
	"time"

	"github.com/pzqf/zCommon/common/id"
)

// TestInstanceManager_ReapEmptyLayers 验证通用实例生命周期（建议③）：
// 建 N 个分线实例(空) → 未到 grace 不回收 → 到 grace 回收 → goroutine 回落基线、登记表清零。
func TestInstanceManager_ReapEmptyLayers(t *testing.T) {
	settle := func() { time.Sleep(60 * time.Millisecond); runtime.GC() }
	settle()
	base := runtime.NumGoroutine()

	mm := NewMapManager()
	const N = 16
	for i := 0; i < N; i++ {
		mm.CreateInstance(InstanceKindLayer, id.MapIdType(1001), 1, "Layer", 500, 500)
	}
	if got := mm.InstanceCount(); got != N {
		t.Fatalf("InstanceCount=%d, 期望 %d", got, N)
	}
	settle()
	afterCreate := runtime.NumGoroutine()
	if afterCreate <= base+N { // 每个实例 ~2 goroutine（actor+spawn）
		t.Fatalf("实例应起 goroutine: base=%d afterCreate=%d (N=%d)", base, afterCreate, N)
	}

	// 空但未到 grace：不回收（设 emptySince 计时）。
	if r := mm.ReapEmpty(1 * time.Hour); r != 0 {
		t.Fatalf("未到 grace 不应回收, got %d", r)
	}
	if mm.InstanceCount() != N {
		t.Fatalf("未到 grace 实例数应不变")
	}

	// 已计时且 grace=0：全部回收。
	if r := mm.ReapEmpty(0); r != N {
		t.Fatalf("到 grace 应回收 %d, got %d", N, r)
	}
	if mm.InstanceCount() != 0 {
		t.Fatalf("回收后登记表应清零, got %d", mm.InstanceCount())
	}

	// goroutine 回落基线（回收=Cleanup 停 actor+spawn）。
	deadline := time.Now().Add(3 * time.Second)
	var got int
	for time.Now().Before(deadline) {
		runtime.GC()
		got = runtime.NumGoroutine()
		if got <= base+3 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got > base+3 {
		t.Fatalf("回收后 goroutine 泄漏: base=%d final=%d (N=%d)", base, got, N)
	}
}

// TestInstanceManager_DungeonExcludedFromReap 副本(Dungeon)不参与空置自动回收——由显式销毁管。
func TestInstanceManager_DungeonExcludedFromReap(t *testing.T) {
	mm := NewMapManager()
	_, mapID := mm.CreateInstance(InstanceKindDungeon, id.MapIdType(2001), 1, "Dungeon", 500, 500)
	defer mm.DestroyInstance(mapID)

	// 空且多次 ReapEmpty：副本不应被回收。
	mm.ReapEmpty(0)
	mm.ReapEmpty(0)
	if mm.InstanceCount() != 1 {
		t.Fatalf("副本不应被空置回收, InstanceCount=%d", mm.InstanceCount())
	}
	if _, ok := mm.GetInstance(mapID); !ok {
		t.Fatalf("副本实例应仍在")
	}
}

// TestInstanceManager_NonEmptyNotReaped 有玩家的实例不被回收；玩家离开后才回收。
func TestInstanceManager_NonEmptyNotReaped(t *testing.T) {
	mm := NewMapManager()
	m, mapID := mm.CreateInstance(InstanceKindLayer, id.MapIdType(3001), 1, "Layer", 500, 500)

	if err := m.AddPlayer(id.PlayerIdType(1), id.ObjectIdType(1), 100, 0, 100); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if m.GetPlayerCount() == 0 {
		t.Fatalf("加人后玩家数应 >0")
	}

	// 有人：多次 ReapEmpty 不回收。
	mm.ReapEmpty(0)
	mm.ReapEmpty(0)
	if mm.InstanceCount() != 1 {
		t.Fatalf("有玩家的实例不应被回收")
	}

	// 玩家离开 → 空置 → 回收。
	m.RemovePlayer(id.PlayerIdType(1))
	mm.ReapEmpty(0) // 计时
	if r := mm.ReapEmpty(0); r != 1 {
		t.Fatalf("清空后应回收, got %d", r)
	}
	if mm.InstanceCount() != 0 {
		t.Fatalf("回收后应清零")
	}
	_ = mapID
}

// TestInstanceManager_DestroyIdempotent DestroyInstance 幂等，重复销毁不 panic。
func TestInstanceManager_DestroyIdempotent(t *testing.T) {
	mm := NewMapManager()
	_, mapID := mm.CreateInstance(InstanceKindBattleground, id.MapIdType(4001), 1, "BG", 500, 500)
	mm.DestroyInstance(mapID)
	mm.DestroyInstance(mapID) // 二次销毁：no-op，不 panic
	if mm.InstanceCount() != 0 {
		t.Fatalf("销毁后应清零")
	}
	if _, ok := mm.GetInstance(mapID); ok {
		t.Fatalf("销毁后不应还查得到")
	}
}
