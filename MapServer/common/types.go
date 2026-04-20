package common

import (
	"github.com/pzqf/zCommon/common/types"
	"github.com/pzqf/zCommon/game"
)

type Vector3 = types.Vector3

type GameObjectType = game.GameObjectType

const (
	GameObjectTypePlayer   = game.GameObjectTypePlayer
	GameObjectTypeMonster  = game.GameObjectTypeMonster
	GameObjectTypeNPC      = game.GameObjectTypeNPC
	GameObjectTypeItem     = game.GameObjectTypeItem
	GameObjectTypeBuilding = game.GameObjectTypeBuilding
	GameObjectTypeResource = game.GameObjectTypeResource
)

type IGameObject = game.IGameObject
