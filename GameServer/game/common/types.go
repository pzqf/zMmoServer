package common

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/common/types"
	"github.com/pzqf/zCommon/game"
	"github.com/pzqf/zEngine/zEvent"
)

type Vector3 = types.Vector3

func NewVector3(x, y, z float32) Vector3 {
	return types.NewVector3(x, y, z)
}

type GameObjectType = game.GameObjectType

const (
	GameObjectTypeBasic    = game.GameObjectTypeBasic
	GameObjectTypeLiving   = game.GameObjectTypeLiving
	GameObjectTypePlayer   = game.GameObjectTypePlayer
	GameObjectTypeNPC      = game.GameObjectTypeNPC
	GameObjectTypeMonster  = game.GameObjectTypeMonster
	GameObjectTypePet      = game.GameObjectTypePet
	GameObjectTypeItem     = game.GameObjectTypeItem
	GameObjectTypeBuilding = game.GameObjectTypeBuilding
	GameObjectTypeResource = game.GameObjectTypeResource
)

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

type ClientSender interface {
	SendToClient(sessionID interface{}, protoId int32, data []byte) error
}

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
