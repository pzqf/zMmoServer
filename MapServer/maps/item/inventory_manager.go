package item

import (
	"sync"

	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"

	"github.com/pzqf/zEngine/zLog"
)

// InventoryItem 背包物品
type InventoryItem struct {
	ItemID    int32 `json:"item_id"`
	Count     int32 `json:"count"`
	SlotIndex int32 `json:"slot_index"`
}

// Inventory 玩家背包
type Inventory struct {
	PlayerID id.PlayerIdType    `json:"player_id"`
	Items    []*InventoryItem   `json:"items"`
	MaxSlots int32              `json:"max_slots"`
}

// InventoryManager 背包管理器
type InventoryManager struct {
	mu         sync.RWMutex
	inventories map[id.PlayerIdType]*Inventory
	configManager *ItemConfigManager
}

// NewInventoryManager 创建背包管理器
func NewInventoryManager() *InventoryManager {
	return &InventoryManager{
		inventories:   make(map[id.PlayerIdType]*Inventory),
		configManager: NewItemConfigManager(),
	}
}

// LoadItemConfig 加载物品配置
func (im *InventoryManager) LoadItemConfig(filePath string) error {
	return im.configManager.LoadConfig(filePath)
}

// GetItemConfig 获取物品配置
func (im *InventoryManager) GetItemConfig(itemID int32) *ItemConfig {
	return im.configManager.GetConfig(itemID)
}

// GetInventory 获取玩家背包
func (im *InventoryManager) GetInventory(playerID id.PlayerIdType) *Inventory {
	im.mu.RLock()
	defer im.mu.RUnlock()

	inventory, exists := im.inventories[playerID]
	if !exists {
		// 创建新背包
		inventory = &Inventory{
			PlayerID: playerID,
			Items:    make([]*InventoryItem, 0),
			MaxSlots: 20, // 默认20个槽位
		}
		im.inventories[playerID] = inventory
	}

	return inventory
}

// AddItem 添加物品到背包
func (im *InventoryManager) AddItem(playerID id.PlayerIdType, itemID int32, count int32) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	// 获取物品配置
	itemConfig := im.configManager.GetConfig(itemID)
	if itemConfig == nil {
		return nil
	}

	// 获取或创建背包
	inventory, exists := im.inventories[playerID]
	if !exists {
		inventory = &Inventory{
			PlayerID: playerID,
			Items:    make([]*InventoryItem, 0),
			MaxSlots: 20,
		}
		im.inventories[playerID] = inventory
	}

	// 检查背包是否已满
	if len(inventory.Items) >= int(inventory.MaxSlots) {
		return nil
	}

	// 检查是否可以堆叠
	if itemConfig.MaxStack > 1 {
		for _, item := range inventory.Items {
			if item.ItemID == itemID && item.Count < itemConfig.MaxStack {
				// 堆叠物品
				item.Count += count
				if item.Count > itemConfig.MaxStack {
					count = item.Count - itemConfig.MaxStack
					item.Count = itemConfig.MaxStack
				} else {
					count = 0
					break
				}
			}
		}
	}

	// 如果还有剩余物品，添加新物品
	if count > 0 {
		// 找到空槽位
		for i := int32(0); i < inventory.MaxSlots; i++ {
			slotFound := false
			for _, item := range inventory.Items {
				if item.SlotIndex == i {
					slotFound = true
					break
				}
			}
			if !slotFound {
				// 添加新物品
				newItem := &InventoryItem{
					ItemID:    itemID,
					Count:     count,
					SlotIndex: i,
				}
				inventory.Items = append(inventory.Items, newItem)
				break
			}
		}
	}

	zLog.Debug("Item added to inventory",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("item_id", itemID),
		zap.Int32("count", count))

	return nil
}

// RemoveItem 从背包中移除物品
func (im *InventoryManager) RemoveItem(playerID id.PlayerIdType, itemID int32, count int32) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	// 获取背包
	inventory, exists := im.inventories[playerID]
	if !exists {
		return nil
	}

	// 查找物品并移除
	remaining := count
	for i := len(inventory.Items) - 1; i >= 0; i-- {
		item := inventory.Items[i]
		if item.ItemID == itemID {
			if item.Count > remaining {
				item.Count -= remaining
				remaining = 0
				break
			} else {
				remaining -= item.Count
				// 移除物品
				inventory.Items = append(inventory.Items[:i], inventory.Items[i+1:]...)
				if remaining <= 0 {
					break
				}
			}
		}
	}

	zLog.Debug("Item removed from inventory",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("item_id", itemID),
		zap.Int32("count", count))

	return nil
}

// UseItem 使用物品
func (im *InventoryManager) UseItem(playerID id.PlayerIdType, itemID int32) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	// 获取物品配置
	itemConfig := im.configManager.GetConfig(itemID)
	if itemConfig == nil {
		return nil
	}

	// 获取背包
	inventory, exists := im.inventories[playerID]
	if !exists {
		return nil
	}

	// 查找物品
	for i, item := range inventory.Items {
		if item.ItemID == itemID {
			// 减少物品数量
			item.Count--
			if item.Count <= 0 {
				// 移除物品
				inventory.Items = append(inventory.Items[:i], inventory.Items[i+1:]...)
			}

			zLog.Debug("Item used",
				zap.Int64("player_id", int64(playerID)),
				zap.Int32("item_id", itemID))

			return nil
		}
	}

	return nil
}

// GetItemCount 获取物品数量
func (im *InventoryManager) GetItemCount(playerID id.PlayerIdType, itemID int32) int32 {
	im.mu.RLock()
	defer im.mu.RUnlock()

	// 获取背包
	inventory, exists := im.inventories[playerID]
	if !exists {
		return 0
	}

	// 计算物品数量
	count := int32(0)
	for _, item := range inventory.Items {
		if item.ItemID == itemID {
			count += item.Count
		}
	}

	return count
}

// GetInventoryItems 获取背包物品列表
func (im *InventoryManager) GetInventoryItems(playerID id.PlayerIdType) []*InventoryItem {
	im.mu.RLock()
	defer im.mu.RUnlock()

	// 获取背包
	inventory, exists := im.inventories[playerID]
	if !exists {
		return []*InventoryItem{}
	}

	// 复制物品列表
	items := make([]*InventoryItem, len(inventory.Items))
	copy(items, inventory.Items)

	return items
}