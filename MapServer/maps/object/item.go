package object

import (
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoShared/common/id"
)

// Item 物品对象
type Item struct {
	id       id.ObjectIdType
	itemID   int32
	name     string
	position common.Vector3
	quantity int32
	itemType ItemType
	rarity   ItemRarity
	effects  []ItemEffect
	isPicked bool
}

// ItemType 物品类型
type ItemType int

const (
	ItemTypeWeapon ItemType = iota
	ItemTypeArmor
	ItemTypeConsumable
	ItemTypeQuest
	ItemTypeMaterial
	ItemTypeCurrency
)

// ItemRarity 物品稀有度
type ItemRarity int

const (
	ItemRarityCommon ItemRarity = iota
	ItemRarityUncommon
	ItemRarityRare
	ItemRarityEpic
	ItemRarityLegendary
)

// ItemEffect 物品效果
type ItemEffect struct {
	EffectType string
	Value      int32
}

// NewItem 创建新物品
func NewItem(objectID id.ObjectIdType, itemID int32, name string, pos common.Vector3, quantity int32, itemType ItemType, rarity ItemRarity) *Item {
	return &Item{
		id:       objectID,
		itemID:   itemID,
		name:     name,
		position: pos,
		quantity: quantity,
		itemType: itemType,
		rarity:   rarity,
		effects:  make([]ItemEffect, 0),
		isPicked: false,
	}
}

// GetID 获取对象ID
func (i *Item) GetID() id.ObjectIdType {
	return i.id
}

// GetType 获取对象类型
func (i *Item) GetType() common.GameObjectType {
	return common.GameObjectTypeItem
}

// GetPosition 获取位置
func (i *Item) GetPosition() common.Vector3 {
	return i.position
}

// SetPosition 设置位置
func (i *Item) SetPosition(pos common.Vector3) {
	i.position = pos
}

// GetItemID 获取物品ID
func (i *Item) GetItemID() int32 {
	return i.itemID
}

// GetName 获取物品名称
func (i *Item) GetName() string {
	return i.name
}

// GetQuantity 获取数量
func (i *Item) GetQuantity() int32 {
	return i.quantity
}

// SetQuantity 设置数量
func (i *Item) SetQuantity(quantity int32) {
	i.quantity = quantity
}

// GetItemType 获取物品类型
func (i *Item) GetItemType() ItemType {
	return i.itemType
}

// GetRarity 获取稀有度
func (i *Item) GetRarity() ItemRarity {
	return i.rarity
}

// AddEffect 添加效果
func (i *Item) AddEffect(effect ItemEffect) {
	i.effects = append(i.effects, effect)
}

// GetEffects 获取效果列表
func (i *Item) GetEffects() []ItemEffect {
	return i.effects
}

// IsPicked 检查是否被拾取
func (i *Item) IsPicked() bool {
	return i.isPicked
}

// SetPicked 设置拾取状态
func (i *Item) SetPicked(picked bool) {
	i.isPicked = picked
}

// CanBePicked 检查是否可以被拾取
func (i *Item) CanBePicked() bool {
	return !i.isPicked && i.quantity > 0
}
