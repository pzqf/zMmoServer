package economy

import (
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

// AuctionStatus 拍卖状态
type AuctionStatus int32

const (
	AuctionStatusActive    AuctionStatus = 1 // 拍卖中
	AuctionStatusCompleted AuctionStatus = 2 // 已完成
	AuctionStatusCancelled AuctionStatus = 3 // 已取消
)

// Auction 拍卖
type Auction struct {
	AuctionID     int64           `json:"auction_id"`
	SellerID      id.PlayerIdType `json:"seller_id"`
	ItemID        int32           `json:"item_id"`
	ItemCount     int32           `json:"item_count"`
	StartingPrice int64           `json:"starting_price"`
	CurrentPrice  int64           `json:"current_price"`
	BuyerID       id.PlayerIdType `json:"buyer_id"`
	EndTime       time.Time       `json:"end_time"`
	Status        AuctionStatus   `json:"status"`
	CurrencyType  CurrencyType    `json:"currency_type"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// AuctionManager 拍卖行管理器
type AuctionManager struct {
	auctions       *zMap.TypedMap[int64, *Auction]
	activeAuctions []*Auction
	activeMu       sync.RWMutex
	currencyManager *CurrencyManager
}

func NewAuctionManager() *AuctionManager {
	return &AuctionManager{
		auctions:       zMap.NewTypedMap[int64, *Auction](),
		activeAuctions: make([]*Auction, 0),
	}
}

func (am *AuctionManager) SetCurrencyManager(cm *CurrencyManager) {
	am.currencyManager = cm
}

// CreateAuction 创建拍卖
func (am *AuctionManager) CreateAuction(sellerID id.PlayerIdType, itemID, itemCount int32, startingPrice int64, duration time.Duration, currencyType CurrencyType) (*Auction, error) {
	// 生成拍卖ID
	auctionID := time.Now().UnixNano()

	// 创建拍卖
	auction := &Auction{
		AuctionID:     auctionID,
		SellerID:      sellerID,
		ItemID:        itemID,
		ItemCount:     itemCount,
		StartingPrice: startingPrice,
		CurrentPrice:  startingPrice,
		BuyerID:       0,
		EndTime:       time.Now().Add(duration),
		Status:        AuctionStatusActive,
		CurrencyType:  currencyType,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 保存拍卖
	am.auctions.Store(auctionID, auction)

	// 保护activeAuctions切片
	am.activeMu.Lock()
	am.activeAuctions = append(am.activeAuctions, auction)
	am.activeMu.Unlock()

	zLog.Debug("Auction created",
		zap.Int64("auction_id", auctionID),
		zap.Int64("seller_id", int64(sellerID)),
		zap.Int32("item_id", itemID),
		zap.Int32("item_count", itemCount),
		zap.Int64("starting_price", startingPrice))

	return auction, nil
}

// PlaceBid 出价
func (am *AuctionManager) PlaceBid(auctionID int64, buyerID id.PlayerIdType, bidPrice int64) error {
	// 检查拍卖是否存在
	auction, exists := am.auctions.Load(auctionID)
	if !exists {
		return nil
	}

	// 检查拍卖状态
	if auction.Status != AuctionStatusActive {
		return nil
	}

	// 检查拍卖是否已结束
	if time.Now().After(auction.EndTime) {
		am.completeAuction(auction)
		return nil
	}

	// 检查出价是否高于当前价格
	if bidPrice <= auction.CurrentPrice {
		return nil
	}

	// 更新拍卖
	auction.CurrentPrice = bidPrice
	auction.BuyerID = buyerID
	auction.UpdatedAt = time.Now()

	zLog.Debug("Bid placed",
		zap.Int64("auction_id", auctionID),
		zap.Int64("buyer_id", int64(buyerID)),
		zap.Int64("bid_price", bidPrice))

	return nil
}

// CancelAuction 取消拍卖
func (am *AuctionManager) CancelAuction(auctionID int64, sellerID id.PlayerIdType) error {
	// 检查拍卖是否存在
	auction, exists := am.auctions.Load(auctionID)
	if !exists {
		return nil
	}

	// 检查拍卖状态
	if auction.Status != AuctionStatusActive {
		return nil
	}

	// 检查是否是卖家
	if auction.SellerID != sellerID {
		return nil
	}

	// 更新拍卖状态
	auction.Status = AuctionStatusCancelled
	auction.UpdatedAt = time.Now()

	// 从活跃拍卖列表中移除
	am.removeFromActiveAuctions(auctionID)

	zLog.Debug("Auction cancelled",
		zap.Int64("auction_id", auctionID),
		zap.Int64("seller_id", int64(sellerID)))

	return nil
}

// GetAuction 获取拍卖信息
func (am *AuctionManager) GetAuction(auctionID int64) *Auction {
	auction, _ := am.auctions.Load(auctionID)
	return auction
}

// GetActiveAuctions 获取活跃拍卖列表
func (am *AuctionManager) GetActiveAuctions() []*Auction {
	am.activeMu.RLock()
	defer am.activeMu.RUnlock()

	// 复制活跃拍卖列表
	auctions := make([]*Auction, len(am.activeAuctions))
	copy(auctions, am.activeAuctions)

	return auctions
}

// GetAuctionsByItemID 根据物品ID获取拍卖列表
func (am *AuctionManager) GetAuctionsByItemID(itemID int32) []*Auction {
	am.activeMu.RLock()
	defer am.activeMu.RUnlock()

	actions := make([]*Auction, 0)
	for _, auction := range am.activeAuctions {
		if auction.ItemID == itemID && auction.Status == AuctionStatusActive {
			actions = append(actions, auction)
		}
	}

	return actions
}

// GetAuctionsBySeller 获取卖家的拍卖列表
func (am *AuctionManager) GetAuctionsBySeller(sellerID id.PlayerIdType) []*Auction {
	actions := make([]*Auction, 0)
	am.auctions.Range(func(_ int64, auction *Auction) bool {
		if auction.SellerID == sellerID {
			actions = append(actions, auction)
		}
		return true
	})

	return actions
}

// GetAuctionsByBuyer 获取买家的拍卖列表
func (am *AuctionManager) GetAuctionsByBuyer(buyerID id.PlayerIdType) []*Auction {
	actions := make([]*Auction, 0)
	am.auctions.Range(func(_ int64, auction *Auction) bool {
		if auction.BuyerID == buyerID && auction.Status == AuctionStatusCompleted {
			actions = append(actions, auction)
		}
		return true
	})

	return actions
}

// UpdateAuctions 更新拍卖状态
func (am *AuctionManager) UpdateAuctions() {
	am.activeMu.Lock()
	defer am.activeMu.Unlock()

	now := time.Now()
	completedAuctions := make([]*Auction, 0)

	// 检查所有活跃拍卖
	for _, auction := range am.activeAuctions {
		if now.After(auction.EndTime) {
			am.completeAuction(auction)
			completedAuctions = append(completedAuctions, auction)
		}
	}

	// 从活跃拍卖列表中移除已完成的拍卖
	for _, auction := range completedAuctions {
		am.removeFromActiveAuctions(auction.AuctionID)
	}
}

// completeAuction 完成拍卖
func (am *AuctionManager) completeAuction(auction *Auction) {
	if auction.BuyerID > 0 {
		auction.Status = AuctionStatusCompleted

		if am.currencyManager != nil {
			if err := am.currencyManager.RemoveCurrency(auction.BuyerID, auction.CurrencyType, auction.CurrentPrice); err != nil {
				zLog.Warn("Failed to deduct buyer currency",
					zap.Int64("auction_id", auction.AuctionID),
					zap.Int64("buyer_id", int64(auction.BuyerID)),
					zap.Error(err))
			} else {
				if err := am.currencyManager.AddCurrency(auction.SellerID, auction.CurrencyType, auction.CurrentPrice); err != nil {
					zLog.Warn("Failed to add seller currency",
						zap.Int64("auction_id", auction.AuctionID),
						zap.Int64("seller_id", int64(auction.SellerID)),
						zap.Error(err))
				}
			}
		}

		zLog.Info("Auction completed with buyer",
			zap.Int64("auction_id", auction.AuctionID),
			zap.Int64("seller_id", int64(auction.SellerID)),
			zap.Int64("buyer_id", int64(auction.BuyerID)),
			zap.Int32("item_id", auction.ItemID),
			zap.Int64("final_price", auction.CurrentPrice))
	} else {
		auction.Status = AuctionStatusCancelled

		zLog.Info("Auction cancelled, no buyer",
			zap.Int64("auction_id", auction.AuctionID),
			zap.Int64("seller_id", int64(auction.SellerID)),
			zap.Int32("item_id", auction.ItemID))
	}
	auction.UpdatedAt = time.Now()
}

// removeFromActiveAuctions 从活跃拍卖列表中移除
func (am *AuctionManager) removeFromActiveAuctions(auctionID int64) {
	newActiveAuctions := make([]*Auction, 0)
	for _, auction := range am.activeAuctions {
		if auction.AuctionID != auctionID {
			newActiveAuctions = append(newActiveAuctions, auction)
		}
	}
	am.activeAuctions = newActiveAuctions
}
