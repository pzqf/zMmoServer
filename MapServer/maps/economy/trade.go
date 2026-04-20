package economy

import (
	"errors"
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

var (
	ErrTradeNotFound      = errors.New("trade not found")
	ErrTradeInvalidStatus = errors.New("invalid trade status")
	ErrTradeNotParticipant = errors.New("player is not a trade participant")
	ErrTradeAlreadyTrading = errors.New("player is already in a trade")
	ErrInsufficientCurrency = errors.New("insufficient currency")
)

// TradeStatus 交易状态
type TradeStatus int32

const (
	TradeStatusPending   TradeStatus = 1 // 等待中
	TradeStatusAccepted  TradeStatus = 2 // 已接受
	TradeStatusCancelled TradeStatus = 3 // 已取消
	TradeStatusCompleted TradeStatus = 4 // 已完成
)

// TradeItem 交易物品
type TradeItem struct {
	ItemID int32 `json:"item_id"`
	Count  int32 `json:"count"`
}

// Trade 交易
type Trade struct {
	TradeID           int64                  `json:"trade_id"`
	InitiatorID       id.PlayerIdType        `json:"initiator_id"`
	TargetID          id.PlayerIdType        `json:"target_id"`
	Status            TradeStatus            `json:"status"`
	InitiatorItems    []*TradeItem           `json:"initiator_items"`
	TargetItems       []*TradeItem           `json:"target_items"`
	InitiatorCurrency map[CurrencyType]int64 `json:"initiator_currency"`
	TargetCurrency    map[CurrencyType]int64 `json:"target_currency"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// TradeManager 交易管理器
type TradeManager struct {
	mu              sync.RWMutex
	trades          map[int64]*Trade
	playerTrades    map[id.PlayerIdType]int64
	currencyManager *CurrencyManager
}

func NewTradeManager() *TradeManager {
	return &TradeManager{
		trades:       make(map[int64]*Trade),
		playerTrades: make(map[id.PlayerIdType]int64),
	}
}

func (tm *TradeManager) SetCurrencyManager(cm *CurrencyManager) {
	tm.currencyManager = cm
}

// InitiateTrade 发起交易
func (tm *TradeManager) InitiateTrade(initiatorID, targetID id.PlayerIdType) (*Trade, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查发起者是否已在交易中
	if _, exists := tm.playerTrades[initiatorID]; exists {
		return nil, nil
	}

	// 检查目标是否已在交易中
	if _, exists := tm.playerTrades[targetID]; exists {
		return nil, nil
	}

	// 生成交易ID
	tradeID := time.Now().UnixNano()

	// 创建交易
	trade := &Trade{
		TradeID:           tradeID,
		InitiatorID:       initiatorID,
		TargetID:          targetID,
		Status:            TradeStatusPending,
		InitiatorItems:    make([]*TradeItem, 0),
		TargetItems:       make([]*TradeItem, 0),
		InitiatorCurrency: make(map[CurrencyType]int64),
		TargetCurrency:    make(map[CurrencyType]int64),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// 保存交易和玩家交易关系
	tm.trades[tradeID] = trade
	tm.playerTrades[initiatorID] = tradeID
	tm.playerTrades[targetID] = tradeID

	zLog.Debug("Trade initiated",
		zap.Int64("trade_id", tradeID),
		zap.Int64("initiator_id", int64(initiatorID)),
		zap.Int64("target_id", int64(targetID)))

	return trade, nil
}

// AcceptTrade 接受交易
func (tm *TradeManager) AcceptTrade(tradeID int64, playerID id.PlayerIdType) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查交易是否存在
	trade, exists := tm.trades[tradeID]
	if !exists {
		return nil
	}

	// 检查交易状态
	if trade.Status != TradeStatusPending {
		return nil
	}

	// 检查玩家是否是交易的目标
	if trade.TargetID != playerID {
		return nil
	}

	// 更新交易状态
	trade.Status = TradeStatusAccepted
	trade.UpdatedAt = time.Now()

	zLog.Debug("Trade accepted",
		zap.Int64("trade_id", tradeID),
		zap.Int64("player_id", int64(playerID)))

	return nil
}

// CancelTrade 取消交易
func (tm *TradeManager) CancelTrade(tradeID int64, playerID id.PlayerIdType) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查交易是否存在
	trade, exists := tm.trades[tradeID]
	if !exists {
		return nil
	}

	// 检查玩家是否是交易的参与者
	if trade.InitiatorID != playerID && trade.TargetID != playerID {
		return nil
	}

	// 更新交易状态
	trade.Status = TradeStatusCancelled
	trade.UpdatedAt = time.Now()

	// 清理玩家交易关系
	delete(tm.playerTrades, trade.InitiatorID)
	delete(tm.playerTrades, trade.TargetID)

	zLog.Debug("Trade cancelled",
		zap.Int64("trade_id", tradeID),
		zap.Int64("player_id", int64(playerID)))

	return nil
}

// CompleteTrade 完成交易
func (tm *TradeManager) CompleteTrade(tradeID int64, playerID id.PlayerIdType) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	trade, exists := tm.trades[tradeID]
	if !exists {
		return ErrTradeNotFound
	}

	if trade.Status != TradeStatusAccepted {
		return ErrTradeInvalidStatus
	}

	if trade.InitiatorID != playerID && trade.TargetID != playerID {
		return ErrTradeNotParticipant
	}

	if tm.currencyManager != nil {
		for currencyType, amount := range trade.InitiatorCurrency {
			if amount > 0 && !tm.currencyManager.HasEnoughCurrency(trade.InitiatorID, currencyType, amount) {
				return ErrInsufficientCurrency
			}
		}
		for currencyType, amount := range trade.TargetCurrency {
			if amount > 0 && !tm.currencyManager.HasEnoughCurrency(trade.TargetID, currencyType, amount) {
				return ErrInsufficientCurrency
			}
		}

		for currencyType, amount := range trade.InitiatorCurrency {
			if amount > 0 {
				if err := tm.currencyManager.RemoveCurrency(trade.InitiatorID, currencyType, amount); err != nil {
					zLog.Warn("Failed to remove currency from initiator",
						zap.Int64("trade_id", tradeID),
						zap.Int32("currency_type", int32(currencyType)),
						zap.Error(err))
				}
				if err := tm.currencyManager.AddCurrency(trade.TargetID, currencyType, amount); err != nil {
					zLog.Warn("Failed to add currency to target",
						zap.Int64("trade_id", tradeID),
						zap.Int32("currency_type", int32(currencyType)),
						zap.Error(err))
				}
			}
		}

		for currencyType, amount := range trade.TargetCurrency {
			if amount > 0 {
				if err := tm.currencyManager.RemoveCurrency(trade.TargetID, currencyType, amount); err != nil {
					zLog.Warn("Failed to remove currency from target",
						zap.Int64("trade_id", tradeID),
						zap.Int32("currency_type", int32(currencyType)),
						zap.Error(err))
				}
				if err := tm.currencyManager.AddCurrency(trade.InitiatorID, currencyType, amount); err != nil {
					zLog.Warn("Failed to add currency to initiator",
						zap.Int64("trade_id", tradeID),
						zap.Int32("currency_type", int32(currencyType)),
						zap.Error(err))
				}
			}
		}
	}

	trade.Status = TradeStatusCompleted
	trade.UpdatedAt = time.Now()

	delete(tm.playerTrades, trade.InitiatorID)
	delete(tm.playerTrades, trade.TargetID)

	zLog.Info("Trade completed",
		zap.Int64("trade_id", tradeID),
		zap.Int64("initiator_id", int64(trade.InitiatorID)),
		zap.Int64("target_id", int64(trade.TargetID)),
		zap.Int("initiator_items", len(trade.InitiatorItems)),
		zap.Int("target_items", len(trade.TargetItems)))

	return nil
}

// AddTradeItem 添加交易物品
func (tm *TradeManager) AddTradeItem(tradeID int64, playerID id.PlayerIdType, itemID, count int32) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查交易是否存在
	trade, exists := tm.trades[tradeID]
	if !exists {
		return nil
	}

	// 检查交易状态
	if trade.Status != TradeStatusPending && trade.Status != TradeStatusAccepted {
		return nil
	}

	// 确定是发起者还是目标
	isInitiator := trade.InitiatorID == playerID

	// 添加物品到交易
	tradeItem := &TradeItem{
		ItemID: itemID,
		Count:  count,
	}

	if isInitiator {
		trade.InitiatorItems = append(trade.InitiatorItems, tradeItem)
	} else {
		trade.TargetItems = append(trade.TargetItems, tradeItem)
	}

	trade.UpdatedAt = time.Now()

	zLog.Debug("Trade item added",
		zap.Int64("trade_id", tradeID),
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("item_id", itemID),
		zap.Int32("count", count))

	return nil
}

// RemoveTradeItem 移除交易物品
func (tm *TradeManager) RemoveTradeItem(tradeID int64, playerID id.PlayerIdType, itemIndex int32) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查交易是否存在
	trade, exists := tm.trades[tradeID]
	if !exists {
		return nil
	}

	// 检查交易状态
	if trade.Status != TradeStatusPending && trade.Status != TradeStatusAccepted {
		return nil
	}

	// 确定是发起者还是目标
	isInitiator := trade.InitiatorID == playerID

	// 移除物品
	if isInitiator {
		if itemIndex >= 0 && int(itemIndex) < len(trade.InitiatorItems) {
			trade.InitiatorItems = append(trade.InitiatorItems[:itemIndex], trade.InitiatorItems[itemIndex+1:]...)
		}
	} else {
		if itemIndex >= 0 && int(itemIndex) < len(trade.TargetItems) {
			trade.TargetItems = append(trade.TargetItems[:itemIndex], trade.TargetItems[itemIndex+1:]...)
		}
	}

	trade.UpdatedAt = time.Now()

	zLog.Debug("Trade item removed",
		zap.Int64("trade_id", tradeID),
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("item_index", itemIndex))

	return nil
}

// AddTradeCurrency 添加交易货币
func (tm *TradeManager) AddTradeCurrency(tradeID int64, playerID id.PlayerIdType, currencyType CurrencyType, amount int64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查交易是否存在
	trade, exists := tm.trades[tradeID]
	if !exists {
		return nil
	}

	// 检查交易状态
	if trade.Status != TradeStatusPending && trade.Status != TradeStatusAccepted {
		return nil
	}

	// 确定是发起者还是目标
	isInitiator := trade.InitiatorID == playerID

	// 添加货币到交易
	if isInitiator {
		trade.InitiatorCurrency[currencyType] = amount
	} else {
		trade.TargetCurrency[currencyType] = amount
	}

	trade.UpdatedAt = time.Now()

	zLog.Debug("Trade currency added",
		zap.Int64("trade_id", tradeID),
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("currency_type", int32(currencyType)),
		zap.Int64("amount", amount))

	return nil
}

// GetTrade 获取交易信息
func (tm *TradeManager) GetTrade(tradeID int64) *Trade {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.trades[tradeID]
}

// GetPlayerTrade 获取玩家正在进行的交易
func (tm *TradeManager) GetPlayerTrade(playerID id.PlayerIdType) *Trade {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tradeID, exists := tm.playerTrades[playerID]
	if !exists {
		return nil
	}

	return tm.trades[tradeID]
}
