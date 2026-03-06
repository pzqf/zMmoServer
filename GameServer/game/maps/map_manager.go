package maps

import (
	"errors"
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// MapManager 地图管理器
type MapManager struct {
	mu       sync.RWMutex
	maps     map[id.MapIdType]*Map
	mapCount int32
}

// NewMapManager 创建地图管理器
func NewMapManager() *MapManager {
	return &MapManager{
		maps:     make(map[id.MapIdType]*Map),
		mapCount: 0,
	}
}

// CreateMap 创建地图
func (mm *MapManager) CreateMap(mapConfigID int32, name string, width, height float32) *Map {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.mapCount++
	mapID := id.MapIdType(mm.mapCount)

	m := NewMap(mapID, mapConfigID, name, width, height)
	mm.maps[mapID] = m

	zLog.Info("Map created",
		zap.Int32("map_id", int32(mapID)),
		zap.Int32("map_config_id", mapConfigID),
		zap.String("name", name))

	return m
}

// GetMap 获取地图
func (mm *MapManager) GetMap(mapID id.MapIdType) (*Map, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	m, exists := mm.maps[mapID]
	if !exists {
		return nil, errors.New("map not found")
	}
	return m, nil
}

// RemoveMap 移除地图
func (mm *MapManager) RemoveMap(mapID id.MapIdType) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	m, exists := mm.maps[mapID]
	if !exists {
		return errors.New("map not found")
	}

	m.mu.Lock()
	for _, obj := range m.objects {
		obj.SetMap(nil)
	}
	m.objects = make(map[id.ObjectIdType]common.IGameObject)
	m.mu.Unlock()

	delete(mm.maps, mapID)

	zLog.Info("Map removed", zap.Int32("map_id", int32(mapID)))
	return nil
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

// GetTotalPlayerCount 获取所有地图的玩家总数
func (mm *MapManager) GetTotalPlayerCount() int {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	total := 0
	for _, m := range mm.maps {
		total += m.GetPlayerCount()
	}
	return total
}

// UpdateAll 更新所有地图
func (mm *MapManager) UpdateAll(deltaTime float64) {
	mm.mu.RLock()
	maps := make([]*Map, 0, len(mm.maps))
	for _, m := range mm.maps {
		maps = append(maps, m)
	}
	mm.mu.RUnlock()

	for _, m := range maps {
		m.Update(deltaTime)
	}
}

// Range 遍历所有地图
func (mm *MapManager) Range(f func(mapID id.MapIdType, m *Map) bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	for mapID, m := range mm.maps {
		if !f(mapID, m) {
			break
		}
	}
}
