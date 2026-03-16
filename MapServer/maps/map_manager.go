package maps

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// MapManager 地图管理器
// 负责管理多个地图实例
type MapManager struct {
	mu   sync.RWMutex
	maps map[id.MapIdType]*Map
}

// NewMapManager 创建新的地图管理器
func NewMapManager() *MapManager {
	return &MapManager{
		maps: make(map[id.MapIdType]*Map),
	}
}

// Start 启动地图管理器
func (mm *MapManager) Start() error {
	// 启动所有地图
	mm.mu.RLock()
	maps := make([]*Map, 0, len(mm.maps))
	for _, m := range mm.maps {
		maps = append(maps, m)
	}
	mm.mu.RUnlock()

	for _, m := range maps {
		// 启动地图的各种系统
		m.InitSpawnSystem()
	}

	zLog.Info("MapManager started")
	return nil
}

// Stop 停止地图管理器
func (mm *MapManager) Stop() {
	// 停止所有地图
	mm.mu.RLock()
	maps := make([]*Map, 0, len(mm.maps))
	for _, m := range mm.maps {
		maps = append(maps, m)
	}
	mm.mu.RUnlock()

	for _, m := range maps {
		// 清理地图资源
		m.Cleanup()
	}

	zLog.Info("MapManager stopped")
}

// CreateMap 创建新地图
func (mm *MapManager) CreateMap(mapID id.MapIdType, mapConfigID int32, name string, width, height float32, connManager *connection.ConnectionManager) *Map {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// 检查地图是否已存在
	if _, exists := mm.maps[mapID]; exists {
		zLog.Warn("Map already exists", zap.Int32("map_id", int32(mapID)))
		return mm.maps[mapID]
	}

	// 创建新地图
	newMap := NewMap(mapID, mapConfigID, name, width, height, connManager)
	mm.maps[mapID] = newMap

	zLog.Info("Map created", zap.Int32("map_id", int32(mapID)), zap.String("name", name))

	// 初始化刷怪系统
	newMap.InitSpawnSystem()

	return newMap
}

// GetMap 获取地图
func (mm *MapManager) GetMap(mapID id.MapIdType) *Map {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	return mm.maps[mapID]
}

// UpdateAllMapsEvents 更新所有地图的事件
func (mm *MapManager) UpdateAllMapsEvents() {
	mm.mu.RLock()
	maps := make([]*Map, 0, len(mm.maps))
	for _, m := range mm.maps {
		maps = append(maps, m)
	}
	mm.mu.RUnlock()

	for _, m := range maps {
		// 处理地图事件
		m.UpdateEvents()
	}
}

// HandlePlayerEnterMap 处理玩家进入地图
func (mm *MapManager) HandlePlayerEnterMap(playerID int64, mapID int64, x, y, z float32) error {
	m := mm.GetMap(id.MapIdType(mapID))
	if m == nil {
		return fmt.Errorf("map not found: %d", mapID)
	}

	return m.AddPlayer(id.PlayerIdType(playerID), id.ObjectIdType(playerID), x, y, z)
}

// HandlePlayerMove 处理玩家移动
func (mm *MapManager) HandlePlayerMove(playerID, objectID, mapID int64, x, y, z float32) error {
	m := mm.GetMap(id.MapIdType(mapID))
	if m == nil {
		return fmt.Errorf("map not found: %d", mapID)
	}

	return m.MovePlayer(id.PlayerIdType(playerID), id.ObjectIdType(objectID), x, y, z)
}

// HandlePlayerAttack 处理玩家攻击
func (mm *MapManager) HandlePlayerAttack(playerID, objectID, mapID, targetID int64) (int64, int64, error) {
	m := mm.GetMap(id.MapIdType(mapID))
	if m == nil {
		return 0, 0, fmt.Errorf("map not found: %d", mapID)
	}

	return m.AttackTarget(id.PlayerIdType(playerID), id.ObjectIdType(objectID), id.ObjectIdType(targetID))
}

// UpdateAllMapsSkills 更新所有地图的技能
func (mm *MapManager) UpdateAllMapsSkills() {
	mm.mu.RLock()
	maps := make([]*Map, 0, len(mm.maps))
	for _, m := range mm.maps {
		maps = append(maps, m)
	}
	mm.mu.RUnlock()

	for _, m := range maps {
		m.UpdateSkills()
	}
}

// UpdateAllMapsAI 更新所有地图的AI
func (mm *MapManager) UpdateAllMapsAI(deltaTime time.Duration) {
	mm.mu.RLock()
	maps := make([]*Map, 0, len(mm.maps))
	for _, m := range mm.maps {
		maps = append(maps, m)
	}
	mm.mu.RUnlock()

	for _, m := range maps {
		if m.aiManager != nil {
			m.aiManager.Update(deltaTime)
		}
	}
}

// UpdateAllMapsBuffs 更新所有地图的Buff
func (mm *MapManager) UpdateAllMapsBuffs(deltaTime time.Duration) {
	mm.mu.RLock()
	maps := make([]*Map, 0, len(mm.maps))
	for _, m := range mm.maps {
		maps = append(maps, m)
	}
	mm.mu.RUnlock()

	for _, m := range maps {
		if m.buffManager != nil {
			m.buffManager.Update(deltaTime)
		}
	}
}

// UpdateAllMapsActivities 更新所有地图的活动
func (mm *MapManager) UpdateAllMapsActivities(deltaTime time.Duration) {
	mm.mu.RLock()
	maps := make([]*Map, 0, len(mm.maps))
	for _, m := range mm.maps {
		maps = append(maps, m)
	}
	mm.mu.RUnlock()

	for _, m := range maps {
		if m.activityManager != nil {
			m.activityManager.Update(deltaTime)
		}
	}
}

// UpdateAllMapsDungeons 更新所有地图的副本
func (mm *MapManager) UpdateAllMapsDungeons(deltaTime time.Duration) {
	mm.mu.RLock()
	maps := make([]*Map, 0, len(mm.maps))
	for _, m := range mm.maps {
		maps = append(maps, m)
	}
	mm.mu.RUnlock()

	for _, m := range maps {
		if m.dungeonManager != nil {
			m.dungeonManager.Update(deltaTime)
		}
	}
}

// RemoveMap 移除地图
func (mm *MapManager) RemoveMap(mapID id.MapIdType) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if m, exists := mm.maps[mapID]; exists {
		// 停止刷新管理器
		if m.spawnManager != nil {
			m.spawnManager.Stop()
		}

		delete(mm.maps, mapID)
		zLog.Info("Map removed", zap.Int32("map_id", int32(mapID)))
	}
}

// GetAllMaps 获取所有地图
func (mm *MapManager) GetAllMaps() []*Map {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	maps := make([]*Map, 0, len(mm.maps))
	for _, m := range mm.maps {
		maps = append(maps, m)
	}

	return maps
}

// GetMapCount 获取地图数量
func (mm *MapManager) GetMapCount() int {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	return len(mm.maps)
}
