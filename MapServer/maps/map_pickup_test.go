package maps

import (
	"testing"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
)

// TestPickupNearest 校验就近拾取：只拾范围内最近的可拾取物、拾后从地图移除、范围外不动、拾空返回 false。
func TestPickupNearest(t *testing.T) {
	m := NewMap(id.MapIdType(1), 1, "PickupMap", 1000, 1000)
	defer m.StopActor()

	const playerID = id.PlayerIdType(100)
	const playerObjID = id.ObjectIdType(100)
	if err := m.AddPlayer(playerID, playerObjID, 100, 0, 100); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	// 范围外掉落物（远）——不应被拾取
	far := object.NewItem(id.ObjectIdType(700), 2001, "Far",
		common.Vector3{X: 900, Y: 0, Z: 900}, 1, object.ItemTypeConsumable, object.ItemRarityCommon)
	m.Do(func() { m.AddObject(far) })

	// 范围内掉落物——应被拾取
	near := object.NewItem(id.ObjectIdType(701), 1003, "Near",
		common.Vector3{X: 101, Y: 0, Z: 101}, 3, object.ItemTypeConsumable, object.ItemRarityCommon)
	m.Do(func() { m.AddObject(near) })

	res := m.PickupNearest(playerID)
	if !res.OK {
		t.Fatalf("应拾到范围内掉落物，但 OK=false")
	}
	if res.ItemID != 1003 || res.Count != 3 {
		t.Fatalf("拾取物错误: itemID=%d count=%d，期望 1003 x3", res.ItemID, res.Count)
	}
	if m.GetObject(id.ObjectIdType(701)) != nil {
		t.Fatalf("拾取后掉落物仍在地图上")
	}
	if m.GetObject(id.ObjectIdType(700)) == nil {
		t.Fatalf("范围外掉落物被误移除")
	}

	// 再次拾取：范围内已无可拾取 → OK=false
	if res2 := m.PickupNearest(playerID); res2.OK {
		t.Fatalf("范围内已无掉落物，应 OK=false，却拾到 itemID=%d", res2.ItemID)
	}
}
