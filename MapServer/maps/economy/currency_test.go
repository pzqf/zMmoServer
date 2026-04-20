package economy

import (
	"testing"

	"github.com/pzqf/zCommon/common/id"
)

func newTestCurrencyManager() *CurrencyManager {
	return NewCurrencyManager()
}

func TestCurrencyManager_AddCurrency(t *testing.T) {
	cm := newTestCurrencyManager()
	playerID := id.PlayerIdType(1)

	err := cm.AddCurrency(playerID, CurrencyTypeGold, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	amount := cm.GetCurrency(playerID, CurrencyTypeGold)
	if amount != 100 {
		t.Errorf("expected 100 gold, got %d", amount)
	}
}

func TestCurrencyManager_AddCurrency_MultipleTimes(t *testing.T) {
	cm := newTestCurrencyManager()
	playerID := id.PlayerIdType(1)

	cm.AddCurrency(playerID, CurrencyTypeGold, 100)
	cm.AddCurrency(playerID, CurrencyTypeGold, 50)

	amount := cm.GetCurrency(playerID, CurrencyTypeGold)
	if amount != 150 {
		t.Errorf("expected 150 gold, got %d", amount)
	}
}

func TestCurrencyManager_AddCurrency_DifferentTypes(t *testing.T) {
	cm := newTestCurrencyManager()
	playerID := id.PlayerIdType(1)

	cm.AddCurrency(playerID, CurrencyTypeGold, 100)
	cm.AddCurrency(playerID, CurrencyTypeDiamond, 50)

	gold := cm.GetCurrency(playerID, CurrencyTypeGold)
	diamond := cm.GetCurrency(playerID, CurrencyTypeDiamond)

	if gold != 100 {
		t.Errorf("expected 100 gold, got %d", gold)
	}
	if diamond != 50 {
		t.Errorf("expected 50 diamond, got %d", diamond)
	}
}

func TestCurrencyManager_RemoveCurrency(t *testing.T) {
	cm := newTestCurrencyManager()
	playerID := id.PlayerIdType(1)

	cm.AddCurrency(playerID, CurrencyTypeGold, 100)
	err := cm.RemoveCurrency(playerID, CurrencyTypeGold, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	amount := cm.GetCurrency(playerID, CurrencyTypeGold)
	if amount != 70 {
		t.Errorf("expected 70 gold, got %d", amount)
	}
}

func TestCurrencyManager_RemoveCurrency_Insufficient(t *testing.T) {
	cm := newTestCurrencyManager()
	playerID := id.PlayerIdType(1)

	cm.AddCurrency(playerID, CurrencyTypeGold, 50)
	err := cm.RemoveCurrency(playerID, CurrencyTypeGold, 100)
	if err == nil {
		t.Error("expected error for insufficient currency")
	}
}

func TestCurrencyManager_RemoveCurrency_NoPlayer(t *testing.T) {
	cm := newTestCurrencyManager()
	playerID := id.PlayerIdType(999)

	err := cm.RemoveCurrency(playerID, CurrencyTypeGold, 10)
	if err == nil {
		t.Error("expected error for non-existent player")
	}
}

func TestCurrencyManager_GetCurrency_NoPlayer(t *testing.T) {
	cm := newTestCurrencyManager()

	amount := cm.GetCurrency(id.PlayerIdType(999), CurrencyTypeGold)
	if amount != 0 {
		t.Errorf("expected 0 for non-existent player, got %d", amount)
	}
}

func TestCurrencyManager_HasEnoughCurrency(t *testing.T) {
	cm := newTestCurrencyManager()
	playerID := id.PlayerIdType(1)

	cm.AddCurrency(playerID, CurrencyTypeGold, 100)

	if !cm.HasEnoughCurrency(playerID, CurrencyTypeGold, 50) {
		t.Error("expected to have enough currency")
	}
	if !cm.HasEnoughCurrency(playerID, CurrencyTypeGold, 100) {
		t.Error("expected to have enough currency (exact amount)")
	}
	if cm.HasEnoughCurrency(playerID, CurrencyTypeGold, 101) {
		t.Error("expected not to have enough currency")
	}
}

func TestCurrencyManager_HasEnoughCurrency_NoPlayer(t *testing.T) {
	cm := newTestCurrencyManager()

	if cm.HasEnoughCurrency(id.PlayerIdType(999), CurrencyTypeGold, 1) {
		t.Error("expected not to have enough currency for non-existent player")
	}
}

func TestCurrencyManager_FullFlow(t *testing.T) {
	cm := newTestCurrencyManager()
	playerID := id.PlayerIdType(1)

	cm.AddCurrency(playerID, CurrencyTypeGold, 1000)
	cm.AddCurrency(playerID, CurrencyTypeDiamond, 100)

	cm.RemoveCurrency(playerID, CurrencyTypeGold, 300)
	cm.RemoveCurrency(playerID, CurrencyTypeDiamond, 50)

	gold := cm.GetCurrency(playerID, CurrencyTypeGold)
	diamond := cm.GetCurrency(playerID, CurrencyTypeDiamond)

	if gold != 700 {
		t.Errorf("expected 700 gold, got %d", gold)
	}
	if diamond != 50 {
		t.Errorf("expected 50 diamond, got %d", diamond)
	}
}
