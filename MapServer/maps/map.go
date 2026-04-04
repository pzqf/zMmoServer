package maps

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/models"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoServer/MapServer/maps/ai"
	"github.com/pzqf/zMmoServer/MapServer/maps/buff"
	"github.com/pzqf/zMmoServer/MapServer/maps/dungeon"
	"github.com/pzqf/zMmoServer/MapServer/maps/economy"
	"github.com/pzqf/zMmoServer/MapServer/maps/event"
	"github.com/pzqf/zMmoServer/MapServer/maps/item"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"github.com/pzqf/zMmoServer/MapServer/maps/skill"
	"github.com/pzqf/zMmoServer/MapServer/maps/task"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

// Region 地图区域
// 用于空间分区，管理区域内的游戏对象
type Region struct {
	regionID id.RegionIdType
	objects  *zMap.TypedMap[id.ObjectIdType, common.IGameObject]
}

// AddObject 添加游戏对象到区域
func (r *Region) AddObject(object common.IGameObject) {
	if object == nil {
		return
	}
	r.objects.Store(object.GetID(), object)
}

// RemoveObject 从区域移除游戏对象
func (r *Region) RemoveObject(objectID id.ObjectIdType) {
	r.objects.Delete(objectID)
}

// GetObject 获取区域中的游戏对象
func (r *Region) GetObject(objectID id.ObjectIdType) common.IGameObject {
	object, _ := r.objects.Load(objectID)
	return object
}

// GetObjects 获取区域中的所有游戏对象
func (r *Region) GetObjects() []common.IGameObject {
	var objects []common.IGameObject
	r.objects.Range(func(_ id.ObjectIdType, obj common.IGameObject) bool {
		objects = append(objects, obj)
		return true
	})
	return objects
}

// Map 游戏地图
// 管理地图中的所有游戏对象、区域、刷新点等
type Map struct {
	mu               sync.RWMutex
	mapID            id.MapIdType
	mapConfigID      int32
	name             string
	width            float32
	height           float32
	regionSize       float32
	objects          *zMap.TypedMap[id.ObjectIdType, common.IGameObject]
	regions          *zMap.TypedMap[id.RegionIdType, *Region]
	spawnPoints      []*models.MapSpawnPoint
	teleportPoints   []*models.MapTeleportPoint
	buildings        []*models.MapBuilding
	events           []*models.MapEvent
	resources        []*models.MapResource
	players          *zMap.TypedMap[id.PlayerIdType, bool]
	spawnManager     *SpawnManager
	eventManager     *event.EventManager
	aiManager        *ai.AIManager
	buffManager      *buff.BuffManager
	dungeonManager   *dungeon.DungeonManager
	skillManager     *skill.SkillManager
	taskManager      *task.TaskManager
	inventoryManager *item.InventoryManager
	currencyManager  *economy.CurrencyManager
	tradeManager     *economy.TradeManager
	auctionManager   *economy.AuctionManager
	shopManager      *economy.ShopManager
	connManager      *connection.ConnectionManager
	createdAt        time.Time
}

// NewMap 创建新地图
func NewMap(mapID id.MapIdType, mapConfigID int32, name string, width, height float32, connManager *connection.ConnectionManager) *Map {
	m := &Map{
		mapID:          mapID,
		mapConfigID:    mapConfigID,
		name:           name,
		width:          width,
		height:         height,
		regionSize:     50,
		objects:        zMap.NewTypedMap[id.ObjectIdType, common.IGameObject](),
		regions:        zMap.NewTypedMap[id.RegionIdType, *Region](),
		spawnPoints:    make([]*models.MapSpawnPoint, 0),
		teleportPoints: make([]*models.MapTeleportPoint, 0),
		buildings:      make([]*models.MapBuilding, 0),
		events:         make([]*models.MapEvent, 0),
		resources:      make([]*models.MapResource, 0),
		players:        zMap.NewTypedMap[id.PlayerIdType, bool](),
		connManager:    connManager,
		createdAt:      time.Now(),
	}

	m.spawnManager = NewSpawnManager(mapID, m)
	m.eventManager = event.NewEventManager()
	m.aiManager = ai.NewAIManager()
	m.aiManager.SetTableManager(tables.GetTableManager())
	m.buffManager = buff.NewBuffManager()
	m.buffManager.SetTableManager(tables.GetTableManager())
	m.dungeonManager = dungeon.NewDungeonManager()
	m.skillManager = skill.NewSkillManager()
	m.taskManager = task.NewTaskManager()
	m.inventoryManager = item.NewInventoryManager()
	m.currencyManager = economy.NewCurrencyManager()
	m.tradeManager = economy.NewTradeManager()
	m.auctionManager = economy.NewAuctionManager()
	m.shopManager = economy.NewShopManager()

	// 创建默认事件
	m.CreateDefaultEvents()

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

// GetWidth 获取地图宽度
func (m *Map) GetWidth() float32 {
	return m.width
}

// GetHeight 获取地图高度
func (m *Map) GetHeight() float32 {
	return m.height
}

// GetRegionSize 获取区域大小
func (m *Map) GetRegionSize() float32 {
	return m.regionSize
}

// SetMaxPlayers 设置地图最大玩家数
func (m *Map) SetMaxPlayers(maxPlayers int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 这里可以添加最大玩家数的设置逻辑
}

// SetDescription 设置地图描述
func (m *Map) SetDescription(description string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 这里可以添加地图描述的设置逻辑
}

// SetWeatherType 设置地图天气类型
func (m *Map) SetWeatherType(weatherType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 这里可以添加天气类型的设置逻辑
}

// SetMinLevel 设置地图最低等级
func (m *Map) SetMinLevel(minLevel int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 这里可以添加最低等级的设置逻辑
}

// SetMaxLevel 设置地图最高等级
func (m *Map) SetMaxLevel(maxLevel int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 这里可以添加最高等级的设置逻辑
}

// SetRegionSize 设置地图区域大小
func (m *Map) SetRegionSize(regionSize float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.regionSize = regionSize
}

// GetObjects 获取地图上的所有游戏对象
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

// GetObject 获取指定ID的游戏对象
func (m *Map) GetObject(objectID id.ObjectIdType) common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()
	obj, _ := m.objects.Load(objectID)
	return obj
}

// AddObject 添加游戏对象到地图
func (m *Map) AddObject(obj common.IGameObject) {
	if obj == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 添加到对象列表
	m.objects.Store(obj.GetID(), obj)

	// 添加到对应的区域
	regionID := m.getRegionID(obj.GetPosition())
	region, exists := m.regions.Load(regionID)
	if !exists {
		region = &Region{
			regionID: regionID,
			objects:  zMap.NewTypedMap[id.ObjectIdType, common.IGameObject](),
		}
		m.regions.Store(regionID, region)
	}
	region.AddObject(obj)

	// 如果是玩家，添加到玩家列表
	if obj.GetType() == common.GameObjectTypePlayer {
		if p, ok := obj.(*object.Player); ok {
			m.players.Store(p.GetPlayerID(), true)
		}
	}

	// 通知周围对象
	m.notifyObjectEnter(obj)
}

// RemoveObject 从地图移除游戏对象
func (m *Map) RemoveObject(objectID id.ObjectIdType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	obj, exists := m.objects.Load(objectID)
	if !exists {
		return
	}

	// 从对象列表移除
	m.objects.Delete(objectID)

	// 从区域移除
	regionID := m.getRegionID(obj.GetPosition())
	if region, exists := m.regions.Load(regionID); exists {
		region.RemoveObject(objectID)
	}

	// 如果是玩家，从玩家列表移除
	if obj.GetType() == common.GameObjectTypePlayer {
		if p, ok := obj.(*object.Player); ok {
			m.players.Delete(p.GetPlayerID())
		}
	}

	// 通知周围对象
	m.notifyObjectLeave(obj)
}

// MoveObject 移动游戏对象
func (m *Map) MoveObject(objectID id.ObjectIdType, newPos common.Vector3) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	object, exists := m.objects.Load(objectID)
	if !exists {
		return fmt.Errorf("object not found")
	}

	oldPos := object.GetPosition()

	// 更新对象位置
	object.SetPosition(newPos)

	// 检查是否需要跨区域
	oldRegionID := m.getRegionID(oldPos)
	newRegionID := m.getRegionID(newPos)

	if oldRegionID != newRegionID {
		// 从旧区域移除
		if oldRegion, exists := m.regions.Load(oldRegionID); exists {
			oldRegion.RemoveObject(objectID)
		}

		// 添加到新区域
		newRegion, exists := m.regions.Load(newRegionID)
		if !exists {
			newRegion = &Region{
				regionID: newRegionID,
				objects:  zMap.NewTypedMap[id.ObjectIdType, common.IGameObject](),
			}
			m.regions.Store(newRegionID, newRegion)
		}
		newRegion.AddObject(object)

		// 通知区域变化
		m.notifyRegionChange(object, oldRegionID, newRegionID)
	}

	// 通知周围对象移动
	m.notifyMovement(object, oldPos, newPos)

	return nil
}

// GetObjectsInRange 获取指定范围内的游戏对象
func (m *Map) GetObjectsInRange(position common.Vector3, radius float32) []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	radiusSquared := radius * radius
	objects := make([]common.IGameObject, 0)

	// 检查周围的区域
	centerRegionID := m.getRegionID(position)
	m.regions.Range(func(regionID id.RegionIdType, region *Region) bool {
		if m.isRegionInRange(regionID, centerRegionID, radius) {
			for _, obj := range region.GetObjects() {
				distance := obj.GetPosition().DistanceTo(position)
				if distance <= radiusSquared {
					objects = append(objects, obj)
				}
			}
		}
		return true
	})

	return objects
}

// GetPlayersInRange 获取指定范围内的玩家
func (m *Map) GetPlayersInRange(position common.Vector3, radius float32) []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	radiusSquared := radius * radius
	players := make([]common.IGameObject, 0)

	// 检查周围的区域
	centerRegionID := m.getRegionID(position)
	m.regions.Range(func(regionID id.RegionIdType, region *Region) bool {
		if m.isRegionInRange(regionID, centerRegionID, radius) {
			for _, obj := range region.GetObjects() {
				if obj.GetType() == common.GameObjectTypePlayer {
					distance := obj.GetPosition().DistanceTo(position)
					if distance <= radiusSquared {
						players = append(players, obj)
					}
				}
			}
		}
		return true
	})

	return players
}

// GetMonstersInRange 获取指定范围内的怪物
func (m *Map) GetMonstersInRange(position common.Vector3, radius float32) []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	radiusSquared := radius * radius
	monsters := make([]common.IGameObject, 0)

	// 检查周围的区域
	centerRegionID := m.getRegionID(position)
	m.regions.Range(func(regionID id.RegionIdType, region *Region) bool {
		if m.isRegionInRange(regionID, centerRegionID, radius) {
			for _, obj := range region.GetObjects() {
				if obj.GetType() == common.GameObjectTypeMonster {
					distance := obj.GetPosition().DistanceTo(position)
					if distance <= radiusSquared {
						monsters = append(monsters, obj)
					}
				}
			}
		}
		return true
	})

	return monsters
}

// GetNPCsInRange 获取指定范围内的NPC
func (m *Map) GetNPCsInRange(position common.Vector3, radius float32) []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	radiusSquared := radius * radius
	npcs := make([]common.IGameObject, 0)

	// 检查周围的区域
	centerRegionID := m.getRegionID(position)
	m.regions.Range(func(regionID id.RegionIdType, region *Region) bool {
		if m.isRegionInRange(regionID, centerRegionID, radius) {
			for _, obj := range region.GetObjects() {
				if obj.GetType() == common.GameObjectTypeNPC {
					distance := obj.GetPosition().DistanceTo(position)
					if distance <= radiusSquared {
						npcs = append(npcs, obj)
					}
				}
			}
		}
		return true
	})

	return npcs
}

// GetObjectsByType 获取指定类型的游戏对象
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

// GetPlayers 获取地图上的所有玩家
func (m *Map) GetPlayers() []*object.Player {
	m.mu.RLock()
	defer m.mu.RUnlock()

	players := make([]*object.Player, 0)
	m.objects.Range(func(objectID id.ObjectIdType, obj common.IGameObject) bool {
		if p, ok := obj.(*object.Player); ok {
			players = append(players, p)
		}
		return true
	})
	return players
}

// GetPlayerCount 获取地图上的玩家数量
func (m *Map) GetPlayerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var count int
	m.players.Range(func(playerID id.PlayerIdType, value bool) bool {
		count++
		return true
	})
	return count
}

// IsObjectInMap 检查游戏对象是否在地图中
func (m *Map) IsObjectInMap(object common.IGameObject) bool {
	if object == nil {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.objects.Load(object.GetID())
	return exists
}

// GetSpawnManager 获取刷怪管理器
func (m *Map) GetSpawnManager() *SpawnManager {
	return m.spawnManager
}

// GetEventManager 获取事件管理器
func (m *Map) GetEventManager() *event.EventManager {
	return m.eventManager
}

// GetAIManager 获取AI管理器
func (m *Map) GetAIManager() *ai.AIManager {
	return m.aiManager
}

// GetBuffManager 获取buff管理器
func (m *Map) GetBuffManager() *buff.BuffManager {
	return m.buffManager
}

// GetSkillManager 获取技能管理器
func (m *Map) GetSkillManager() *skill.SkillManager {
	return m.skillManager
}

// GetTaskManager 获取任务管理器
func (m *Map) GetTaskManager() *task.TaskManager {
	return m.taskManager
}

// GetInventoryManager 获取背包管理器
func (m *Map) GetInventoryManager() *item.InventoryManager {
	return m.inventoryManager
}

// GetCurrencyManager 获取货币管理器
func (m *Map) GetCurrencyManager() *economy.CurrencyManager {
	return m.currencyManager
}

// GetTradeManager 获取交易管理器
func (m *Map) GetTradeManager() *economy.TradeManager {
	return m.tradeManager
}

// GetAuctionManager 获取拍卖行管理器
func (m *Map) GetAuctionManager() *economy.AuctionManager {
	return m.auctionManager
}

// GetShopManager 获取商店管理器
func (m *Map) GetShopManager() *economy.ShopManager {
	return m.shopManager
}

// AddSpawnPoint 添加刷新点
func (m *Map) AddSpawnPoint(spawnPoint *models.MapSpawnPoint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spawnPoints = append(m.spawnPoints, spawnPoint)
}

// GetSpawnPoints 获取刷新点列表
func (m *Map) GetSpawnPoints() []*models.MapSpawnPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.spawnPoints
}

// AddTeleportPoint 添加传送点
func (m *Map) AddTeleportPoint(teleportPoint *models.MapTeleportPoint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.teleportPoints = append(m.teleportPoints, teleportPoint)
}

// AddTeleportPointFromResource 从资源添加传送点
func (m *Map) AddTeleportPointFromResource(id int32, x, y, z float32, targetMapID id.MapIdType, targetX, targetY, targetZ float32, name string, requiredLevel, requiredItem int32, isActive bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	teleportPoint := &models.MapTeleportPoint{
		ID:            id,
		MapID:         int32(m.mapID),
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
	}

	m.teleportPoints = append(m.teleportPoints, teleportPoint)
}

// GetTeleportPoints 获取传送点列表
func (m *Map) GetTeleportPoints() []*models.MapTeleportPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.teleportPoints
}

// AddBuilding 添加建筑
func (m *Map) AddBuilding(building *models.MapBuilding) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buildings = append(m.buildings, building)
}

// AddBuildingFromResource 从资源添加建筑
func (m *Map) AddBuildingFromResource(id int32, x, y, z, width, height float32, buildingType, name string, level, hp, faction int32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	building := &models.MapBuilding{
		ID:      id,
		MapID:   int32(m.mapID),
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
	}

	m.buildings = append(m.buildings, building)
}

// GetBuildings 获取建筑列表
func (m *Map) GetBuildings() []*models.MapBuilding {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.buildings
}

// AddEvent 添加事件
func (m *Map) AddEvent(event *models.MapEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

// GetEvents 获取事件列表
func (m *Map) GetEvents() []*models.MapEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.events
}

// AddResource 添加资源点
func (m *Map) AddResource(resource *models.MapResource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources = append(m.resources, resource)
}

// AddResourceFromResource 从资源添加资源点
func (m *Map) AddResourceFromResource(resourceID int32, resourceType string, x, y, z float32, respawnTime, itemID, quantity, level int32, isGathering bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	resource := &models.MapResource{
		ResourceID:  resourceID,
		MapID:       int32(m.mapID),
		Type:        resourceType,
		X:           float64(x),
		Y:           float64(y),
		Z:           float64(z),
		RespawnTime: respawnTime,
		ItemID:      itemID,
		Quantity:    quantity,
		Level:       level,
		IsGathering: isGathering,
	}

	m.resources = append(m.resources, resource)
}

// GetResources 获取资源点列表
func (m *Map) GetResources() []*models.MapResource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resources
}

// GetTeleportPointByID 根据ID获取传送点
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

// GetBuildingByID 根据ID获取建筑
func (m *Map) GetBuildingByID(buildingID int32) *models.MapBuilding {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, building := range m.buildings {
		if building.ID == buildingID {
			return building
		}
	}
	return nil
}

// GetEventByID 根据ID获取事件
func (m *Map) GetEventByID(eventID int32) *models.MapEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, event := range m.events {
		if event.EventID == eventID {
			return event
		}
	}
	return nil
}

// GetResourceByID 根据ID获取资源点
func (m *Map) GetResourceByID(resourceID int32) *models.MapResource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, resource := range m.resources {
		if resource.ResourceID == resourceID {
			return resource
		}
	}
	return nil
}

// HandleObjectInteraction 处理对象交互
func (m *Map) HandleObjectInteraction(player *object.Player, targetObject common.IGameObject) error {
	if targetObject == nil {
		return fmt.Errorf("target object not found")
	}

	switch targetObject.GetType() {
	case common.GameObjectTypeNPC:
		return m.handleNPCInteraction(player, targetObject)
	case common.GameObjectTypeMonster:
		return m.handleMonsterInteraction(player, targetObject)
	case common.GameObjectTypeItem:
		return m.handleItemInteraction(player, targetObject)
	default:
		return fmt.Errorf("unsupported object type")
	}
}

// handleNPCInteraction 处理与NPC的交互
func (m *Map) handleNPCInteraction(player *object.Player, npc common.IGameObject) error {
	// 这里可以根据NPC的类型和对话内容进行处理
	zLog.Info("Player interacted with NPC",
		zap.Int64("player_id", int64(player.GetPlayerID())),
		zap.Int64("npc_id", int64(npc.GetID())))

	return nil
}

// handleMonsterInteraction 处理与怪物的交互
func (m *Map) handleMonsterInteraction(player *object.Player, monster common.IGameObject) error {
	// 这里可以处理攻击怪物的逻辑
	zLog.Info("Player attacked monster",
		zap.Int64("player_id", int64(player.GetPlayerID())),
		zap.Int64("monster_id", int64(monster.GetID())))

	return nil
}

// handleItemInteraction 处理与物品的交互
func (m *Map) handleItemInteraction(player *object.Player, item common.IGameObject) error {
	// 这里可以处理拾取物品的逻辑
	zLog.Info("Player picked up item",
		zap.Int64("player_id", int64(player.GetPlayerID())),
		zap.Int64("item_id", int64(item.GetID())))

	// 从地图上移除物品
	m.RemoveObject(item.GetID())

	// 添加到玩家背包
	if inventoryItem, ok := item.(*object.Item); ok {
		return m.AddItem(player, inventoryItem.GetItemID(), 1)
	}

	return nil
}

// HandleSkillUse 处理技能释放
func (m *Map) HandleSkillUse(caster *object.Player, skillID int32, targetID id.ObjectIdType, targetPos common.Vector3) error {
	// 检查技能是否存在
	skillConfig := m.skillManager.GetSkillConfig(skillID)
	if skillConfig == nil {
		return fmt.Errorf("skill not found")
	}

	// 检查技能冷却
	if caster.IsSkillInCooldown(skillID) {
		return fmt.Errorf("skill is on cooldown")
	}

	// 检查技能消耗
	if caster.GetMana() < skillConfig.ManaCost {
		return fmt.Errorf("not enough mana")
	}

	// 检查目标
	var target common.IGameObject
	if targetID != 0 {
		target = m.GetObject(targetID)
		if target == nil {
			return fmt.Errorf("target not found")
		}

		// 检查目标是否在技能范围内
		if !m.ValidateTarget(caster, target, skillConfig.Range) {
			return fmt.Errorf("target out of range")
		}
	} else {
		// 检查技能释放位置是否在地图范围内
		if !m.IsPositionInMap(targetPos) {
			return fmt.Errorf("invalid skill position")
		}
	}
	// 扣除技能消耗
	caster.SetMana(caster.GetMana() - skillConfig.ManaCost)

	// 设置技能冷却
	caster.SetSkillCooldown(skillID, time.Duration(skillConfig.Cooldown)*time.Millisecond)

	// 创建技能对象
	skillObjectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)
	newSkill := skill.NewSkill(
		skillObjectID,
		skillID,
		caster.GetID(),
		targetID,
		targetPos,
		skillConfig.Damage,
		skillConfig.Range,
		skillConfig.Type,
		time.Duration(skillConfig.Duration)*time.Millisecond,
		time.Duration(skillConfig.Cooldown)*time.Millisecond,
	)

	// 设置技能等级和施法者攻击力
	newSkill.SetLevel(1) // 暂时使用等级1
	newSkill.SetCasterAttack(caster.GetAttack())

	// 添加技能到技能管理器
	m.skillManager.AddSkill(newSkill)

	// 处理技能效果
	m.handleSkillEffect(newSkill)

	// 通知周围玩家技能释放
	// 由于我们现在使用统一的协议，这里暂时注释掉技能通知
	// 后续需要根据新的协议结构重新实现
	// m.notifySkillUse(caster, newSkill)

	// 发送技能释放响应给施法者
	// 由于我们现在使用统一的协议，这里暂时注释掉技能响应
	// 后续需要根据新的协议结构重新实现
	// m.sendSkillUseResponse(caster, newSkill, nil)

	zLog.Debug("Skill used",
		zap.Int64("caster_id", int64(caster.GetID())),
		zap.Int32("skill_id", skillID),
		zap.Int64("target_id", int64(targetID)),
		zap.Float32("x", targetPos.X),
		zap.Float32("y", targetPos.Y))

	return nil
}

// handleSkillEffect 处理技能效果
func (m *Map) handleSkillEffect(skill *skill.Skill) {
	// 获取技能范围内的对象
	objects := m.GetObjectsInRange(skill.GetPosition(), skill.GetRange())

	for _, obj := range objects {
		// 跳过施法者自己
		if obj.GetID() == skill.GetCasterID() {
			continue
		}

		// 检查目标类型是否符合技能要求
		targetType := obj.GetType()
		validTarget := false

		switch skill.GetEffectType() {
		case 1: // 伤害
			validTarget = targetType == common.GameObjectTypeMonster || targetType == common.GameObjectTypePlayer
		case 2: // 治疗
			validTarget = targetType == common.GameObjectTypePlayer
		case 3: // 增益
			validTarget = targetType == common.GameObjectTypePlayer
		case 4: // 减益
			validTarget = targetType == common.GameObjectTypeMonster || targetType == common.GameObjectTypePlayer
		}

		if !validTarget {
			continue
		}

		// 应用技能效果
		switch skill.GetEffectType() {
		case 1: // 伤害
			if obj.GetType() == common.GameObjectTypeMonster {
				if monster, ok := obj.(*object.Monster); ok {
					damage := skill.CalculateDamageToMonster(monster)
					monster.TakeDamage(damage)
					if monster.GetHealth() <= 0 {
						// 怪物死亡
						m.handleMonsterDeath(monster, skill.GetCasterID())
					}
				}
			} else if obj.GetType() == common.GameObjectTypePlayer {
				if player, ok := obj.(*object.Player); ok {
					damage := skill.CalculateDamage(player)
					player.TakeDamage(damage)
					if player.GetHealth() <= 0 {
						// 玩家死亡
						m.handlePlayerDeath(player)
					}
				}
			}
		case 2: // 治疗
			if player, ok := obj.(*object.Player); ok {
				healAmount := skill.GetDamage() // 这里用伤害值作为治疗量
				newHealth := player.GetHealth() + healAmount
				if newHealth > player.GetMaxHealth() {
					newHealth = player.GetMaxHealth()
				}
				player.SetHealth(newHealth)
			}
		case 3: // 增益
			// 这里可以添加增益效果
		case 4: // 减益
			// 这里可以添加减益效果
		}
	}
}

// handleMonsterDeath 处理怪物死亡
func (m *Map) handleMonsterDeath(monster *object.Monster, killerID id.ObjectIdType) {
	// 从地图上移除怪物
	m.RemoveObject(monster.GetID())

	// 查找 killer
	killer := m.GetObject(killerID)
	if killer != nil && killer.GetType() == common.GameObjectTypePlayer {
		if player, ok := killer.(*object.Player); ok {
			// 给玩家经验和掉落
			exp := monster.GetExp()
			player.AddExp(exp)

			// 掉落物品
			m.dropItems(monster.GetPosition(), monster.GetLevel())

			zLog.Info("Monster killed",
				zap.Int64("player_id", int64(player.GetPlayerID())),
				zap.Int64("monster_id", int64(monster.GetID())),
				zap.Int64("exp_reward", exp))
		}
	}
}

// handlePlayerDeath 处理玩家死亡
func (m *Map) handlePlayerDeath(player *object.Player) {
	// 这里可以添加玩家死亡的处理逻辑
	zLog.Info("Player died", zap.Int64("player_id", int64(player.GetPlayerID())))

	// 可以将玩家传送到复活点
	// 或者添加死亡惩罚
}

// dropItems 掉落物品
func (m *Map) dropItems(position common.Vector3, monsterLevel int32) {
	// 这里可以添加物品掉落逻辑
	// 暂时简单实现
	if rand.Float32() < 0.5 { // 50% 掉落率
		itemID := int32(1) // 简单起见，只掉落ID为1的物品
		itemObjectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)
		newItem := object.NewItem(itemObjectID, itemID, "Test Item", position, 1, object.ItemTypeConsumable, object.ItemRarityCommon)
		m.AddObject(newItem)

		zLog.Debug("Item dropped", zap.Int32("item_id", itemID), zap.Float32("x", position.X), zap.Float32("y", position.Y))
	}
}

// HandleTaskAccept 处理任务接取
func (m *Map) HandleTaskAccept(player *object.Player, taskID int32) error {
	if m.taskManager == nil {
		return fmt.Errorf("task manager not initialized")
	}

	return m.taskManager.AcceptTask(player.GetPlayerID(), taskID)
}

// HandleTaskReward 处理任务奖励领取
func (m *Map) HandleTaskReward(player *object.Player, taskID int32) error {
	if m.taskManager == nil {
		return fmt.Errorf("task manager not initialized")
	}

	// 领取任务奖励
	err := m.taskManager.RewardTask(player.GetPlayerID(), taskID)
	if err != nil {
		return err
	}

	// 发放任务奖励
	taskConfig := m.taskManager.GetTaskConfig(taskID)
	if taskConfig != nil {
		for _, reward := range taskConfig.Rewards {
			switch reward.Type {
			case "exp":
				// 增加经验
				player.AddExp(int64(reward.Count))
			case "gold":
				// 增加金币
				err := m.AddCurrency(player, economy.CurrencyTypeGold, int64(reward.Count))
				if err != nil {
					return err
				}
			case "item":
				// 增加物品
				err := m.AddItem(player, reward.Target, reward.Count)
				if err != nil {
					return err
				}
			case "skill_point":
				// 增加技能点
				// 这里需要实现增加技能点的逻辑
			}
		}
	}

	return nil
}

// UpdateTaskProgress 更新任务进度
func (m *Map) UpdateTaskProgress(player *object.Player, conditionType string, target int32, count int32) {
	if m.taskManager != nil {
		m.taskManager.UpdateTaskProgress(player.GetPlayerID(), conditionType, target, count)
	}
}

// GetPlayerTasks 获取玩家的任务列表
func (m *Map) GetPlayerTasks(player *object.Player) []*task.PlayerTask {
	if m.taskManager == nil {
		return []*task.PlayerTask{}
	}

	return m.taskManager.GetPlayerTasks(player.GetPlayerID())
}

// GetAvailableTasks 获取玩家可接取的任务
func (m *Map) GetAvailableTasks(player *object.Player) []*task.TaskConfig {
	if m.taskManager == nil {
		return []*task.TaskConfig{}
	}

	return m.taskManager.GetAvailableTasks(player.GetPlayerID(), player.GetLevel())
}

// AddItem 添加物品到背包
func (m *Map) AddItem(player *object.Player, itemID int32, count int32) error {
	if m.inventoryManager == nil {
		return fmt.Errorf("inventory manager not initialized")
	}

	return m.inventoryManager.AddItem(player.GetPlayerID(), itemID, count)
}

// RemoveItem 从背包中移除物品
func (m *Map) RemoveItem(player *object.Player, itemID int32, count int32) error {
	if m.inventoryManager == nil {
		return fmt.Errorf("inventory manager not initialized")
	}

	return m.inventoryManager.RemoveItem(player.GetPlayerID(), itemID, count)
}

// GetItemCount 获取背包中物品的数量
func (m *Map) GetItemCount(player *object.Player, itemID int32) int32 {
	if m.inventoryManager == nil {
		return 0
	}

	return m.inventoryManager.GetItemCount(player.GetPlayerID(), itemID)
}

// GetInventoryItems 获取背包物品列表
func (m *Map) GetInventoryItems(player *object.Player) []*item.InventoryItem {
	if m.inventoryManager == nil {
		return []*item.InventoryItem{}
	}

	return m.inventoryManager.GetInventoryItems(player.GetPlayerID())
}

// GetCurrency 获取玩家货币数量
func (m *Map) GetCurrency(player *object.Player, currencyType economy.CurrencyType) int64 {
	if m.currencyManager == nil {
		return 0
	}

	return m.currencyManager.GetCurrency(player.GetPlayerID(), currencyType)
}

// AddCurrency 增加玩家货币
func (m *Map) AddCurrency(player *object.Player, currencyType economy.CurrencyType, amount int64) error {
	if m.currencyManager == nil {
		return fmt.Errorf("currency manager not initialized")
	}

	return m.currencyManager.AddCurrency(player.GetPlayerID(), currencyType, amount)
}

// RemoveCurrency 减少玩家货币
func (m *Map) RemoveCurrency(player *object.Player, currencyType economy.CurrencyType, amount int64) error {
	if m.currencyManager == nil {
		return fmt.Errorf("currency manager not initialized")
	}

	return m.currencyManager.RemoveCurrency(player.GetPlayerID(), currencyType, amount)
}

// HasEnoughCurrency 检查玩家是否有足够的货币
func (m *Map) HasEnoughCurrency(player *object.Player, currencyType economy.CurrencyType, amount int64) bool {
	if m.currencyManager == nil {
		return false
	}

	return m.currencyManager.HasEnoughCurrency(player.GetPlayerID(), currencyType, amount)
}

// StartTrade 开始交易
func (m *Map) StartTrade(initiator, target *object.Player) (*economy.Trade, error) {
	if m.tradeManager == nil {
		return nil, fmt.Errorf("trade manager not initialized")
	}

	return m.tradeManager.InitiateTrade(initiator.GetPlayerID(), target.GetPlayerID())
}

// AddTradeItem 添加交易物品
func (m *Map) AddTradeItem(player *object.Player, tradeID int64, itemID int32, count int32) error {
	if m.tradeManager == nil {
		return fmt.Errorf("trade manager not initialized")
	}

	return m.tradeManager.AddTradeItem(tradeID, player.GetPlayerID(), itemID, count)
}

// AddTradeCurrency 添加交易货币
func (m *Map) AddTradeCurrency(player *object.Player, tradeID int64, currencyType economy.CurrencyType, amount int64) error {
	if m.tradeManager == nil {
		return fmt.Errorf("trade manager not initialized")
	}

	return m.tradeManager.AddTradeCurrency(tradeID, player.GetPlayerID(), currencyType, amount)
}

// AcceptTrade 接受交易
func (m *Map) AcceptTrade(player *object.Player, tradeID int64) error {
	if m.tradeManager == nil {
		return fmt.Errorf("trade manager not initialized")
	}

	return m.tradeManager.AcceptTrade(tradeID, player.GetPlayerID())
}

// CancelTrade 取消交易
func (m *Map) CancelTrade(player *object.Player, tradeID int64) error {
	if m.tradeManager == nil {
		return fmt.Errorf("trade manager not initialized")
	}

	return m.tradeManager.CancelTrade(tradeID, player.GetPlayerID())
}

// CreateAuction 创建拍卖
func (m *Map) CreateAuction(player *object.Player, itemID int32, count int32, startingPrice int64, duration int32, currencyType economy.CurrencyType) (int64, error) {
	if m.auctionManager == nil {
		return 0, fmt.Errorf("auction manager not initialized")
	}

	auction, err := m.auctionManager.CreateAuction(
		player.GetPlayerID(),
		itemID,
		count,
		startingPrice,
		time.Duration(duration)*time.Second,
		currencyType,
	)
	if err != nil {
		return 0, err
	}

	return auction.AuctionID, nil
}

// PlaceBid 出价
func (m *Map) PlaceBid(auctionID int64, player *object.Player, bidPrice int64) error {
	if m.auctionManager == nil {
		return fmt.Errorf("auction manager not initialized")
	}

	return m.auctionManager.PlaceBid(auctionID, player.GetPlayerID(), bidPrice)
}

// CancelAuction 取消拍卖
func (m *Map) CancelAuction(auctionID int64, player *object.Player) error {
	if m.auctionManager == nil {
		return fmt.Errorf("auction manager not initialized")
	}

	return m.auctionManager.CancelAuction(auctionID, player.GetPlayerID())
}

// BuyItem 购买物品
func (m *Map) BuyItem(shopID, itemID, count int32, player *object.Player) (int64, economy.CurrencyType, error) {
	if m.shopManager == nil {
		return 0, 0, fmt.Errorf("shop manager not initialized")
	}

	return m.shopManager.BuyItem(shopID, itemID, count, player.GetLevel(), player.GetClass())
}

// LoadShopConfig 加载商店配置
func (m *Map) LoadShopConfig(filePath string) error {
	if m.shopManager != nil {
		return m.shopManager.LoadShopConfig(filePath)
	}
	return nil
}

// GetTargetInRange 获取指定范围内的目标
func (m *Map) GetTargetInRange(position common.Vector3, skillRange float32, casterID id.ObjectIdType, targetTypes []common.GameObjectType) []common.IGameObject {
	objects := m.GetObjectsInRange(position, skillRange)
	targets := make([]common.IGameObject, 0)

	for _, obj := range objects {
		// 跳过施法者自己
		if obj.GetID() == casterID {
			continue
		}

		// 检查目标类型是否符合要求
		objType := obj.GetType()
		for _, targetType := range targetTypes {
			if objType == targetType {
				targets = append(targets, obj)
				break
			}
		}
	}

	return targets
}

// GetNearestTarget 获取最近的目标
func (m *Map) GetNearestTarget(position common.Vector3, skillRange float32, casterID id.ObjectIdType, targetTypes []common.GameObjectType) common.IGameObject {
	targets := m.GetTargetInRange(position, skillRange, casterID, targetTypes)
	if len(targets) == 0 {
		return nil
	}

	var nearestTarget common.IGameObject
	minDistance := float32(math.MaxFloat32)

	for _, target := range targets {
		targetPos := target.GetPosition()
		distance := m.CalculateDistance(position, targetPos)
		if distance < minDistance {
			minDistance = distance
			nearestTarget = target
		}
	}

	return nearestTarget
}

// isRegionInRange 检查区域是否在范围内
func (m *Map) isRegionInRange(regionID, centerRegionID id.RegionIdType, radius float32) bool {
	// 这里可以实现区域是否在范围内的逻辑
	// 暂时简单实现，总是返回true
	return true
}

// notifyObjectEnter 通知对象进入
func (m *Map) notifyObjectEnter(obj common.IGameObject) {
	// 这里可以实现通知逻辑
}

// notifyObjectLeave 通知对象离开
func (m *Map) notifyObjectLeave(obj common.IGameObject) {
	// 这里可以实现通知逻辑
}

// notifyMovement 通知对象移动
func (m *Map) notifyMovement(obj common.IGameObject, oldPos, newPos common.Vector3) {
	// 这里可以实现通知逻辑
}

// notifyRegionChange 通知区域变化
func (m *Map) notifyRegionChange(obj common.IGameObject, oldRegionID, newRegionID id.RegionIdType) {
	// 这里可以实现通知逻辑
}

// getRegionID 根据位置获取区域ID
func (m *Map) getRegionID(position common.Vector3) id.RegionIdType {
	// 这里可以实现根据位置计算区域ID的逻辑
	// 暂时简单实现，返回固定值
	return 1
}

// IsPositionInMap 检查位置是否在地图范围内
func (m *Map) IsPositionInMap(position common.Vector3) bool {
	// 这里可以实现检查位置是否在地图范围内的逻辑
	// 暂时简单实现，总是返回true
	return true
}

// ValidateTarget 验证目标是否有效
func (m *Map) ValidateTarget(caster *object.Player, target common.IGameObject, skillRange float32) bool {
	// 这里可以实现验证目标是否有效的逻辑
	// 暂时简单实现，总是返回true
	return true
}

// CalculateDistance 计算两点之间的距离
func (m *Map) CalculateDistance(pos1, pos2 common.Vector3) float32 {
	// 这里可以实现计算距离的逻辑
	// 暂时简单实现，返回固定值
	return 0
}

// CreateDefaultEvents 创建默认事件
func (m *Map) CreateDefaultEvents() {
	// 这里可以实现创建默认事件的逻辑
}

// Cleanup 清理地图资源
func (m *Map) Cleanup() {
	// 清理地图资源的逻辑
	m.mu.Lock()
	defer m.mu.Unlock()

	// 清理所有对象
	m.objects.Range(func(objectID id.ObjectIdType, obj common.IGameObject) bool {
		m.objects.Delete(objectID)
		return true
	})

	// 清理所有区域
	m.regions.Range(func(regionID id.RegionIdType, region *Region) bool {
		m.regions.Delete(regionID)
		return true
	})

	// 清理玩家列表
	m.players.Range(func(playerID id.PlayerIdType, value bool) bool {
		m.players.Delete(playerID)
		return true
	})

	// 清理其他资源
	m.spawnPoints = nil
	m.teleportPoints = nil
	m.buildings = nil
	m.events = nil
	m.resources = nil
}

// UpdateEvents 更新地图事件
func (m *Map) UpdateEvents() {
	// 更新地图事件的逻辑
	if m.eventManager != nil {
		m.eventManager.UpdateEvents()
	}
}

// AddPlayer 添加玩家到地图
func (m *Map) AddPlayer(playerID id.PlayerIdType, objectID id.ObjectIdType, x, y, z float32) error {
	position := common.Vector3{X: x, Y: y, Z: z}
	player := object.NewPlayer(objectID, playerID, "Player", position, 1)
	m.AddObject(player)
	return nil
}

// RemovePlayer 移除玩家
func (m *Map) RemovePlayer(playerID id.PlayerIdType) {
	m.RemoveObject(id.ObjectIdType(playerID))
}

// MovePlayer 移动玩家
func (m *Map) MovePlayer(playerID id.PlayerIdType, objectID id.ObjectIdType, x, y, z float32) error {
	// 移动玩家的逻辑
	position := common.Vector3{X: x, Y: y, Z: z}
	return m.MoveObject(objectID, position)
}

// AttackTarget 攻击目标
func (m *Map) AttackTarget(playerID id.PlayerIdType, objectID id.ObjectIdType, targetID id.ObjectIdType) (int64, int64, error) {
	// 攻击目标的逻辑
	object := m.GetObject(objectID)
	if object == nil {
		return 0, 0, fmt.Errorf("attacker not found")
	}

	target := m.GetObject(targetID)
	if target == nil {
		return 0, 0, fmt.Errorf("target not found")
	}

	// 简单实现，返回固定值
	return 10, 0, nil
}

// UpdateSkills 更新地图技能
func (m *Map) UpdateSkills() {
	// 更新地图技能的逻辑
	if m.skillManager != nil {
		m.skillManager.Update()
	}
}
