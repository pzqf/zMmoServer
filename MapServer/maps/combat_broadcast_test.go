package maps

import (
	"sync"
	"testing"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
)

// recordingAOINotifier 记录所有 NotifyAOI 事件，用于断言状态变更广播（业务④ 场景同步增强）。
// 加锁：NotifyAOI 由地图 actor goroutine 同步调用，测试主 goroutine 读取，-race 下需同步。
type recordingAOINotifier struct {
	mu     sync.Mutex
	events []recordedAOIEvt
}

type recordedAOIEvt struct {
	watcher   int64
	eventType uint32
	targetID  int64
	pos       common.Vector3
}

func (r *recordingAOINotifier) NotifyAOI(watcherPlayerID int64, mapID int32, eventType uint32, targetID int64, pos common.Vector3) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, recordedAOIEvt{watcher: watcherPlayerID, eventType: eventType, targetID: targetID, pos: pos})
}

func (r *recordingAOINotifier) countByType(t uint32) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if e.eventType == t {
			n++
		}
	}
	return n
}

func (r *recordingAOINotifier) lastAttrFor(target int64) (recordedAOIEvt, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var found recordedAOIEvt
	ok := false
	for _, e := range r.events {
		if e.eventType == crossserver.MsgInternalAOIAttr && e.targetID == target {
			found = e
			ok = true
		}
	}
	return found, ok
}

// TestCombatBroadcast_AttrAndDeath 校验战斗结算把 target 的血量/死亡经 AOI 广播给视野内玩家。
// 攻击者与目标同在视野内 → 攻击者应作为 watcher 收到每次伤害后的血量广播，以及最终死亡广播。
func TestCombatBroadcast_AttrAndDeath(t *testing.T) {
	m := NewMap(id.MapIdType(1), 1, "BroadcastMap", 1000, 1000)
	defer m.StopActor()

	rec := &recordingAOINotifier{}
	m.SetAOINotifier(rec)

	const playerID = id.PlayerIdType(100)
	const playerObjID = id.ObjectIdType(100)
	if err := m.AddPlayer(playerID, playerObjID, 100, 0, 100); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	const monsterObjID = id.ObjectIdType(500)
	monster := object.NewMonster(monsterObjID, 1, "M", common.Vector3{X: 101, Y: 0, Z: 101}, 1)
	m.Do(func() { m.AddObject(monster) })

	// 反复攻击直至击杀（level1 怪 → 120 HP）。上限足够高，避免因单次伤害较低而未触达。
	killed := false
	for i := 0; i < 1000 && !killed; i++ {
		if _, _, err := m.AttackTarget(playerID, playerObjID, monsterObjID); err != nil {
			t.Fatalf("AttackTarget[%d]: %v", i, err)
		}
		if rec.countByType(crossserver.MsgInternalAOIDeath) > 0 {
			killed = true
		}
	}
	if !killed {
		t.Fatalf("怪未被击杀，未收到死亡广播；血量广播=%d 条", rec.countByType(crossserver.MsgInternalAOIAttr))
	}

	attrCount := rec.countByType(crossserver.MsgInternalAOIAttr)
	if attrCount == 0 {
		t.Fatalf("未收到任何血量变更广播")
	}
	deathCount := rec.countByType(crossserver.MsgInternalAOIDeath)
	if deathCount == 0 {
		t.Fatalf("未收到死亡广播")
	}

	// 末次血量广播：watcher 应为攻击者玩家（非对象 ID），HP 载荷应满足 0<=cur<=max。
	last, ok := rec.lastAttrFor(int64(monsterObjID))
	if !ok {
		t.Fatalf("无针对该怪的血量广播")
	}
	if last.watcher != int64(playerID) {
		t.Fatalf("血量广播 watcher=%d, 期望攻击者 playerID=%d", last.watcher, playerID)
	}
	if last.pos.X < 0 || last.pos.X > last.pos.Y {
		t.Fatalf("血量载荷异常: cur=%v max=%v", last.pos.X, last.pos.Y)
	}

	t.Logf("血量广播=%d 条, 死亡广播=%d 条, 末次 HP=%.0f/%.0f, watcher(playerID)=%d",
		attrCount, deathCount, last.pos.X, last.pos.Y, last.watcher)
}

// TestCombatBroadcast_NoWatchersNoPanic 无 AOINotifier 时战斗结算不应 panic（兜底路径）。
func TestCombatBroadcast_NoWatchersNoPanic(t *testing.T) {
	m := NewMap(id.MapIdType(2), 1, "NoNotifier", 1000, 1000)
	defer m.StopActor()

	const playerID = id.PlayerIdType(100)
	const playerObjID = id.ObjectIdType(100)
	_ = m.AddPlayer(playerID, playerObjID, 100, 0, 100)
	monster := object.NewMonster(id.ObjectIdType(500), 1, "M", common.Vector3{X: 101, Y: 0, Z: 101}, 1)
	m.Do(func() { m.AddObject(monster) })

	if _, _, err := m.AttackTarget(playerID, playerObjID, id.ObjectIdType(500)); err != nil {
		t.Fatalf("AttackTarget: %v", err)
	}
}
