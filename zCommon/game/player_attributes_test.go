package game

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPlayerAttributes(t *testing.T) {
	attrs := NewPlayerAttributes()
	assert.NotNil(t, attrs)
	assert.Equal(t, int32(0), attrs.GetLevel())
	assert.Equal(t, int64(0), attrs.GetExp())
	assert.Equal(t, int64(0), attrs.GetGold())
	assert.Equal(t, int64(0), attrs.GetDiamond())
}

func TestPlayerAttributesLevel(t *testing.T) {
	attrs := NewPlayerAttributes()

	attrs.SetLevel(10)
	assert.Equal(t, int32(10), attrs.GetLevel())

	attrs.SetLevel(99)
	assert.Equal(t, int32(99), attrs.GetLevel())
}

func TestPlayerAttributesExp(t *testing.T) {
	attrs := NewPlayerAttributes()

	attrs.SetExp(1000)
	assert.Equal(t, int64(1000), attrs.GetExp())

	result := attrs.AddExp(500)
	assert.Equal(t, int64(1500), result)
	assert.Equal(t, int64(1500), attrs.GetExp())
}

func TestPlayerAttributesGold(t *testing.T) {
	attrs := NewPlayerAttributes()

	attrs.SetGold(10000)
	assert.Equal(t, int64(10000), attrs.GetGold())

	result := attrs.AddGold(5000)
	assert.Equal(t, int64(15000), result)
}

func TestPlayerAttributesGoldSafe(t *testing.T) {
	attrs := NewPlayerAttributes()
	attrs.SetGold(100)

	result, ok := attrs.AddGoldSafe(50)
	assert.True(t, ok)
	assert.Equal(t, int64(150), result)

	result, ok = attrs.AddGoldSafe(-200)
	assert.False(t, ok)
	assert.Equal(t, int64(150), result)
}

func TestPlayerAttributesDiamondSafe(t *testing.T) {
	attrs := NewPlayerAttributes()
	attrs.SetDiamond(100)

	result, ok := attrs.AddDiamondSafe(50)
	assert.True(t, ok)
	assert.Equal(t, int64(150), result)

	result, ok = attrs.AddDiamondSafe(-200)
	assert.False(t, ok)
	assert.Equal(t, int64(150), result)
}

func TestPlayerAttributesHPMP(t *testing.T) {
	attrs := NewPlayerAttributes()

	attrs.SetHP(500)
	attrs.SetMaxHP(1000)
	attrs.SetMP(200)
	attrs.SetMaxMP(500)

	assert.Equal(t, int64(500), attrs.GetHP())
	assert.Equal(t, int64(1000), attrs.GetMaxHP())
	assert.Equal(t, int64(200), attrs.GetMP())
	assert.Equal(t, int64(500), attrs.GetMaxMP())
}

func TestPlayerAttributesCombatStats(t *testing.T) {
	attrs := NewPlayerAttributes()

	attrs.SetStrength(10)
	attrs.SetAgility(8)
	attrs.SetIntelligence(15)
	attrs.SetStamina(12)
	attrs.SetSpirit(7)

	assert.Equal(t, int32(10), attrs.GetStrength())
	assert.Equal(t, int32(8), attrs.GetAgility())
	assert.Equal(t, int32(15), attrs.GetIntelligence())
	assert.Equal(t, int32(12), attrs.GetStamina())
	assert.Equal(t, int32(7), attrs.GetSpirit())
}

func TestPlayerAttributesConcurrentAccess(t *testing.T) {
	attrs := NewPlayerAttributes()
	attrs.SetGold(10000)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			attrs.AddGold(10)
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(11000), attrs.GetGold())
}
