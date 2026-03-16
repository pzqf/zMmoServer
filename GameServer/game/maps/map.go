package maps

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// Region 地图区域
type Region struct {
	mu       sync.RWMutex
	regionID id.RegionIdType
	objects  map[id.ObjectIdType]common.IGameObject
}

// NewRegion 创建新区域
func NewRegion(regionID id.RegionIdType) *Region {
	return &Region{
		regionID: regionID,
		objects:  make(map[id.ObjectIdType]common.IGameObject),
	}
}

// AddObject 添加游戏对象到区域
func (r *Region) AddObject(object common.IGameObject) {
	if object == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.objects[object.GetID()] = object
}

// RemoveObject 从区域移除游戏对象
func (r *Region) RemoveObject(objectID id.ObjectIdType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.objects, objectID)
}

// GetObjects 获取区域内所有对象
func (r *Region) GetObjects() []common.IGameObject {
	r.mu.RLock()
	defer r.mu.RUnlock()
	objects := make([]common.IGameObject, 0, len(r.objects))
	for _, obj := range r.objects {
		objects = append(objects, obj)
	}
	return objects
}

// GetObjectCount 获取区域内对象数量
func (r *Region) GetObjectCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.objects)
}

// Map 游戏地图
type Map struct {
	mu           sync.RWMutex
	mapID        id.MapIdType
	mapConfigID  int32
	name         string
	width        float32
	height       float32
	regionSize   float32
	objects      map[id.ObjectIdType]common.IGameObject
	regions      map[id.RegionIdType]*Region
	players      map[id.PlayerIdType]bool
	createdAt    time.Time
}

// NewMap 创建新地图
func NewMap(mapID id.MapIdType, mapConfigID int32, name string, width, height float32) *Map {
	m := &Map{
		mapID:        mapID,
		mapConfigID:  mapConfigID,
		name:         name,
		width:        width,
		height:       height,
		regionSize:   50,
		objects:      make(map[id.ObjectIdType]common.IGameObject),
		regions:      make(map[id.RegionIdType]*Region),
		players:      make(map[id.PlayerIdType]bool),
		createdAt:    time.Now(),
	}
	return m
}

// GetID 获取地图ID
func (m *Map) GetID() id.MapIdType {
	return m.mapID
}

// GetName 获取地图名称
func (m *Map) GetName() string {
	return m.name
}

// GetMapConfigID 获取地图配置ID
func (m *Map) GetMapConfigID() int32 {
	return m.mapConfigID
}

// GetSize 获取地图尺寸
func (m *Map) GetSize() (float32, float32) {
	return m.width, m.height
}

// GetCreatedAt 获取创建时间
func (m *Map) GetCreatedAt() time.Time {
	return m.createdAt
}

// getRegionID 根据坐标计算区域ID
func (m *Map) getRegionID(pos common.Vector3) id.RegionIdType {
	if m.regionSize <= 0 {
		return 0
	}
	xRegion := uint64(pos.X / m.regionSize)
	yRegion := uint64(pos.Y / m.regionSize)
	return id.RegionIdType(xRegion*1000000 + yRegion)
}

// AddObject 添加游戏对象到地图
func (m *Map) AddObject(object common.IGameObject) {
	if object == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	objectID := object.GetID()
	m.objects[objectID] = object

	regionID := m.getRegionID(object.GetPosition())
	if _, exists := m.regions[regionID]; !exists {
		m.regions[regionID] = NewRegion(regionID)
	}
	m.regions[regionID].AddObject(object)

	object.SetMap(m)

	zLog.Debug("Object added to map",
		zap.Int64("object_id", int64(objectID)),
		zap.Int32("map_id", int32(m.mapID)),
		zap.Int32("region_id", int32(regionID)))
}

// RemoveObject 从地图移除游戏对象
func (m *Map) RemoveObject(objectID id.ObjectIdType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	object, exists := m.objects[objectID]
	if !exists {
		return
	}

	delete(m.objects, objectID)

	regionID := m.getRegionID(object.GetPosition())
	if region, ok := m.regions[regionID]; ok {
		region.RemoveObject(objectID)
	}

	object.SetMap(nil)

	zLog.Debug("Object removed from map",
		zap.Int64("object_id", int64(objectID)),
		zap.Int32("map_id", int32(m.mapID)))
}

// MoveObject 移动游戏对象
func (m *Map) MoveObject(object common.IGameObject, targetPos common.Vector3) error {
	oldPos := object.GetPosition()
	oldRegionID := m.getRegionID(oldPos)
	newRegionID := m.getRegionID(targetPos)

	if oldRegionID == newRegionID {
		object.SetPosition(targetPos)
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if region, exists := m.regions[oldRegionID]; exists {
		region.RemoveObject(object.GetID())
	}

	if _, exists := m.regions[newRegionID]; !exists {
		m.regions[newRegionID] = NewRegion(newRegionID)
	}
	m.regions[newRegionID].AddObject(object)

	object.SetPosition(targetPos)

	return nil
}

// TeleportObject 传送游戏对象
func (m *Map) TeleportObject(object common.IGameObject, targetPos common.Vector3) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldPos := object.GetPosition()
	oldRegionID := m.getRegionID(oldPos)
	newRegionID := m.getRegionID(targetPos)

	if region, exists := m.regions[oldRegionID]; exists {
		region.RemoveObject(object.GetID())
	}

	if _, exists := m.regions[newRegionID]; !exists {
		m.regions[newRegionID] = NewRegion(newRegionID)
	}
	m.regions[newRegionID].AddObject(object)

	object.SetPosition(targetPos)

	return nil
}

// GetObjectsInRange 获取指定范围内的游戏对象
func (m *Map) GetObjectsInRange(pos common.Vector3, radius float32) []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	objects := make([]common.IGameObject, 0)
	for _, obj := range m.objects {
		distance := obj.GetPosition().DistanceTo(pos)
		if distance <= radius {
			objects = append(objects, obj)
		}
	}
	return objects
}

// GetObjectsByType 获取指定类型的游戏对象
func (m *Map) GetObjectsByType(objectType common.GameObjectType) []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	objects := make([]common.IGameObject, 0)
	for _, obj := range m.objects {
		if obj.GetType() == objectType {
			objects = append(objects, obj)
		}
	}
	return objects
}

// GetObject 获取指定对象
func (m *Map) GetObject(objectID id.ObjectIdType) common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.objects[objectID]
}

// GetAllObjects 获取所有对象
func (m *Map) GetAllObjects() []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	objects := make([]common.IGameObject, 0, len(m.objects))
	for _, obj := range m.objects {
		objects = append(objects, obj)
	}
	return objects
}

// GetObjectCount 获取对象数量
func (m *Map) GetObjectCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.objects)
}

// AddPlayer 添加玩家
func (m *Map) AddPlayer(playerID id.PlayerIdType, object common.IGameObject) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.players[playerID] = true
	m.objects[object.GetID()] = object

	regionID := m.getRegionID(object.GetPosition())
	if _, exists := m.regions[regionID]; !exists {
		m.regions[regionID] = NewRegion(regionID)
	}
	m.regions[regionID].AddObject(object)

	object.SetMap(m)

	event.Publish(event.NewEvent(event.EventPlayerEnterMap, m, &event.PlayerMapEventData{
		PlayerID: playerID,
		MapID:    m.mapID,
		PosX:     object.GetPosition().X,
		PosY:     object.GetPosition().Y,
		PosZ:     object.GetPosition().Z,
	}))

	zLog.Info("Player entered map",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(m.mapID)))
}

// RemovePlayer 移除玩家
func (m *Map) RemovePlayer(playerID id.PlayerIdType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.players, playerID)

	for _, obj := range m.objects {
		if obj.GetType() == common.GameObjectTypePlayer {
			if obj.GetID() == id.ObjectIdType(playerID) {
				regionID := m.getRegionID(obj.GetPosition())
				if region, ok := m.regions[regionID]; ok {
					region.RemoveObject(obj.GetID())
				}
				delete(m.objects, obj.GetID())
				obj.SetMap(nil)
				break
			}
		}
	}

	event.Publish(event.NewEvent(event.EventPlayerLeaveMap, m, &event.PlayerMapEventData{
		PlayerID: playerID,
		MapID:    m.mapID,
	}))

	zLog.Info("Player left map",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(m.mapID)))
}

// GetPlayerCount 获取玩家数量
func (m *Map) GetPlayerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.players)
}

// GetPlayers 获取所有玩家ID
func (m *Map) GetPlayers() []id.PlayerIdType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	players := make([]id.PlayerIdType, 0, len(m.players))
	for playerID := range m.players {
		players = append(players, playerID)
	}
	return players
}

// Update 更新地图
func (m *Map) Update(deltaTime float64) {
	m.mu.RLock()
	objects := make([]common.IGameObject, 0, len(m.objects))
	for _, obj := range m.objects {
		objects = append(objects, obj)
	}
	m.mu.RUnlock()

	for _, obj := range objects {
		if obj.IsActive() {
			obj.Update(deltaTime)
		}
	}
}
