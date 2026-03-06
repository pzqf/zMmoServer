package shop

import (
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// ShopManager 商城管理器
type ShopManager struct {
	items      map[int64]*ShopItem
	categories map[int]*ShopCategory
	playerData map[id.PlayerIdType]*PlayerShopData
	mutex      sync.RWMutex
}

// NewShopManager 创建商城管理器
func NewShopManager() *ShopManager {
	return &ShopManager{
		items:      make(map[int64]*ShopItem),
		categories: make(map[int]*ShopCategory),
		playerData: make(map[id.PlayerIdType]*PlayerShopData),
	}
}

// AddItem 添加商品
func (sm *ShopManager) AddItem(item *ShopItem) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.items[item.ItemID] = item
	zLog.Info("Shop item added",
		zap.Int64("item_id", item.ItemID),
		zap.String("name", item.Name),
		zap.Int64("price", item.Price))
}

// GetItem 获取商品
func (sm *ShopManager) GetItem(itemID int64) *ShopItem {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.items[itemID]
}

// GetAllItems 获取所有商品
func (sm *ShopManager) GetAllItems() []*ShopItem {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	items := make([]*ShopItem, 0, len(sm.items))
	for _, item := range sm.items {
		items = append(items, item)
	}

	return items
}

// GetItemsByCategory 获取分类商品
func (sm *ShopManager) GetItemsByCategory(categoryID int) []*ShopItem {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	items := make([]*ShopItem, 0)
	for _, item := range sm.items {
		if item.Category == categoryID {
			items = append(items, item)
		}
	}

	return items
}

// AddCategory 添加分类
func (sm *ShopManager) AddCategory(category *ShopCategory) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.categories[category.CategoryID] = category
}

// GetCategory 获取分类
func (sm *ShopManager) GetCategory(categoryID int) *ShopCategory {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.categories[categoryID]
}

// GetAllCategories 获取所有分类
func (sm *ShopManager) GetAllCategories() []*ShopCategory {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	categories := make([]*ShopCategory, 0, len(sm.categories))
	for _, category := range sm.categories {
		categories = append(categories, category)
	}

	return categories
}

// GetPlayerData 获取玩家商城数据
func (sm *ShopManager) GetPlayerData(playerID id.PlayerIdType) *PlayerShopData {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if data, exists := sm.playerData[playerID]; exists {
		data.CheckAndReset()
		return data
	}

	// 创建新的玩家数据
	data := NewPlayerShopData(playerID)
	sm.playerData[playerID] = data
	return data
}

// CanBuy 检查是否可以购买
func (sm *ShopManager) CanBuy(playerID id.PlayerIdType, itemID int64, count int) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	item := sm.items[itemID]
	if item == nil {
		return false
	}

	data := sm.GetPlayerData(playerID)
	return data.CanBuy(item, count)
}

// BuyItem 购买商品
func (sm *ShopManager) BuyItem(playerID id.PlayerIdType, itemID int64, count int) (*ShopItem, bool) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	item := sm.items[itemID]
	if item == nil {
		return nil, false
	}

	data := sm.GetPlayerData(playerID)
	if !data.CanBuy(item, count) {
		return nil, false
	}

	// 记录购买
	data.RecordBuy(itemID, count)

	totalPrice := item.Price * int64(count)
	zLog.Info("Player bought item",
		zap.Uint64("player_id", uint64(playerID)),
		zap.Int64("item_id", itemID),
		zap.String("item_name", item.Name),
		zap.Int("count", count),
		zap.Int64("total_price", totalPrice))

	return item, true
}

// GetHotItems 获取热销商品
func (sm *ShopManager) GetHotItems() []*ShopItem {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	items := make([]*ShopItem, 0)
	for _, item := range sm.items {
		if item.IsHot {
			items = append(items, item)
		}
	}

	return items
}

// GetNewItems 获取新品
func (sm *ShopManager) GetNewItems() []*ShopItem {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	items := make([]*ShopItem, 0)
	for _, item := range sm.items {
		if item.IsNew {
			items = append(items, item)
		}
	}

	return items
}

// GetDiscountItems 获取折扣商品
func (sm *ShopManager) GetDiscountItems() []*ShopItem {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	items := make([]*ShopItem, 0)
	for _, item := range sm.items {
		if item.OriginalPrice > item.Price {
			items = append(items, item)
		}
	}

	return items
}

// ResetAllDaily 重置所有玩家每日购买记录
func (sm *ShopManager) ResetAllDaily() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for _, data := range sm.playerData {
		data.ResetDaily()
	}

	zLog.Info("All players daily shop records reset")
}

// ResetAllWeekly 重置所有玩家每周购买记录
func (sm *ShopManager) ResetAllWeekly() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for _, data := range sm.playerData {
		data.ResetWeekly()
	}

	zLog.Info("All players weekly shop records reset")
}
