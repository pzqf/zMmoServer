package shop

import (
	"github.com/pzqf/zMmoShared/common/id"
	"time"
)

// ShopItem 商城商品
type ShopItem struct {
	ItemID       int64     // 商品ID（对应配置表）
	Name         string    // 商品名称
	Description  string    // 商品描述
	Icon         string    // 商品图标
	PriceType    int       // 价格类型（1:金币, 2:钻石, 3:代币）
	Price        int64     // 价格
	OriginalPrice int64    // 原价（用于显示折扣）
	Category     int       // 商品分类
	ItemType     int       // 物品类型
	ItemCount    int       // 物品数量
	LevelReq     int       // 等级要求
	VipReq       int       // VIP等级要求
	LimitType    int       // 限购类型（0:不限购, 1:每日限购, 2:每周限购, 3:永久限购）
	LimitCount   int       // 限购数量
	IsHot        bool      // 是否热销
	IsNew        bool      // 是否新品
	SortOrder    int       // 排序权重
	StartTime    time.Time // 上架时间
	EndTime      time.Time // 下架时间
}

// ShopCategory 商城分类
type ShopCategory struct {
	CategoryID   int    // 分类ID
	Name         string // 分类名称
	Icon         string // 分类图标
	SortOrder    int    // 排序权重
	IsVisible    bool   // 是否可见
}

// PlayerShopRecord 玩家商城购买记录
type PlayerShopRecord struct {
	PlayerID   id.PlayerIdType // 玩家ID
	ItemID     int64           // 商品ID
	BuyCount   int             // 购买数量
	TotalPrice int64           // 总价格
	BuyTime    time.Time       // 购买时间
}

// PlayerShopData 玩家商城数据
type PlayerShopData struct {
	PlayerID      id.PlayerIdType          // 玩家ID
	DailyBuys     map[int64]int            // 每日购买记录（商品ID -> 数量）
	WeeklyBuys    map[int64]int            // 每周购买记录（商品ID -> 数量）
	TotalBuys     map[int64]int            // 永久购买记录（商品ID -> 数量）
	LastResetTime time.Time                // 上次重置时间
}

// NewPlayerShopData 创建玩家商城数据
func NewPlayerShopData(playerID id.PlayerIdType) *PlayerShopData {
	return &PlayerShopData{
		PlayerID:      playerID,
		DailyBuys:     make(map[int64]int),
		WeeklyBuys:    make(map[int64]int),
		TotalBuys:     make(map[int64]int),
		LastResetTime: time.Now(),
	}
}

// CanBuy 检查是否可以购买
func (psd *PlayerShopData) CanBuy(item *ShopItem, buyCount int) bool {
	// 检查时间限制
	now := time.Now()
	if now.Before(item.StartTime) || now.After(item.EndTime) {
		return false
	}

	// 检查限购
	switch item.LimitType {
	case 1: // 每日限购
		if psd.DailyBuys[item.ItemID]+buyCount > item.LimitCount {
			return false
		}
	case 2: // 每周限购
		if psd.WeeklyBuys[item.ItemID]+buyCount > item.LimitCount {
			return false
		}
	case 3: // 永久限购
		if psd.TotalBuys[item.ItemID]+buyCount > item.LimitCount {
			return false
		}
	}

	return true
}

// RecordBuy 记录购买
func (psd *PlayerShopData) RecordBuy(itemID int64, count int) {
	psd.TotalBuys[itemID] += count
	psd.DailyBuys[itemID] += count
	psd.WeeklyBuys[itemID] += count
}

// ResetDaily 重置每日购买记录
func (psd *PlayerShopData) ResetDaily() {
	psd.DailyBuys = make(map[int64]int)
}

// ResetWeekly 重置每周购买记录
func (psd *PlayerShopData) ResetWeekly() {
	psd.WeeklyBuys = make(map[int64]int)
}

// CheckAndReset 检查并重置购买记录
func (psd *PlayerShopData) CheckAndReset() {
	now := time.Now()
	lastReset := psd.LastResetTime

	// 检查是否需要重置每日记录
	if now.Day() != lastReset.Day() || now.Month() != lastReset.Month() || now.Year() != lastReset.Year() {
		psd.ResetDaily()
	}

	// 检查是否需要重置每周记录
	_, lastWeek := lastReset.ISOWeek()
	_, currentWeek := now.ISOWeek()
	if lastWeek != currentWeek {
		psd.ResetWeekly()
	}

	psd.LastResetTime = now
}
