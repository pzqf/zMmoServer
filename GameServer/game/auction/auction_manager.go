package auction

import (
	"errors"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// AuctionManager 拍卖管理器
type AuctionManager struct {
	mu            sync.RWMutex
	auctions     map[id.AuctionIdType]*AuctionItem
	auctionCount  int32
	maxAuctions   int32
}

// NewAuctionManager 创建拍卖管理器
func NewAuctionManager(maxAuctions int32) *AuctionManager {
	return &AuctionManager{
		auctions:    make(map[id.AuctionIdType]*AuctionItem),
		auctionCount: 0,
		maxAuctions: maxAuctions,
	}
}

// GetAuctionCount 获取拍卖数量
func (am *AuctionManager) GetAuctionCount() int32 {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.auctionCount
}

// GetMaxAuctions 获取最大拍卖数
func (am *AuctionManager) GetMaxAuctions() int32 {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.maxAuctions
}

// IsFull 检查拍卖行是否已满
func (am *AuctionManager) IsFull() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.auctionCount >= am.maxAuctions
}

// AddAuction 添加拍卖
func (am *AuctionManager) AddAuction(auction *AuctionItem) error {
	if auction == nil {
		return errors.New("auction is nil")
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	if am.auctionCount >= am.maxAuctions {
		return errors.New("auction is full")
	}

	am.auctionCount++
	auction.SetAuctionID(id.AuctionIdType(am.auctionCount))
	am.auctions[auction.GetAuctionID()] = auction

	zLog.Debug("Auction added",
		zap.Int64("auction_id", int64(auction.GetAuctionID())),
		zap.Int64("seller_id", int64(auction.GetSellerID())),
		zap.Int32("item_id", auction.GetItemConfigID()))

	return nil
}

// RemoveAuction 移除拍卖
func (am *AuctionManager) RemoveAuction(auctionID id.AuctionIdType) (*AuctionItem, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	auction, exists := am.auctions[auctionID]
	if !exists {
		return nil, errors.New("auction not found")
	}

	delete(am.auctions, auctionID)
	am.auctionCount--

	zLog.Debug("Auction removed",
		zap.Int64("auction_id", int64(auctionID)))

	return auction, nil
}

// GetAuction 获取拍卖
func (am *AuctionManager) GetAuction(auctionID id.AuctionIdType) (*AuctionItem, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	auction, exists := am.auctions[auctionID]
	if !exists {
		return nil, errors.New("auction not found")
	}
	return auction, nil
}

// GetAllAuctions 获取所有拍卖
func (am *AuctionManager) GetAllAuctions() []*AuctionItem {
	am.mu.RLock()
	defer am.mu.RUnlock()

	auctions := make([]*AuctionItem, 0, len(am.auctions))
	for _, auction := range am.auctions {
		auctions = append(auctions, auction)
	}
	return auctions
}

// GetActiveAuctions 获取活跃拍卖
func (am *AuctionManager) GetActiveAuctions() []*AuctionItem {
	am.mu.RLock()
	defer am.mu.RUnlock()

	auctions := make([]*AuctionItem, 0)
	for _, auction := range am.auctions {
		if auction.IsActive() {
			auctions = append(auctions, auction)
		}
	}
	return auctions
}

// GetAuctionsByType 获取指定类型的拍卖
func (am *AuctionManager) GetAuctionsByType(auctionType AuctionType) []*AuctionItem {
	am.mu.RLock()
	defer am.mu.RUnlock()

	auctions := make([]*AuctionItem, 0)
	for _, auction := range am.auctions {
		if auction.GetAuctionType() == auctionType {
			auctions = append(auctions, auction)
		}
	}
	return auctions
}

// GetAuctionsBySeller 获取指定卖家的拍卖
func (am *AuctionManager) GetAuctionsBySeller(sellerID id.PlayerIdType) []*AuctionItem {
	am.mu.RLock()
	defer am.mu.RUnlock()

	auctions := make([]*AuctionItem, 0)
	for _, auction := range am.auctions {
		if auction.GetSellerID() == sellerID {
			auctions = append(auctions, auction)
		}
	}
	return auctions
}

// Bid 竞拍
func (am *AuctionManager) Bid(auctionID id.AuctionIdType, buyerID id.PlayerIdType, buyerName string, bidPrice int64) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	auction, exists := am.auctions[auctionID]
	if !exists {
		return errors.New("auction not found")
	}

	if !auction.CanBid(bidPrice) {
		return errors.New("cannot bid")
	}

	if !auction.Bid(buyerID, buyerName, bidPrice) {
		return errors.New("bid failed")
	}

	am.publishAuctionBidEvent(auction, buyerID)

	zLog.Info("Auction bid",
		zap.Int64("auction_id", int64(auctionID)),
		zap.Int64("buyer_id", int64(buyerID)),
		zap.Int64("price", bidPrice))

	return nil
}

// Buyout 一口价购买
func (am *AuctionManager) Buyout(auctionID id.AuctionIdType, buyerID id.PlayerIdType, buyerName string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	auction, exists := am.auctions[auctionID]
	if !exists {
		return errors.New("auction not found")
	}

	if !auction.CanBuyout() {
		return errors.New("cannot buyout")
	}

	if !auction.Buyout(buyerID, buyerName) {
		return errors.New("buyout failed")
	}

	am.publishAuctionSoldEvent(auction, buyerID)

	zLog.Info("Auction buyout",
		zap.Int64("auction_id", int64(auctionID)),
		zap.Int64("buyer_id", int64(buyerID)),
		zap.Int64("price", auction.GetBuyoutPrice()))

	return nil
}

// Cancel 取消拍卖
func (am *AuctionManager) Cancel(auctionID id.AuctionIdType, sellerID id.PlayerIdType) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	auction, exists := am.auctions[auctionID]
	if !exists {
		return errors.New("auction not found")
	}

	if auction.GetSellerID() != sellerID {
		return errors.New("not seller")
	}

	if !auction.Cancel() {
		return errors.New("cancel failed")
	}

	am.publishAuctionCancelEvent(auction)

	zLog.Info("Auction canceled",
		zap.Int64("auction_id", int64(auctionID)),
		zap.Int64("seller_id", int64(sellerID)))

	return nil
}

// UpdateExpiredAuctions 更新过期拍卖
func (am *AuctionManager) UpdateExpiredAuctions() int32 {
	am.mu.Lock()
	defer am.mu.Unlock()

	currentTime := time.Now().UnixMilli()
	expiredCount := int32(0)

	for _, auction := range am.auctions {
		if auction.IsActive() && auction.IsExpired(currentTime) {
			auction.SetStatus(AuctionStatusExpired)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		zLog.Info("Expired auctions updated",
			zap.Int32("count", expiredCount))
	}

	return expiredCount
}

// Clear 清空拍卖行
func (am *AuctionManager) Clear() {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.auctions = make(map[id.AuctionIdType]*AuctionItem)
	am.auctionCount = 0
}

// publishAuctionBidEvent 发布拍卖竞拍事件
func (am *AuctionManager) publishAuctionBidEvent(auction *AuctionItem, buyerID id.PlayerIdType) {
	event.Publish(event.NewEvent(event.EventAuctionBid, am, &event.AuctionEventData{
		AuctionID: auction.GetAuctionID(),
		SellerID:  auction.GetSellerID(),
		BuyerID:   buyerID,
		Price:     auction.GetCurrentPrice(),
	}))
}

// publishAuctionSoldEvent 发布拍卖售出事件
func (am *AuctionManager) publishAuctionSoldEvent(auction *AuctionItem, buyerID id.PlayerIdType) {
	event.Publish(event.NewEvent(event.EventAuctionSold, am, &event.AuctionEventData{
		AuctionID: auction.GetAuctionID(),
		SellerID:  auction.GetSellerID(),
		BuyerID:   buyerID,
		Price:     auction.GetCurrentPrice(),
	}))
}

// publishAuctionCancelEvent 发布拍卖取消事件
func (am *AuctionManager) publishAuctionCancelEvent(auction *AuctionItem) {
	event.Publish(event.NewEvent(event.EventAuctionCancel, am, &event.AuctionEventData{
		AuctionID: auction.GetAuctionID(),
		SellerID:  auction.GetSellerID(),
	}))
}
