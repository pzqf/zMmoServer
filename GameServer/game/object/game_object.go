package object

import (
	"sync"

	"github.com/pzqf/zEngine/zEvent"
	"github.com/pzqf/zEngine/zObject"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoServer/GameServer/game/object/component"
	"github.com/pzqf/zCommon/common/id"
)

// GameObject 基础游戏对象
type GameObject struct {
	*zObject.BaseObject
	mu           sync.RWMutex
	name         string
	objectType   common.GameObjectType
	position     common.Vector3
	isActive     bool
	eventEmitter *zEvent.EventBus
	components   *component.ComponentManager
	mapObject    common.IMap
}

// NewGameObject 创建新的游戏对象
func NewGameObject(objectID id.ObjectIdType, name string) *GameObject {
	goObj := &GameObject{
		BaseObject:   &zObject.BaseObject{},
		name:         name,
		objectType:   common.GameObjectTypeBasic,
		position:     common.NewVector3(0, 0, 0),
		isActive:     true,
		eventEmitter: zEvent.NewEventBus(),
	}
	goObj.components = component.NewComponentManager(goObj)
	goObj.SetId(objectID)
	return goObj
}

// NewGameObjectWithType 创建指定类型的游戏对象
func NewGameObjectWithType(objectID id.ObjectIdType, name string, objectType common.GameObjectType) *GameObject {
	goObj := &GameObject{
		BaseObject:   &zObject.BaseObject{},
		name:         name,
		objectType:   objectType,
		position:     common.NewVector3(0, 0, 0),
		isActive:     true,
		eventEmitter: zEvent.NewEventBus(),
	}
	goObj.components = component.NewComponentManager(goObj)
	goObj.SetId(objectID)
	return goObj
}

// GetID 获取对象ID
func (goObj *GameObject) GetID() id.ObjectIdType {
	goObj.mu.RLock()
	defer goObj.mu.RUnlock()
	return goObj.GetId().(id.ObjectIdType)
}

// SetID 设置对象ID
func (goObj *GameObject) SetID(objectID id.ObjectIdType) {
	goObj.mu.Lock()
	defer goObj.mu.Unlock()
	goObj.SetId(objectID)
}

// GetName 获取对象名称
func (goObj *GameObject) GetName() string {
	goObj.mu.RLock()
	defer goObj.mu.RUnlock()
	return goObj.name
}

// SetName 设置对象名称
func (goObj *GameObject) SetName(name string) {
	goObj.mu.Lock()
	defer goObj.mu.Unlock()
	goObj.name = name
}

// GetType 获取对象类型
func (goObj *GameObject) GetType() common.GameObjectType {
	goObj.mu.RLock()
	defer goObj.mu.RUnlock()
	return goObj.objectType
}

// SetType 设置对象类型
func (goObj *GameObject) SetType(objectType common.GameObjectType) {
	goObj.mu.Lock()
	defer goObj.mu.Unlock()
	goObj.objectType = objectType
}

// GetPosition 获取对象位置
func (goObj *GameObject) GetPosition() common.Vector3 {
	goObj.mu.RLock()
	defer goObj.mu.RUnlock()
	return goObj.position
}

// SetPosition 设置对象位置
func (goObj *GameObject) SetPosition(position common.Vector3) {
	goObj.mu.Lock()
	defer goObj.mu.Unlock()
	goObj.position = position
}

// IsActive 检查对象是否激活
func (goObj *GameObject) IsActive() bool {
	goObj.mu.RLock()
	defer goObj.mu.RUnlock()
	return goObj.isActive
}

// SetActive 设置对象激活状态
func (goObj *GameObject) SetActive(active bool) {
	goObj.mu.Lock()
	defer goObj.mu.Unlock()
	goObj.isActive = active
}

// GetEventEmitter 获取事件总线
func (goObj *GameObject) GetEventEmitter() *zEvent.EventBus {
	return goObj.eventEmitter
}

// GetMap 获取所属地图
func (goObj *GameObject) GetMap() common.IMap {
	goObj.mu.RLock()
	defer goObj.mu.RUnlock()
	return goObj.mapObject
}

// SetMap 设置所属地图
func (goObj *GameObject) SetMap(mapObj common.IMap) {
	goObj.mu.Lock()
	defer goObj.mu.Unlock()
	goObj.mapObject = mapObj
}

// AddComponent 添加组件
func (goObj *GameObject) AddComponent(comp common.IComponent) {
	goObj.components.AddComponent(comp)
}

// GetComponent 获取组件
func (goObj *GameObject) GetComponent(componentID string) common.IComponent {
	return goObj.components.GetComponent(componentID)
}

// RemoveComponent 移除组件
func (goObj *GameObject) RemoveComponent(componentID string) {
	goObj.components.RemoveComponent(componentID)
}

// HasComponent 检查是否有指定组件
func (goObj *GameObject) HasComponent(componentID string) bool {
	return goObj.components.HasComponent(componentID)
}

// GetAllComponents 获取所有组件
func (goObj *GameObject) GetAllComponents() []common.IComponent {
	return goObj.components.GetAllComponents()
}

// Update 更新逻辑
func (goObj *GameObject) Update(deltaTime float64) {
	goObj.components.Update(deltaTime)
}

// Destroy 销毁对象
func (goObj *GameObject) Destroy() {
	goObj.mu.Lock()
	defer goObj.mu.Unlock()

	if goObj.mapObject != nil {
		goObj.mapObject.RemoveObject(goObj.GetID())
		goObj.mapObject = nil
	}

	goObj.components.Destroy()
	goObj.isActive = false
}

// MoveTo 移动到指定位置
func (goObj *GameObject) MoveTo(targetPos common.Vector3) error {
	goObj.mu.Lock()
	defer goObj.mu.Unlock()

	if goObj.mapObject != nil {
		return goObj.mapObject.MoveObject(goObj, targetPos)
	}

	goObj.position = targetPos
	return nil
}

// Teleport 传送到指定位置
func (goObj *GameObject) Teleport(targetPos common.Vector3) error {
	goObj.mu.Lock()
	defer goObj.mu.Unlock()

	if goObj.mapObject != nil {
		return goObj.mapObject.TeleportObject(goObj, targetPos)
	}

	goObj.position = targetPos
	return nil
}

// InRange 检查是否在指定范围内
func (goObj *GameObject) InRange(targetPos common.Vector3, radius float32) bool {
	goObj.mu.RLock()
	defer goObj.mu.RUnlock()
	return goObj.position.DistanceTo(targetPos) <= radius
}

// GetDistance 获取到目标对象的距离
func (goObj *GameObject) GetDistance(target common.IGameObject) float32 {
	goObj.mu.RLock()
	defer goObj.mu.RUnlock()
	targetPos := target.GetPosition()
	return goObj.position.DistanceTo(targetPos)
}

// GetNeighbors 获取周围的对象
func (goObj *GameObject) GetNeighbors(radius float32) []common.IGameObject {
	goObj.mu.RLock()
	defer goObj.mu.RUnlock()

	if goObj.mapObject == nil {
		return nil
	}

	return goObj.mapObject.GetObjectsInRange(goObj.position, radius)
}

