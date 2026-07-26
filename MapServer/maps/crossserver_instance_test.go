package maps

import (
	"testing"

	"github.com/pzqf/zCommon/common/id"
)

// TestCrossServerInstance_TrackedAndReaped 验证 §5.1：跨服临时实例经 ③ 登记为 InstanceKindCrossServer，
// 享有统一实例生命周期——是跨服图、派生 mapID、空置被 ReapEmpty 自动回收（打完人走即销毁）。
func TestCrossServerInstance_TrackedAndReaped(t *testing.T) {
	mm := NewMapManager()

	crossMap := mm.CreateCrossServerMap(5001, "CrossWar", 500, 500, MapModeCrossGroup, 7)
	if crossMap == nil {
		t.Fatalf("CreateCrossServerMap 返回 nil")
	}
	mapID := crossMap.GetID()

	// 登记为 CrossServer 种类的实例（③ 统一生命周期）。
	inst, ok := mm.GetInstance(mapID)
	if !ok {
		t.Fatalf("跨服图应登记为实例, 却查不到")
	}
	if inst.Kind != InstanceKindCrossServer {
		t.Fatalf("实例种类应为 CrossServer, got %v", inst.Kind)
	}
	if !crossMap.IsCrossServer() {
		t.Fatalf("应为跨服图(MapModeCrossGroup)")
	}
	if int64(mapID) < 100000 {
		t.Fatalf("跨服实例应是派生 mapID(>=100000), got %d", mapID)
	}

	// 跨服玩家（来自不同归属服——归属由 net 层 PlayerGameServerManager 跟踪，战斗结算由 ④ AttrGrant
	// 按归属回流各自家服；此处只验实例生命周期）。加两人。
	if err := crossMap.AddPlayer(id.PlayerIdType(1), id.ObjectIdType(1), 100, 0, 100); err != nil {
		t.Fatalf("AddPlayer 1: %v", err)
	}
	if err := crossMap.AddPlayer(id.PlayerIdType(2), id.ObjectIdType(2), 100, 0, 100); err != nil {
		t.Fatalf("AddPlayer 2: %v", err)
	}

	// 有人时不回收。
	mm.ReapEmpty(0)
	mm.ReapEmpty(0)
	if mm.InstanceCount() != 1 {
		t.Fatalf("有玩家的跨服实例不应被回收")
	}

	// 打完人走 → 空置 → 自动回收。
	crossMap.RemovePlayer(id.PlayerIdType(1))
	crossMap.RemovePlayer(id.PlayerIdType(2))
	mm.ReapEmpty(0) // 计时
	if r := mm.ReapEmpty(0); r != 1 {
		t.Fatalf("空置跨服实例应被回收, got %d", r)
	}
	if _, ok := mm.GetInstance(mapID); ok {
		t.Fatalf("回收后不应还查得到")
	}
}
