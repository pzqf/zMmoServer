package maps

import (
	"fmt"
	"sync"
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
		lm.mm.ReleaseLayerReservation(layerMapID)
		t.Fatalf("AddPlayer: %v", err)
	}
	lm.mm.ReleaseLayerReservation(layerMapID) // 玩家已加入，归还在途预留（同 HandlePlayerEnterMap）
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
	m := mm.GetMap(l1)                 // 拿到该层（不经 AllocateLayer，避免占用在途预留干扰回收）
	m.RemovePlayer(id.PlayerIdType(1)) // 清空

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

// TestLayer_ConcurrentEnter_NoCapBreach ②M1 守护：多人并发进同一可分线图，softCap/hardCap 不被击穿。
// 修复前 AllocateLayer 读的 GetPlayerCount 滞后于 AddPlayer，N 个并发进图看到同一份人数、一起选中同层
// → 单层超 cap；引入在途预留(reserved 计入 effective 人数)后不再击穿。SoftCap=HardCap=3、n=30 → 每层
// 恰好 3 人、共 10 层。(AllocateLayer 全程持 lm.mu 串行 + reserved 原子，故打包紧凑且不超限。)
func TestLayer_ConcurrentEnter_NoCapBreach(t *testing.T) {
	mm := NewMapManager()
	lm := NewLayerManager(mm)
	const logical = id.MapIdType(1010)
	const cap = 3
	lm.RegisterLayerable(logical, LayerConfig{MapConfigID: 1, Name: "Field", Width: 1000, Height: 1000, SoftCap: cap, HardCap: cap})

	const n = 30
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 1; i <= n; i++ {
		wg.Add(1)
		go func(pid int) {
			defer wg.Done()
			m, layerMapID, ok := lm.AllocateLayer(logical, 0)
			if !ok {
				errs <- fmt.Errorf("player %d: AllocateLayer 失败", pid)
				return
			}
			err := m.AddPlayer(id.PlayerIdType(pid), id.ObjectIdType(pid), 100, 0, 100)
			mm.ReleaseLayerReservation(layerMapID) // 无论成败都归还（同 HandlePlayerEnterMap 的 defer）
			if err != nil {
				errs <- fmt.Errorf("player %d: AddPlayer: %w", pid, err)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}

	layers := mm.GetInstancesByLogical(logical, InstanceKindLayer)
	total := 0
	for _, inst := range layers {
		c := inst.Map.GetPlayerCount()
		if c > cap {
			t.Fatalf("层 %d 超 cap(%d): %d 人（softCap/hardCap 被击穿）", inst.MapID, cap, c)
		}
		total += c
	}
	if total != n {
		t.Fatalf("总人数应=%d, got %d", n, total)
	}
	if len(layers) != n/cap {
		t.Fatalf("应恰好 %d 层(每层 %d 人), got %d 层", n/cap, cap, len(layers))
	}
}

// TestLayer_ReservedBlocksReap ②M2 守护：AllocateLayer 选中/新建层后（reserved>0、玩家尚未 AddPlayer），
// 该层被视为非空、不被 ReapEmpty 回收——否则"选中→AddPlayer"窗口内空层被回收，玩家被加进已销毁的死图。
func TestLayer_ReservedBlocksReap(t *testing.T) {
	mm := NewMapManager()
	lm := NewLayerManager(mm)
	const logical = id.MapIdType(1011)
	lm.RegisterLayerable(logical, LayerConfig{MapConfigID: 1, Name: "Field", Width: 500, Height: 500, SoftCap: 5, HardCap: 8})

	// 模拟"已分配、玩家尚未加入"：AllocateLayer 后 reserved=1、count=0。
	m, layerMapID, ok := lm.AllocateLayer(logical, 0)
	if !ok {
		t.Fatalf("AllocateLayer 应成功")
	}
	mm.ReapEmpty(0)
	mm.ReapEmpty(0)
	if _, ok := mm.GetInstance(layerMapID); !ok {
		t.Fatalf("在途预留(reserved>0)的层不应被 ReapEmpty 回收")
	}

	// 玩家加入并归还预留后再清空，才可被回收。
	if err := m.AddPlayer(id.PlayerIdType(1), id.ObjectIdType(1), 100, 0, 100); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	mm.ReleaseLayerReservation(layerMapID)
	m.RemovePlayer(id.PlayerIdType(1))
	mm.ReapEmpty(0)
	mm.ReapEmpty(0)
	if _, ok := mm.GetInstance(layerMapID); ok {
		t.Fatalf("归还预留且空置后应被 ReapEmpty 回收")
	}
}
