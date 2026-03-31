package maps

import (
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"go.uber.org/zap"
)

// SpawnManager 刷新管理器
// 负责管理地图中的怪物和其他游戏对象的刷新
type SpawnManager struct {
	mu          sync.RWMutex
	mapID       id.MapIdType
	mapObj      *Map
	spawnPoints []*SpawnPoint
	running     bool
}

// SpawnPoint 刷新点
type SpawnPoint struct {
	spawnID      int32
	spawnType    string
	objectID     int32
	position     common.Vector3
	respawnTime  int
	maxCount     int
	currentCount int
	active       bool
	lastSpawn    time.Time
}

// NewSpawnManager 创建新的刷新管理器
func NewSpawnManager(mapID id.MapIdType, mapObj *Map) *SpawnManager {
	return &SpawnManager{
		mapID:       mapID,
		mapObj:      mapObj,
		spawnPoints: make([]*SpawnPoint, 0),
		running:     false,
	}
}

// Init 初始化刷新管理器
func (sm *SpawnManager) Init(mapConfigID int32) {
	zLog.Info("Initializing spawn manager", zap.Int32("map_id", int32(sm.mapID)))

	// 模拟从配置表加载刷新点
	// 实际应用中，应该从配置文件或数据库加载
	spawnPoints := []struct {
		ID          int32
		SpawnType   string
		SpawnID     int32
		PositionX   float32
		PositionY   float32
		PositionZ   float32
		RespawnTime int
		MaxCount    int
		Active      bool
	}{
		{1, "monster", 101, 100, 100, 0, 30, 5, true},
		{2, "monster", 102, 200, 200, 0, 45, 3, true},
		{3, "npc", 201, 300, 300, 0, 0, 1, true},
		{4, "item", 301, 400, 400, 0, 60, 2, true},
	}

	for _, spawnData := range spawnPoints {
		spawnPoint := &SpawnPoint{
			spawnID:   spawnData.ID,
			spawnType: spawnData.SpawnType,
			objectID:  spawnData.SpawnID,
			position: common.Vector3{
				X: spawnData.PositionX,
				Y: spawnData.PositionY,
				Z: spawnData.PositionZ,
			},
			respawnTime:  spawnData.RespawnTime,
			maxCount:     spawnData.MaxCount,
			currentCount: 0,
			active:       spawnData.Active,
			lastSpawn:    time.Now(),
		}
		sm.spawnPoints = append(sm.spawnPoints, spawnPoint)
	}

	zLog.Info("Spawn manager initialized", zap.Int32("map_id", int32(sm.mapID)), zap.Int("spawn_points", len(sm.spawnPoints)))

	// 开始刷新循环
	sm.running = true
	go sm.spawnLoop()
}

// spawnLoop 刷新循环
func (sm *SpawnManager) spawnLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for sm.running {
		select {
		case <-ticker.C:
			sm.checkSpawnPoints()
		}
	}
}

// checkSpawnPoints 检查刷新点
func (sm *SpawnManager) checkSpawnPoints() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()

	for _, spawnPoint := range sm.spawnPoints {
		if !spawnPoint.active {
			continue
		}

		if spawnPoint.currentCount < spawnPoint.maxCount {
			elapsed := now.Sub(spawnPoint.lastSpawn).Seconds()
			if elapsed >= float64(spawnPoint.respawnTime) {
				sm.spawnObject(spawnPoint)
				spawnPoint.lastSpawn = now
			}
		}
	}
}

// spawnObject 刷新游戏对象
func (sm *SpawnManager) spawnObject(spawnPoint *SpawnPoint) {
	// 根据spawnType创建不同类型的游戏对象
	switch spawnPoint.spawnType {
	case "monster":
		sm.spawnMonster(spawnPoint)
	case "npc":
		sm.spawnNPC(spawnPoint)
	case "item":
		sm.spawnItem(spawnPoint)
	default:
		zLog.Warn("Unknown spawn type", zap.String("type", spawnPoint.spawnType))
	}
}

// spawnMonster 刷新怪物
func (sm *SpawnManager) spawnMonster(spawnPoint *SpawnPoint) {
	// 生成唯一的对象ID
	objectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)

	// 创建怪物对象
	monster := object.NewMonster(objectID, spawnPoint.objectID, "Monster_"+string(rune('A'+spawnPoint.objectID%26)), spawnPoint.position, 1)

	// 添加到地图
	sm.mapObj.AddObject(monster)

	zLog.Debug("Spawning monster",
		zap.Int32("monster_id", spawnPoint.objectID),
		zap.Int64("object_id", int64(objectID)),
		zap.Float32("x", spawnPoint.position.X),
		zap.Float32("y", spawnPoint.position.Y))

	spawnPoint.currentCount++
}

// spawnNPC 刷新NPC
func (sm *SpawnManager) spawnNPC(spawnPoint *SpawnPoint) {
	// 生成唯一的对象ID
	objectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)

	// 创建NPC对象
	npc := object.NewNPC(objectID, spawnPoint.objectID, "NPC_"+string(rune('A'+spawnPoint.objectID%26)), spawnPoint.position, "Hello, adventurer!")

	// 添加到地图
	sm.mapObj.AddObject(npc)

	zLog.Debug("Spawning NPC",
		zap.Int32("npc_id", spawnPoint.objectID),
		zap.Int64("object_id", int64(objectID)),
		zap.Float32("x", spawnPoint.position.X),
		zap.Float32("y", spawnPoint.position.Y))

	spawnPoint.currentCount++
}

// spawnItem 刷新物品
func (sm *SpawnManager) spawnItem(spawnPoint *SpawnPoint) {
	// 生成唯一的对象ID
	objectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)

	// 创建物品对象
	item := object.NewItem(objectID, spawnPoint.objectID, "Item_"+string(rune('A'+spawnPoint.objectID%26)), spawnPoint.position, 1, object.ItemTypeMaterial, object.ItemRarityCommon)

	// 添加到地图
	sm.mapObj.AddObject(item)

	zLog.Debug("Spawning item",
		zap.Int32("item_id", spawnPoint.objectID),
		zap.Int64("object_id", int64(objectID)),
		zap.Float32("x", spawnPoint.position.X),
		zap.Float32("y", spawnPoint.position.Y))

	spawnPoint.currentCount++
}

// RemoveObject 移除游戏对象
func (sm *SpawnManager) RemoveObject(objectID id.ObjectIdType) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 这里简化处理，实际应该根据objectID找到对应的刷新点
	// 由于我们没有维护对象到刷新点的映射，可以遍历刷新点并减少currentCount
	// 更优化的实现应该是在spawnObject时记录对象ID和刷新点的对应关系
	for _, spawnPoint := range sm.spawnPoints {
		if spawnPoint.currentCount > 0 {
			spawnPoint.currentCount--
			zLog.Debug("Object removed from spawn point",
				zap.Int32("spawn_id", spawnPoint.spawnID),
				zap.Int("current_count", spawnPoint.currentCount))
			break
		}
	}
}

// Stop 停止刷新管理器
func (sm *SpawnManager) Stop() {
	sm.running = false
	zLog.Info("Spawn manager stopped", zap.Int32("map_id", int32(sm.mapID)))
}
