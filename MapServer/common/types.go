package common

import "github.com/pzqf/zCommon/common/id"

// Vector3 三维向量
type Vector3 struct {
	X, Y, Z float32
}

// DistanceTo 计算到另一个点的距离平方
func (v Vector3) DistanceTo(other Vector3) float32 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	dz := v.Z - other.Z
	return dx*dx + dy*dy + dz*dz
}

// GameObjectType 游戏对象类型
type GameObjectType int

const (
	GameObjectTypePlayer GameObjectType = iota
	GameObjectTypeMonster
	GameObjectTypeNPC
	GameObjectTypeItem
	GameObjectTypeBuilding
	GameObjectTypeResource
)

// IGameObject 游戏对象接口
type IGameObject interface {
	GetID() id.ObjectIdType
	GetType() GameObjectType
	GetPosition() Vector3
	SetPosition(pos Vector3)
}

