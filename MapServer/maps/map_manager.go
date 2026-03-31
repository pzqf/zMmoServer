package maps

import (
	"fmt"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

// MapManager 地图管理器
// 负责管理多个地图实例
type MapManager struct {
	maps *zMap.TypedMap[id.MapIdType, *Map]
}

// NewMapManager 创建新的地图管理器
func NewMapManager() *MapManager {
	return &MapManager{
		maps: zMap.NewTypedMap[id.MapIdType, *Map](),
	}
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
func (mm *MapManager) CreateMap(mapID id.MapIdType, mapConfigID int32, name string, width, height float32, connManager *connection.ConnectionManager) *Map {
	// 检查地图是否已存在
	if existingMap, exists := mm.maps.Load(mapID); exists {
		zLog.Warn("Map already exists", zap.Int32("map_id", int32(mapID)))
		return existingMap
	}

	// 创建新地图
	newMap := NewMap(mapID, mapConfigID, name, width, height, connManager)
	mm.maps.Store(mapID, newMap)

	zLog.Info("Map created", zap.Int32("map_id", int32(mapID)), zap.String("name", name))

	// 初始化刷怪系统
	// newMap.InitSpawnSystem() // spawnManager is already initialized in NewMap

	return newMap
}

// CreateMapFromResource 从资源创建地图
func (mm *MapManager) CreateMapFromResource(resource *MapResource, connManager *connection.ConnectionManager) *Map {
	mapID := id.MapIdType(resource.MapID)

	// 检查地图是否已存在
	if existingMap, exists := mm.maps.Load(mapID); exists {
		zLog.Warn("Map already exists", zap.Int32("map_id", int32(mapID)))
		return existingMap
	}

	// 创建新地图
	newMap := NewMap(mapID, resource.MapID, resource.Name, float32(resource.Width), float32(resource.Height), connManager)
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
	maps := make([]*Map, 0)
	mm.maps.Range(func(key id.MapIdType, value *Map) bool {
		maps = append(maps, value)
		return true
	})

	for _, m := range maps {
		m.UpdateSkills()
	}
}

// UpdateAllMapsAI 更新所有地图的AI
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
