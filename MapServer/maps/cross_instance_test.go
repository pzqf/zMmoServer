package maps

import (
	"testing"

	"github.com/pzqf/zCommon/common/id"
)

// 跨服活动实例（跨 realm 物理路由 ④）的核心不变量：**同一 activityID 只有一张实例地图**。
// 多个 realm 的 GameServer 会各自来请求，若各建一张，跨服玩家永远碰不到面——这正是这套
// 物理路由要解决的问题本身，故用测试钉死。

// TestEnsureCrossInstance_IdempotentAcrossRealms 三个不同 realm 的 GameServer 为同一活动请求，
// 必须拿到同一张实例地图。
func TestEnsureCrossInstance_IdempotentAcrossRealms(t *testing.T) {
	mm := NewMapManager()
	defer mm.Stop()

	const activityID = int64(90001)

	first, mapID, created := mm.EnsureCrossServerInstance(activityID, 5001, "CrossWar", 500, 500, 101)
	if first == nil || mapID == 0 {
		t.Fatalf("首次请求应建出实例, map=%v id=%d", first, mapID)
	}
	if !created {
		t.Fatalf("首次请求 created 应为 true")
	}

	// 模拟另外两个 realm 的 GameServer 来问（同一活动、同一配置）。
	for i := 0; i < 2; i++ {
		m, gotID, gotCreated := mm.EnsureCrossServerInstance(activityID, 5001, "CrossWar", 500, 500, 201)
		if gotID != mapID {
			t.Fatalf("不同 realm 必须收敛到同一张实例: want %d got %d", mapID, gotID)
		}
		if m != first {
			t.Fatalf("应返回同一张 *Map")
		}
		if gotCreated {
			t.Fatalf("复用时 created 应为 false")
		}
	}

	if mm.InstanceCount() != 1 {
		t.Fatalf("同一活动只应有 1 个实例, got %d", mm.InstanceCount())
	}
}

// TestEnsureCrossInstance_DifferentActivitiesDoNotShare 不同活动各自独立一张地图。
func TestEnsureCrossInstance_DifferentActivitiesDoNotShare(t *testing.T) {
	mm := NewMapManager()
	defer mm.Stop()

	_, mapA, _ := mm.EnsureCrossServerInstance(90002, 5001, "", 0, 0, 101)
	_, mapB, _ := mm.EnsureCrossServerInstance(90003, 5001, "", 0, 0, 101)

	if mapA == 0 || mapB == 0 || mapA == mapB {
		t.Fatalf("不同活动应各有一张实例: a=%d b=%d", mapA, mapB)
	}
	if mm.InstanceCount() != 2 {
		t.Fatalf("expected 2 instances, got %d", mm.InstanceCount())
	}
}

// TestEnsureCrossInstance_IsCrossServerKind 建出来的必须是跨服种类的实例地图——
// 于是它享有统一实例生命周期（空置由 ReapEmpty 自动回收），而不是一张永不回收的常驻图。
func TestEnsureCrossInstance_IsCrossServerKind(t *testing.T) {
	mm := NewMapManager()
	defer mm.Stop()

	crossMap, mapID, _ := mm.EnsureCrossServerInstance(90004, 5001, "", 0, 0, 77)

	inst, ok := mm.GetInstance(mapID)
	if !ok {
		t.Fatalf("实例应登记在池里")
	}
	if inst.Kind != InstanceKindCrossServer {
		t.Fatalf("种类应为 crossserver, got %s", inst.Kind)
	}
	if !crossMap.IsCrossServer() {
		t.Fatalf("地图应为跨服模式")
	}
	if crossMap.GetServerGroupID() != 77 {
		t.Fatalf("承载服务器组应记在地图上, got %d", crossMap.GetServerGroupID())
	}
}

// TestEnsureCrossInstance_RecreatedAfterReap 实例空置被回收后，再请求会重建并改绑——
// 绑定表是**惰性失效**的，不能返回一个已经销毁的死图。
func TestEnsureCrossInstance_RecreatedAfterReap(t *testing.T) {
	mm := NewMapManager()
	defer mm.Stop()

	const activityID = int64(90005)
	_, firstID, _ := mm.EnsureCrossServerInstance(activityID, 5001, "", 0, 0, 101)

	if got, ok := mm.GetCrossServerInstanceMapID(activityID); !ok || got != firstID {
		t.Fatalf("绑定应指向首个实例: ok=%v got=%d want=%d", ok, got, firstID)
	}

	// 空置回收（Reap 需两次：首次计时、二次回收，见 crossserver_instance_test）。
	mm.ReapEmpty(0)
	mm.ReapEmpty(0)
	if mm.InstanceCount() != 0 {
		t.Fatalf("空置实例应被回收, 剩 %d", mm.InstanceCount())
	}
	if _, ok := mm.GetCrossServerInstanceMapID(activityID); ok {
		t.Fatalf("实例已回收，绑定查询不应再报有效")
	}

	_, secondID, created := mm.EnsureCrossServerInstance(activityID, 5001, "", 0, 0, 101)
	if secondID == 0 || secondID == firstID {
		t.Fatalf("回收后应重建出新实例: first=%d second=%d", firstID, secondID)
	}
	if !created {
		t.Fatalf("重建时 created 应为 true")
	}
	if m := mm.GetMap(secondID); m == nil {
		t.Fatalf("重建的实例地图应可用")
	}
}

// TestReapCrossActivityBindings 回收后清理绑定表，防其随活动数无界增长。
func TestReapCrossActivityBindings(t *testing.T) {
	mm := NewMapManager()
	defer mm.Stop()

	mm.EnsureCrossServerInstance(90006, 5001, "", 0, 0, 101)
	mm.EnsureCrossServerInstance(90007, 5001, "", 0, 0, 101)

	if n := mm.ReapCrossActivityBindings(); n != 0 {
		t.Fatalf("实例还活着时不应清绑定, got %d", n)
	}

	mm.ReapEmpty(0)
	mm.ReapEmpty(0)

	if n := mm.ReapCrossActivityBindings(); n != 2 {
		t.Fatalf("实例回收后应清掉 2 条绑定, got %d", n)
	}
	if _, ok := mm.GetCrossServerInstanceMapID(90006); ok {
		t.Fatalf("绑定应已清")
	}
}

// TestEnsureCrossInstance_RejectsInvalidActivity 非法 activityID 明确拒绝（不静默建一张野图）。
func TestEnsureCrossInstance_RejectsInvalidActivity(t *testing.T) {
	mm := NewMapManager()
	defer mm.Stop()

	m, mapID, created := mm.EnsureCrossServerInstance(0, 5001, "", 0, 0, 101)
	if m != nil || mapID != 0 || created {
		t.Fatalf("activityID<=0 应拒绝: m=%v id=%d created=%v", m, mapID, created)
	}
	if mm.InstanceCount() != 0 {
		t.Fatalf("不应建出任何实例, got %d", mm.InstanceCount())
	}
}

// TestEnsureCrossInstance_PlayersShareOneMap 两个来自不同 realm 的玩家进同一活动，
// 应落在同一张地图上、能互相看见（AOI 的前提是同一张 *Map）。
func TestEnsureCrossInstance_PlayersShareOneMap(t *testing.T) {
	mm := NewMapManager()
	defer mm.Stop()

	const activityID = int64(90008)
	_, mapID, _ := mm.EnsureCrossServerInstance(activityID, 5001, "", 0, 0, 101)

	// realm A 的玩家 与 realm B 的玩家（GameServer 不同，但都被送进这张实例）。
	if err := mm.HandlePlayerEnterMap(11001, int64(mapID), 100, 100, 0, 10, 500); err != nil {
		t.Fatalf("realm A 玩家进图失败: %v", err)
	}
	if err := mm.HandlePlayerEnterMap(22001, int64(mapID), 105, 100, 0, 12, 600); err != nil {
		t.Fatalf("realm B 玩家进图失败: %v", err)
	}

	m := mm.GetMap(mapID)
	if m == nil {
		t.Fatalf("实例地图应存在")
	}
	if got := m.GetPlayerCount(); got != 2 {
		t.Fatalf("两个 realm 的玩家应在同一张实例里, got %d", got)
	}
	if m.GetObject(id.ObjectIdType(11001)) == nil {
		t.Fatalf("realm A 玩家应在图内")
	}
	if m.GetObject(id.ObjectIdType(22001)) == nil {
		t.Fatalf("realm B 玩家应在图内")
	}
}
