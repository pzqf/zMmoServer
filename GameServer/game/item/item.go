package item

import (
	"sync"

	"github.com/pzqf/zMmoShared/common/id"
)

// ItemType 物品类型
type ItemType int32

const (
	ItemTypeWeapon   ItemType = 1 // 武器
	ItemTypeArmor    ItemType = 2 // 防具
	ItemTypeAccessory ItemType = 3 // 饰品
	ItemTypeConsumable ItemType = 4 // 消耗品
	ItemTypeMaterial ItemType = 5 // 材料
	ItemTypeQuest    ItemType = 6 // 任务物品
	ItemTypeCurrency ItemType = 7 // 货币
)

// ItemQuality 物品品质
type ItemQuality int32

const (
	ItemQualityNormal   ItemQuality = 1 // 普通
	ItemQualityGood     ItemQuality = 2 // 良好
	ItemQualityRare     ItemQuality = 3 // 稀有
	ItemQualityEpic     ItemQuality = 4 // 史诗
	ItemQualityLegendary ItemQuality = 5 // 传说
)

// BindType 绑定类型
type BindType int32

const (
	BindTypeNone   BindType = 0 // 不绑定
	BindTypePickup BindType = 1 // 拾取绑定
	BindTypeEquip  BindType = 2 // 装备绑定
)

// Item 物品结构
type Item struct {
	mu           sync.RWMutex
	itemID       id.ItemIdType
	itemConfigID int32
	itemType     ItemType
	itemName     string
	count        int32
	maxStack     int32
	quality      ItemQuality
	level        int32
	bindType     BindType
	isBound      bool
	expireTime   int64
	attributes   map[string]int32
}

// NewItem 创建新物品
func NewItem(itemConfigID int32, itemType ItemType, itemName string, count int32, maxStack int32) *Item {
	return &Item{
		itemConfigID: itemConfigID,
		itemType:     itemType,
		itemName:     itemName,
		count:        count,
		maxStack:     maxStack,
		quality:      ItemQualityNormal,
		level:        1,
		bindType:     BindTypeNone,
		isBound:      false,
		expireTime:   0,
		attributes:   make(map[string]int32),
	}
}

// GetItemID 获取物品ID
func (i *Item) GetItemID() id.ItemIdType {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.itemID
}

// SetItemID 设置物品ID
func (i *Item) SetItemID(itemID id.ItemIdType) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.itemID = itemID
}

// GetItemConfigID 获取物品配置ID
func (i *Item) GetItemConfigID() int32 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.itemConfigID
}

// GetItemType 获取物品类型
func (i *Item) GetItemType() ItemType {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.itemType
}

// GetItemName 获取物品名称
func (i *Item) GetItemName() string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.itemName
}

// GetCount 获取物品数量
func (i *Item) GetCount() int32 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.count
}

// SetCount 设置物品数量
func (i *Item) SetCount(count int32) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.count = count
}

// AddCount 增加物品数量
func (i *Item) AddCount(count int32) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.count += count
}

// ReduceCount 减少物品数量
func (i *Item) ReduceCount(count int32) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.count < count {
		return false
	}
	i.count -= count
	return true
}

// GetMaxStack 获取最大堆叠数量
func (i *Item) GetMaxStack() int32 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.maxStack
}

// GetQuality 获取物品品质
func (i *Item) GetQuality() ItemQuality {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.quality
}

// SetQuality 设置物品品质
func (i *Item) SetQuality(quality ItemQuality) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.quality = quality
}

// GetLevel 获取物品等级
func (i *Item) GetLevel() int32 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.level
}

// SetLevel 设置物品等级
func (i *Item) SetLevel(level int32) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.level = level
}

// GetBindType 获取绑定类型
func (i *Item) GetBindType() BindType {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.bindType
}

// SetBindType 设置绑定类型
func (i *Item) SetBindType(bindType BindType) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.bindType = bindType
}

// IsBound 检查是否已绑定
func (i *Item) IsBound() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.isBound
}

// SetBound 设置绑定状态
func (i *Item) SetBound(bound bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.isBound = bound
}

// CanStack 检查是否可以堆叠
func (i *Item) CanStack(other *Item) bool {
	if other == nil {
		return false
	}
	return i.itemConfigID == other.itemConfigID &&
		i.quality == other.quality &&
		i.level == other.level &&
		i.isBound == other.isBound
}

// IsFull 检查是否已满
func (i *Item) IsFull() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.count >= i.maxStack
}

// GetSpace 获取剩余空间
func (i *Item) GetSpace() int32 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	space := i.maxStack - i.count
	if space < 0 {
		return 0
	}
	return space
}

// GetExpireTime 获取过期时间
func (i *Item) GetExpireTime() int64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.expireTime
}

// SetExpireTime 设置过期时间
func (i *Item) SetExpireTime(expireTime int64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.expireTime = expireTime
}

// IsExpired 检查是否已过期
func (i *Item) IsExpired(currentTime int64) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.expireTime > 0 && currentTime > i.expireTime
}

// GetAttribute 获取属性值
func (i *Item) GetAttribute(name string) int32 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.attributes[name]
}

// SetAttribute 设置属性值
func (i *Item) SetAttribute(name string, value int32) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.attributes[name] = value
}

// GetAttributes 获取所有属性
func (i *Item) GetAttributes() map[string]int32 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result := make(map[string]int32)
	for k, v := range i.attributes {
		result[k] = v
	}
	return result
}

// Clone 克隆物品
func (i *Item) Clone() *Item {
	i.mu.RLock()
	defer i.mu.RUnlock()

	clone := &Item{
		itemConfigID: i.itemConfigID,
		itemType:     i.itemType,
		itemName:     i.itemName,
		count:        i.count,
		maxStack:     i.maxStack,
		quality:      i.quality,
		level:        i.level,
		bindType:     i.bindType,
		isBound:      i.isBound,
		expireTime:   i.expireTime,
		attributes:   make(map[string]int32),
	}

	for k, v := range i.attributes {
		clone.attributes[k] = v
	}

	return clone
}
