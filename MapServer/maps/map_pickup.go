package maps

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
)

// pickupRadius 就近拾取半径：玩家该范围内最近的可拾取掉落物会被拾走。
const pickupRadius = 5.0

// PickupResult 一次拾取的结果。OK=false 表示范围内无可拾取物。
type PickupResult struct {
	ItemID int32
	Count  int32
	OK     bool
}

// PickupNearest 玩家就近拾取一件掉落物（网络入口，经 m.Do 串行到地图 goroutine——MAP-2）。
// 取玩家 pickupRadius 内最近的可拾取 Item：标记已拾取 + 从地图移除（RemoveObject 经 AOI
// 触发离开视野，客户端侧掉落物随之消失）。返回拾到的 itemID/count；无可拾取时 OK=false。
// 物品权威在地图侧——是否移除由这里判定；随后由上层把 grant 推给 GameServer 落背包。
func (m *Map) PickupNearest(playerID id.PlayerIdType) PickupResult {
	var res PickupResult
	m.Do(func() {
		player := m.GetObject(id.ObjectIdType(playerID))
		if player == nil {
			return
		}
		ppos := player.GetPosition()

		var nearest *object.Item
		var best float32
		for _, obj := range m.GetObjectsInRange(ppos, pickupRadius) {
			if obj.GetType() != common.GameObjectTypeItem {
				continue
			}
			it, ok := obj.(*object.Item)
			if !ok || !it.CanBePicked() {
				continue
			}
			d := ppos.DistanceTo(obj.GetPosition())
			if nearest == nil || d < best {
				nearest, best = it, d
			}
		}
		if nearest == nil {
			return
		}
		nearest.SetPicked(true)
		m.RemoveObject(nearest.GetID())
		res = PickupResult{ItemID: nearest.GetItemID(), Count: nearest.GetQuantity(), OK: true}
	})
	return res
}

// DropTestLoot 仅测试用（ZMMO_TEST_LOOT=1）：在给定位置旁放一件可拾取测试掉落物，
// 供拾取闭环 E2E 使用（map 1001 无怪、无真实掉落）。生产路径的掉落来自 combat 击杀。
func (m *Map) DropTestLoot(x, y, z float32) {
	m.Do(func() {
		it := object.NewItem(nextMapObjectID(), 1003, "TestLoot",
			common.Vector3{X: x + 1, Y: y, Z: z + 1}, 1,
			object.ItemTypeConsumable, object.ItemRarityCommon)
		m.AddObject(it)
	})
}
