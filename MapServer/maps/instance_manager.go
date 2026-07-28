package maps

import (
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// —— 通用实例地图生命周期（realm 架构建议③）——
//
// "实例地图" = 副本 / 分线(layer) / 战场 / 跨服临时实例：都是一张**派生 mapID 的独立 *Map**
// （独立单写者 goroutine + 独立 AOI + 独立 tick）。地图级的"建/销/空置回收/在途预留"由 zEngine 的
// zInstance.Pool（通用实例池）承载（见 map_manager.go 的 instPool）；本文件是它在地图域的接线：
//   - 按逻辑图分组（Pool 的 key）；派生 mapID = 池内部 id；
//   - 副本(Dungeon)=pinned，不空置回收、由玩法生命周期显式 DestroyInstance；分线/战场/跨服=可回收；
//   - 分线的"选层+在途预留额度"复用 Pool.Acquire（见 layer_manager.go）。
// 副本的**玩法**生命周期(波次/奖励)仍留在 dungeon 包，只是地图的建/销/回收统一走这里。

// InstanceKind 实例地图种类。
type InstanceKind int

const (
	InstanceKindDungeon      InstanceKind = iota + 1 // 副本：显式销毁 / 由 dungeon 玩法生命周期管，不空置自动回收
	InstanceKindLayer                                // 分线：开放世界热点图的平行副本，空置自动回收
	InstanceKindBattleground                         // 战场：匹配对局，空置自动回收
	InstanceKindCrossServer                          // 跨服临时实例，空置自动回收
)

func (k InstanceKind) String() string {
	switch k {
	case InstanceKindDungeon:
		return "dungeon"
	case InstanceKindLayer:
		return "layer"
	case InstanceKindBattleground:
		return "battleground"
	case InstanceKindCrossServer:
		return "crossserver"
	default:
		return "unknown"
	}
}

// MapInstance 一个实例地图的生命周期记录。实现 zInstance.Instance：Occupancy=玩家数、Close=地图 Cleanup。
type MapInstance struct {
	MapID        id.MapIdType // 派生的物理实例 mapID（= 池内部 id / 身份 / mm.maps 键）
	LogicalMapID id.MapIdType // 逻辑地图（副本=dungeonID 语义 / 分线=逻辑图 ID）
	Kind         InstanceKind
	Map          *Map
	CreatedAt    time.Time
}

// Occupancy 当前实例内玩家数（供池判空置回收）。
func (mi *MapInstance) Occupancy() int { return mi.Map.GetPlayerCount() }

// Close 释放实例地图（先 StopActor 再停 spawn，防 goroutine 泄漏）。由池在回收/销毁时调用。
func (mi *MapInstance) Close() { mi.Map.Cleanup() }

// newMapInstance 建一张派生实例地图（供 Pool 的 build 闭包调用；newID 即派生 mapID）。
func (mm *MapManager) newMapInstance(newID uint64, kind InstanceKind, logicalMapID id.MapIdType, mapConfigID int32, name string, width, height float32) *MapInstance {
	mapID := id.MapIdType(newID)
	m := NewMap(mapID, mapConfigID, name, width, height)
	m.SetMapMode(MapModeSingleServer)
	if mm.aoiNotifier != nil {
		m.SetAOINotifier(mm.aoiNotifier)
	}
	mm.maps.Store(mapID, m)
	return &MapInstance{
		MapID:        mapID,
		LogicalMapID: logicalMapID,
		Kind:         kind,
		Map:          m,
		CreatedAt:    time.Now(),
	}
}

// CreateInstance 通用实例地图创建：经池登记一张新实例地图（Dungeon=pinned 不回收）。
// 返回承载的 *Map 与其派生 mapID。副本以外的种类空置会被 ReapEmpty 自动回收。
func (mm *MapManager) CreateInstance(kind InstanceKind, logicalMapID id.MapIdType, mapConfigID int32, name string, width, height float32) (*Map, id.MapIdType) {
	_, inst := mm.instPool.Add(logicalMapID, kind == InstanceKindDungeon, func(newID uint64) *MapInstance {
		return mm.newMapInstance(newID, kind, logicalMapID, mapConfigID, name, width, height)
	})
	zLog.Info("Map instance created",
		zap.String("kind", kind.String()),
		zap.Int32("map_id", int32(inst.MapID)),
		zap.Int32("logical_map_id", int32(logicalMapID)))
	return inst.Map, inst.MapID
}

// DestroyInstance 通用实例地图销毁（幂等）：经池摘除 → onEvict(从 mm.maps 删) → Close(Cleanup=先 StopActor 再停 spawn)。
func (mm *MapManager) DestroyInstance(mapID id.MapIdType) {
	inst, ok := mm.instPool.Get(uint64(mapID))
	if !ok {
		return
	}
	kind := inst.Kind
	mm.instPool.Destroy(uint64(mapID))
	zLog.Info("Map instance destroyed",
		zap.String("kind", kind.String()),
		zap.Int32("map_id", int32(mapID)))
}

// ReapEmpty 回收"空置持续超过 grace"的自动回收实例（分线/战场/跨服；副本=pinned 不在此列）。
// ⚠ 必须由驱动 PostTickAll 的**同一 goroutine**周期调用——销毁会 Cleanup/StopActor 地图，与该 goroutine
// 串行才能与 tick 投递互不干扰。返回本次回收的实例数。
func (mm *MapManager) ReapEmpty(grace time.Duration) int {
	return mm.instPool.Reap(grace)
}

// GetInstance 查一个实例地图的生命周期记录。
func (mm *MapManager) GetInstance(mapID id.MapIdType) (*MapInstance, bool) {
	return mm.instPool.Get(uint64(mapID))
}

// InstanceCount 当前登记的实例地图数（含副本）。
func (mm *MapManager) InstanceCount() int {
	return mm.instPool.Count()
}

// GetInstancesByLogical 返回某逻辑地图下、指定种类的所有存活实例（供分线分配器/查询）。
func (mm *MapManager) GetInstancesByLogical(logicalMapID id.MapIdType, kind InstanceKind) []*MapInstance {
	var out []*MapInstance
	mm.instPool.RangeByKey(logicalMapID, func(_ uint64, inst *MapInstance) bool {
		if inst.Kind == kind {
			out = append(out, inst)
		}
		return true
	})
	return out
}

// ReleaseLayerReservation 玩家 AddPlayer 完成（成功计入 count / 或失败）后归还 1 个在途预留额度。
// 委托池的 Release（分线 AllocateLayer 经 Pool.Acquire 占的预留，见 layer_manager.go）。
func (mm *MapManager) ReleaseLayerReservation(mapID id.MapIdType) {
	mm.instPool.Release(uint64(mapID))
}
