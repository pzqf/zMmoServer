package economy

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// ShopType 商店类型
type ShopType int32

const (
	ShopTypeNPC     ShopType = 1 // NPC商店
	ShopTypeSpecial ShopType = 2 // 特殊商店
	ShopTypeEvent   ShopType = 3 // 活动商店
)

// ShopItem 商店物品
type ShopItem struct {
	ItemID       int32        `json:"item_id"`
	Price        int64        `json:"price"`
	CurrencyType CurrencyType `json:"currency_type"`
	Stock        int32        `json:"stock"`
	MaxStock     int32        `json:"max_stock"`
	RefreshTime  int64        `json:"refresh_time"`
	LevelReq     int32        `json:"level_req"`
	ClassReq     int32        `json:"class_req"`
}

// Shop 商店
type Shop struct {
	ShopID          int32       `json:"shop_id"`
	Name            string      `json:"name"`
	Type            ShopType    `json:"type"`
	NPCID           int32       `json:"npc_id"`
	Items           []*ShopItem `json:"items"`
	RefreshInterval int64       `json:"refresh_interval"` // 刷新间隔（秒）
}

// ShopManager 商店管理器
type ShopManager struct {
	mu    sync.RWMutex
	shops map[int32]*Shop
}

// NewShopManager 创建商店管理器
func NewShopManager() *ShopManager {
	return &ShopManager{
		shops: make(map[int32]*Shop),
	}
}

// LoadShopConfig 加载商店配置
func (sm *ShopManager) LoadShopConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		zLog.Error("Failed to read shop config file", zap.Error(err))
		return err
	}

	var shops []*Shop
	if err := json.Unmarshal(data, &shops); err != nil {
		zLog.Error("Failed to unmarshal shop config", zap.Error(err))
		return err
	}

	for _, shop := range shops {
		sm.shops[shop.ShopID] = shop
	}

	zLog.Info("Shop config loaded successfully", zap.Int("count", len(sm.shops)))
	return nil
}

// GetShop 获取商店信息
func (sm *ShopManager) GetShop(shopID int32) *Shop {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.shops[shopID]
}

// GetShopsByNPC 获取NPC的商店列表
func (sm *ShopManager) GetShopsByNPC(npcID int32) []*Shop {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shops := make([]*Shop, 0)
	for _, shop := range sm.shops {
		if shop.NPCID == npcID {
			shops = append(shops, shop)
		}
	}

	return shops
}

// GetShopItem 获取商店物品
func (sm *ShopManager) GetShopItem(shopID, itemID int32) *ShopItem {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shop, exists := sm.shops[shopID]
	if !exists {
		return nil
	}

	for _, item := range shop.Items {
		if item.ItemID == itemID {
			return item
		}
	}

	return nil
}

// BuyItem 购买物品
func (sm *ShopManager) BuyItem(shopID, itemID, count int32, playerLevel, playerClass int32) (int64, CurrencyType, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 检查商店是否存在
	shop, exists := sm.shops[shopID]
	if !exists {
		return 0, 0, nil
	}

	// 查找物品
	var shopItem *ShopItem
	for _, item := range shop.Items {
		if item.ItemID == itemID {
			shopItem = item
			break
		}
	}

	if shopItem == nil {
		return 0, 0, nil
	}

	// 检查等级要求
	if playerLevel < shopItem.LevelReq {
		return 0, 0, nil
	}

	// 检查职业要求
	if shopItem.ClassReq > 0 && playerClass != shopItem.ClassReq {
		return 0, 0, nil
	}

	// 检查库存
	if shopItem.Stock < count {
		return 0, 0, nil
	}

	// 计算总价
	totalPrice := shopItem.Price * int64(count)

	// 减少库存
	shopItem.Stock -= count

	zLog.Debug("Item bought from shop",
		zap.Int32("shop_id", shopID),
		zap.Int32("item_id", itemID),
		zap.Int32("count", count),
		zap.Int64("total_price", totalPrice),
		zap.Int32("currency_type", int32(shopItem.CurrencyType)))

	return totalPrice, shopItem.CurrencyType, nil
}

// RefreshShop 刷新商店
func (sm *ShopManager) RefreshShop(shopID int32) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 检查商店是否存在
	shop, exists := sm.shops[shopID]
	if !exists {
		return nil
	}

	// 刷新物品库存
	for _, item := range shop.Items {
		item.Stock = item.MaxStock
		item.RefreshTime = time.Now().Unix()
	}

	zLog.Debug("Shop refreshed", zap.Int32("shop_id", shopID))

	return nil
}

// UpdateShops 更新商店状态
func (sm *ShopManager) UpdateShops() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now().Unix()

	for _, shop := range sm.shops {
		// 检查是否需要刷新
		if shop.RefreshInterval > 0 {
			for _, item := range shop.Items {
				if now-item.RefreshTime >= shop.RefreshInterval {
					item.Stock = item.MaxStock
					item.RefreshTime = now
				}
			}
		}
	}
}
