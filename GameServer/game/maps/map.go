package maps

import (
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
	"go.uber.org/zap"
)

// Map GameServer 侧的游戏地图。
//
// 职责边界（②-b 后）：GameServer 的 Map 只是**逻辑图内的对象注册表 + 玩家集**——
// 供进出图登记、在场校验、玩家计数与 EventPlayerEnterMap/LeaveMap 广播。
//   - 客户端视野（AOI enter/leave/move）由 **MapServer 的分层 AOI 单一权威**驱动，
//     经 crossserver 500-505 → map_service.HandleAOINotify → 推客户端。
//   - 因此 GameServer **不再自建 AOI/分区（grid AOI + Region 网格）**：原来那两套网格记账
//     写而不读（listener 恒 nil、范围查询走 objects 暴力遍历）、且按逻辑图ID键值无法感知
//     ②-b 分线，已随 ②-b 修复一并移除。范围查询统一走 objects 遍历（本图对象量级小）。
type Map struct {
	mu          sync.RWMutex
	mapID       id.MapIdType
	mapConfigID int32
	name        string
	width       float32
	height      float32
	objects     map[id.ObjectIdType]common.IGameObject
	players     map[id.PlayerIdType]bool
	createdAt   time.Time
}

// NewMap 创建新地图
func NewMap(mapID id.MapIdType, mapConfigID int32, name string, width, height float32) *Map {
	return &Map{
		mapID:       mapID,
		mapConfigID: mapConfigID,
		name:        name,
		width:       width,
		height:      height,
		objects:     make(map[id.ObjectIdType]common.IGameObject),
		players:     make(map[id.PlayerIdType]bool),
		createdAt:   time.Now(),
	}
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

// AddObject 添加游戏对象到地图
func (m *Map) AddObject(object common.IGameObject) {
	if object == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	objectID := object.GetID()
	m.objects[objectID] = object
	object.SetMap(m)

	zLog.Debug("Object added to map",
		zap.Int64("object_id", int64(objectID)),
		zap.Int32("map_id", int32(m.mapID)))
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
	object.SetMap(nil)

	zLog.Debug("Object removed from map",
		zap.Int64("object_id", int64(objectID)),
		zap.Int32("map_id", int32(m.mapID)))
}

// MoveObject 移动游戏对象（GameServer 侧仅更新对象坐标；AOI/视野由 MapServer 权威处理）
func (m *Map) MoveObject(object common.IGameObject, targetPos common.Vector3) error {
	object.SetPosition(targetPos)
	return nil
}

// TeleportObject 传送游戏对象（同 MoveObject，仅更新坐标）
func (m *Map) TeleportObject(object common.IGameObject, targetPos common.Vector3) error {
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
		if obj.GetType() == common.GameObjectTypePlayer && obj.GetID() == id.ObjectIdType(playerID) {
			delete(m.objects, obj.GetID())
			obj.SetMap(nil)
			break
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
