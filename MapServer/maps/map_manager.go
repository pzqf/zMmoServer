package maps

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/maps/dungeon"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

// MapManager 地图管理器
// 负责管理多个地图实例
type MapManager struct {
	maps                *zMap.TypedMap[id.MapIdType, *Map]
	dungeonLifecycleMgr *dungeon.DungeonLifecycleManager
	nextInstanceMapID   int64 // 实例地图（副本/分线/战场/跨服）派生 mapID 的自增源，从 100000 起
	aoiNotifier         AOINotifier

	// 通用实例地图登记表（建议③）：key=派生 mapID。见 instance_manager.go。
	instMu    sync.Mutex
	instances map[id.MapIdType]*MapInstance

	// 热点图动态分线（建议②-b）：layerMgr=分配器（nil=未启用）；playerMap 记每个玩家实际所在 mapID
	// （分线玩家在派生层图，普通玩家在逻辑图），供 op 从"客户端传的逻辑 mapID"解析到真实地图。见 layer_manager.go。
	layerMgr    *LayerManager
	playerMapMu sync.Mutex
	playerMap   map[int64]id.MapIdType
}

// SetAOINotifier 注入 AOI 回程通知器，后续所创建的地图都会套用。
func (mm *MapManager) SetAOINotifier(n AOINotifier) {
	mm.aoiNotifier = n
	// 已创建的地图也补上
	mm.maps.Range(func(_ id.MapIdType, m *Map) bool {
		m.SetAOINotifier(n)
		return true
	})
}

// NewMapManager 创建新的地图管理器
func NewMapManager() *MapManager {
	return &MapManager{
		maps:              zMap.NewTypedMap[id.MapIdType, *Map](),
		nextInstanceMapID: 100000,
		instances:         make(map[id.MapIdType]*MapInstance),
		playerMap:         make(map[int64]id.MapIdType),
	}
}

func (mm *MapManager) SetDungeonLifecycleManager(dlm *dungeon.DungeonLifecycleManager) {
	mm.dungeonLifecycleMgr = dlm
}

// Start 启动地图管理器
func (mm *MapManager) Start() error {
	// 启动所有地图
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		maps = append(maps, value)
		return true
	})

	// 地图的各种系统在NewMap时已经初始化

	zLog.Info("MapManager started")
	return nil
}

// Stop 停止地图管理器
func (mm *MapManager) Stop() {
	// 停止所有地图
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		maps = append(maps, value)
		return true
	})

	for _, m := range maps {
		// 清理地图资源
		m.Cleanup()
	}

	zLog.Info("MapManager stopped")
}

// CreateMap 创建新地图
func (mm *MapManager) CreateMap(mapID id.MapIdType, mapConfigID int32, name string, width, height float32) *Map {
	// 检查地图是否已存在
	if existingMap, exists := mm.maps.Load(mapID); exists {
		zLog.Warn("Map already exists", zap.Int32("map_id", int32(mapID)))
		return existingMap
	}

	// 创建新地图
	newMap := NewMap(mapID, mapConfigID, name, width, height)
	newMap.SetAOINotifier(mm.aoiNotifier)
	mm.maps.Store(mapID, newMap)

	zLog.Info("Map created", zap.Int32("map_id", int32(mapID)), zap.String("name", name))

	// 初始化刷怪系统
	// newMap.InitSpawnSystem() // spawnManager is already initialized in NewMap

	return newMap
}

// CreateMapFromResource 从资源创建地图
func (mm *MapManager) CreateMapFromResource(resource *MapResource) *Map {
	mapID := id.MapIdType(resource.MapID)

	// 检查地图是否已存在
	if existingMap, exists := mm.maps.Load(mapID); exists {
		zLog.Warn("Map already exists", zap.Int32("map_id", int32(mapID)))
		return existingMap
	}

	// 创建新地图
	newMap := NewMap(mapID, resource.MapID, resource.Name, float32(resource.Width), float32(resource.Height))
	mm.maps.Store(mapID, newMap)

	// 设置地图属性
	newMap.SetMaxPlayers(resource.MaxPlayers)
	newMap.SetDescription(resource.Description)
	newMap.SetWeatherType(resource.WeatherType)
	newMap.SetMinLevel(resource.MinLevel)
	newMap.SetMaxLevel(resource.MaxLevel)

	// 加载传送点
	for _, tp := range resource.TeleportPoints {
		newMap.AddTeleportPointFromResource(tp.ID, float32(tp.X), float32(tp.Y), float32(tp.Z),
			id.MapIdType(tp.TargetMapID), float32(tp.TargetX), float32(tp.TargetY), float32(tp.TargetZ),
			tp.Name, tp.RequiredLevel, tp.RequiredItem, tp.IsActive)
	}

	// 加载建筑
	for _, building := range resource.Buildings {
		newMap.AddBuildingFromResource(building.ID, float32(building.X), float32(building.Y), float32(building.Z),
			float32(building.Width), float32(building.Height), building.Type, building.Name,
			building.Level, building.HP, building.Faction)
	}

	// 加载资源点
	for _, resourcePoint := range resource.Resources {
		newMap.AddResourceFromResource(resourcePoint.ResourceID, resourcePoint.Type,
			float32(resourcePoint.X), float32(resourcePoint.Y), float32(resourcePoint.Z),
			resourcePoint.RespawnTime, resourcePoint.ItemID, resourcePoint.Quantity,
			resourcePoint.Level, resourcePoint.IsGathering)
	}

	zLog.Info("Map created from resource", zap.Int32("map_id", resource.MapID), zap.String("name", resource.Name))

	return newMap
}

// GetMap 获取地图
func (mm *MapManager) GetMap(mapID id.MapIdType) *Map {
	m, _ := mm.maps.Load(mapID)
	return m
}

// UpdateAllMapsEvents 更新所有地图的事件
func (mm *MapManager) UpdateAllMapsEvents() {
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		maps = append(maps, value)
		return true
	})

	for _, m := range maps {
		// 处理地图事件
		m.UpdateEvents()
	}
}

// HandlePlayerEnterMap 处理玩家进入地图
func (mm *MapManager) HandlePlayerEnterMap(playerID int64, mapID int64, x, y, z float32) error {
	var m *Map
	actualMapID := id.MapIdType(mapID)
	// 可分线逻辑图：经分配器选/建一层，玩家实际进的是派生层图（对客户端/GameServer 透明，②-b）。
	if mm.layerMgr != nil && mm.layerMgr.IsLayerable(id.MapIdType(mapID)) {
		layerMap, layerMapID, ok := mm.layerMgr.AllocateLayer(id.MapIdType(mapID), 0) // 亲和=0（组队同层待接组队系统）
		if !ok {
			return fmt.Errorf("allocate layer for map %d failed", mapID)
		}
		m, actualMapID = layerMap, layerMapID
	} else {
		m = mm.GetMap(id.MapIdType(mapID))
	}
	if m == nil {
		return fmt.Errorf("map not found: %d", mapID)
	}

	if err := m.AddPlayer(id.PlayerIdType(playerID), id.ObjectIdType(playerID), x, y, z); err != nil {
		return err
	}
	mm.setPlayerMap(playerID, actualMapID) // 记玩家实际所在地图，供后续 op resolve
	// 测试用掉落种子：map 1001 无怪、无真实掉落，故用 ZMMO_TEST_LOOT 在玩家落点旁放一件可拾取物，
	// 供拾取闭环 E2E。生产路径掉落来自 combat 击杀（见 handleMonsterDeath）。
	if os.Getenv("ZMMO_TEST_LOOT") == "1" {
		m.DropTestLoot(x, y, z)
	}
	return nil
}

// HandlePlayerPickup 处理玩家就近拾取（GameServer → MapServer）。地图侧权威判定并移除掉落物，
// 返回拾到的 itemID/count；上层据此把 grant 推回 GameServer 落背包。
func (mm *MapManager) HandlePlayerPickup(playerID, mapID int64) (int32, int32, bool) {
	m := mm.resolveMap(playerID, mapID)
	if m == nil {
		return 0, 0, false
	}
	r := m.PickupNearest(id.PlayerIdType(playerID))
	return r.ItemID, r.Count, r.OK
}

// HandlePlayerLeaveMap 处理玩家离开地图
func (mm *MapManager) HandlePlayerLeaveMap(playerID int64, mapID int64) error {
	m := mm.resolveMap(playerID, mapID)
	if m == nil {
		return fmt.Errorf("map not found: %d", mapID)
	}

	m.RemovePlayer(id.PlayerIdType(playerID))
	mm.clearPlayerMap(playerID) // 玩家离图，清实际地图记录（②-b）
	return nil
}

// HandlePlayerMove 处理玩家移动
func (mm *MapManager) HandlePlayerMove(playerID, objectID, mapID int64, x, y, z float32) error {
	m := mm.resolveMap(playerID, mapID)
	if m == nil {
		return fmt.Errorf("map not found: %d", mapID)
	}

	return m.MovePlayer(id.PlayerIdType(playerID), id.ObjectIdType(objectID), x, y, z)
}

// HandlePlayerAttack 处理玩家攻击
func (mm *MapManager) HandlePlayerAttack(playerID, objectID, mapID, targetID int64) (int64, int64, error) {
	m := mm.resolveMap(playerID, mapID)
	if m == nil {
		return 0, 0, fmt.Errorf("map not found: %d", mapID)
	}

	return m.AttackTarget(id.PlayerIdType(playerID), id.ObjectIdType(objectID), id.ObjectIdType(targetID))
}

// UpdateAllMapsSkills 更新所有地图的技能
func (mm *MapManager) UpdateAllMapsSkills() {
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		maps = append(maps, value)
		return true
	})

	for _, m := range maps {
		m.UpdateSkills()
	}
}

// PostTickAll 向每张地图的单写者 goroutine 投递一帧更新（MAP-2）。
// 取代此前由游戏主循环直接调用各 UpdateAllMaps* 的做法——那样帧更新跑在主循环 goroutine 上，
// 与网络分发 goroutine 并发改动同一批对象，构成数据竞争。改为投递到每张地图各自的 goroutine，
// 使帧更新与网络命令在该地图内严格串行。队列满则合帧丢弃，见 map_actor.go。
func (mm *MapManager) PostTickAll(deltaTime time.Duration) {
	mm.maps.Range(func(_ id.MapIdType, m *Map) bool {
		m.postTick(deltaTime)
		return true
	})
}

// ⚠ 以下 UpdateAllMaps{AI,Buffs,Players,Skills,Events} 均为 MAP-2 前的旧路径，现已无任何调用方
// （游戏主循环只用 PostTickAll）。**切勿把它们重新接回主循环**：它们在主循环 goroutine 上直接跑
// AIManager.Update，会与 AddObject 形成 m.mu↔am.mu 的锁序反转——目前二者都只在地图 actor
// goroutine 上串行执行才安全，一旦从主循环并发调用即死锁。保留仅为兼容旧调用签名。
//
// UpdateAllMapsAI 更新所有地图的AI（DEAD，勿用，见上）
func (mm *MapManager) UpdateAllMapsAI(deltaTime time.Duration) {
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		maps = append(maps, value)
		return true
	})

	for _, m := range maps {
		if m.aiManager != nil {
			m.aiManager.Update(deltaTime)
		}
	}
}

// UpdateAllMapsBuffs 更新所有地图的Buff
func (mm *MapManager) UpdateAllMapsBuffs(deltaTime time.Duration) {
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		maps = append(maps, value)
		return true
	})

	for _, m := range maps {
		if m.buffManager != nil {
			m.buffManager.Update(deltaTime)
		}
	}
}

// UpdateAllMapsPlayers 更新所有地图的玩家状态
func (mm *MapManager) UpdateAllMapsPlayers() {
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		maps = append(maps, value)
		return true
	})

	for _, m := range maps {
		m.UpdatePlayers()
	}
}

// UpdateAllMapsDungeons 更新所有地图的副本
func (mm *MapManager) UpdateAllMapsDungeons(deltaTime time.Duration) {
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		maps = append(maps, value)
		return true
	})

	for _, m := range maps {
		if m.dungeonManager != nil {
			m.dungeonManager.Update(deltaTime)
		}
	}
}

// RemoveMap 移除地图
func (mm *MapManager) RemoveMap(mapID id.MapIdType) {
	m, exists := mm.maps.Load(mapID)
	if !exists {
		return
	}

	// 停止刷新管理器
	if m.spawnManager != nil {
		m.spawnManager.Stop()
	}

	mm.maps.Delete(mapID)
	zLog.Info("Map removed", zap.Int32("map_id", int32(mapID)))
}

// GetAllMaps 获取所有地图
func (mm *MapManager) GetAllMaps() []*Map {
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		maps = append(maps, value)
		return true
	})

	return maps
}

// GetMapCount 获取地图数量
func (mm *MapManager) GetMapCount() int {
	count := 0
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		count++
		return true
	})
	return count
}

// GetAllMapIDs 获取所有地图ID
func (mm *MapManager) GetAllMapIDs() []int32 {
	var mapIDs []int32
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		mapIDs = append(mapIDs, int32(key))
		return true
	})
	return mapIDs
}

func (mm *MapManager) CreateDungeonMap(dungeonID id.DungeonIdType, players []id.PlayerIdType) (*Map, *dungeon.DungeonInstance, error) {
	// 地图级生命周期统一走通用实例路径（建议③）；副本的玩法生命周期仍由 dungeonLifecycleMgr 管。
	dungeonMap, dungeonMapID := mm.CreateInstance(InstanceKindDungeon, id.MapIdType(dungeonID), int32(dungeonID),
		fmt.Sprintf("Dungeon_%d", dungeonID), 500, 500)
	dungeonMap.SetIsDungeon(true)

	if mm.dungeonLifecycleMgr == nil {
		dm := dungeon.NewDungeonManager()
		mm.dungeonLifecycleMgr = dungeon.NewDungeonLifecycleManager(dm)
	}

	instance, err := mm.dungeonLifecycleMgr.CreateAndStartDungeon(dungeonID, players, dungeonMapID)
	if err != nil {
		mm.DestroyInstance(dungeonMapID)
		return nil, nil, fmt.Errorf("create dungeon lifecycle: %w", err)
	}

	dungeonMap.SetDungeonInstanceID(instance.InstanceID)

	zLog.Info("Dungeon map created",
		zap.Int32("dungeon_id", int32(dungeonID)),
		zap.Int32("map_id", int32(dungeonMapID)),
		zap.Int64("instance_id", int64(instance.InstanceID)),
		zap.Int("player_count", len(players)))

	return dungeonMap, instance, nil
}

func (mm *MapManager) DestroyDungeonMap(instanceID id.InstanceIdType) error {
	if mm.dungeonLifecycleMgr == nil {
		return fmt.Errorf("dungeon lifecycle manager not initialized")
	}

	lifecycle, exists := mm.dungeonLifecycleMgr.GetLifecycle(instanceID)
	if !exists {
		return fmt.Errorf("dungeon lifecycle not found: %d", instanceID)
	}

	mapID := lifecycle.MapID

	if err := mm.dungeonLifecycleMgr.DestroyDungeon(instanceID); err != nil {
		return fmt.Errorf("destroy dungeon: %w", err)
	}

	mm.DestroyInstance(mapID) // 统一走通用实例销毁（Cleanup + 摘除 + 注销）

	zLog.Info("Dungeon map destroyed",
		zap.Int32("map_id", int32(mapID)),
		zap.Int64("instance_id", int64(instanceID)))

	return nil
}

// CreateCrossServerMap 建一张跨服临时实例地图（realm §5.1）。经 ③ 的通用实例路径登记为
// InstanceKindCrossServer——于是它享有统一实例生命周期：空置由 ReapEmpty 自动回收（打完人走即销毁）。
// 跨服玩家来自多个归属服，其战斗/结算的**持久变更**由 ④ 的 AttrGrant 按玩家归属(PlayerGameServerManager)
// 回流到各自的家服落库（跨服实例只产事件、不碰持久资产）。
func (mm *MapManager) CreateCrossServerMap(mapConfigID int32, name string, width, height float32, mode MapMode, serverGroupID int32) *Map {
	crossMap, crossMapID := mm.CreateInstance(InstanceKindCrossServer, id.MapIdType(mapConfigID), mapConfigID, name, width, height)
	crossMap.SetMapMode(mode) // 覆盖 CreateInstance 默认的 SingleServer
	crossMap.SetServerGroupID(serverGroupID)

	zLog.Info("Cross-server map created (tracked instance)",
		zap.Int32("map_config_id", mapConfigID),
		zap.Int32("map_id", int32(crossMapID)),
		zap.Int("mode", int(mode)),
		zap.Int32("server_group_id", serverGroupID))

	return crossMap
}

func (mm *MapManager) GetDungeonLifecycleManager() *dungeon.DungeonLifecycleManager {
	return mm.dungeonLifecycleMgr
}

func (mm *MapManager) GetDungeonMapByInstanceID(instanceID id.InstanceIdType) *Map {
	if mm.dungeonLifecycleMgr == nil {
		return nil
	}

	lifecycle, exists := mm.dungeonLifecycleMgr.GetLifecycle(instanceID)
	if !exists {
		return nil
	}

	m, _ := mm.maps.Load(lifecycle.MapID)
	return m
}

func (mm *MapManager) UpdateDungeonLifecycles(deltaTime time.Duration) {
	if mm.dungeonLifecycleMgr != nil {
		mm.dungeonLifecycleMgr.Update(deltaTime)
	}
}

func (mm *MapManager) GetCrossServerMaps() []*Map {
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		if value.IsCrossServer() {
			maps = append(maps, value)
		}
		return true
	})
	return maps
}

func (mm *MapManager) GetDungeonMaps() []*Map {
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		if value.IsDungeon() {
			maps = append(maps, value)
		}
		return true
	})
	return maps
}
