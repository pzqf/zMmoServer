package maps

import (
	"sort"
	"testing"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
)

// TestGetObjectsInRange_MatchesBruteForce 验证 OPT-2：AOI 网格缩候选 + 平方距离的
// GetObjectsInRange 与朴素全量扫描结果完全一致，覆盖 radius ≤ 单格(走 AOI 路径) 与
// radius > 单格(回退全量) 两种情形。这是空间优化的正确性护栏——缩候选若漏格就会丢目标。
func TestGetObjectsInRange_MatchesBruteForce(t *testing.T) {
	m := NewMap(id.MapIdType(3), 3, "RangeMap", 1000, 1000)
	defer m.StopActor()

	// 在全图铺开若干对象（跨多个 AOI 网格）。
	positions := []common.Vector3{
		{X: 10, Y: 0, Z: 10}, {X: 12, Y: 0, Z: 13}, {X: 48, Y: 0, Z: 49},
		{X: 55, Y: 0, Z: 55}, {X: 100, Y: 0, Z: 100}, {X: 105, Y: 0, Z: 98},
		{X: 300, Y: 0, Z: 300}, {X: 500, Y: 0, Z: 20}, {X: 20, Y: 0, Z: 500},
		{X: 0, Y: 0, Z: 0}, {X: 999, Y: 0, Z: 999},
	}
	for i, p := range positions {
		mon := object.NewMonster(id.ObjectIdType(1000+i), 1, "M", p, 1)
		m.Do(func() { m.AddObject(mon) })
	}

	bruteForce := func(center common.Vector3, radius float32) []int64 {
		var ids []int64
		for _, obj := range m.GetObjects() {
			if center.DistanceTo(obj.GetPosition()) <= radius {
				ids = append(ids, int64(obj.GetID()))
			}
		}
		sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
		return ids
	}
	actual := func(center common.Vector3, radius float32) []int64 {
		var ids []int64
		for _, obj := range m.GetObjectsInRange(center, radius) {
			ids = append(ids, int64(obj.GetID()))
		}
		sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
		return ids
	}
	eq := func(a, b []int64) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	cases := []struct {
		center common.Vector3
		radius float32
	}{
		{common.Vector3{X: 11, Y: 0, Z: 11}, 5},      // 小半径,AOI 路径
		{common.Vector3{X: 50, Y: 0, Z: 50}, 10},     // 跨格边界,AOI 路径
		{common.Vector3{X: 100, Y: 0, Z: 100}, 50},   // 恰好单格尺寸,AOI 路径
		{common.Vector3{X: 100, Y: 0, Z: 100}, 200},  // 超过单格,回退全量
		{common.Vector3{X: 500, Y: 0, Z: 500}, 800},  // 大半径,回退全量
		{common.Vector3{X: 0, Y: 0, Z: 0}, 1},        // 边角小半径
	}
	for _, c := range cases {
		want := bruteForce(c.center, c.radius)
		got := actual(c.center, c.radius)
		if !eq(want, got) {
			t.Errorf("range query mismatch at center=%v radius=%.0f: want %v, got %v", c.center, c.radius, want, got)
		}
	}
}
