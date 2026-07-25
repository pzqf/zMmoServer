package maps

import (
	"runtime"
	"testing"
	"time"

	"github.com/pzqf/zCommon/common/id"
)

// TestMapCleanup_StopsGoroutines 验证 MAP-11 / 实例地图生命周期：每张 Map 起两条 goroutine
// （单写者 actor + 刷怪 spawnLoop），Cleanup() 必须把它们都停掉，否则副本/镜像/跨服实例地图
// 销毁后 goroutine 永久泄漏。批量建图→Cleanup→断言 goroutine 数回落到基线附近。
func TestMapCleanup_StopsGoroutines(t *testing.T) {
	settle := func() { time.Sleep(60 * time.Millisecond); runtime.GC() }
	settle()
	base := runtime.NumGoroutine()

	const N = 24
	maps := make([]*Map, 0, N)
	for i := 0; i < N; i++ {
		maps = append(maps, NewMap(id.MapIdType(9000+i), 1, "Instance", 500, 500))
	}
	settle()
	afterCreate := runtime.NumGoroutine()
	if afterCreate <= base+N { // 期望 ~2N（actor+spawn）
		t.Fatalf("expected maps to spawn goroutines: base=%d afterCreate=%d (N=%d)", base, afterCreate, N)
	}

	for _, m := range maps {
		m.Cleanup()
	}

	// 等 goroutine 退出（actor/spawn 均由 channel close 立即触发退出）。
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
		t.Fatalf("goroutines leaked after Cleanup: base=%d final=%d (created %d maps, ~%d goroutines)", base, got, N, 2*N)
	}

	// Cleanup 幂等：二次调用不 panic。
	maps[0].Cleanup()
}
