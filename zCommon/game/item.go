package game

import (
	"fmt"
	"sync"

	"github.com/pzqf/zUtil/zMap"
)

type ItemRarity int

const (
	ItemRarityCommon    ItemRarity = 0
	ItemRarityUncommon  ItemRarity = 1
	ItemRarityRare      ItemRarity = 2
	ItemRarityEpic      ItemRarity = 3
	ItemRarityLegendary ItemRarity = 4
)

type ItemType int

const (
	ItemTypeWeapon     ItemType = 0
	ItemTypeArmor      ItemType = 1
	ItemTypeConsumable ItemType = 2
	ItemTypeQuest      ItemType = 3
	ItemTypeMaterial   ItemType = 4
	ItemTypeCurrency   ItemType = 5
)

type EquipSlot int

const (
	EquipSlotWeapon  EquipSlot = 0
	EquipSlotArmor   EquipSlot = 1
	EquipSlotHelmet  EquipSlot = 2
	EquipSlotBoots   EquipSlot = 3
	EquipSlotGloves  EquipSlot = 4
	EquipSlotNecklace EquipSlot = 5
	EquipSlotRing1   EquipSlot = 6
	EquipSlotRing2   EquipSlot = 7
	EquipSlotBelt    EquipSlot = 8
	EquipSlotShoulder EquipSlot = 9
)

type Item struct {
	mu         sync.RWMutex
	ItemID     int64
	ConfigID   int32
	Name       string
	Type       ItemType
	Rarity     ItemRarity
	Count      int32
	MaxStack   int32
	Level      int32
	Bind       bool
	Properties map[string]int32
}

func NewItem(itemID int64, configID int32, name string, itemType ItemType, count int32) *Item {
	return &Item{
		ItemID:     itemID,
		ConfigID:   configID,
		Name:       name,
		Type:       itemType,
		Rarity:     ItemRarityCommon,
		Count:      count,
		MaxStack:   99,
		Properties: make(map[string]int32),
	}
}

func (item *Item) GetCount() int32 {
	item.mu.RLock()
	defer item.mu.RUnlock()
	return item.Count
}

func (item *Item) SetCount(count int32) {
	item.mu.Lock()
	defer item.mu.Unlock()
	item.Count = count
}

func (item *Item) AddCount(count int32) int32 {
	item.mu.Lock()
	defer item.mu.Unlock()
	item.Count += count
	return item.Count
}

type Inventory struct {
	BaseComponent
	mu       sync.RWMutex
	items    *zMap.TypedMap[int32, *Item]
	size     int32
	freeSlot int32
}

func NewInventory(size int32) *Inventory {
	return &Inventory{
		BaseComponent: NewBaseComponent("inventory"),
		items:         zMap.NewTypedMap[int32, *Item](),
		size:          size,
		freeSlot:      0,
	}
}

func (inv *Inventory) GetSize() int32 {
	return inv.size
}

func (inv *Inventory) GetItem(slot int32) (*Item, bool) {
	return inv.items.Load(slot)
}

func (inv *Inventory) GetAllItems() map[int32]*Item {
	result := make(map[int32]*Item)
	inv.items.Range(func(slot int32, item *Item) bool {
		result[slot] = item
		return true
	})
	return result
}

func (inv *Inventory) AddItem(item *Item) (int32, error) {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	for slot := int32(0); slot < inv.size; slot++ {
		existing, ok := inv.items.Load(slot)
		if ok && existing.ConfigID == item.ConfigID && existing.Count < existing.MaxStack {
			remaining := existing.MaxStack - existing.Count
			if item.Count <= remaining {
				existing.AddCount(item.Count)
				return slot, nil
			}
			existing.AddCount(remaining)
			item.Count -= remaining
		}
	}

	if item.Count > 0 {
		slot := inv.findFreeSlot()
		if slot < 0 {
			return -1, fmt.Errorf("inventory full")
		}
		inv.items.Store(slot, item)
		return slot, nil
	}

	return -1, nil
}

func (inv *Inventory) RemoveItem(slot int32, count int32) (*Item, error) {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	item, ok := inv.items.Load(slot)
	if !ok {
		return nil, fmt.Errorf("item not found in slot %d", slot)
	}

	if item.Count < count {
		return nil, fmt.Errorf("not enough items: have %d, need %d", item.Count, count)
	}

	item.Count -= count
	if item.Count <= 0 {
		inv.items.Delete(slot)
	}
	return item, nil
}

func (inv *Inventory) HasSpace(configID int32, count int32) bool {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	remaining := count
	inv.items.Range(func(slot int32, item *Item) bool {
		if item.ConfigID == configID && item.Count < item.MaxStack {
			canStack := item.MaxStack - item.Count
			if remaining <= canStack {
				remaining = 0
				return false
			}
			remaining -= canStack
		}
		return true
	})

	if remaining <= 0 {
		return true
	}

	freeSlots := inv.size - int32(inv.items.Len())
	return freeSlots > 0
}

func (inv *Inventory) Count() int64 {
	return inv.items.Len()
}

func (inv *Inventory) findFreeSlot() int32 {
	for slot := int32(0); slot < inv.size; slot++ {
		if _, ok := inv.items.Load(slot); !ok {
			return slot
		}
	}
	return -1
}

type Equipment struct {
	BaseComponent
	mu        sync.RWMutex
	equipments *zMap.TypedMap[EquipSlot, *Item]
}

func NewEquipment() *Equipment {
	return &Equipment{
		BaseComponent: NewBaseComponent("equipment"),
		equipments:    zMap.NewTypedMap[EquipSlot, *Item](),
	}
}

func (e *Equipment) Equip(slot EquipSlot, item *Item) (*Item, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	old, _ := e.equipments.Load(slot)
	e.equipments.Store(slot, item)
	return old, nil
}

func (e *Equipment) Unequip(slot EquipSlot) (*Item, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	item, ok := e.equipments.Load(slot)
	if !ok {
		return nil, fmt.Errorf("no equipment in slot %d", slot)
	}
	e.equipments.Delete(slot)
	return item, nil
}

func (e *Equipment) GetEquipment(slot EquipSlot) (*Item, bool) {
	return e.equipments.Load(slot)
}

func (e *Equipment) GetAllEquipments() map[EquipSlot]*Item {
	result := make(map[EquipSlot]*Item)
	e.equipments.Range(func(slot EquipSlot, item *Item) bool {
		result[slot] = item
		return true
	})
	return result
}

func (e *Equipment) CalculateTotalProperty(propertyName string) int32 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	total := int32(0)
	e.equipments.Range(func(slot EquipSlot, item *Item) bool {
		if item.Properties != nil {
			if val, ok := item.Properties[propertyName]; ok {
				total += val
			}
		}
		return true
	})
	return total
}

func (e *Equipment) CalculateAttackBonus() int32 {
	return e.CalculateTotalProperty("attack")
}

func (e *Equipment) CalculateDefenseBonus() int32 {
	return e.CalculateTotalProperty("defense")
}

func (e *Equipment) CalculateHPBonus() int32 {
	return e.CalculateTotalProperty("hp")
}

func (e *Equipment) CalculateMPBonus() int32 {
	return e.CalculateTotalProperty("mp")
}

func (e *Equipment) CalculateCritRateBonus() float32 {
	return float32(e.CalculateTotalProperty("crit_rate")) / 100.0
}

func (e *Equipment) CalculateSpeedBonus() float32 {
	return float32(e.CalculateTotalProperty("speed")) / 100.0
}
