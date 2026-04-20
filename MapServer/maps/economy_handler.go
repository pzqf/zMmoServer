package maps

import (
	"fmt"
	"time"

	"github.com/pzqf/zMmoServer/MapServer/maps/economy"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
)

func (m *Map) GetCurrency(player *object.Player, currencyType economy.CurrencyType) int64 {
	if m.currencyManager == nil {
		return 0
	}
	return m.currencyManager.GetCurrency(player.GetPlayerID(), currencyType)
}

func (m *Map) AddCurrency(player *object.Player, currencyType economy.CurrencyType, amount int64) error {
	if m.currencyManager == nil {
		return fmt.Errorf("currency manager not initialized")
	}
	return m.currencyManager.AddCurrency(player.GetPlayerID(), currencyType, amount)
}

func (m *Map) RemoveCurrency(player *object.Player, currencyType economy.CurrencyType, amount int64) error {
	if m.currencyManager == nil {
		return fmt.Errorf("currency manager not initialized")
	}
	return m.currencyManager.RemoveCurrency(player.GetPlayerID(), currencyType, amount)
}

func (m *Map) HasEnoughCurrency(player *object.Player, currencyType economy.CurrencyType, amount int64) bool {
	if m.currencyManager == nil {
		return false
	}
	return m.currencyManager.HasEnoughCurrency(player.GetPlayerID(), currencyType, amount)
}

func (m *Map) StartTrade(initiator, target *object.Player) (*economy.Trade, error) {
	if m.tradeManager == nil {
		return nil, fmt.Errorf("trade manager not initialized")
	}
	return m.tradeManager.InitiateTrade(initiator.GetPlayerID(), target.GetPlayerID())
}

func (m *Map) AddTradeItem(player *object.Player, tradeID int64, itemID int32, count int32) error {
	if m.tradeManager == nil {
		return fmt.Errorf("trade manager not initialized")
	}
	return m.tradeManager.AddTradeItem(tradeID, player.GetPlayerID(), itemID, count)
}

func (m *Map) AddTradeCurrency(player *object.Player, tradeID int64, currencyType economy.CurrencyType, amount int64) error {
	if m.tradeManager == nil {
		return fmt.Errorf("trade manager not initialized")
	}
	return m.tradeManager.AddTradeCurrency(tradeID, player.GetPlayerID(), currencyType, amount)
}

func (m *Map) AcceptTrade(player *object.Player, tradeID int64) error {
	if m.tradeManager == nil {
		return fmt.Errorf("trade manager not initialized")
	}
	return m.tradeManager.AcceptTrade(tradeID, player.GetPlayerID())
}

func (m *Map) CancelTrade(player *object.Player, tradeID int64) error {
	if m.tradeManager == nil {
		return fmt.Errorf("trade manager not initialized")
	}
	return m.tradeManager.CancelTrade(tradeID, player.GetPlayerID())
}

func (m *Map) CreateAuction(player *object.Player, itemID int32, count int32, startingPrice int64, duration int32, currencyType economy.CurrencyType) (int64, error) {
	if m.auctionManager == nil {
		return 0, fmt.Errorf("auction manager not initialized")
	}

	auction, err := m.auctionManager.CreateAuction(
		player.GetPlayerID(),
		itemID,
		count,
		startingPrice,
		time.Duration(duration)*time.Second,
		currencyType,
	)
	if err != nil {
		return 0, err
	}

	return auction.AuctionID, nil
}

func (m *Map) PlaceBid(auctionID int64, player *object.Player, bidPrice int64) error {
	if m.auctionManager == nil {
		return fmt.Errorf("auction manager not initialized")
	}
	return m.auctionManager.PlaceBid(auctionID, player.GetPlayerID(), bidPrice)
}

func (m *Map) CancelAuction(auctionID int64, player *object.Player) error {
	if m.auctionManager == nil {
		return fmt.Errorf("auction manager not initialized")
	}
	return m.auctionManager.CancelAuction(auctionID, player.GetPlayerID())
}

func (m *Map) BuyItem(shopID, itemID, count int32, player *object.Player) (int64, economy.CurrencyType, error) {
	if m.shopManager == nil {
		return 0, 0, fmt.Errorf("shop manager not initialized")
	}
	return m.shopManager.BuyItem(shopID, itemID, count, player.GetLevel(), player.GetClass())
}

func (m *Map) LoadShopConfig(filePath string) error {
	if m.shopManager != nil {
		return m.shopManager.LoadShopConfig(filePath)
	}
	return nil
}
