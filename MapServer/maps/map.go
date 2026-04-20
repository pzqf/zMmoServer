package maps

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zCommon/aoi"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/models"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoServer/MapServer/maps/ai"
	"github.com/pzqf/zMmoServer/MapServer/maps/buff"
	"github.com/pzqf/zMmoServer/MapServer/maps/combat"
	"github.com/pzqf/zMmoServer/MapServer/maps/dungeon"
	"github.com/pzqf/zMmoServer/MapServer/maps/economy"
	"github.com/pzqf/zMmoServer/MapServer/maps/event"
	"github.com/pzqf/zMmoServer/MapServer/maps/item"
	"github.com/pzqf/zMmoServer/MapServer/maps/loot"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"github.com/pzqf/zMmoServer/MapServer/maps/skill"
	"github.com/pzqf/zMmoServer/MapServer/maps/task"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type MapMode int

const (
	MapModeSingleServer MapMode = iota
	MapModeCrossGroup
	MapModeMirror
)

type Map struct {
	mu                sync.RWMutex
	mapID             id.MapIdType
	mapConfigID       int32
	name              string
	width             float32
	height            float32
	mapMode           MapMode
	serverGroupID     int32
	isDungeon         bool
	dungeonInstanceID id.InstanceIdType
	objects           *zMap.TypedMap[id.ObjectIdType, common.IGameObject]
	players           *zMap.TypedMap[id.PlayerIdType, bool]
	aoiManager        *aoi.GridManager
	spawnPoints       []*models.MapSpawnPoint
	teleportPoints    []*models.MapTeleportPoint
	buildings         []*models.MapBuilding
	events            []*models.MapEvent
	resources         []*models.MapResource
	spawnManager      *SpawnManager
	eventManager      *event.EventManager
	aiManager         *ai.AIManager
	buffManager       *buff.BuffManager
	dungeonManager    *dungeon.DungeonManager
	skillManager      *skill.SkillManager
	taskManager       *task.TaskManager
	inventoryManager  *item.InventoryManager
	currencyManager   *economy.CurrencyManager
	tradeManager      *economy.TradeManager
	auctionManager    *economy.AuctionManager
	shopManager       *economy.ShopManager
	combatSystem      *combat.CombatSystem
	lootSystem        *loot.LootSystem
	connManager       *connection.ConnectionManager
	createdAt         time.Time
}

const defaultGridSize = 50.0

func NewMap(mapID id.MapIdType, mapConfigID int32, name string, width, height float32, connManager *connection.ConnectionManager) *Map {
	m := &Map{
		mapID:          mapID,
		mapConfigID:    mapConfigID,
		name:           name,
		width:          width,
		height:         height,
		objects:        zMap.NewTypedMap[id.ObjectIdType, common.IGameObject](),
		players:        zMap.NewTypedMap[id.PlayerIdType, bool](),
		aoiManager:     aoi.NewGridManager(float64(width), float64(height), defaultGridSize, defaultGridSize),
		spawnPoints:    make([]*models.MapSpawnPoint, 0),
		teleportPoints: make([]*models.MapTeleportPoint, 0),
		buildings:      make([]*models.MapBuilding, 0),
		events:         make([]*models.MapEvent, 0),
		resources:      make([]*models.MapResource, 0),
		connManager:    connManager,
		createdAt:      time.Now(),
	}

	m.aoiManager.SetListener(m.handleAOIEvent)

	m.spawnManager = NewSpawnManager(mapID, m)
	m.eventManager = event.NewEventManager()
	m.aiManager = ai.NewAIManager()
	m.aiManager.SetTableManager(tables.GetTableManager())
	m.buffManager = buff.NewBuffManager()
	m.buffManager.SetTableManager(tables.GetTableManager())
	m.dungeonManager = dungeon.NewDungeonManager()
	m.skillManager = skill.NewSkillManager()
	m.skillManager.SetTableManager(tables.GetTableManager())
	m.taskManager = task.NewTaskManager()
	m.taskManager.SetTableManager(tables.GetTableManager())
	m.inventoryManager = item.NewInventoryManager()
	m.inventoryManager.SetTableManager(tables.GetTableManager())
	m.currencyManager = economy.NewCurrencyManager()
	m.tradeManager = economy.NewTradeManager()
	m.auctionManager = economy.NewAuctionManager()
	m.shopManager = economy.NewShopManager()
	m.combatSystem = combat.NewCombatSystem()
	m.lootSystem = loot.NewLootSystem(tables.GetTableManager())

	m.tradeManager.SetCurrencyManager(m.currencyManager)
	m.auctionManager.SetCurrencyManager(m.currencyManager)

	m.CreateDefaultEvents()

	return m
}

func (m *Map) handleAOIEvent(evt aoi.AOIEvent) {
	switch evt.Type {
	case aoi.AOIEventEnter:
		zLog.Debug("AOI: entity entered view",
			zap.Int64("watcher", evt.Watcher),
			zap.Int64("target", evt.Target))
	case aoi.AOIEventLeave:
		zLog.Debug("AOI: entity left view",
			zap.Int64("watcher", evt.Watcher),
			zap.Int64("target", evt.Target))
	case aoi.AOIEventMove:
		zLog.Debug("AOI: entity moved",
			zap.Int64("target", evt.Target))
	}
}

func (m *Map) posToCoord(pos common.Vector3) aoi.Coord {
	return aoi.Coord{X: float64(pos.X), Y: float64(pos.Y)}
}

func (m *Map) GetID() id.MapIdType {
	return m.mapID
}

func (m *Map) GetName() string {
	return m.name
}

func (m *Map) GetWidth() float32 {
	return m.width
}

func (m *Map) GetHeight() float32 {
	return m.height
}

func (m *Map) GetAOIManager() *aoi.GridManager {
	return m.aoiManager
}

func (m *Map) SetMaxPlayers(maxPlayers int32) {}

func (m *Map) SetDescription(description string) {}

func (m *Map) SetWeatherType(weatherType string) {}

func (m *Map) SetMinLevel(minLevel int32) {}

func (m *Map) SetMaxLevel(maxLevel int32) {}

func (m *Map) GetObjects() []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()
	objects := make([]common.IGameObject, 0)
	m.objects.Range(func(objectID id.ObjectIdType, obj common.IGameObject) bool {
		objects = append(objects, obj)
		return true
	})
	return objects
}

func (m *Map) GetObject(objectID id.ObjectIdType) common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()
	obj, _ := m.objects.Load(objectID)
	return obj
}

func (m *Map) AddObject(obj common.IGameObject) {
	if obj == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.objects.Store(obj.GetID(), obj)

	m.aoiManager.AddEntity(int64(obj.GetID()), m.posToCoord(obj.GetPosition()))

	if obj.GetType() == common.GameObjectTypePlayer {
		if p, ok := obj.(*object.Player); ok {
			m.players.Store(p.GetPlayerID(), true)
			p.SetBuffManager(m.buffManager)
		}
	}

	if obj.GetType() == common.GameObjectTypeMonster {
		if monster, ok := obj.(*object.Monster); ok {
			m.aiManager.CreateMonsterAI(monster, "melee", m)
		}
	}
}

func (m *Map) RemoveObject(objectID id.ObjectIdType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	obj, ok := m.objects.Load(objectID)
	if !ok {
		return
	}

	m.aoiManager.RemoveEntity(int64(objectID), m.posToCoord(obj.GetPosition()))

	if obj.GetType() == common.GameObjectTypePlayer {
		if p, ok := obj.(*object.Player); ok {
			m.players.Delete(p.GetPlayerID())
		}
	}

	if obj.GetType() == common.GameObjectTypeMonster {
		m.aiManager.RemoveMonsterAI(objectID)
	}

	m.objects.Delete(objectID)
}

func (m *Map) MoveObject(objectID id.ObjectIdType, newPos common.Vector3) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	obj, ok := m.objects.Load(objectID)
	if !ok {
		return fmt.Errorf("object not found: %d", objectID)
	}

	oldPos := obj.GetPosition()
	obj.SetPosition(newPos)
	m.aoiManager.MoveEntity(int64(objectID), m.posToCoord(oldPos), m.posToCoord(newPos))

	return nil
}

func (m *Map) GetObjectsInRange(position common.Vector3, radius float32) []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	objects := make([]common.IGameObject, 0)
	m.objects.Range(func(objectID id.ObjectIdType, obj common.IGameObject) bool {
		distance := position.DistanceTo(obj.GetPosition())
		if distance <= radius {
			objects = append(objects, obj)
		}
		return true
	})

	return objects
}

func (m *Map) GetPlayersInRange(position common.Vector3, radius float32) []common.IGameObject {
	return m.getObjectsInRangeByType(position, radius, common.GameObjectTypePlayer)
}

func (m *Map) GetMonstersInRange(position common.Vector3, radius float32) []common.IGameObject {
	return m.getObjectsInRangeByType(position, radius, common.GameObjectTypeMonster)
}

func (m *Map) GetNPCsInRange(position common.Vector3, radius float32) []common.IGameObject {
	return m.getObjectsInRangeByType(position, radius, common.GameObjectTypeNPC)
}

func (m *Map) getObjectsInRangeByType(position common.Vector3, radius float32, objType common.GameObjectType) []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	objects := make([]common.IGameObject, 0)
	m.objects.Range(func(objectID id.ObjectIdType, obj common.IGameObject) bool {
		if obj.GetType() == objType {
			distance := position.DistanceTo(obj.GetPosition())
			if distance <= radius {
				objects = append(objects, obj)
			}
		}
		return true
	})

	return objects
}

func (m *Map) GetObjectsByType(objectType common.GameObjectType) []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	objects := make([]common.IGameObject, 0)
	m.objects.Range(func(objectID id.ObjectIdType, obj common.IGameObject) bool {
		if obj.GetType() == objectType {
			objects = append(objects, obj)
		}
		return true
	})

	return objects
}

func (m *Map) GetPlayers() []*object.Player {
	m.mu.RLock()
	defer m.mu.RUnlock()

	players := make([]*object.Player, 0)
	m.objects.Range(func(objectID id.ObjectIdType, obj common.IGameObject) bool {
		if obj.GetType() == common.GameObjectTypePlayer {
			if player, ok := obj.(*object.Player); ok {
				players = append(players, player)
			}
		}
		return true
	})

	return players
}

func (m *Map) GetPlayerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int(m.players.Len())
}

func (m *Map) IsObjectInMap(object common.IGameObject) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.objects.Load(object.GetID())
	return ok
}

func (m *Map) GetSpawnManager() *SpawnManager {
	return m.spawnManager
}

func (m *Map) GetEventManager() *event.EventManager {
	return m.eventManager
}

func (m *Map) GetAIManager() *ai.AIManager {
	return m.aiManager
}

func (m *Map) GetBuffManager() *buff.BuffManager {
	return m.buffManager
}

func (m *Map) GetSkillManager() *skill.SkillManager {
	return m.skillManager
}

func (m *Map) GetTaskManager() *task.TaskManager {
	return m.taskManager
}

func (m *Map) GetInventoryManager() *item.InventoryManager {
	return m.inventoryManager
}

func (m *Map) GetCurrencyManager() *economy.CurrencyManager {
	return m.currencyManager
}

func (m *Map) GetTradeManager() *economy.TradeManager {
	return m.tradeManager
}

func (m *Map) GetAuctionManager() *economy.AuctionManager {
	return m.auctionManager
}

func (m *Map) GetShopManager() *economy.ShopManager {
	return m.shopManager
}

func (m *Map) AddSpawnPoint(spawnPoint *models.MapSpawnPoint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spawnPoints = append(m.spawnPoints, spawnPoint)
}

func (m *Map) GetSpawnPoints() []*models.MapSpawnPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.MapSpawnPoint, len(m.spawnPoints))
	copy(result, m.spawnPoints)
	return result
}

func (m *Map) AddTeleportPoint(teleportPoint *models.MapTeleportPoint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.teleportPoints = append(m.teleportPoints, teleportPoint)
}

func (m *Map) AddTeleportPointFromResource(id int32, x, y, z float32, targetMapID id.MapIdType, targetX, targetY, targetZ float32, name string, requiredLevel, requiredItem int32, isActive bool) {
	m.AddTeleportPoint(&models.MapTeleportPoint{
		ID:            id,
		X:             float64(x),
		Y:             float64(y),
		Z:             float64(z),
		TargetMapID:   int32(targetMapID),
		TargetX:       float64(targetX),
		TargetY:       float64(targetY),
		TargetZ:       float64(targetZ),
		Name:          name,
		RequiredLevel: requiredLevel,
		RequiredItem:  requiredItem,
		IsActive:      isActive,
	})
}

func (m *Map) GetTeleportPoints() []*models.MapTeleportPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.MapTeleportPoint, len(m.teleportPoints))
	copy(result, m.teleportPoints)
	return result
}

func (m *Map) AddBuilding(building *models.MapBuilding) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buildings = append(m.buildings, building)
}

func (m *Map) AddBuildingFromResource(id int32, x, y, z, width, height float32, buildingType, name string, level, hp, faction int32) {
	m.AddBuilding(&models.MapBuilding{
		ID:      id,
		X:       float64(x),
		Y:       float64(y),
		Z:       float64(z),
		Width:   float64(width),
		Height:  float64(height),
		Type:    buildingType,
		Name:    name,
		Level:   level,
		HP:      hp,
		Faction: faction,
	})
}

func (m *Map) GetBuildings() []*models.MapBuilding {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.MapBuilding, len(m.buildings))
	copy(result, m.buildings)
	return result
}

func (m *Map) AddEvent(event *models.MapEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *Map) GetEvents() []*models.MapEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.MapEvent, len(m.events))
	copy(result, m.events)
	return result
}

func (m *Map) AddResource(resource *models.MapResource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources = append(m.resources, resource)
}

func (m *Map) AddResourceFromResource(resourceID int32, resourceType string, x, y, z float32, respawnTime, itemID, quantity, level int32, isGathering bool) {
	m.AddResource(&models.MapResource{
		ResourceID:  resourceID,
		Type:        resourceType,
		X:           float64(x),
		Y:           float64(y),
		Z:           float64(z),
		RespawnTime: respawnTime,
		ItemID:      itemID,
		Quantity:    quantity,
		Level:       level,
		IsGathering: isGathering,
	})
}

func (m *Map) GetResources() []*models.MapResource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.MapResource, len(m.resources))
	copy(result, m.resources)
	return result
}

func (m *Map) GetTeleportPointByID(teleportPointID int32) *models.MapTeleportPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, tp := range m.teleportPoints {
		if tp.ID == teleportPointID {
			return tp
		}
	}
	return nil
}

func (m *Map) GetBuildingByID(buildingID int32) *models.MapBuilding {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, b := range m.buildings {
		if b.ID == buildingID {
			return b
		}
	}
	return nil
}

func (m *Map) GetEventByID(eventID int32) *models.MapEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.events {
		if e.EventID == eventID {
			return e
		}
	}
	return nil
}

func (m *Map) GetResourceByID(resourceID int32) *models.MapResource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.resources {
		if r.ResourceID == resourceID {
			return r
		}
	}
	return nil
}

func (m *Map) CreateDefaultEvents() {}

func (m *Map) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.objects.Range(func(objectID id.ObjectIdType, obj common.IGameObject) bool {
		m.objects.Delete(objectID)
		return true
	})

	m.players.Range(func(playerID id.PlayerIdType, value bool) bool {
		m.players.Delete(playerID)
		return true
	})

	m.spawnPoints = nil
	m.teleportPoints = nil
	m.buildings = nil
	m.events = nil
	m.resources = nil
}

func (m *Map) UpdateEvents() {
	if m.eventManager != nil {
		m.eventManager.UpdateEvents()
	}
}

func (m *Map) AddPlayer(playerID id.PlayerIdType, objectID id.ObjectIdType, x, y, z float32) error {
	position := common.Vector3{X: x, Y: y, Z: z}
	player := object.NewPlayer(objectID, playerID, "Player", position, 1)
	m.AddObject(player)
	return nil
}

func (m *Map) RemovePlayer(playerID id.PlayerIdType) {
	m.RemoveObject(id.ObjectIdType(playerID))
}

func (m *Map) MovePlayer(playerID id.PlayerIdType, objectID id.ObjectIdType, x, y, z float32) error {
	position := common.Vector3{X: x, Y: y, Z: z}
	return m.MoveObject(objectID, position)
}

func (m *Map) UpdatePlayers() {
	m.objects.Range(func(objID id.ObjectIdType, obj common.IGameObject) bool {
		if obj.GetType() == common.GameObjectTypePlayer {
			if player, ok := obj.(*object.Player); ok {
				player.UpdateStatus()
			}
		}
		return true
	})
}

func (m *Map) GetMapMode() MapMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mapMode
}

func (m *Map) SetMapMode(mode MapMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mapMode = mode
}

func (m *Map) GetServerGroupID() int32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.serverGroupID
}

func (m *Map) SetServerGroupID(groupID int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serverGroupID = groupID
}

func (m *Map) IsDungeon() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isDungeon
}

func (m *Map) SetIsDungeon(isDungeon bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isDungeon = isDungeon
}

func (m *Map) GetDungeonInstanceID() id.InstanceIdType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dungeonInstanceID
}

func (m *Map) SetDungeonInstanceID(instanceID id.InstanceIdType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dungeonInstanceID = instanceID
}

func (m *Map) GetDungeonManager() *dungeon.DungeonManager {
	return m.dungeonManager
}

func (m *Map) IsCrossServer() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mapMode == MapModeCrossGroup || m.mapMode == MapModeMirror
}
