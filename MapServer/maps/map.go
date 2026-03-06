package maps

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoServer/MapServer/maps/achievement"
	"github.com/pzqf/zMmoServer/MapServer/maps/activity"
	"github.com/pzqf/zMmoServer/MapServer/maps/ai"
	"github.com/pzqf/zMmoServer/MapServer/maps/buff"
	"github.com/pzqf/zMmoServer/MapServer/maps/dungeon"
	"github.com/pzqf/zMmoServer/MapServer/maps/economy"
	"github.com/pzqf/zMmoServer/MapServer/maps/event"
	"github.com/pzqf/zMmoServer/MapServer/maps/item"
	"github.com/pzqf/zMmoServer/MapServer/maps/mount"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"github.com/pzqf/zMmoServer/MapServer/maps/pet"
	"github.com/pzqf/zMmoServer/MapServer/maps/skill"
	"github.com/pzqf/zMmoServer/MapServer/maps/social"
	"github.com/pzqf/zMmoServer/MapServer/maps/task"
	pb "github.com/pzqf/zMmoServer/resources/protocol/net/protocol"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/config/models"
	"github.com/pzqf/zMmoShared/config/tables"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// Region 地图区域
// 用于空间分区，管理区域内的游戏对象
type Region struct {
	mu       sync.RWMutex
	regionID id.RegionIdType
	objects  map[id.ObjectIdType]common.IGameObject
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

// Map 游戏地图
// 管理地图中的所有游戏对象、区域、刷新点等
type Map struct {
	mu                 sync.RWMutex
	mapID              id.MapIdType
	mapConfigID        int32
	name               string
	width              float32
	height             float32
	regionSize         float32
	objects            map[id.ObjectIdType]common.IGameObject
	regions            map[id.RegionIdType]*Region
	spawnPoints        []*models.MapSpawnPoint
	teleportPoints     []*models.MapTeleportPoint
	buildings          []*models.MapBuilding
	events             []*models.MapEvent
	resources          []*models.MapResource
	players            map[id.PlayerIdType]bool
	spawnManager       *SpawnManager
	eventManager       *event.EventManager
	aiManager          *ai.AIManager
	buffManager        *buff.BuffManager
	activityManager    *activity.ActivityManager
	dungeonManager     *dungeon.DungeonManager
	skillManager       *skill.SkillManager
	taskManager        *task.TaskManager
	inventoryManager   *item.InventoryManager
	teamManager        *social.TeamManager
	guildManager       *social.GuildManager
	currencyManager    *economy.CurrencyManager
	tradeManager       *economy.TradeManager
	auctionManager     *economy.AuctionManager
	shopManager        *economy.ShopManager
	achievementManager *achievement.AchievementManager
	petManager         *pet.PetManager
	mountManager       *mount.MountManager
	connManager        *connection.ConnectionManager
	createdAt          time.Time
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
		objects:        make(map[id.ObjectIdType]common.IGameObject),
		regions:        make(map[id.RegionIdType]*Region),
		spawnPoints:    make([]*models.MapSpawnPoint, 0),
		teleportPoints: make([]*models.MapTeleportPoint, 0),
		buildings:      make([]*models.MapBuilding, 0),
		events:         make([]*models.MapEvent, 0),
		resources:      make([]*models.MapResource, 0),
		players:        make(map[id.PlayerIdType]bool),
		connManager:    connManager,
		createdAt:      time.Now(),
	}

	m.spawnManager = NewSpawnManager(mapID, m)
	m.eventManager = event.NewEventManager()
	m.aiManager = ai.NewAIManager()
	m.aiManager.SetTableManager(tables.GetTableManager())
	m.buffManager = buff.NewBuffManager()
	m.buffManager.SetTableManager(tables.GetTableManager())
	m.activityManager = activity.NewActivityManager()
	m.dungeonManager = dungeon.NewDungeonManager()
	m.skillManager = skill.NewSkillManager()
	m.taskManager = task.NewTaskManager()
	m.inventoryManager = item.NewInventoryManager()
	m.teamManager = social.NewTeamManager()
	m.guildManager = social.NewGuildManager()
	m.currencyManager = economy.NewCurrencyManager()
	m.tradeManager = economy.NewTradeManager()
	m.auctionManager = economy.NewAuctionManager()
	m.shopManager = economy.NewShopManager()
	m.achievementManager = achievement.NewAchievementManager()
	m.petManager = pet.NewPetManager()
	m.mountManager = mount.NewMountManager()

	// 创建默认事件
	m.CreateDefaultEvents()

	return m
}

// InitSpawnSystem 初始化刷怪系统
func (m *Map) InitSpawnSystem() {
	if m.spawnManager != nil {
		m.spawnManager.Init(m.mapConfigID)
	}
}

// GetSpawnManager 获取刷新管理器
func (m *Map) GetSpawnManager() *SpawnManager {
	return m.spawnManager
}

// SpawnDroppedItem 生成掉落物品到地图
func (m *Map) SpawnDroppedItem(itemID int32, itemName string, pos common.Vector3, quantity int32, itemType object.ItemType, rarity object.ItemRarity) *object.Item {
	objectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)

	item := object.NewItem(objectID, itemID, itemName, pos, quantity, itemType, rarity)

	m.AddObject(item)

	zLog.Debug("Spawned dropped item",
		zap.Int32("item_id", itemID),
		zap.Int64("object_id", int64(objectID)),
		zap.Float32("x", pos.X),
		zap.Float32("y", pos.Y))

	return item
}

// GetID 获取地图ID
func (m *Map) GetID() id.MapIdType {
	return m.mapID
}

// GetName 获取地图名称
func (m *Map) GetName() string {
	return m.name
}

// IsInMap 检查位置是否在地图范围内
func (m *Map) IsInMap(pos common.Vector3) bool {
	return pos.X >= 0 && pos.X < m.width && pos.Y >= 0 && pos.Y < m.height
}

// IsObjectInMap 检查对象是否在地图上
func (m *Map) IsObjectInMap(obj common.IGameObject) bool {
	if obj == nil {
		return false
	}

	pos := obj.GetPosition()
	return m.IsInMap(pos)
}

// CalculateDistance 计算两点之间的距离
func (m *Map) CalculateDistance(pos1, pos2 common.Vector3) float32 {
	dx := pos2.X - pos1.X
	dy := pos2.Y - pos1.Y
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// CheckCollision 检查位置是否有碰撞（是否有阻挡物体）
func (m *Map) CheckCollision(pos common.Vector3, excludeID id.ObjectIdType) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	collisionRadius := float32(2.0) // 碰撞半径

	for _, obj := range m.objects {
		if obj.GetID() == excludeID {
			continue
		}

		// 怪物和NPC会产生碰撞
		objType := obj.GetType()
		if objType == common.GameObjectTypeMonster || objType == common.GameObjectTypeNPC {
			objPos := obj.GetPosition()
			dx := pos.X - objPos.X
			dy := pos.Y - objPos.Y
			distSq := dx*dx + dy*dy
			if distSq <= collisionRadius*collisionRadius {
				return true
			}
		}
	}

	return false
}

// GetObjectsInRange 获取指定范围内的游戏对象
func (m *Map) GetObjectsInRange(center common.Vector3, radius float32) []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	objects := make([]common.IGameObject, 0)

	// 计算需要检查的区域范围
	radiusSq := radius * radius
	startRegionX := int((center.X - radius) / m.regionSize)
	startRegionY := int((center.Y - radius) / m.regionSize)
	endRegionX := int((center.X + radius) / m.regionSize)
	endRegionY := int((center.Y + radius) / m.regionSize)

	// 只检查周围的区域，而不是所有对象
	for regionX := startRegionX; regionX <= endRegionX; regionX++ {
		for regionY := startRegionY; regionY <= endRegionY; regionY++ {
			regionID := id.RegionIdType(regionX*1000 + regionY)
			if region, exists := m.regions[regionID]; exists {
				region.mu.RLock()
				for _, obj := range region.objects {
					distanceSq := obj.GetPosition().DistanceTo(center)
					if distanceSq <= radiusSq {
						objects = append(objects, obj)
					}
				}
				region.mu.RUnlock()
			}
		}
	}

	return objects
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

	// 添加到对应的区域
	regionID := m.getRegionID(object.GetPosition())
	if _, exists := m.regions[regionID]; !exists {
		m.regions[regionID] = &Region{
			regionID: regionID,
			objects:  make(map[id.ObjectIdType]common.IGameObject),
		}
	}

	m.regions[regionID].AddObject(object)
}

// RemoveObject 从地图移除游戏对象
func (m *Map) RemoveObject(objectID id.ObjectIdType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.objects, objectID)

	// 从区域中移除
	for regionID, region := range m.regions {
		if _, exists := region.objects[objectID]; exists {
			delete(m.regions[regionID].objects, objectID)
			break
		}
	}

	// 通知刷新管理器移除对象
	if m.spawnManager != nil {
		m.spawnManager.RemoveObject(objectID)
	}
}

// MoveObject 移动游戏对象
func (m *Map) MoveObject(obj common.IGameObject, targetPos common.Vector3) error {
	// 验证目标位置是否在地图范围内
	if !m.IsInMap(targetPos) {
		return fmt.Errorf("target position out of map bounds")
	}

	// 验证移动距离是否合理
	oldPos := obj.GetPosition()
	distance := m.CalculateDistance(oldPos, targetPos)
	maxMoveDistance := float32(50.0) // 最大单次移动距离
	if distance > maxMoveDistance {
		return fmt.Errorf("target position too far")
	}

	// 碰撞检测
	if m.CheckCollision(targetPos, obj.GetID()) {
		return fmt.Errorf("target position blocked")
	}

	oldRegionID := m.getRegionID(oldPos)
	newRegionID := m.getRegionID(targetPos)

	// 计算移动前的AOI对象
	oldAOIObjects := m.GetObjectsInRange(oldPos, 100) // 100为AOI范围

	// 同一区域内移动
	if oldRegionID == newRegionID {
		obj.SetPosition(targetPos)
		// 计算移动后的AOI对象
		newAOIObjects := m.GetObjectsInRange(targetPos, 100)
		// 处理AOI变化
		m.handleAOIChange(obj, oldAOIObjects, newAOIObjects)
		// 通知周围玩家移动
		m.notifyMovement(obj, oldPos, targetPos)
		// 触发事件（如果是玩家）
		if p, ok := obj.(*object.Player); ok {
			m.TriggerEvents(p)
		}
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 从旧区域移除
	if _, exists := m.regions[oldRegionID]; exists {
		delete(m.regions[oldRegionID].objects, obj.GetID())
	}

	// 添加到新区域
	if _, exists := m.regions[newRegionID]; !exists {
		m.regions[newRegionID] = &Region{
			regionID: newRegionID,
			objects:  make(map[id.ObjectIdType]common.IGameObject),
		}
	}

	m.regions[newRegionID].AddObject(obj)
	obj.SetPosition(targetPos)

	// 计算移动后的AOI对象
	newAOIObjects := m.GetObjectsInRange(targetPos, 100)
	// 处理AOI变化
	m.handleAOIChange(obj, oldAOIObjects, newAOIObjects)
	// 通知周围玩家移动
	m.notifyMovement(obj, oldPos, targetPos)
	// 触发事件（如果是玩家）
	if p, ok := obj.(*object.Player); ok {
		m.TriggerEvents(p)
	}

	return nil
}

// handleAOIChange 处理AOI变化
func (m *Map) handleAOIChange(object common.IGameObject, oldObjects, newObjects []common.IGameObject) {
	// 计算进入视野的对象
	enteredObjects := make([]common.IGameObject, 0)
	for _, newObj := range newObjects {
		found := false
		for _, oldObj := range oldObjects {
			if newObj.GetID() == oldObj.GetID() {
				found = true
				break
			}
		}
		if !found {
			enteredObjects = append(enteredObjects, newObj)
		}
	}

	// 计算离开视野的对象
	leftObjects := make([]common.IGameObject, 0)
	for _, oldObj := range oldObjects {
		found := false
		for _, newObj := range newObjects {
			if oldObj.GetID() == newObj.GetID() {
				found = true
				break
			}
		}
		if !found {
			leftObjects = append(leftObjects, oldObj)
		}
	}

	// 通知对象进入视野
	for _, enteredObj := range enteredObjects {
		m.notifyObjectEntered(object, enteredObj)
	}

	// 通知对象离开视野
	for _, leftObj := range leftObjects {
		m.notifyObjectLeft(object, leftObj)
	}
}

// notifyMovement 通知周围玩家移动
func (m *Map) notifyMovement(object common.IGameObject, oldPos, newPos common.Vector3) {
	// 获取周围的玩家
	players := m.getPlayersInRange(newPos, 100)
	// 通知玩家移动
	for _, player := range players {
		// 排除自己
		if player.GetID() != object.GetID() {
			m.sendMovementUpdate(player, object, newPos)
		}
	}
}

// getPlayersInRange 获取指定范围内的玩家
func (m *Map) getPlayersInRange(pos common.Vector3, radius float32) []common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	players := make([]common.IGameObject, 0)

	for _, obj := range m.objects {
		if obj.GetType() == common.GameObjectTypePlayer {
			distance := obj.GetPosition().DistanceTo(pos)
			if distance <= radius*radius {
				players = append(players, obj)
			}
		}
	}

	return players
}

// notifyObjectEntered 通知对象进入视野
func (m *Map) notifyObjectEntered(watcher, target common.IGameObject) {
	if m.connManager == nil {
		return
	}

	// 只有玩家需要收到 AOI 通知
	if watcher.GetType() != common.GameObjectTypePlayer {
		return
	}

	playerID := id.PlayerIdType(0)
	if p, ok := watcher.(*object.Player); ok {
		playerID = p.GetPlayerID()
	}

	if playerID == 0 {
		return
	}

	// 构建 AOI 进入通知
	aoiObjects := make([]*pb.AoiObjectInfo, 0)
	objInfo := &pb.AoiObjectInfo{
		ObjectId:   int64(target.GetID()),
		ObjectType: int32(target.GetType()),
		X:          target.GetPosition().X,
		Y:          target.GetPosition().Y,
		Z:          target.GetPosition().Z,
	}

	// 根据对象类型设置额外信息
	switch target.GetType() {
	case common.GameObjectTypePlayer:
		if p, ok := target.(*object.Player); ok {
			objInfo.EntityId = int64(p.GetPlayerID())
			objInfo.Name = p.GetName()
		}
	case common.GameObjectTypeMonster:
		if m, ok := target.(*object.Monster); ok {
			objInfo.EntityId = int64(m.GetMonsterID())
			objInfo.Name = m.GetName()
		}
	case common.GameObjectTypeNPC:
		if n, ok := target.(*object.NPC); ok {
			objInfo.EntityId = int64(n.GetNPCID())
			objInfo.Name = n.GetName()
		}
	}

	aoiObjects = append(aoiObjects, objInfo)

	// 发送 AOI 进入通知
	m.sendAOIEnterNotify(playerID, aoiObjects)

	// 记录日志
	zLog.Debug("Object entered AOI",
		zap.Int64("watcher_id", int64(watcher.GetID())),
		zap.Int64("target_id", int64(target.GetID())),
		zap.Int32("target_type", int32(target.GetType())))
}

// notifyObjectLeft 通知对象离开视野
func (m *Map) notifyObjectLeft(watcher, target common.IGameObject) {
	if m.connManager == nil {
		return
	}

	// 只有玩家需要收到 AOI 通知
	if watcher.GetType() != common.GameObjectTypePlayer {
		return
	}

	playerID := id.PlayerIdType(0)
	if p, ok := watcher.(*object.Player); ok {
		playerID = p.GetPlayerID()
	}

	if playerID == 0 {
		return
	}

	// 发送 AOI 离开通知
	objectIDs := []int64{int64(target.GetID())}
	m.sendAOILeaveNotify(playerID, objectIDs)

	// 记录日志
	zLog.Debug("Object left AOI",
		zap.Int64("watcher_id", int64(watcher.GetID())),
		zap.Int64("target_id", int64(target.GetID())))
}

// sendAOIEnterNotify 发送 AOI 进入通知
func (m *Map) sendAOIEnterNotify(playerID id.PlayerIdType, objects []*pb.AoiObjectInfo) {
	if m.connManager == nil {
		return
	}

	notify := &pb.AoiEnterNotify{
		PlayerId: int64(playerID),
		MapId:    int64(m.mapID),
		Objects:  objects,
	}

	data, err := proto.Marshal(notify)
	if err != nil {
		zLog.Error("Failed to marshal AOI enter notify", zap.Error(err))
		return
	}

	// 构建消息包：长度 + 消息ID + 数据
	msgID := uint32(protocol.MsgIdAoiEnterNotify)
	length := uint32(4 + len(data))
	buffer := make([]byte, 4+4+len(data))
	binary.BigEndian.PutUint32(buffer[:4], length)
	binary.BigEndian.PutUint32(buffer[4:8], msgID)
	copy(buffer[8:], data)

	// 发送到 GameServer
	err = m.connManager.SendToGameServerByMap(int(m.mapID), buffer)
	if err != nil {
		zLog.Error("Failed to send AOI enter notify", zap.Error(err))
	}
}

// sendAOILeaveNotify 发送 AOI 离开通知
func (m *Map) sendAOILeaveNotify(playerID id.PlayerIdType, objectIDs []int64) {
	if m.connManager == nil {
		return
	}

	notify := &pb.AoiLeaveNotify{
		PlayerId:  int64(playerID),
		MapId:     int64(m.mapID),
		ObjectIds: objectIDs,
	}

	data, err := proto.Marshal(notify)
	if err != nil {
		zLog.Error("Failed to marshal AOI leave notify", zap.Error(err))
		return
	}

	// 构建消息包：长度 + 消息ID + 数据
	msgID := uint32(protocol.MsgIdAoiLeaveNotify)
	length := uint32(4 + len(data))
	buffer := make([]byte, 4+4+len(data))
	binary.BigEndian.PutUint32(buffer[:4], length)
	binary.BigEndian.PutUint32(buffer[4:8], msgID)
	copy(buffer[8:], data)

	// 发送到 GameServer
	err = m.connManager.SendToGameServerByMap(int(m.mapID), buffer)
	if err != nil {
		zLog.Error("Failed to send AOI leave notify", zap.Error(err))
	}
}

// sendMovementUpdate 发送移动更新
func (m *Map) sendMovementUpdate(player, movingObject common.IGameObject, newPos common.Vector3) {
	if m.connManager == nil {
		return
	}

	// 只有玩家需要收到移动更新
	if player.GetType() != common.GameObjectTypePlayer {
		return
	}

	targetPlayerID := id.PlayerIdType(0)
	if p, ok := player.(*object.Player); ok {
		targetPlayerID = p.GetPlayerID()
	}

	if targetPlayerID == 0 {
		return
	}

	// 构建移动同步消息
	moveSync := &pb.MapMoveSync{
		ObjectId:  int64(movingObject.GetID()),
		MapId:     int64(m.mapID),
		X:         newPos.X,
		Y:         newPos.Y,
		Z:         newPos.Z,
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := proto.Marshal(moveSync)
	if err != nil {
		zLog.Error("Failed to marshal map move sync", zap.Error(err))
		return
	}

	// 构建消息包：长度 + 消息ID + 数据
	msgID := uint32(protocol.MsgIdMapMoveSync)
	length := uint32(4 + len(data))
	buffer := make([]byte, 4+4+len(data))
	binary.BigEndian.PutUint32(buffer[:4], length)
	binary.BigEndian.PutUint32(buffer[4:8], msgID)
	copy(buffer[8:], data)

	// 发送到 GameServer
	err = m.connManager.SendToGameServerByMap(int(m.mapID), buffer)
	if err != nil {
		zLog.Error("Failed to send map move sync", zap.Error(err))
	}

	// 记录日志
	zLog.Debug("Sent movement update",
		zap.Int64("moving_object_id", int64(movingObject.GetID())),
		zap.Int64("target_player_id", int64(targetPlayerID)),
		zap.Float32("x", newPos.X),
		zap.Float32("y", newPos.Y))
}

// TeleportObject 传送游戏对象
func (m *Map) TeleportObject(object common.IGameObject, targetPos common.Vector3) error {
	object.SetPosition(targetPos)
	return nil
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

// GetSize 获取地图尺寸
func (m *Map) GetSize() (float32, float32) {
	return m.width, m.height
}

// SetRegionSize 设置区域大小
func (m *Map) SetRegionSize(regionSize float32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.regionSize = regionSize
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

// GetObjectByID 根据ID获取游戏对象
func (m *Map) GetObjectByID(objectID int64) common.IGameObject {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, obj := range m.objects {
		if int64(id) == objectID {
			return obj
		}
	}

	return nil
}

// GetEventManager 获取事件管理器
func (m *Map) GetEventManager() *event.EventManager {
	return m.eventManager
}

// TriggerEvents 触发地图事件
func (m *Map) TriggerEvents(player *object.Player) {
	if m.eventManager != nil {
		m.eventManager.TriggerEvents(player)
	}
}

// UpdateEvents 更新地图事件
func (m *Map) UpdateEvents() {
	if m.eventManager != nil {
		m.eventManager.UpdateEvents()
	}
}

// CreateDefaultEvents 创建默认事件
func (m *Map) CreateDefaultEvents() {
	if m.eventManager != nil {
		m.eventManager.CreateDefaultEvents()
	}
}

// AddTeleportPoint 添加传送点
func (m *Map) AddTeleportPoint(teleportPoint *models.MapTeleportPoint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.teleportPoints = append(m.teleportPoints, teleportPoint)
}

// GetTeleportPoints 获取所有传送点
func (m *Map) GetTeleportPoints() []*models.MapTeleportPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.teleportPoints
}

// GetTeleportPointByID 根据ID获取传送点
func (m *Map) GetTeleportPointByID(teleportID int32) *models.MapTeleportPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, tp := range m.teleportPoints {
		if tp.ID == teleportID {
			return tp
		}
	}

	return nil
}

// GetTeleportPointsInRange 获取指定范围内的传送点
func (m *Map) GetTeleportPointsInRange(pos common.Vector3, radius float32) []*models.MapTeleportPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.MapTeleportPoint
	for _, tp := range m.teleportPoints {
		tpPos := common.Vector3{X: float32(tp.X), Y: float32(tp.Y), Z: float32(tp.Z)}
		if tpPos.DistanceTo(pos) <= radius*radius {
			result = append(result, tp)
		}
	}

	return result
}

// AddBuilding 添加建筑
func (m *Map) AddBuilding(building *models.MapBuilding) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buildings = append(m.buildings, building)
}

// GetBuildings 获取所有建筑
func (m *Map) GetBuildings() []*models.MapBuilding {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.buildings
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

// GetBuildingsInRange 获取指定范围内的建筑
func (m *Map) GetBuildingsInRange(pos common.Vector3, radius float32) []*models.MapBuilding {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*models.MapBuilding
	for _, building := range m.buildings {
		buildingPos := common.Vector3{X: float32(building.X), Y: float32(building.Y), Z: float32(building.Z)}
		if buildingPos.DistanceTo(pos) <= radius*radius {
			result = append(result, building)
		}
	}

	return result
}

// UseSkill 释放技能
func (m *Map) UseSkill(caster *object.Player, skillID int32, targetID id.ObjectIdType, targetPos common.Vector3) error {
	// 检查施法者是否存在
	if caster == nil {
		return fmt.Errorf("caster not found")
	}

	// 检查技能是否存在
	if !caster.HasSkill(skillID) {
		return fmt.Errorf("skill not found")
	}

	// 检查技能是否在冷却中
	if caster.IsSkillInCooldown(skillID) {
		return fmt.Errorf("skill is in cooldown")
	}

	// 从技能配置中获取技能信息
	skillConfig := m.skillManager.GetSkillConfig(skillID)
	if skillConfig == nil {
		return fmt.Errorf("skill config not found")
	}

	// 检查目标位置是否在地图范围内
	if !m.IsInMap(targetPos) {
		return fmt.Errorf("target position out of map bounds")
	}

	// 处理目标选择
	var targetObj common.IGameObject
	if targetID > 0 {
		// 有指定目标ID，尝试找到目标
		targetObj = m.GetObjectByID(int64(targetID))
		if targetObj != nil {
			// 验证目标是否有效
			if !m.ValidateTarget(caster, targetObj, skillConfig.Range) {
				targetObj = nil
			}
		}
	}

	// 如果没有指定目标或目标无效，自动选择最近的目标
	if targetObj == nil {
		targetTypes := []common.GameObjectType{common.GameObjectTypeMonster, common.GameObjectTypePlayer}
		targetObj = m.GetNearestTarget(targetPos, skillConfig.Range, caster.GetID(), targetTypes)
		if targetObj != nil {
			targetID = targetObj.GetID()
			targetPos = targetObj.GetPosition()
		}
	}

	// 检查施法者与目标的距离
	casterPos := caster.GetPosition()
	distance := m.CalculateDistance(casterPos, targetPos)
	if distance > skillConfig.Range {
		return fmt.Errorf("target too far")
	}

	// 检查施法者的 mana 是否足够
	if caster.GetMana() < skillConfig.ManaCost {
		return fmt.Errorf("not enough mana")
	}

	// 消耗 mana
	caster.SetMana(caster.GetMana() - skillConfig.ManaCost)

	// 创建技能对象
	skillObjectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)
	newSkill := skill.NewSkill(
		skillObjectID,
		skillID,
		caster.GetID(),
		targetID,
		targetPos,
		skillConfig.Damage, // 从配置中获取伤害
		skillConfig.Range,  // 从配置中获取范围
		skillConfig.Type,   // 从配置中获取效果类型
		time.Duration(skillConfig.Duration)*time.Millisecond, // 从配置中获取持续时间
		time.Duration(skillConfig.Cooldown)*time.Millisecond, // 从配置中获取冷却时间
	)

	// 设置技能等级和施法者攻击力
	newSkill.SetLevel(skillConfig.Level) // 从配置中获取等级
	newSkill.SetCasterAttack(caster.GetAttack())
	newSkill.SetEffectID(skillConfig.EffectID) // 从配置中获取特效ID

	// 添加技能到技能管理器
	m.skillManager.AddSkill(newSkill)

	// 设置技能冷却时间
	caster.SetSkillCooldown(skillID, time.Duration(skillConfig.Cooldown)*time.Millisecond)

	// 添加技能到历史记录
	caster.AddSkillToHistory(skillID)

	// 检查技能组合
	m.checkSkillCombo(caster)

	// 处理技能效果
	m.handleSkillEffect(newSkill)

	// 通知周围玩家技能释放
	m.notifySkillUse(caster, newSkill)

	// 发送技能释放响应给施法者
	m.sendSkillUseResponse(caster, newSkill, nil)

	zLog.Debug("Skill used",
		zap.Int64("caster_id", int64(caster.GetID())),
		zap.Int32("skill_id", skillID),
		zap.Int64("target_id", int64(targetID)),
		zap.Float32("x", targetPos.X),
		zap.Float32("y", targetPos.Y))

	return nil
}

// sendSkillUseResponse 发送技能释放响应给施法者
func (m *Map) sendSkillUseResponse(caster *object.Player, skill *skill.Skill, err error) {
	if m.connManager == nil {
		return
	}

	// 构建技能释放响应
	skillResponse := &pb.SkillUseResponse{
		Success:  err == nil,
		SkillId:  int64(skill.GetSkillConfigID()),
		TargetId: int64(skill.GetTargetID()),
		Cooldown: int64(skill.GetRemainingCooldown().Milliseconds()),
		Damage:   int64(skill.GetDamage()),
	}

	if err != nil {
		skillResponse.ErrorMsg = err.Error()
	}

	data, err := proto.Marshal(skillResponse)
	if err != nil {
		zLog.Error("Failed to marshal skill use response", zap.Error(err))
		return
	}

	// 构建消息包：长度 + 消息ID + 数据
	msgID := uint32(protocol.MsgIdSkillUseResult)
	length := uint32(4 + len(data))
	buffer := make([]byte, 4+4+len(data))
	binary.BigEndian.PutUint32(buffer[:4], length)
	binary.BigEndian.PutUint32(buffer[4:8], msgID)
	copy(buffer[8:], data)

	// 发送到 GameServer
	err = m.connManager.SendToGameServerByMap(int(m.mapID), buffer)
	if err != nil {
		zLog.Error("Failed to send skill use response", zap.Error(err))
	}
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

		// 只对怪物和玩家造成伤害
		objType := obj.GetType()
		if objType == common.GameObjectTypeMonster || objType == common.GameObjectTypePlayer {
			// 检查技能是否命中目标
			if !skill.CheckHit(obj.GetPosition()) {
				continue
			}

			// 计算伤害
			var damage int32
			switch target := obj.(type) {
			case *object.Player:
				damage = skill.CalculateDamage(target)
				damage = damage - target.GetDefense()
				if damage < 1 {
					damage = 1 // 确保至少造成1点伤害
				}
				target.TakeDamage(damage)
			case *object.Monster:
				damage = skill.CalculateDamageToMonster(target)
				damage = damage - target.GetDefense()
				if damage < 1 {
					damage = 1 // 确保至少造成1点伤害
				}
				target.TakeDamage(damage)
			}

			zLog.Debug("Skill hit target",
				zap.Int64("skill_id", int64(skill.GetID())),
				zap.Int64("target_id", int64(obj.GetID())),
				zap.Int32("damage", damage))
		}
	}
}

// notifySkillUse 通知周围玩家技能释放
func (m *Map) notifySkillUse(caster *object.Player, skill *skill.Skill) {
	// 获取周围的玩家
	players := m.getPlayersInRange(caster.GetPosition(), 100)

	for _, player := range players {
		// 排除施法者自己
		if player.GetID() != caster.GetID() {
			// 发送技能释放通知
			m.sendSkillUseNotify(player, caster, skill)
		}
	}
}

// sendSkillUseNotify 发送技能释放通知
func (m *Map) sendSkillUseNotify(player common.IGameObject, caster *object.Player, skill *skill.Skill) {
	if m.connManager == nil {
		return
	}

	// 只有玩家需要收到技能释放通知
	if player.GetType() != common.GameObjectTypePlayer {
		return
	}

	targetPlayerID := id.PlayerIdType(0)
	if p, ok := player.(*object.Player); ok {
		targetPlayerID = p.GetPlayerID()
	}

	if targetPlayerID == 0 {
		return
	}

	// 构建技能释放通知
	skillNotify := &pb.SkillUseNotify{
		CasterId:  int64(caster.GetID()),
		SkillId:   int64(skill.GetSkillConfigID()),
		TargetId:  int64(skill.GetTargetID()),
		TargetX:   skill.GetPosition().X,
		TargetY:   skill.GetPosition().Y,
		TargetZ:   skill.GetPosition().Z,
		MapId:     int64(m.mapID),
		EffectId:  skill.GetEffectID(),
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := proto.Marshal(skillNotify)
	if err != nil {
		zLog.Error("Failed to marshal skill use notify", zap.Error(err))
		return
	}

	// 构建消息包：长度 + 消息ID + 数据
	msgID := uint32(protocol.MsgIdSkillUse)
	length := uint32(4 + len(data))
	buffer := make([]byte, 4+4+len(data))
	binary.BigEndian.PutUint32(buffer[:4], length)
	binary.BigEndian.PutUint32(buffer[4:8], msgID)
	copy(buffer[8:], data)

	// 发送到 GameServer
	err = m.connManager.SendToGameServerByMap(int(m.mapID), buffer)
	if err != nil {
		zLog.Error("Failed to send skill use notify", zap.Error(err))
	}
}

// UpdateSkills 更新技能状态
func (m *Map) UpdateSkills() {
	if m.skillManager != nil {
		m.skillManager.Update()
	}
}

// GetTaskManager 获取任务管理器
func (m *Map) GetTaskManager() *task.TaskManager {
	return m.taskManager
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
				// 这里需要实现增加经验的逻辑
			case "gold":
				// 增加金币
				// 这里需要实现增加金币的逻辑
			case "item":
				// 增加物品
				// 这里需要实现增加物品的逻辑
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

// GetInventoryManager 获取背包管理器
func (m *Map) GetInventoryManager() *item.InventoryManager {
	return m.inventoryManager
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

// UseItem 使用物品
func (m *Map) UseItem(player *object.Player, itemID int32) error {
	if m.inventoryManager == nil {
		return fmt.Errorf("inventory manager not initialized")
	}

	// 使用物品
	err := m.inventoryManager.UseItem(player.GetPlayerID(), itemID)
	if err != nil {
		return err
	}

	// 处理物品效果
	itemConfig := m.inventoryManager.GetItemConfig(itemID)
	if itemConfig != nil {
		for _, effect := range itemConfig.Effects {
			switch effect.Type {
			case "health":
				// 增加生命值
				// 这里需要实现增加生命值的逻辑
			case "mana":
				// 增加魔法值
				// 这里需要实现增加魔法值的逻辑
			case "attack":
				// 增加攻击力
				// 这里需要实现增加攻击力的逻辑
			case "defense":
				// 增加防御力
				// 这里需要实现增加防御力的逻辑
			case "speed":
				// 增加移动速度
				// 这里需要实现增加移动速度的逻辑
			}
		}
	}

	return nil
}

// GetItemCount 获取物品数量
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

// GetTeamManager 获取队伍管理器
func (m *Map) GetTeamManager() *social.TeamManager {
	return m.teamManager
}

// GetGuildManager 获取公会管理器
func (m *Map) GetGuildManager() *social.GuildManager {
	return m.guildManager
}

// CreateTeam 创建队伍
func (m *Map) CreateTeam(player *object.Player) (*social.Team, error) {
	if m.teamManager == nil {
		return nil, fmt.Errorf("team manager not initialized")
	}

	return m.teamManager.CreateTeam(
		player.GetPlayerID(),
		player.GetName(),
		player.GetLevel(),
		player.GetClass(),
	)
}

// InviteToTeam 邀请玩家加入队伍
func (m *Map) InviteToTeam(teamID id.TeamIdType, inviter, target *object.Player) error {
	if m.teamManager == nil {
		return fmt.Errorf("team manager not initialized")
	}

	return m.teamManager.InviteToTeam(teamID, inviter.GetPlayerID(), target.GetPlayerID())
}

// AcceptTeamInvite 接受队伍邀请
func (m *Map) AcceptTeamInvite(teamID id.TeamIdType, player *object.Player) error {
	if m.teamManager == nil {
		return fmt.Errorf("team manager not initialized")
	}

	return m.teamManager.AcceptTeamInvite(
		teamID,
		player.GetPlayerID(),
		player.GetName(),
		player.GetLevel(),
		player.GetClass(),
	)
}

// LeaveTeam 离开队伍
func (m *Map) LeaveTeam(player *object.Player) error {
	if m.teamManager == nil {
		return fmt.Errorf("team manager not initialized")
	}

	return m.teamManager.LeaveTeam(player.GetPlayerID())
}

// KickFromTeam 踢出队伍
func (m *Map) KickFromTeam(teamID id.TeamIdType, leader, target *object.Player) error {
	if m.teamManager == nil {
		return fmt.Errorf("team manager not initialized")
	}

	return m.teamManager.KickFromTeam(teamID, leader.GetPlayerID(), target.GetPlayerID())
}

// DisbandTeam 解散队伍
func (m *Map) DisbandTeam(teamID id.TeamIdType, leader *object.Player) error {
	if m.teamManager == nil {
		return fmt.Errorf("team manager not initialized")
	}

	return m.teamManager.DisbandTeam(teamID, leader.GetPlayerID())
}

// GetPlayerTeam 获取玩家所在队伍
func (m *Map) GetPlayerTeam(player *object.Player) *social.Team {
	if m.teamManager == nil {
		return nil
	}

	return m.teamManager.GetPlayerTeam(player.GetPlayerID())
}

// CreateGuild 创建公会
func (m *Map) CreateGuild(player *object.Player, guildName string) (*social.Guild, error) {
	if m.guildManager == nil {
		return nil, fmt.Errorf("guild manager not initialized")
	}

	return m.guildManager.CreateGuild(
		player.GetPlayerID(),
		guildName,
		player.GetName(),
		player.GetLevel(),
		player.GetClass(),
	)
}

// InviteToGuild 邀请玩家加入公会
func (m *Map) InviteToGuild(guildID id.GuildIdType, inviter, target *object.Player) error {
	if m.guildManager == nil {
		return fmt.Errorf("guild manager not initialized")
	}

	return m.guildManager.InviteToGuild(guildID, inviter.GetPlayerID(), target.GetPlayerID())
}

// AcceptGuildInvite 接受公会邀请
func (m *Map) AcceptGuildInvite(guildID id.GuildIdType, player *object.Player) error {
	if m.guildManager == nil {
		return fmt.Errorf("guild manager not initialized")
	}

	return m.guildManager.AcceptGuildInvite(
		guildID,
		player.GetPlayerID(),
		player.GetName(),
		player.GetLevel(),
		player.GetClass(),
	)
}

// LeaveGuild 离开公会
func (m *Map) LeaveGuild(player *object.Player) error {
	if m.guildManager == nil {
		return fmt.Errorf("guild manager not initialized")
	}

	return m.guildManager.LeaveGuild(player.GetPlayerID())
}

// KickFromGuild 踢出公会
func (m *Map) KickFromGuild(guildID id.GuildIdType, operator, target *object.Player) error {
	if m.guildManager == nil {
		return fmt.Errorf("guild manager not initialized")
	}

	return m.guildManager.KickFromGuild(guildID, operator.GetPlayerID(), target.GetPlayerID())
}

// DisbandGuild 解散公会
func (m *Map) DisbandGuild(guildID id.GuildIdType, leader *object.Player) error {
	if m.guildManager == nil {
		return fmt.Errorf("guild manager not initialized")
	}

	return m.guildManager.DisbandGuild(guildID, leader.GetPlayerID())
}

// UpdateGuildNotice 更新公会公告
func (m *Map) UpdateGuildNotice(guildID id.GuildIdType, leader *object.Player, notice string) error {
	if m.guildManager == nil {
		return fmt.Errorf("guild manager not initialized")
	}

	return m.guildManager.UpdateGuildNotice(guildID, leader.GetPlayerID(), notice)
}

// GetPlayerGuild 获取玩家所在公会
func (m *Map) GetPlayerGuild(player *object.Player) *social.Guild {
	if m.guildManager == nil {
		return nil
	}

	return m.guildManager.GetPlayerGuild(player.GetPlayerID())
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

// GetAchievementManager 获取成就管理器
func (m *Map) GetAchievementManager() *achievement.AchievementManager {
	return m.achievementManager
}

// GetPetManager 获取宠物管理器
func (m *Map) GetPetManager() *pet.PetManager {
	return m.petManager
}

// GetMountManager 获取坐骑管理器
func (m *Map) GetMountManager() *mount.MountManager {
	return m.mountManager
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

// InitiateTrade 发起交易
func (m *Map) InitiateTrade(initiator, target *object.Player) (*economy.Trade, error) {
	if m.tradeManager == nil {
		return nil, fmt.Errorf("trade manager not initialized")
	}

	return m.tradeManager.InitiateTrade(initiator.GetPlayerID(), target.GetPlayerID())
}

// AcceptTrade 接受交易
func (m *Map) AcceptTrade(tradeID int64, player *object.Player) error {
	if m.tradeManager == nil {
		return fmt.Errorf("trade manager not initialized")
	}

	return m.tradeManager.AcceptTrade(tradeID, player.GetPlayerID())
}

// CancelTrade 取消交易
func (m *Map) CancelTrade(tradeID int64, player *object.Player) error {
	if m.tradeManager == nil {
		return fmt.Errorf("trade manager not initialized")
	}

	return m.tradeManager.CancelTrade(tradeID, player.GetPlayerID())
}

// CompleteTrade 完成交易
func (m *Map) CompleteTrade(tradeID int64, player *object.Player) error {
	if m.tradeManager == nil {
		return fmt.Errorf("trade manager not initialized")
	}

	return m.tradeManager.CompleteTrade(tradeID, player.GetPlayerID())
}

// CreateAuction 创建拍卖
func (m *Map) CreateAuction(player *object.Player, itemID, itemCount int32, startingPrice int64, duration time.Duration, currencyType economy.CurrencyType) (*economy.Auction, error) {
	if m.auctionManager == nil {
		return nil, fmt.Errorf("auction manager not initialized")
	}

	return m.auctionManager.CreateAuction(player.GetPlayerID(), itemID, itemCount, startingPrice, duration, currencyType)
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

// ValidateTarget 验证目标是否有效
func (m *Map) ValidateTarget(caster common.IGameObject, target common.IGameObject, skillRange float32) bool {
	if target == nil {
		return false
	}

	// 检查目标是否在地图上
	if !m.IsObjectInMap(target) {
		return false
	}

	// 检查目标是否在技能范围内
	casterPos := caster.GetPosition()
	targetPos := target.GetPosition()
	distance := m.CalculateDistance(casterPos, targetPos)
	if distance > skillRange {
		return false
	}

	return true
}

// checkSkillCombo 检查技能组合
func (m *Map) checkSkillCombo(caster *object.Player) {
	// 获取技能释放历史
	skillHistory := caster.GetSkillHistory()
	if len(skillHistory) < 2 {
		return
	}

	// 获取技能组合管理器
	comboManager := m.skillManager.GetSkillComboManager()
	if comboManager == nil {
		return
	}

	// 检查技能组合
	skillHistoryTime := caster.GetSkillHistoryTime()
	combo := comboManager.CheckSkillCombo(skillHistory, skillHistoryTime)
	if combo != nil {
		// 触发技能组合效果
		m.triggerSkillCombo(caster, combo)

		// 清理技能历史记录
		caster.ClearSkillHistory()
	}
}

// triggerSkillCombo 触发技能组合效果
func (m *Map) triggerSkillCombo(caster *object.Player, combo *skill.SkillCombo) {
	// 获取施法者位置
	casterPos := caster.GetPosition()

	// 创建技能组合特效
	skillObjectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)
	comboSkill := skill.NewSkill(
		skillObjectID,
		int32(combo.ID),
		caster.GetID(),
		0, // 无特定目标
		casterPos,
		combo.BonusDamage,
		15.0,           // 组合技能范围
		1,              // 效果类型：伤害
		time.Second*3,  // 持续时间
		time.Second*10, // 冷却时间
	)

	// 设置技能等级和施法者攻击力
	comboSkill.SetLevel(1)
	comboSkill.SetCasterAttack(caster.GetAttack())
	comboSkill.SetEffectID(combo.EffectID)

	// 添加技能到技能管理器
	m.skillManager.AddSkill(comboSkill)

	// 处理技能效果
	m.handleSkillEffect(comboSkill)

	// 通知周围玩家技能组合释放
	m.notifySkillUse(caster, comboSkill)

	zLog.Debug("Skill combo triggered",
		zap.Int32("combo_id", combo.ID),
		zap.String("combo_name", combo.Name),
		zap.Int32("caster_id", int32(caster.GetID())))
}

// CanUseTeleport 检查玩家是否可以使用传送点
func (m *Map) CanUseTeleport(player *object.Player, teleportPoint *models.MapTeleportPoint) bool {
	// 检查传送点是否激活
	if !teleportPoint.IsActive {
		return false
	}

	// 检查玩家等级是否满足要求
	if player.GetLevel() < teleportPoint.RequiredLevel {
		return false
	}

	// 检查玩家是否拥有所需物品
	// 这里可以添加物品检查逻辑

	return true
}

// UseTeleport 使用传送点
func (m *Map) UseTeleport(player *object.Player, teleportPoint *models.MapTeleportPoint) (id.MapIdType, common.Vector3, error) {
	if !m.CanUseTeleport(player, teleportPoint) {
		return 0, common.Vector3{}, nil
	}

	// 计算目标位置
	targetPos := common.Vector3{
		X: float32(teleportPoint.TargetX),
		Y: float32(teleportPoint.TargetY),
		Z: float32(teleportPoint.TargetZ),
	}

	// 返回目标地图ID和位置
	return id.MapIdType(teleportPoint.TargetMapID), targetPos, nil
}

// InteractWithBuilding 与建筑交互
func (m *Map) InteractWithBuilding(player *object.Player, building *models.MapBuilding) bool {
	// 检查玩家是否在建筑范围内
	buildingPos := common.Vector3{X: float32(building.X), Y: float32(building.Y), Z: float32(building.Z)}
	distance := player.GetPosition().DistanceTo(buildingPos)

	// 建筑的交互范围设为建筑宽度的一半
	interactionRadius := float32(building.Width) / 2
	if distance > interactionRadius*interactionRadius {
		return false
	}

	// 根据建筑类型执行不同的交互逻辑
	switch building.Type {
	case "town_hall":
		// 城镇大厅交互逻辑
		return true
	case "blacksmith":
		// 铁匠铺交互逻辑
		return true
	case "inn":
		// 旅馆交互逻辑
		return true
	case "shop":
		// 商店交互逻辑
		return true
	default:
		return false
	}
}

// UpdateBuilding 更新建筑状态
func (m *Map) UpdateBuilding(building *models.MapBuilding) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, b := range m.buildings {
		if b.ID == building.ID {
			m.buildings[i] = building
			break
		}
	}
}

// LoadTeleportPoints 从配置加载传送点
func (m *Map) LoadTeleportPoints() {
	// 这里可以从配置文件或数据库加载传送点
	// 暂时添加一些默认传送点
	defaultTeleportPoints := []*models.MapTeleportPoint{
		{
			ID:            1,
			MapID:         int32(m.mapID),
			X:             100,
			Y:             100,
			Z:             0,
			TargetMapID:   int32(m.mapID),
			TargetX:       200,
			TargetY:       200,
			TargetZ:       0,
			Name:          "Test Teleport",
			RequiredLevel: 1,
			RequiredItem:  0,
			IsActive:      true,
		},
	}

	for _, tp := range defaultTeleportPoints {
		m.AddTeleportPoint(tp)
	}
}

// LoadBuildings 从配置加载建筑
func (m *Map) LoadBuildings() {
	// 这里可以从配置文件或数据库加载建筑
	// 暂时添加一些默认建筑
	defaultBuildings := []*models.MapBuilding{
		{
			ID:      1,
			MapID:   int32(m.mapID),
			X:       150,
			Y:       150,
			Z:       0,
			Width:   10,
			Height:  10,
			Type:    "town_hall",
			Name:    "Town Hall",
			Level:   1,
			HP:      1000,
			Faction: 0,
		},
		{
			ID:      2,
			MapID:   int32(m.mapID),
			X:       160,
			Y:       160,
			Z:       0,
			Width:   8,
			Height:  8,
			Type:    "blacksmith",
			Name:    "Blacksmith",
			Level:   1,
			HP:      500,
			Faction: 0,
		},
	}

	for _, building := range defaultBuildings {
		m.AddBuilding(building)
	}
}
