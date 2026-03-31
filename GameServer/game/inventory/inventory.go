package inventory

import (
	"errors"
	"sync"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
	"github.com/pzqf/zMmoServer/GameServer/game/item"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

// Inventory 背包系统
type Inventory struct {
	mu       sync.RWMutex
	playerID id.PlayerIdType
	items    *zMap.TypedMap[int32, *item.Item]
	size     int32
	maxSize  int32
}

// NewInventory 创建背包
func NewInventory(playerID id.PlayerIdType, maxSize int32) *Inventory {
	return &Inventory{
		playerID: playerID,
		items:    zMap.NewTypedMap[int32, *item.Item](),
		size:     0,
		maxSize:  maxSize,
	}
}

// GetPlayerID 获取玩家ID
func (inv *Inventory) GetPlayerID() id.PlayerIdType {
	return inv.playerID
}

// GetSize 获取当前占用格子数
func (inv *Inventory) GetSize() int32 {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.size
}

// GetMaxSize 获取背包最大容量
func (inv *Inventory) GetMaxSize() int32 {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.maxSize
}

// GetFreeSpace 获取剩余空间
func (inv *Inventory) GetFreeSpace() int32 {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.maxSize - inv.size
}

// IsFull 检查背包是否已满
func (inv *Inventory) IsFull() bool {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.size >= inv.maxSize
}

// AddItem 添加物品
func (inv *Inventory) AddItem(newItem *item.Item) (int32, error) {
	if newItem == nil {
		return -1, errors.New("item is nil")
	}

	inv.mu.Lock()
	defer inv.mu.Unlock()

	// 尝试堆叠到已有物品
	if newItem.GetMaxStack() > 1 {
		inv.items.Range(func(slotIndex int32, existingItem *item.Item) bool {
			if existingItem.CanStack(newItem) && !existingItem.IsFull() {
				space := existingItem.GetSpace()
				if space >= newItem.GetCount() {
					existingItem.AddCount(newItem.GetCount())
					inv.publishItemAddEvent(existingItem, slotIndex, newItem.GetCount())
					return false
				}
				// 部分堆叠
				existingItem.AddCount(space)
				newItem.ReduceCount(space)
				inv.publishItemAddEvent(existingItem, slotIndex, space)
			}
			return true
		})
	}

	// 放入新格子
	if inv.size >= inv.maxSize {
		return -1, errors.New("inventory is full")
	}

	slotIndex := inv.findEmptySlot()
	if slotIndex < 0 {
		return -1, errors.New("no empty slot")
	}

	inv.items.Store(slotIndex, newItem)
	inv.size++

	inv.publishItemAddEvent(newItem, slotIndex, newItem.GetCount())

	zLog.Debug("Item added to inventory",
		zap.Int64("player_id", int64(inv.playerID)),
		zap.Int32("slot", slotIndex),
		zap.Int32("item_config_id", newItem.GetItemConfigID()),
		zap.Int32("count", newItem.GetCount()))

	return slotIndex, nil
}

// AddItemByConfigID 通过配置ID添加物品
func (inv *Inventory) AddItemByConfigID(itemConfigID int32, itemType item.ItemType, itemName string, count int32, maxStack int32) (int32, error) {
	newItem := item.NewItem(itemConfigID, itemType, itemName, count, maxStack)
	return inv.AddItem(newItem)
}

// RemoveItem 移除物品
func (inv *Inventory) RemoveItem(slotIndex int32, count int32) (*item.Item, error) {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	existingItem, exists := inv.items.Load(slotIndex)
	if !exists {
		return nil, errors.New("item not found")
	}

	if existingItem.GetCount() < count {
		return nil, errors.New("not enough items")
	}

	// 全部移除
	if existingItem.GetCount() == count {
		inv.items.Delete(slotIndex)
		inv.size--
		inv.publishItemRemoveEvent(existingItem, slotIndex, count)
		return existingItem, nil
	}

	// 部分移除
	existingItem.ReduceCount(count)
	inv.publishItemRemoveEvent(existingItem, slotIndex, count)

	// 返回克隆的物品
	clone := existingItem.Clone()
	clone.SetCount(count)
	return clone, nil
}

// RemoveItemByConfigID 通过配置ID移除物品
func (inv *Inventory) RemoveItemByConfigID(itemConfigID int32, count int32) (int32, error) {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	removedCount := int32(0)
	inv.items.Range(func(slotIndex int32, existingItem *item.Item) bool {
		if existingItem.GetItemConfigID() == itemConfigID {
			itemCount := existingItem.GetCount()
			if itemCount <= count-removedCount {
				inv.items.Delete(slotIndex)
				inv.size--
				removedCount += itemCount
				inv.publishItemRemoveEvent(existingItem, slotIndex, itemCount)
			} else {
				needRemove := count - removedCount
				existingItem.ReduceCount(needRemove)
				removedCount += needRemove
				inv.publishItemRemoveEvent(existingItem, slotIndex, needRemove)
			}

			if removedCount >= count {
				return false
			}
		}
		return true
	})

	if removedCount < count {
		return removedCount, errors.New("not enough items")
	}

	return removedCount, nil
}

// GetItem 获取指定格子的物品
func (inv *Inventory) GetItem(slotIndex int32) (*item.Item, error) {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	existingItem, exists := inv.items.Load(slotIndex)
	if !exists {
		return nil, errors.New("item not found")
	}
	return existingItem, nil
}

// GetItemByConfigID 通过配置ID获取物品
func (inv *Inventory) GetItemByConfigID(itemConfigID int32) (*item.Item, int32) {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	var resultItem *item.Item
	var resultIndex int32 = -1
	inv.items.Range(func(slotIndex int32, existingItem *item.Item) bool {
		if existingItem.GetItemConfigID() == itemConfigID {
			resultItem = existingItem
			resultIndex = slotIndex
			return false
		}
		return true
	})
	return resultItem, resultIndex
}

// GetItemCount 获取物品数量
func (inv *Inventory) GetItemCount(itemConfigID int32) int32 {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	count := int32(0)
	inv.items.Range(func(_ int32, existingItem *item.Item) bool {
		if existingItem.GetItemConfigID() == itemConfigID {
			count += existingItem.GetCount()
		}
		return true
	})
	return count
}

// HasItem 检查是否有指定物品
func (inv *Inventory) HasItem(itemConfigID int32, count int32) bool {
	return inv.GetItemCount(itemConfigID) >= count
}

// GetAllItems 获取所有物品
func (inv *Inventory) GetAllItems() map[int32]*item.Item {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	result := make(map[int32]*item.Item)
	inv.items.Range(func(slotIndex int32, existingItem *item.Item) bool {
		result[slotIndex] = existingItem
		return true
	})
	return result
}

// MoveItem 移动物品
func (inv *Inventory) MoveItem(fromSlot int32, toSlot int32) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	if fromSlot == toSlot {
		return nil
	}

	fromItem, fromExists := inv.items.Load(fromSlot)
	if !fromExists {
		return errors.New("source item not found")
	}

	toItem, toExists := inv.items.Load(toSlot)

	// 目标格子为空，直接移动
	if !toExists {
		inv.items.Store(toSlot, fromItem)
		inv.items.Delete(fromSlot)
		return nil
	}

	// 尝试堆叠
	if fromItem.CanStack(toItem) && !toItem.IsFull() {
		space := toItem.GetSpace()
		if space >= fromItem.GetCount() {
			toItem.AddCount(fromItem.GetCount())
			inv.items.Delete(fromSlot)
			inv.size--
		} else {
			fromItem.ReduceCount(space)
			toItem.AddCount(space)
		}
		return nil
	}

	// 交换位置
	inv.items.Store(fromSlot, toItem)
	inv.items.Store(toSlot, fromItem)
	return nil
}

// UseItem 使用物品
func (inv *Inventory) UseItem(slotIndex int32, count int32) (*item.Item, error) {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	existingItem, exists := inv.items.Load(slotIndex)
	if !exists {
		return nil, errors.New("item not found")
	}

	if existingItem.GetCount() < count {
		return nil, errors.New("not enough items")
	}

	// 发布使用事件
	event.Publish(event.NewEvent(event.EventPlayerItemUse, inv, &event.PlayerItemEventData{
		PlayerID:  inv.playerID,
		ItemID:    existingItem.GetItemID(),
		ItemCfgID: existingItem.GetItemConfigID(),
		Count:     count,
		Slot:      slotIndex,
	}))

	// 消耗物品
	if existingItem.GetCount() == count {
		inv.items.Delete(slotIndex)
		inv.size--
		return existingItem, nil
	}

	existingItem.ReduceCount(count)
	clone := existingItem.Clone()
	clone.SetCount(count)
	return clone, nil
}

// Expand 扩展背包容量
func (inv *Inventory) Expand(addSize int32) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	if addSize <= 0 {
		return errors.New("invalid expand size")
	}

	inv.maxSize += addSize
	return nil
}

// Clear 清空背包
func (inv *Inventory) Clear() {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	inv.items.Range(func(slotIndex int32, existingItem *item.Item) bool {
		inv.publishItemRemoveEvent(existingItem, slotIndex, existingItem.GetCount())
		inv.items.Delete(slotIndex)
		return true
	})

	inv.size = 0
}

// findEmptySlot 查找空格子
func (inv *Inventory) findEmptySlot() int32 {
	for i := int32(0); i < inv.maxSize; i++ {
		if _, exists := inv.items.Load(i); !exists {
			return i
		}
	}
	return -1
}

// publishItemAddEvent 发布物品添加事件
func (inv *Inventory) publishItemAddEvent(item *item.Item, slotIndex int32, count int32) {
	event.Publish(event.NewEvent(event.EventPlayerItemAdd, inv, &event.PlayerItemEventData{
		PlayerID:  inv.playerID,
		ItemID:    item.GetItemID(),
		ItemCfgID: item.GetItemConfigID(),
		Count:     count,
		Slot:      slotIndex,
	}))
}

// publishItemRemoveEvent 发布物品移除事件
func (inv *Inventory) publishItemRemoveEvent(item *item.Item, slotIndex int32, count int32) {
	event.Publish(event.NewEvent(event.EventPlayerItemRemove, inv, &event.PlayerItemEventData{
		PlayerID:  inv.playerID,
		ItemID:    item.GetItemID(),
		ItemCfgID: item.GetItemConfigID(),
		Count:     count,
		Slot:      slotIndex,
	}))
}
