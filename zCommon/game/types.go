package game

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/common/types"
)

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
	GameObjectTypeResource GameObjectType = 8
)

type Vector3 = types.Vector3

type Position = types.Position

type IGameObject interface {
	GetID() id.ObjectIdType
	GetType() GameObjectType
	GetPosition() Vector3
	SetPosition(pos Vector3)
}

type ILivingObject interface {
	IGameObject
	GetHP() int64
	SetHP(hp int64)
	GetMaxHP() int64
	GetMP() int64
	SetMP(mp int64)
	GetMaxMP() int64
	GetLevel() int32
	IsAlive() bool
}

type IPlayerObject interface {
	ILivingObject
	GetPlayerID() id.PlayerIdType
	GetName() string
	GetGold() int64
	GetDiamond() int64
}
