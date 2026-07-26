package maps

import (
	"testing"

	"github.com/pzqf/zCommon/common/id"
)

// enter 把玩家加进分配到的层（模拟进图），返回选中的 layerMapID。失败则 t.Fatal。
func allocAndEnter(t *testing.T, lm *LayerManager, logicalMapID, affinity id.MapIdType, pid int64) id.MapIdType {
	t.Helper()
	m, layerMapID, ok := lm.AllocateLayer(logicalMapID, affinity)
	if !ok {
		t.Fatalf("AllocateLayer(%d) 应成功", logicalMapID)
	}
	if err := m.AddPlayer(id.PlayerIdType(pid), id.ObjectIdType(pid), 100, 0, 100); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return layerMapID
}

// TestLayer_FillThenNewLayer 填满软上限就开新层。
func TestLayer_FillThenNewLayer(t *testing.T) {
	mm := NewMapManager()
	lm := NewLayerManager(mm)
	const logical = id.MapIdType(1001)
	lm.RegisterLayerable(logical, LayerConfig{MapConfigID: 1, Name: "Field", Width: 500, Height: 500, SoftCap: 2, HardCap: 3})

	// 前 2 人（softCap=2）：无亲和 → 都进第一层。
	l1 := allocAndEnter(t, lm, logical, 0, 1)
	l2 := allocAndEnter(t, lm, logical, 0, 2)
	if l1 != l2 {
		t.Fatalf("前 2 人应同层: %d vs %d", l1, l2)
	}
	if n := len(mm.GetInstancesByLogical(logical, InstanceKindLayer)); n != 1 {
		t.Fatalf("此时应只有 1 层, got %d", n)
	}

	// 第 3 人：第一层已达 softCap → 开新层。
	l3 := allocAndEnter(t, lm, logical, 0, 3)
	if l3 == l1 {
		t.Fatalf("第 3 人应进新层, 却进了第一层")
	}
	if n := len(mm.GetInstancesByLogical(logical, InstanceKindLayer)); n != 2 {
		t.Fatalf("应有 2 层, got %d", n)
	}
}

// TestLayer_Affinity 亲和：同队目标层未到硬上限时优先进该层（可超软上限）。
func TestLayer_Affinity(t *testing.T) {
	mm := NewMapManager()
	lm := NewLayerManager(mm)
	const logical = id.MapIdType(1002)
	lm.RegisterLayerable(logical, LayerConfig{MapConfigID: 1, Name: "Field", Width: 500, Height: 500, SoftCap: 1, HardCap: 3})

	l1 := allocAndEnter(t, lm, logical, 0, 1) // 第一层 count=1（=softCap）

	// 队友带亲和进 l1：虽已达 softCap，但 <hardCap → 仍进 l1。
	l2 := allocAndEnter(t, lm, logical, l1, 2)
	if l2 != l1 {
		t.Fatalf("亲和应进同层: %d vs %d", l2, l1)
	}
	l3 := allocAndEnter(t, lm, logical, l1, 3) // count 2→3（=hardCap）
	if l3 != l1 {
		t.Fatalf("亲和(未到 hardCap)应进同层")
	}
	if n := len(mm.GetInstancesByLogical(logical, InstanceKindLayer)); n != 1 {
		t.Fatalf("亲和挤同层, 应仍 1 层, got %d", n)
	}

	// 第 4 人再带亲和 l1：l1 已到 hardCap=3 → 亲和失效 → 无层 <softCap → 开新层。
	l4 := allocAndEnter(t, lm, logical, l1, 4)
	if l4 == l1 {
		t.Fatalf("l1 已满 hardCap, 第 4 人应进新层")
	}
	if n := len(mm.GetInstancesByLogical(logical, InstanceKindLayer)); n != 2 {
		t.Fatalf("应有 2 层, got %d", n)
	}
}

// TestLayer_NonLayerable 未注册的逻辑地图不可分线。
func TestLayer_NonLayerable(t *testing.T) {
	mm := NewMapManager()
	lm := NewLayerManager(mm)
	if lm.IsLayerable(id.MapIdType(9999)) {
		t.Fatalf("未注册不应可分线")
	}
	if _, _, ok := lm.AllocateLayer(id.MapIdType(9999), 0); ok {
		t.Fatalf("未注册的逻辑地图 AllocateLayer 应返回 false")
	}
}

// TestLayer_ReapThenRecreate 空层被 ReapEmpty 回收后，再进图会重建。
func TestLayer_ReapThenRecreate(t *testing.T) {
	mm := NewMapManager()
	lm := NewLayerManager(mm)
	const logical = id.MapIdType(1003)
	lm.RegisterLayerable(logical, LayerConfig{MapConfigID: 1, Name: "Field", Width: 500, Height: 500, SoftCap: 5, HardCap: 8})

	l1 := allocAndEnter(t, lm, logical, 0, 1)
	m, _, _ := lm.AllocateLayer(logical, l1) // 拿到该层
	m.RemovePlayer(id.PlayerIdType(1))       // 清空

	mm.ReapEmpty(0) // 计时
	mm.ReapEmpty(0) // 回收空层
	if n := len(mm.GetInstancesByLogical(logical, InstanceKindLayer)); n != 0 {
		t.Fatalf("空层应被回收, 却剩 %d 层", n)
	}

	// 再进图 → 重建一层。
	allocAndEnter(t, lm, logical, 0, 2)
	if n := len(mm.GetInstancesByLogical(logical, InstanceKindLayer)); n != 1 {
		t.Fatalf("再进图应重建 1 层, got %d", n)
	}
}
