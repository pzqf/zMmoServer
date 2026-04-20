package player

import (
	"sync"
	"testing"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/stretchr/testify/assert"
)

func TestNewPlayer(t *testing.T) {
	playerID := id.PlayerIdType(1001)
	accountID := id.AccountIdType(2001)
	name := "TestPlayer"

	p := NewPlayer(playerID, accountID, name)

	assert.NotNil(t, p)
	assert.Equal(t, playerID, p.GetPlayerID())
	assert.Equal(t, accountID, p.GetAccountID())
	assert.Equal(t, name, p.GetName())
	assert.Equal(t, id.MapIdType(0), p.GetCurrentMapID())
}

func TestPlayerGold(t *testing.T) {
	p := NewPlayer(1, 1, "TestPlayer")

	assert.Equal(t, int64(0), p.GetGold())

	p.SetGold(1000)
	assert.Equal(t, int64(1000), p.GetGold())

	p.AddGold(500)
	assert.Equal(t, int64(1500), p.GetGold())

	ok := p.ReduceGold(300)
	assert.True(t, ok)
	assert.Equal(t, int64(1200), p.GetGold())

	ok = p.ReduceGold(2000)
	assert.False(t, ok)
	assert.Equal(t, int64(1200), p.GetGold())
}

func TestPlayerDiamond(t *testing.T) {
	p := NewPlayer(1, 1, "TestPlayer")

	assert.Equal(t, int64(0), p.GetDiamond())

	p.SetDiamond(100)
	assert.Equal(t, int64(100), p.GetDiamond())

	p.AddDiamond(50)
	assert.Equal(t, int64(150), p.GetDiamond())

	ok := p.ReduceDiamond(30)
	assert.True(t, ok)
	assert.Equal(t, int64(120), p.GetDiamond())

	ok = p.ReduceDiamond(200)
	assert.False(t, ok)
	assert.Equal(t, int64(120), p.GetDiamond())
}

func TestPlayerMapState(t *testing.T) {
	p := NewPlayer(1, 1, "TestPlayer")

	assert.Equal(t, id.MapIdType(0), p.GetCurrentMapID())

	p.SetCurrentMapID(100)
	assert.Equal(t, id.MapIdType(100), p.GetCurrentMapID())

	p.SetCurrentMapID(0)
	assert.Equal(t, id.MapIdType(0), p.GetCurrentMapID())
}

func TestPlayerMapOperator(t *testing.T) {
	p := NewPlayer(1, 1, "TestPlayer")

	assert.Nil(t, p.mapOp)

	mockOp := &mockMapOperator{}
	p.SetMapOperator(mockOp)
	assert.NotNil(t, p.mapOp)
}

func TestPlayerConcurrentGold(t *testing.T) {
	p := NewPlayer(1, 1, "TestPlayer")
	p.SetGold(10000)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.AddGold(10)
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(11000), p.GetGold())
}

type mockMapOperator struct{}

func (m *mockMapOperator) EnterMap(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	return nil
}

func (m *mockMapOperator) LeaveMap(playerID id.PlayerIdType, mapID id.MapIdType) error {
	return nil
}

func (m *mockMapOperator) Move(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	return nil
}

func (m *mockMapOperator) Attack(playerID id.PlayerIdType, mapID id.MapIdType, targetID id.ObjectIdType) (int64, int64, error) {
	return 0, 0, nil
}
