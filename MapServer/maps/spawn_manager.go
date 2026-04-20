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

func (sm *SpawnManager) spawnObject(spawnPoint *SpawnPoint) {
	switch spawnPoint.spawnType {
	case 1:
		sm.spawnMonster(spawnPoint)
	case 2:
		sm.spawnNPC(spawnPoint)
	default:
		zLog.Warn("Unknown spawn type", zap.Int32("type", spawnPoint.spawnType))
	}
}

func (sm *SpawnManager) spawnMonster(spawnPoint *SpawnPoint) {
	objectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)

	monster := object.NewMonster(objectID, spawnPoint.objectID, "Monster_"+string(rune('A'+spawnPoint.objectID%26)), spawnPoint.position, 1)

	sm.mapObj.AddObject(monster)

	zLog.Debug("Spawning monster",
		zap.Int32("monster_id", spawnPoint.objectID),
		zap.Int64("object_id", int64(objectID)),
		zap.Float32("x", spawnPoint.position.X),
		zap.Float32("y", spawnPoint.position.Y))

	spawnPoint.currentCount++
}

func (sm *SpawnManager) spawnNPC(spawnPoint *SpawnPoint) {
	objectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)

	npc := object.NewNPC(objectID, spawnPoint.objectID, "NPC_"+string(rune('A'+spawnPoint.objectID%26)), spawnPoint.position, "Hello, adventurer!")

	sm.mapObj.AddObject(npc)

	zLog.Debug("Spawning NPC",
		zap.Int32("npc_id", spawnPoint.objectID),
		zap.Int64("object_id", int64(objectID)),
		zap.Float32("x", spawnPoint.position.X),
		zap.Float32("y", spawnPoint.position.Y))

	spawnPoint.currentCount++
}

func (sm *SpawnManager) RemoveObject(objectID id.ObjectIdType) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

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

func (sm *SpawnManager) Stop() {
	sm.running = false
	zLog.Info("Spawn manager stopped", zap.Int32("map_id", int32(sm.mapID)))
}
