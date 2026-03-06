package economy

import (
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// CurrencyType 货币类型
type CurrencyType int32

const (
	CurrencyTypeGold    CurrencyType = 1 // 金币
	CurrencyTypeDiamond CurrencyType = 2 // 钻石
	CurrencyTypeToken   CurrencyType = 3 // 代币
	CurrencyTypeCoin    CurrencyType = 4 // 铜币
)

// Currency 货币数据
type Currency struct {
	Type   CurrencyType `json:"type"`
	Amount int64        `json:"amount"`
}

// CurrencyManager 货币管理器
type CurrencyManager struct {
	mu        sync.RWMutex
	currencies map[id.PlayerIdType]map[CurrencyType]int64
}

// NewCurrencyManager 创建货币管理器
func NewCurrencyManager() *CurrencyManager {
	return &CurrencyManager{
		currencies: make(map[id.PlayerIdType]map[CurrencyType]int64),
	}
}

// GetCurrency 获取玩家货币数量
func (cm *CurrencyManager) GetCurrency(playerID id.PlayerIdType, currencyType CurrencyType) int64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	playerCurrencies, exists := cm.currencies[playerID]
	if !exists {
		return 0
	}

	return playerCurrencies[currencyType]
}

// AddCurrency 增加玩家货币
func (cm *CurrencyManager) AddCurrency(playerID id.PlayerIdType, currencyType CurrencyType, amount int64) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 确保玩家货币数据存在
	if _, exists := cm.currencies[playerID]; !exists {
		cm.currencies[playerID] = make(map[CurrencyType]int64)
	}

	// 增加货币
	cm.currencies[playerID][currencyType] += amount

	zLog.Debug("Currency added",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("currency_type", int32(currencyType)),
		zap.Int64("amount", amount),
		zap.Int64("new_amount", cm.currencies[playerID][currencyType]))

	return nil
}

// RemoveCurrency 减少玩家货币
func (cm *CurrencyManager) RemoveCurrency(playerID id.PlayerIdType, currencyType CurrencyType, amount int64) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 确保玩家货币数据存在
	if _, exists := cm.currencies[playerID]; !exists {
		return nil
	}

	// 检查货币是否足够
	currentAmount := cm.currencies[playerID][currencyType]
	if currentAmount < amount {
		return nil
	}

	// 减少货币
	cm.currencies[playerID][currencyType] -= amount

	zLog.Debug("Currency removed",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("currency_type", int32(currencyType)),
		zap.Int64("amount", amount),
		zap.Int64("new_amount", cm.currencies[playerID][currencyType]))

	return nil
}

// SetCurrency 设置玩家货币数量
func (cm *CurrencyManager) SetCurrency(playerID id.PlayerIdType, currencyType CurrencyType, amount int64) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 确保玩家货币数据存在
	if _, exists := cm.currencies[playerID]; !exists {
		cm.currencies[playerID] = make(map[CurrencyType]int64)
	}

	// 设置货币
	cm.currencies[playerID][currencyType] = amount

	zLog.Debug("Currency set",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("currency_type", int32(currencyType)),
		zap.Int64("amount", amount))

	return nil
}

// GetAllCurrencies 获取玩家所有货币
func (cm *CurrencyManager) GetAllCurrencies(playerID id.PlayerIdType) map[CurrencyType]int64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	playerCurrencies, exists := cm.currencies[playerID]
	if !exists {
		return make(map[CurrencyType]int64)
	}

	// 复制货币数据
	currencies := make(map[CurrencyType]int64)
	for currencyType, amount := range playerCurrencies {
		currencies[currencyType] = amount
	}

	return currencies
}

// HasEnoughCurrency 检查玩家是否有足够的货币
func (cm *CurrencyManager) HasEnoughCurrency(playerID id.PlayerIdType, currencyType CurrencyType, amount int64) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	playerCurrencies, exists := cm.currencies[playerID]
	if !exists {
		return false
	}

	return playerCurrencies[currencyType] >= amount
}