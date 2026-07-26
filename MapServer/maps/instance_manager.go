package maps

import (
	"sync/atomic"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// —— 通用实例地图生命周期（realm 架构建议③）——
//
// "实例地图" = 副本 / 分线(layer) / 战场 / 跨服临时实例：都是一张**派生 mapID 的独立 *Map**
// （独立单写者 goroutine + 独立 AOI + 独立 tick）。此前只有副本(dungeon)一条私有的创建/销毁路径，
// 本文件把"地图级"的实例生命周期抽成 MapManager 的通用能力，供分线/跨服实例复用；副本的**玩法**
// 生命周期(波次/奖励)仍留在 dungeon 包，只是地图的建/销/回收统一走这里。
//
// 身份键 = 派生的物理 mapID（每个实例唯一）。副本另有自己的玩法 InstanceID，二者不冲突。

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

// MapInstance 一个实例地图的通用生命周期记录。
type MapInstance struct {
	MapID             id.MapIdType // 派生的物理实例 mapID（registry key / 身份）
	LogicalMapID      id.MapIdType // 逻辑地图（副本=dungeonID 语义 / 分线=逻辑图 ID）
	Kind              InstanceKind
	Map               *Map
	CreatedAt         time.Time
	autoReapWhenEmpty bool      // 空置自动回收（分线/战场/跨服=true；副本=false）
	emptySince        time.Time // 变空的时刻（零值=当前非空 / 尚未开始计时）
}

// CreateInstance 通用实例地图创建：分配派生 mapID、建独立 *Map、套 AOI 通知器、登记。
// 返回承载的 *Map 与其派生 mapID。副本以外的种类空置会被 ReapEmpty 自动回收。
func (mm *MapManager) CreateInstance(kind InstanceKind, logicalMapID id.MapIdType, mapConfigID int32, name string, width, height float32) (*Map, id.MapIdType) {
	mapID := id.MapIdType(atomic.AddInt64(&mm.nextInstanceMapID, 1))

	m := NewMap(mapID, mapConfigID, name, width, height)
	m.SetMapMode(MapModeSingleServer)
	if mm.aoiNotifier != nil {
		m.SetAOINotifier(mm.aoiNotifier)
	}
	mm.maps.Store(mapID, m)

	mm.instMu.Lock()
	mm.instances[mapID] = &MapInstance{
		MapID:             mapID,
		LogicalMapID:      logicalMapID,
		Kind:              kind,
		Map:               m,
		CreatedAt:         time.Now(),
		autoReapWhenEmpty: kind != InstanceKindDungeon,
	}
	mm.instMu.Unlock()

	zLog.Info("Map instance created",
		zap.String("kind", kind.String()),
		zap.Int32("map_id", int32(mapID)),
		zap.Int32("logical_map_id", int32(logicalMapID)))
	return m, mapID
}

// DestroyInstance 通用实例地图销毁：干净停 goroutine（Cleanup=先 StopActor 再停 spawn）、从地图表摘除、注销。幂等。
func (mm *MapManager) DestroyInstance(mapID id.MapIdType) {
	mm.instMu.Lock()
	inst, ok := mm.instances[mapID]
	delete(mm.instances, mapID)
	mm.instMu.Unlock()

	if m, exists := mm.maps.Load(mapID); exists {
		m.Cleanup()
		mm.maps.Delete(mapID)
	}
	if ok {
		zLog.Info("Map instance destroyed",
			zap.String("kind", inst.Kind.String()),
			zap.Int32("map_id", int32(mapID)))
	}
}

// ReapEmpty 回收"空置持续超过 grace"的自动回收实例（分线/战场/跨服）；副本(autoReapWhenEmpty=false)不在此列。
// ⚠ 必须由驱动 PostTickAll 的**同一 goroutine**周期调用——销毁会 Cleanup/StopActor 地图，与该 goroutine
// 串行才能与 tick 投递互不干扰。返回本次回收的实例数。
func (mm *MapManager) ReapEmpty(grace time.Duration) int {
	now := time.Now()
	var toReap []id.MapIdType

	mm.instMu.Lock()
	for mapID, inst := range mm.instances {
		if !inst.autoReapWhenEmpty {
			continue
		}
		if inst.Map.GetPlayerCount() > 0 {
			inst.emptySince = time.Time{} // 非空，清空计时
			continue
		}
		if inst.emptySince.IsZero() {
			inst.emptySince = now // 刚变空，开始计时
			continue
		}
		if now.Sub(inst.emptySince) >= grace {
			toReap = append(toReap, mapID)
		}
	}
	mm.instMu.Unlock()

	for _, mapID := range toReap {
		mm.DestroyInstance(mapID)
	}
	return len(toReap)
}

// GetInstance 查一个实例地图的生命周期记录。
func (mm *MapManager) GetInstance(mapID id.MapIdType) (*MapInstance, bool) {
	mm.instMu.Lock()
	defer mm.instMu.Unlock()
	inst, ok := mm.instances[mapID]
	return inst, ok
}

// InstanceCount 当前登记的实例地图数（含副本）。
func (mm *MapManager) InstanceCount() int {
	mm.instMu.Lock()
	defer mm.instMu.Unlock()
	return len(mm.instances)
}

// GetInstancesByLogical 返回某逻辑地图下、指定种类的所有存活实例快照（供分线分配器选层，建议②）。
func (mm *MapManager) GetInstancesByLogical(logicalMapID id.MapIdType, kind InstanceKind) []*MapInstance {
	mm.instMu.Lock()
	defer mm.instMu.Unlock()
	var out []*MapInstance
	for _, inst := range mm.instances {
		if inst.LogicalMapID == logicalMapID && inst.Kind == kind {
			out = append(out, inst)
		}
	}
	return out
}
