package maps

import (
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"go.uber.org/zap"
)

type SpawnManager struct {
	mu          sync.RWMutex
	mapID       id.MapIdType
	mapConfigID int32
	mapObj      *Map
	spawnPoints []*SpawnPoint
	running     bool
	// spawnedBy: 记录每个刷出对象归属的刷怪点（MAP-8），供对象移除时精确回退 currentCount，
	// 否则刷怪点计数只增不减→达 maxCount 后永久停摆。
	spawnedBy map[id.ObjectIdType]*SpawnPoint
}

type SpawnPoint struct {
	spawnID      int32
	spawnType    int32
	objectID     int32
	position     common.Vector3
	respawnTime  int
	maxCount     int
	currentCount int
	active       bool
	lastSpawn    time.Time
	radius       float32
	patrolRange  float32
}

func NewSpawnManager(mapID id.MapIdType, mapObj *Map) *SpawnManager {
	return &SpawnManager{
		mapID:       mapID,
		mapObj:      mapObj,
		spawnPoints: make([]*SpawnPoint, 0),
		running:     false,
		spawnedBy:   make(map[id.ObjectIdType]*SpawnPoint),
	}
}

func (sm *SpawnManager) Init(mapConfigID int32) {
	sm.mapConfigID = mapConfigID

	tm := tables.GetTableManager()
	if tm == nil {
		zLog.Warn("TableManager not initialized, spawn manager uses no spawn points",
			zap.Int32("map_id", int32(sm.mapID)))
		sm.running = true
		go sm.spawnLoop()
		return
	}

	spawnLoader := tm.GetSpawnPointLoader()
	if spawnLoader == nil {
		zLog.Warn("SpawnPointLoader not available",
			zap.Int32("map_id", int32(sm.mapID)))
		sm.running = true
		go sm.spawnLoop()
		return
	}

	configPoints := spawnLoader.GetSpawnPointsByMap(mapConfigID)
	for _, cp := range configPoints {
		spawnPoint := &SpawnPoint{
			spawnID:   cp.SpawnID,
			spawnType: int32(cp.SpawnType),
			objectID:  cp.MonsterID,
			position: common.Vector3{
				X: cp.PosX,
				Y: cp.PosY,
				Z: cp.PosZ,
			},
			respawnTime:  int(cp.SpawnInterval),
			maxCount:     int(cp.MaxCount),
			currentCount: 0,
			active:       true,
			lastSpawn:    time.Now(),
			radius:       cp.Radius,
			patrolRange:  cp.PatrolRange,
		}
		sm.spawnPoints = append(sm.spawnPoints, spawnPoint)
	}

	zLog.Info("Spawn manager initialized from config",
		zap.Int32("map_id", int32(sm.mapID)),
		zap.Int32("map_config_id", mapConfigID),
		zap.Int("spawn_points", len(sm.spawnPoints)))

	sm.running = true
	go sm.spawnLoop()
}

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

func (sm *SpawnManager) checkSpawnPoints() {
	now := time.Now()

	// 第一阶段：持 sm.mu 挑选到期刷怪点、创建对象、登记归属并递增计数，收集待加入对象。
	sm.mu.Lock()
	var toSpawn []common.IGameObject
	for _, spawnPoint := range sm.spawnPoints {
		if !spawnPoint.active {
			continue
		}
		if spawnPoint.currentCount < spawnPoint.maxCount {
			elapsed := now.Sub(spawnPoint.lastSpawn).Seconds()
			if elapsed >= float64(spawnPoint.respawnTime) {
				if obj := sm.createSpawnObject(spawnPoint); obj != nil {
					toSpawn = append(toSpawn, obj)
					spawnPoint.lastSpawn = now
				}
			}
		}
	}
	sm.mu.Unlock()

	// 第二阶段：在锁外把对象串行加入地图（经 actor）。
	// 关键：绝不能持 sm.mu 跨阻塞的 Do——那样会与地图 goroutine 上（持 m.mu）需要 sm.mu 的
	// RemoveObject 形成锁序死锁（MAP-8 修复的前提）。
	for _, obj := range toSpawn {
		o := obj
		sm.mapObj.Do(func() { sm.mapObj.AddObject(o) })
	}
}

// createSpawnObject 在持 sm.mu 时创建刷出对象、登记归属并递增 currentCount，返回待加入地图的对象。
func (sm *SpawnManager) createSpawnObject(spawnPoint *SpawnPoint) common.IGameObject {
	var objectID id.ObjectIdType
	var obj common.IGameObject
	switch spawnPoint.spawnType {
	case 1:
		objectID = nextMapObjectID()
		obj = object.NewMonster(objectID, spawnPoint.objectID, "Monster_"+string(rune('A'+spawnPoint.objectID%26)), spawnPoint.position, 1)
		zLog.Debug("Spawning monster",
			zap.Int32("monster_id", spawnPoint.objectID),
			zap.Int64("object_id", int64(objectID)),
			zap.Float32("x", spawnPoint.position.X),
			zap.Float32("y", spawnPoint.position.Y))
	case 2:
		objectID = nextMapObjectID()
		obj = object.NewNPC(objectID, spawnPoint.objectID, "NPC_"+string(rune('A'+spawnPoint.objectID%26)), spawnPoint.position, "Hello, adventurer!")
		zLog.Debug("Spawning NPC",
			zap.Int32("npc_id", spawnPoint.objectID),
			zap.Int64("object_id", int64(objectID)),
			zap.Float32("x", spawnPoint.position.X),
			zap.Float32("y", spawnPoint.position.Y))
	default:
		zLog.Warn("Unknown spawn type", zap.Int32("type", spawnPoint.spawnType))
		return nil
	}

	sm.spawnedBy[objectID] = spawnPoint
	spawnPoint.currentCount++
	return obj
}

// RemoveObject 在对象移除时精确回退其归属刷怪点的 currentCount（MAP-8）。
// 通过 spawnedBy 定位归属点，避免此前"递减第一个非零点"的错配（会让满员点被误减而超刷）。
// 非本管理器刷出的对象（如死亡重生走 scheduleMonsterRespawn 直接 AddObject 的）不在表中，忽略。
func (sm *SpawnManager) RemoveObject(objectID id.ObjectIdType) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	spawnPoint, ok := sm.spawnedBy[objectID]
	if !ok {
		return
	}
	delete(sm.spawnedBy, objectID)
	if spawnPoint.currentCount > 0 {
		spawnPoint.currentCount--
	}
	zLog.Debug("Object removed from spawn point",
		zap.Int32("spawn_id", spawnPoint.spawnID),
		zap.Int("current_count", spawnPoint.currentCount))
}

func (sm *SpawnManager) Stop() {
	sm.running = false
	zLog.Info("Spawn manager stopped", zap.Int32("map_id", int32(sm.mapID)))
}
