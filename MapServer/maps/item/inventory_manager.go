package item

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/models"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type InventoryItem struct {
	ItemID    int32 `json:"item_id"`
	Count     int32 `json:"count"`
	SlotIndex int32 `json:"slot_index"`
}

type Inventory struct {
	PlayerID id.PlayerIdType  `json:"player_id"`
	Items    []*InventoryItem `json:"items"`
	MaxSlots int32            `json:"max_slots"`
}

type InventoryManager struct {
	inventories   *zMap.TypedMap[id.PlayerIdType, *Inventory]
	tableManager  *tables.TableManager
}

func NewInventoryManager() *InventoryManager {
	return &InventoryManager{
		inventories: zMap.NewTypedMap[id.PlayerIdType, *Inventory](),
	}
}

func (im *InventoryManager) SetTableManager(tm *tables.TableManager) {
	im.tableManager = tm
}

func (im *InventoryManager) GetItemConfig(itemID int32) *models.ItemBase {
	if im.tableManager == nil {
		return nil
	}
	item, ok := im.tableManager.GetItemLoader().GetItem(itemID)
	if !ok {
		return nil
	}
	return item
}

func (im *InventoryManager) GetInventory(playerID id.PlayerIdType) *Inventory {
	inventory, exists := im.inventories.Load(playerID)
	if !exists {
		inventory = &Inventory{
			PlayerID: playerID,
			Items:    make([]*InventoryItem, 0),
			MaxSlots: 20,
		}
		im.inventories.Store(playerID, inventory)
	}

	return inventory
}

func (im *InventoryManager) AddItem(playerID id.PlayerIdType, itemID int32, count int32) error {
	itemConfig := im.GetItemConfig(itemID)
	if itemConfig == nil {
		return nil
	}

	inventory, exists := im.inventories.Load(playerID)
	if !exists {
		inventory = &Inventory{
			PlayerID: playerID,
			Items:    make([]*InventoryItem, 0),
			MaxSlots: 20,
		}
		im.inventories.Store(playerID, inventory)
	}

	if len(inventory.Items) >= int(inventory.MaxSlots) {
		return nil
	}

	maxStack := itemConfig.StackLimit
	if maxStack > 1 {
		for _, item := range inventory.Items {
			if item.ItemID == itemID && item.Count < maxStack {
				item.Count += count
				if item.Count > maxStack {
					count = item.Count - maxStack
					item.Count = maxStack
				} else {
					count = 0
					break
				}
			}
		}
	}

	if count > 0 {
		for i := int32(0); i < inventory.MaxSlots; i++ {
			slotFound := false
			for _, item := range inventory.Items {
				if item.SlotIndex == i {
					slotFound = true
					break
				}
			}
			if !slotFound {
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

func (im *InventoryManager) RemoveItem(playerID id.PlayerIdType, itemID int32, count int32) error {
	inventory, exists := im.inventories.Load(playerID)
	if !exists {
		return nil
	}

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

func (im *InventoryManager) UseItem(playerID id.PlayerIdType, itemID int32) error {
	itemConfig := im.GetItemConfig(itemID)
	if itemConfig == nil {
		return nil
	}

	inventory, exists := im.inventories.Load(playerID)
	if !exists {
		return nil
	}

	for i, item := range inventory.Items {
		if item.ItemID == itemID {
			item.Count--
			if item.Count <= 0 {
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

func (im *InventoryManager) GetItemCount(playerID id.PlayerIdType, itemID int32) int32 {
	inventory, exists := im.inventories.Load(playerID)
	if !exists {
		return 0
	}

	count := int32(0)
	for _, item := range inventory.Items {
		if item.ItemID == itemID {
			count += item.Count
		}
	}

	return count
}

func (im *InventoryManager) GetInventoryItems(playerID id.PlayerIdType) []*InventoryItem {
	inventory, exists := im.inventories.Load(playerID)
	if !exists {
		return []*InventoryItem{}
	}

	items := make([]*InventoryItem, len(inventory.Items))
	copy(items, inventory.Items)

	return items
}
