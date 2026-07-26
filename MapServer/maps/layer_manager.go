package maps

import (
	"sync"

	"github.com/pzqf/zCommon/common/id"
)

// —— 热点图动态分线（Layering，realm 建议②）——
//
// 一张"开放世界"逻辑地图爆满时，开多份平行副本（层/line），玩家分摊进去、跨层互不可见——治"每张 map
// 单 goroutine、一核封顶"的天花板。**纯单服内伸缩，不涉及跨服。** 层 = InstanceKindLayer 的实例地图
// （复用 ③ 的 CreateInstance/ReapEmpty：空层由 ReapEmpty 自动回收）。本文件是"层分配器"：进图时选/建一层。
//
// 分配策略（AllocateLayer）：
//  1. 亲和：若给了 affinity 层且其人数 < hardCap，进该层（同队/组队目标同层）。
//  2. 最少人数：在人数 < softCap 的层里选最空的。
//  3. 都不满足（无层 < softCap）：开新层。
// 即"填到 softCap 就开新层；亲和可挤到 hardCap"。

// LayerConfig 一张可分线逻辑地图的分线参数。
type LayerConfig struct {
	MapConfigID int32
	Name        string
	Width       float32
	Height      float32
	SoftCap     int // 单层软上限：到达后新玩家开新层（除非有亲和）
	HardCap     int // 单层硬上限：亲和也不能超过
}

// LayerManager 分线分配器。依附 MapManager（层实例经其 CreateInstance 创建、GetInstancesByLogical 查询、
// ReapEmpty 空置回收）。
type LayerManager struct {
	mm  *MapManager
	mu  sync.Mutex
	cfg map[id.MapIdType]LayerConfig // logicalMapID → 分线配置
}

func NewLayerManager(mm *MapManager) *LayerManager {
	return &LayerManager{mm: mm, cfg: make(map[id.MapIdType]LayerConfig)}
}

// —— MapManager 侧的分线接线（②-b）——

// EnableLayering 启用分线（惰性创建分配器）。幂等。返回分配器供注册可分线图。
func (mm *MapManager) EnableLayering() *LayerManager {
	if mm.layerMgr == nil {
		mm.layerMgr = NewLayerManager(mm)
	}
	return mm.layerMgr
}

// resolveMap 把"客户端传的逻辑 mapID"解析到玩家实际所在地图：分线玩家在派生层图，普通玩家在逻辑图。
// 进图后各 op（移动/攻击/拾取/离开）都经它定位，使分线对 GameServer 路由透明（仍按逻辑 mapID 路由到本 MapServer）。
func (mm *MapManager) resolveMap(playerID int64, mapID int64) *Map {
	mm.playerMapMu.Lock()
	actual, ok := mm.playerMap[playerID]
	mm.playerMapMu.Unlock()
	if ok {
		return mm.GetMap(actual)
	}
	return mm.GetMap(id.MapIdType(mapID))
}

func (mm *MapManager) setPlayerMap(playerID int64, mapID id.MapIdType) {
	mm.playerMapMu.Lock()
	mm.playerMap[playerID] = mapID
	mm.playerMapMu.Unlock()
}

func (mm *MapManager) clearPlayerMap(playerID int64) {
	mm.playerMapMu.Lock()
	delete(mm.playerMap, playerID)
	mm.playerMapMu.Unlock()
}

// RegisterLayerable 声明某逻辑地图可分线及其参数。未注册的逻辑地图 = 不可分线（走普通单图）。
func (lm *LayerManager) RegisterLayerable(logicalMapID id.MapIdType, cfg LayerConfig) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.cfg[logicalMapID] = cfg
}

// IsLayerable 该逻辑地图是否可分线。
func (lm *LayerManager) IsLayerable(logicalMapID id.MapIdType) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	_, ok := lm.cfg[logicalMapID]
	return ok
}

// AllocateLayer 为进入 logicalMapID 的玩家选/建一层。affinityLayerMapID != 0 表示优先进该层（同队同层）。
// 返回选中层的 *Map 与其 layerMapID；非可分线逻辑地图返回 (nil, 0, false)。
// 注意：只负责"选/建层"，不负责把玩家加进去——由调用方拿到 *Map 后 AddPlayer。
// 全程持锁串行化同一 LayerManager 的分配，避免并发进图重复开层。
func (lm *LayerManager) AllocateLayer(logicalMapID, affinityLayerMapID id.MapIdType) (*Map, id.MapIdType, bool) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	cfg, ok := lm.cfg[logicalMapID]
	if !ok {
		return nil, 0, false
	}

	// 1&2. 在既有层里按 cap（计入在途预留，防并发进图击穿 softCap/hardCap = M1 TOCTOU）选一层：
	//      亲和优先 → 否则 effective 人数最少且 < softCap。选中即占 1 个在途名额（reserved++）。
	if m, layerMapID, ok := lm.mm.reserveExistingLayer(logicalMapID, affinityLayerMapID, cfg.SoftCap, cfg.HardCap); ok {
		return m, layerMapID, true
	}

	// 3. 无可用层 → 开新层，并即刻预留 1 个在途名额：使"新建层→玩家 AddPlayer"窗口内该层被视为非空、
	//    不被 ReapEmpty 回收、也不被并发分配当空层挤入（M2 空层回收竞态）。
	m, layerMapID := lm.mm.CreateInstance(InstanceKindLayer, logicalMapID, cfg.MapConfigID, cfg.Name, cfg.Width, cfg.Height)
	lm.mm.reserveInstance(layerMapID)
	return m, layerMapID, true
}

// 调用方（HandlePlayerEnterMap）在 AddPlayer 完成后必须 ReleaseLayerReservation(layerMapID) 归还名额，
// 无论成功或失败——否则该层 reserved 永不归零、cap 被永久占用、且永不被 ReapEmpty 回收。
