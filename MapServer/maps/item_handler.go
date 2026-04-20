package maps

import (
	"fmt"

	"github.com/pzqf/zMmoServer/MapServer/maps/item"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
)

func (m *Map) AddItem(player *object.Player, itemID int32, count int32) error {
	if m.inventoryManager == nil {
		return fmt.Errorf("inventory manager not initialized")
	}
	return m.inventoryManager.AddItem(player.GetPlayerID(), itemID, count)
}

func (m *Map) RemoveItem(player *object.Player, itemID int32, count int32) error {
	if m.inventoryManager == nil {
		return fmt.Errorf("inventory manager not initialized")
	}
	return m.inventoryManager.RemoveItem(player.GetPlayerID(), itemID, count)
}

func (m *Map) GetItemCount(player *object.Player, itemID int32) int32 {
	if m.inventoryManager == nil {
		return 0
	}
	return m.inventoryManager.GetItemCount(player.GetPlayerID(), itemID)
}

func (m *Map) GetInventoryItems(player *object.Player) []*item.InventoryItem {
	if m.inventoryManager == nil {
		return []*item.InventoryItem{}
	}
	return m.inventoryManager.GetInventoryItems(player.GetPlayerID())
}
