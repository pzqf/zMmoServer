package common

import (
	"math"

	"github.com/pzqf/zEngine/zEvent"
	"github.com/pzqf/zMmoShared/common/id"
)

// GameObjectType 游戏对象类型
type GameObjectType int

const (
	GameObjectTypeBasic    GameObjectType = 0
	GameObjectTypeLiving   GameObjectType = 1
	GameObjectTypePlayer   GameObjectType = 2
	GameObjectTypeNPC      GameObjectType = 3
	GameObjectTypeMonster  GameObjectType = 4
	GameObjectTypePet      GameObjectType = 5
	GameObjectTypeItem     GameObjectType = 6
	GameObjectTypeBuilding GameObjectType = 7
)

// Vector3 三维向量
type Vector3 struct {
	X, Y, Z float32
}

// NewVector3 创建三维向量
func NewVector3(x, y, z float32) Vector3 {
	return Vector3{X: x, Y: y, Z: z}
}

// Add 向量相加
func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

// Subtract 向量相减
func (v Vector3) Subtract(other Vector3) Vector3 {
	return Vector3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

// MultiplyScalar 向量乘以标量
func (v Vector3) MultiplyScalar(scalar float32) Vector3 {
	return Vector3{X: v.X * scalar, Y: v.Y * scalar, Z: v.Z * scalar}
}

// DistanceTo 计算两个向量之间的距离
func (v Vector3) DistanceTo(other Vector3) float32 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	dz := v.Z - other.Z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}

// Length 向量长度
func (v Vector3) Length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

// Normalize 标准化向量
func (v Vector3) Normalize() Vector3 {
	length := v.Length()
	if length == 0 {
		return Vector3{}
	}
	return Vector3{X: v.X / length, Y: v.Y / length, Z: v.Z / length}
}

// IComponent 组件接口
type IComponent interface {
	GetID() string
	GetGameObject() IGameObject
	SetGameObject(obj IGameObject)
	Init() error
	Update(deltaTime float64)
	Destroy()
	IsActive() bool
	SetActive(active bool)
}

// IGameObject 游戏对象接口
type IGameObject interface {
	GetID() id.ObjectIdType
	GetName() string
	GetType() GameObjectType
	GetPosition() Vector3
	SetPosition(pos Vector3)
	Update(deltaTime float64)
	Destroy()
	IsActive() bool
	SetActive(active bool)
	GetEventEmitter() *zEvent.EventBus
	GetMap() IMap
	SetMap(mapObj IMap)
	AddComponent(component IComponent)
	GetComponent(componentID string) IComponent
	RemoveComponent(componentID string)
	HasComponent(componentID string) bool
	GetAllComponents() []IComponent
}

// IMap 地图接口
type IMap interface {
	GetID() id.MapIdType
	GetName() string
	GetObjectsInRange(pos Vector3, radius float32) []IGameObject
	GetObjectsByType(objectType GameObjectType) []IGameObject
	AddObject(object IGameObject)
	RemoveObject(objectID id.ObjectIdType)
	MoveObject(object IGameObject, targetPos Vector3) error
	TeleportObject(object IGameObject, targetPos Vector3) error
}
