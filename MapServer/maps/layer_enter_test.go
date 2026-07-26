package maps

import (
	"testing"

	"github.com/pzqf/zCommon/common/id"
)

// TestLayerEnter_SplitAndResolve 验证 ②-b 进图接线：可分线图上两玩家进同一逻辑图 → 分到不同层；
// 后续 op 经 resolveMap 定位到各自的层；离图清理。
func TestLayerEnter_SplitAndResolve(t *testing.T) {
	mm := NewMapManager()
	const logical = int64(1001)
	// softCap=1 → 第二个玩家会被分到新层。
	mm.EnableLayering().RegisterLayerable(id.MapIdType(logical), LayerConfig{
		MapConfigID: 1, Name: "Field", Width: 500, Height: 500, SoftCap: 1, HardCap: 2,
	})

	// 玩家 1、2 进同一逻辑图。
	if err := mm.HandlePlayerEnterMap(1, logical, 100, 0, 100); err != nil {
		t.Fatalf("p1 enter: %v", err)
	}
	if err := mm.HandlePlayerEnterMap(2, logical, 100, 0, 100); err != nil {
		t.Fatalf("p2 enter: %v", err)
	}

	// 应建了 2 个层实例。
	layers := mm.GetInstancesByLogical(id.MapIdType(logical), InstanceKindLayer)
	if len(layers) != 2 {
		t.Fatalf("两玩家(softCap=1)应分到 2 层, got %d", len(layers))
	}

	// resolveMap：两玩家解析到不同的层图，且各自层里有该玩家、互不可见。
	m1 := mm.resolveMap(1, logical)
	m2 := mm.resolveMap(2, logical)
	if m1 == nil || m2 == nil {
		t.Fatalf("resolveMap 不应为 nil")
	}
	if m1.GetID() == m2.GetID() {
		t.Fatalf("两玩家应在不同层: 都在 %d", m1.GetID())
	}
	if m1.GetPlayerCount() != 1 || m2.GetPlayerCount() != 1 {
		t.Fatalf("每层应各 1 人, got %d / %d", m1.GetPlayerCount(), m2.GetPlayerCount())
	}
	// 逻辑图本身没有 layerMapID，两层都是派生 ID（>=100000）。
	if int64(m1.GetID()) < 100000 || int64(m2.GetID()) < 100000 {
		t.Fatalf("层应是派生 mapID(>=100000): %d / %d", m1.GetID(), m2.GetID())
	}

	// 玩家 1 离图：其层清空 + playerMap 记录清理（resolveMap 回落到逻辑图）。
	if err := mm.HandlePlayerLeaveMap(1, logical); err != nil {
		t.Fatalf("p1 leave: %v", err)
	}
	if m1.GetPlayerCount() != 0 {
		t.Fatalf("p1 离图后其层应空")
	}
	// 离图后 resolveMap(1) 回落到逻辑图（此处逻辑图未创建 → nil，验证不再指向层）。
	if got := mm.resolveMap(1, logical); got != nil && int64(got.GetID()) >= 100000 {
		t.Fatalf("p1 离图后不应再解析到层图")
	}
}

// TestLayerEnter_NonLayerableUsesLogical 非可分线图：进图仍走逻辑图本身（不建层）。
func TestLayerEnter_NonLayerableUsesLogical(t *testing.T) {
	mm := NewMapManager()
	const logical = int64(2001)
	m := mm.CreateMap(id.MapIdType(logical), 1, "Town", 500, 500) // 建逻辑图
	if m == nil {
		t.Fatalf("CreateMap 失败")
	}

	if err := mm.HandlePlayerEnterMap(7, logical, 10, 0, 10); err != nil {
		t.Fatalf("enter: %v", err)
	}
	// 未启用分线 → 不建层，玩家在逻辑图本身。
	if n := len(mm.GetInstancesByLogical(id.MapIdType(logical), InstanceKindLayer)); n != 0 {
		t.Fatalf("非可分线图不应建层, got %d", n)
	}
	if mm.resolveMap(7, logical).GetID() != id.MapIdType(logical) {
		t.Fatalf("非可分线图玩家应在逻辑图本身")
	}
	if m.GetPlayerCount() != 1 {
		t.Fatalf("逻辑图应有 1 人")
	}
}
