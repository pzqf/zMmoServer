package auction

import (
	"sync"

	"github.com/pzqf/zCommon/common/id"
)

// AuctionType 拍卖类型
type AuctionType int32

const (
	AuctionTypeFixed   AuctionType = 1 // 一口价
	AuctionTypeBidding AuctionType = 2 // 竞拍
)

// AuctionStatus 拍卖状�?
type AuctionStatus int32

const (
	AuctionStatusPending  AuctionStatus = 0 // 待上�?
	AuctionStatusActive   AuctionStatus = 1 // 拍卖�?
	AuctionStatusSold     AuctionStatus = 2 // 已售�?
	AuctionStatusExpired  AuctionStatus = 3 // 已过�?
	AuctionStatusCanceled AuctionStatus = 4 // 已取�?
)

// AuctionItem 拍卖物品
type AuctionItem struct {
	mu           sync.RWMutex
	auctionID    id.AuctionIdType
	sellerID     id.PlayerIdType
	sellerName   string
	itemConfigID int32
	itemName     string
	count        int32
	auctionType  AuctionType
	startPrice   int64
	currentPrice int64
	buyoutPrice  int64
	buyerID      id.PlayerIdType
	buyerName    string
	status       AuctionStatus
	startTime    int64
	endTime      int64
}

// NewAuctionItem 创建拍卖物品
func NewAuctionItem(sellerID id.PlayerIdType, sellerName string, itemConfigID int32, itemName string, count int32) *AuctionItem {
	return &AuctionItem{
		sellerID:     sellerID,
		sellerName:   sellerName,
		itemConfigID: itemConfigID,
		itemName:     itemName,
		count:        count,
		auctionType:  AuctionTypeFixed,
		startPrice:   0,
		currentPrice: 0,
		buyoutPrice:  0,
		buyerID:      0,
		buyerName:    "",
		status:       AuctionStatusPending,
		startTime:    0,
		endTime:      0,
	}
}

// GetAuctionID 获取拍卖ID
func (ai *AuctionItem) GetAuctionID() id.AuctionIdType {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.auctionID
}

// SetAuctionID 设置拍卖ID
func (ai *AuctionItem) SetAuctionID(auctionID id.AuctionIdType) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.auctionID = auctionID
}

// GetSellerID 获取卖家ID
func (ai *AuctionItem) GetSellerID() id.PlayerIdType {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.sellerID
}

// GetSellerName 获取卖家名称
func (ai *AuctionItem) GetSellerName() string {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.sellerName
}

// GetItemConfigID 获取物品配置ID
func (ai *AuctionItem) GetItemConfigID() int32 {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.itemConfigID
}

// GetItemName 获取物品名称
func (ai *AuctionItem) GetItemName() string {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.itemName
}

// GetCount 获取数量
func (ai *AuctionItem) GetCount() int32 {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.count
}

// GetAuctionType 获取拍卖类型
func (ai *AuctionItem) GetAuctionType() AuctionType {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.auctionType
}

// SetAuctionType 设置拍卖类型
func (ai *AuctionItem) SetAuctionType(auctionType AuctionType) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.auctionType = auctionType
}

// GetStartPrice 获取起拍价格
func (ai *AuctionItem) GetStartPrice() int64 {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.startPrice
}

// SetStartPrice 设置起拍价格
func (ai *AuctionItem) SetStartPrice(price int64) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.startPrice = price
	ai.currentPrice = price
}

// GetCurrentPrice 获取当前价格
func (ai *AuctionItem) GetCurrentPrice() int64 {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.currentPrice
}

// SetCurrentPrice 设置当前价格
func (ai *AuctionItem) SetCurrentPrice(price int64) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.currentPrice = price
}

// GetBuyoutPrice 获取一口价
func (ai *AuctionItem) GetBuyoutPrice() int64 {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.buyoutPrice
}

// SetBuyoutPrice 设置一口价
func (ai *AuctionItem) SetBuyoutPrice(price int64) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.buyoutPrice = price
}

// GetBuyerID 获取买家ID
func (ai *AuctionItem) GetBuyerID() id.PlayerIdType {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.buyerID
}

// GetBuyerName 获取买家名称
func (ai *AuctionItem) GetBuyerName() string {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.buyerName
}

// GetStatus 获取状态
func (ai *AuctionItem) GetStatus() AuctionStatus {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.status
}

// SetStatus 设置状态
func (ai *AuctionItem) SetStatus(status AuctionStatus) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.status = status
}

// GetStartTime 获取开始时间
func (ai *AuctionItem) GetStartTime() int64 {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.startTime
}

// GetEndTime 获取结束时间
func (ai *AuctionItem) GetEndTime() int64 {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.endTime
}

// SetEndTime 设置结束时间
func (ai *AuctionItem) SetEndTime(endTime int64) {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.endTime = endTime
}

// IsActive 检查是否活跃
func (ai *AuctionItem) IsActive() bool {
	return ai.GetStatus() == AuctionStatusActive
}

// IsExpired 检查是否过期
func (ai *AuctionItem) IsExpired(currentTime int64) bool {
	ai.mu.RLock()
	defer ai.mu.RUnlock()
	return ai.endTime > 0 && currentTime > ai.endTime
}

// CanBid 检查是否可以竞拍
func (ai *AuctionItem) CanBid(bidPrice int64) bool {
	ai.mu.RLock()
	defer ai.mu.RUnlock()

	if ai.status != AuctionStatusActive {
		return false
	}

	if ai.auctionType != AuctionTypeBidding {
		return false
	}

	return bidPrice > ai.currentPrice
}

// CanBuyout 检查是否可以一口价购买
func (ai *AuctionItem) CanBuyout() bool {
	ai.mu.RLock()
	defer ai.mu.RUnlock()

	if ai.status != AuctionStatusActive {
		return false
	}

	return ai.buyoutPrice > 0
}

// Bid 竞拍
func (ai *AuctionItem) Bid(buyerID id.PlayerIdType, buyerName string, bidPrice int64) bool {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	if ai.status != AuctionStatusActive {
		return false
	}

	if ai.auctionType != AuctionTypeBidding {
		return false
	}

	if bidPrice <= ai.currentPrice {
		return false
	}

	ai.currentPrice = bidPrice
	ai.buyerID = buyerID
	ai.buyerName = buyerName

	return true
}

// Buyout 一口价购买
func (ai *AuctionItem) Buyout(buyerID id.PlayerIdType, buyerName string) bool {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	if ai.status != AuctionStatusActive {
		return false
	}

	if ai.buyoutPrice <= 0 {
		return false
	}

	ai.currentPrice = ai.buyoutPrice
	ai.buyerID = buyerID
	ai.buyerName = buyerName
	ai.status = AuctionStatusSold

	return true
}

// Cancel 取消拍卖
func (ai *AuctionItem) Cancel() bool {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	if ai.status != AuctionStatusActive && ai.status != AuctionStatusPending {
		return false
	}

	ai.status = AuctionStatusCanceled
	return true
}

// Clone 克隆拍卖物品
func (ai *AuctionItem) Clone() *AuctionItem {
	ai.mu.RLock()
	defer ai.mu.RUnlock()

	return &AuctionItem{
		auctionID:    ai.auctionID,
		sellerID:     ai.sellerID,
		sellerName:   ai.sellerName,
		itemConfigID: ai.itemConfigID,
		itemName:     ai.itemName,
		count:        ai.count,
		auctionType:  ai.auctionType,
		startPrice:   ai.startPrice,
		currentPrice: ai.currentPrice,
		buyoutPrice:  ai.buyoutPrice,
		buyerID:      ai.buyerID,
		buyerName:    ai.buyerName,
		status:       ai.status,
		startTime:    ai.startTime,
		endTime:      ai.endTime,
	}
}
