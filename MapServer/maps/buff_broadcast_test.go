package maps

import (
	"testing"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/models"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zCommon/crossserver"
)

func (r *recordingAOINotifier) lastBuffFor(target int64) (recordedAOIEvt, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var found recordedAOIEvt
	ok := false
	for _, e := range r.events {
		if e.eventType == crossserver.MsgInternalAOIBuff && e.targetID == target {
			found = e
			ok = true
		}
	}
	return found, ok
}

func (r *recordingAOINotifier) countBuffOps(target int64) (added, removed int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.events {
		if e.eventType == crossserver.MsgInternalAOIBuff && e.targetID == target {
			if e.pos.Y > 0.5 {
				added++
			} else {
				removed++
			}
		}
	}
	return
}

// TestBuffBroadcast_AddAndExpire 校验 buff 增/删经 AOI 广播给视野内玩家（场景同步增强）。
// 走真实 buffManager：施加 1 秒 buff → 广播「获得」；到期后 tick 触发 Update → 广播「移除」。
func TestBuffBroadcast_AddAndExpire(t *testing.T) {
	// 种子一条 1 秒 buff 配置到全局表管理器（地图的 buffManager 共用它）。
	tables.GetTableManager().GetBuffLoader().SetBuffConfig(&models.Buff{
		BuffID: 9001, Name: "TestBuff", Type: "增益", Duration: 1, Value: 10, Property: "attack", IsPermanent: false,
	})

	m := NewMap(id.MapIdType(3), 1, "BuffMap", 1000, 1000)
	defer m.StopActor()
	rec := &recordingAOINotifier{}
	m.SetAOINotifier(rec)

	const playerID = id.PlayerIdType(100)
	const playerObjID = id.ObjectIdType(100)
	if err := m.AddPlayer(playerID, playerObjID, 100, 0, 100); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	// 施加 buff 并广播「获得」（模拟 skill_handler 的施加点，全程在 actor goroutine 内）。
	m.Do(func() {
		if err := m.buffManager.AddBuff(playerID, 9001, playerObjID); err != nil {
			t.Errorf("AddBuff: %v", err)
			return
		}
		if obj := m.GetObject(playerObjID); obj != nil {
			m.broadcastEntityBuff(obj, 9001, true, m.buffRemainingSec(playerID, 9001))
		}
	})

	addEvt, ok := rec.lastBuffFor(int64(playerObjID))
	if !ok {
		t.Fatalf("未收到 buff 广播")
	}
	if int32(addEvt.pos.X) != 9001 {
		t.Fatalf("buffID=%v, 期望 9001", addEvt.pos.X)
	}
	if addEvt.pos.Y < 0.5 {
		t.Fatalf("added 标志应为 1（获得）, got %v", addEvt.pos.Y)
	}
	if addEvt.pos.Z <= 0 {
		t.Fatalf("剩余秒数应 >0, got %v", addEvt.pos.Z)
	}
	if addEvt.watcher != int64(playerID) {
		t.Fatalf("watcher=%d, 期望 playerID=%d", addEvt.watcher, playerID)
	}

	// 到期后 tick 触发 Update → 广播「移除」。
	time.Sleep(1100 * time.Millisecond)
	m.Do(func() { m.tick(time.Second) })

	remEvt, ok := rec.lastBuffFor(int64(playerObjID))
	if !ok {
		t.Fatalf("无 buff 广播")
	}
	if remEvt.pos.Y > 0.5 {
		t.Fatalf("过期后末次应为移除(added=0), got %v", remEvt.pos.Y)
	}
	added, removed := rec.countBuffOps(int64(playerObjID))
	if added == 0 || removed == 0 {
		t.Fatalf("buff 获得/移除广播均应 >0, got added=%d removed=%d", added, removed)
	}
	t.Logf("buff 获得广播=%d, 移除广播=%d, watcher(playerID)=%d, 获得时剩余=%.2fs",
		added, removed, addEvt.watcher, addEvt.pos.Z)
}
