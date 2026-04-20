package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInventory(t *testing.T) {
	inv := NewInventory(30)

	assert.NotNil(t, inv)
	assert.Equal(t, int32(30), inv.GetSize())
	assert.Equal(t, int64(0), inv.Count())
}

func TestInventoryAddItem(t *testing.T) {
	inv := NewInventory(30)
	item := NewItem(1, 1001, "Sword", ItemTypeWeapon, 1)

	_, err := inv.AddItem(item)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), inv.Count())
}

func TestInventoryAddItemStack(t *testing.T) {
	inv := NewInventory(30)

	item1 := NewItem(1, 1001, "Potion", ItemTypeConsumable, 50)
	item1.MaxStack = 99
	slot1, err := inv.AddItem(item1)
	assert.NoError(t, err)

	item2 := NewItem(2, 1001, "Potion", ItemTypeConsumable, 30)
	item2.MaxStack = 99
	slot2, err := inv.AddItem(item2)
	assert.NoError(t, err)

	assert.Equal(t, slot1, slot2)
	assert.Equal(t, int64(1), inv.Count())

	stored, ok := inv.GetItem(slot1)
	assert.True(t, ok)
	assert.Equal(t, int32(80), stored.GetCount())
}

func TestInventoryRemoveItem(t *testing.T) {
	inv := NewInventory(30)
	item := NewItem(1, 1001, "Sword", ItemTypeWeapon, 1)
	_, _ = inv.AddItem(item)

	_, err := inv.RemoveItem(0, 1)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), inv.Count())
}

func TestInventoryRemoveItemPartial(t *testing.T) {
	inv := NewInventory(30)
	item := NewItem(1, 1001, "Potion", ItemTypeConsumable, 10)
	_, _ = inv.AddItem(item)

	_, err := inv.RemoveItem(0, 3)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), inv.Count())

	stored, ok := inv.GetItem(0)
	assert.True(t, ok)
	assert.Equal(t, int32(7), stored.GetCount())
}

func TestInventoryRemoveItemNotFound(t *testing.T) {
	inv := NewInventory(30)

	_, err := inv.RemoveItem(0, 1)
	assert.Error(t, err)
}

func TestInventoryRemoveItemNotEnough(t *testing.T) {
	inv := NewInventory(30)
	item := NewItem(1, 1001, "Potion", ItemTypeConsumable, 5)
	_, _ = inv.AddItem(item)

	_, err := inv.RemoveItem(0, 10)
	assert.Error(t, err)
}

func TestInventoryHasSpace(t *testing.T) {
	inv := NewInventory(2)

	assert.True(t, inv.HasSpace(1001, 1))

	item1 := NewItem(1, 1001, "Sword", ItemTypeWeapon, 1)
	_, _ = inv.AddItem(item1)

	assert.True(t, inv.HasSpace(1001, 1))
}

func TestInventoryFull(t *testing.T) {
	inv := NewInventory(1)

	item1 := NewItem(1, 1001, "Sword", ItemTypeWeapon, 1)
	_, _ = inv.AddItem(item1)

	item2 := NewItem(2, 2001, "Shield", ItemTypeArmor, 1)
	_, err := inv.AddItem(item2)
	assert.Error(t, err)
}

func TestInventoryGetAllItems(t *testing.T) {
	inv := NewInventory(30)

	item1 := NewItem(1, 1001, "Sword", ItemTypeWeapon, 1)
	item2 := NewItem(2, 1002, "Shield", ItemTypeArmor, 1)
	_, _ = inv.AddItem(item1)
	_, _ = inv.AddItem(item2)

	items := inv.GetAllItems()
	assert.Len(t, items, 2)
}

func TestEquipmentEquip(t *testing.T) {
	equip := NewEquipment()

	item := NewItem(1, 1001, "Sword", ItemTypeWeapon, 1)
	old, err := equip.Equip(EquipSlotWeapon, item)

	assert.NoError(t, err)
	assert.Nil(t, old)

	stored, ok := equip.GetEquipment(EquipSlotWeapon)
	assert.True(t, ok)
	assert.Equal(t, item, stored)
}

func TestEquipmentReplace(t *testing.T) {
	equip := NewEquipment()

	item1 := NewItem(1, 1001, "Sword", ItemTypeWeapon, 1)
	_, _ = equip.Equip(EquipSlotWeapon, item1)

	item2 := NewItem(2, 1002, "Axe", ItemTypeWeapon, 1)
	old, err := equip.Equip(EquipSlotWeapon, item2)

	assert.NoError(t, err)
	assert.Equal(t, item1, old)

	stored, _ := equip.GetEquipment(EquipSlotWeapon)
	assert.Equal(t, item2, stored)
}

func TestEquipmentUnequip(t *testing.T) {
	equip := NewEquipment()

	item := NewItem(1, 1001, "Sword", ItemTypeWeapon, 1)
	_, _ = equip.Equip(EquipSlotWeapon, item)

	removed, err := equip.Unequip(EquipSlotWeapon)

	assert.NoError(t, err)
	assert.Equal(t, item, removed)

	_, ok := equip.GetEquipment(EquipSlotWeapon)
	assert.False(t, ok)
}

func TestEquipmentUnequipEmpty(t *testing.T) {
	equip := NewEquipment()

	_, err := equip.Unequip(EquipSlotWeapon)
	assert.Error(t, err)
}

func TestEquipmentGetAllEquipments(t *testing.T) {
	equip := NewEquipment()

	weapon := NewItem(1, 1001, "Sword", ItemTypeWeapon, 1)
	armor := NewItem(2, 1002, "Armor", ItemTypeArmor, 1)
	_, _ = equip.Equip(EquipSlotWeapon, weapon)
	_, _ = equip.Equip(EquipSlotArmor, armor)

	all := equip.GetAllEquipments()
	assert.Len(t, all, 2)
}
