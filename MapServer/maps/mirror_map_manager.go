package maps

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type MirrorMapConfig struct {
	SourceMapConfigID     int32
	MaxPlayersPerInstance int32
	MinPlayersForCreate   int32
	IdleTimeout           time.Duration
	MaxInstances          int32
}

type MirrorMapInstance struct {
	MapID         id.MapIdType
	SourceMapID   int32
	PlayerCount   int32
	CreatedAt     time.Time
	LastActiveAt  time.Time
	ServerGroupID int32
}

type MirrorMapManager struct {
	mu          sync.RWMutex
	mapManager  *MapManager
	connManager *connection.ConnectionManager
	config      MirrorMapConfig
	instances   *zMap.TypedMap[id.MapIdType, *MirrorMapInstance]
	bySource    *zMap.TypedMap[int32, []*MirrorMapInstance]
	nextMapID   int64
}

func NewMirrorMapManager(mapManager *MapManager, connManager *connection.ConnectionManager, config MirrorMapConfig) *MirrorMapManager {
	return &MirrorMapManager{
		mapManager:  mapManager,
		connManager: connManager,
		config:      config,
		instances:   zMap.NewTypedMap[id.MapIdType, *MirrorMapInstance](),
		bySource:    zMap.NewTypedMap[int32, []*MirrorMapInstance](),
		nextMapID:   200000,
	}
}

func (mmm *MirrorMapManager) GetOrCreateMirrorMap(sourceMapConfigID int32, serverGroupID int32) (id.MapIdType, error) {
	mmm.mu.Lock()
	defer mmm.mu.Unlock()

	instances, exists := mmm.bySource.Load(sourceMapConfigID)
	if exists {
		for _, inst := range instances {
			if inst.ServerGroupID == serverGroupID && inst.PlayerCount < mmm.config.MaxPlayersPerInstance {
				inst.PlayerCount++
				inst.LastActiveAt = time.Now()
				return inst.MapID, nil
			}
		}
	}

	totalInstances := int32(0)
	mmm.instances.Range(func(key id.MapIdType, value *MirrorMapInstance) bool {
		totalInstances++
		return true
	})

	if mmm.config.MaxInstances > 0 && totalInstances >= mmm.config.MaxInstances {
		return 0, fmt.Errorf("max mirror instances reached: %d", mmm.config.MaxInstances)
	}

	_ = id.MapIdType(atomic.AddInt64(&mmm.nextMapID, 1))

	mirrorMap := mmm.mapManager.CreateCrossServerMap(
		sourceMapConfigID,
		fmt.Sprintf("Mirror_%d", sourceMapConfigID),
		500, 500,
		MapModeMirror,
		serverGroupID,
		mmm.connManager,
	)

	inst := &MirrorMapInstance{
		MapID:         mirrorMap.GetID(),
		SourceMapID:   sourceMapConfigID,
		PlayerCount:   1,
		CreatedAt:     time.Now(),
		LastActiveAt:  time.Now(),
		ServerGroupID: serverGroupID,
	}

	mmm.instances.Store(mirrorMap.GetID(), inst)

	existingList, _ := mmm.bySource.Load(sourceMapConfigID)
	updatedList := append(existingList, inst)
	mmm.bySource.Store(sourceMapConfigID, updatedList)

	zLog.Info("Mirror map created",
		zap.Int32("source_map_config_id", sourceMapConfigID),
		zap.Int32("map_id", int32(mirrorMap.GetID())),
		zap.Int32("server_group_id", serverGroupID))

	return mirrorMap.GetID(), nil
}

func (mmm *MirrorMapManager) PlayerLeave(mapID id.MapIdType) {
	mmm.mu.Lock()
	defer mmm.mu.Unlock()

	inst, exists := mmm.instances.Load(mapID)
	if !exists {
		return
	}

	inst.PlayerCount--
	inst.LastActiveAt = time.Now()

	if inst.PlayerCount <= 0 {
		mmm.removeInstanceLocked(inst)
	}
}

func (mmm *MirrorMapManager) removeInstanceLocked(inst *MirrorMapInstance) {
	mmm.instances.Delete(inst.MapID)

	instances, exists := mmm.bySource.Load(inst.SourceMapID)
	if exists {
		updated := make([]*MirrorMapInstance, 0, len(instances))
		for _, i := range instances {
			if i.MapID != inst.MapID {
				updated = append(updated, i)
			}
		}
		if len(updated) == 0 {
			mmm.bySource.Delete(inst.SourceMapID)
		} else {
			mmm.bySource.Store(inst.SourceMapID, updated)
		}
	}

	zLog.Info("Mirror map removed",
		zap.Int32("map_id", int32(inst.MapID)),
		zap.Int32("source_map_config_id", inst.SourceMapID))
}

func (mmm *MirrorMapManager) Update(deltaTime time.Duration) {
	mmm.mu.Lock()
	defer mmm.mu.Unlock()

	now := time.Now()
	toRemove := make([]*MirrorMapInstance, 0)

	mmm.instances.Range(func(key id.MapIdType, inst *MirrorMapInstance) bool {
		if inst.PlayerCount <= 0 && now.Sub(inst.LastActiveAt) >= mmm.config.IdleTimeout {
			toRemove = append(toRemove, inst)
		}
		return true
	})

	for _, inst := range toRemove {
		mmm.removeInstanceLocked(inst)
		zLog.Info("Mirror map auto-destroyed (idle timeout)",
			zap.Int32("map_id", int32(inst.MapID)))
	}
}

func (mmm *MirrorMapManager) GetInstanceCount() int {
	count := 0
	mmm.instances.Range(func(key id.MapIdType, value *MirrorMapInstance) bool {
		count++
		return true
	})
	return count
}

func (mmm *MirrorMapManager) GetInstancesBySource(sourceMapConfigID int32) []*MirrorMapInstance {
	instances, exists := mmm.bySource.Load(sourceMapConfigID)
	if !exists {
		return nil
	}
	return instances
}

func (mmm *MirrorMapManager) GetTotalPlayerCount() int32 {
	total := int32(0)
	mmm.instances.Range(func(key id.MapIdType, value *MirrorMapInstance) bool {
		total += value.PlayerCount
		return true
	})
	return total
}
