package player

import (
	"testing"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/stretchr/testify/assert"
)

func TestNewPlayerManager(t *testing.T) {
	pm := NewPlayerManager()

	assert.NotNil(t, pm)
	assert.Equal(t, int64(0), pm.GetPlayerCount())
}

func TestPlayerManagerCreatePlayer(t *testing.T) {
	pm := NewPlayerManager()

	player, err := pm.CreatePlayer(1, 100, "TestPlayer")

	assert.NoError(t, err)
	assert.NotNil(t, player)
	assert.Equal(t, id.PlayerIdType(1), player.GetPlayerID())
	assert.Equal(t, id.AccountIdType(100), player.GetAccountID())
	assert.Equal(t, int64(1), pm.GetPlayerCount())
}

func TestPlayerManagerCreatePlayerDuplicate(t *testing.T) {
	pm := NewPlayerManager()

	_, err := pm.CreatePlayer(1, 100, "TestPlayer1")
	assert.NoError(t, err)

	_, err = pm.CreatePlayer(1, 100, "TestPlayer2")
	assert.Equal(t, ErrPlayerAlreadyExists, err)
}

func TestPlayerManagerGetPlayer(t *testing.T) {
	pm := NewPlayerManager()

	_, err := pm.CreatePlayer(1, 100, "TestPlayer")
	assert.NoError(t, err)

	player, err := pm.GetPlayer(1)
	assert.NoError(t, err)
	assert.Equal(t, id.PlayerIdType(1), player.GetPlayerID())
}

func TestPlayerManagerGetPlayerNotFound(t *testing.T) {
	pm := NewPlayerManager()

	_, err := pm.GetPlayer(999)
	assert.Equal(t, ErrPlayerNotFound, err)
}

func TestPlayerManagerGetPlayerByAccount(t *testing.T) {
	pm := NewPlayerManager()

	_, err := pm.CreatePlayer(1, 100, "TestPlayer")
	assert.NoError(t, err)

	player, err := pm.GetPlayerByAccount(100)
	assert.NoError(t, err)
	assert.Equal(t, id.AccountIdType(100), player.GetAccountID())
}

func TestPlayerManagerRemovePlayer(t *testing.T) {
	pm := NewPlayerManager()

	_, err := pm.CreatePlayer(1, 100, "TestPlayer")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), pm.GetPlayerCount())

	err = pm.RemovePlayer(1)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), pm.GetPlayerCount())

	_, err = pm.GetPlayer(1)
	assert.Equal(t, ErrPlayerNotFound, err)
}

func TestPlayerManagerRemovePlayerNotFound(t *testing.T) {
	pm := NewPlayerManager()

	err := pm.RemovePlayer(999)
	assert.Equal(t, ErrPlayerNotFound, err)
}

func TestPlayerManagerHasPlayer(t *testing.T) {
	pm := NewPlayerManager()

	assert.False(t, pm.HasPlayer(1))

	_, _ = pm.CreatePlayer(1, 100, "TestPlayer")

	assert.True(t, pm.HasPlayer(1))
	assert.True(t, pm.HasPlayerByAccount(100))
	assert.False(t, pm.HasPlayer(2))
	assert.False(t, pm.HasPlayerByAccount(200))
}

func TestPlayerManagerAddPlayer(t *testing.T) {
	pm := NewPlayerManager()

	p := NewPlayer(1, 100, "TestPlayer")

	err := pm.AddPlayer(p)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), pm.GetPlayerCount())
}

func TestPlayerManagerAddPlayerNil(t *testing.T) {
	pm := NewPlayerManager()

	err := pm.AddPlayer(nil)
	assert.Equal(t, ErrPlayerNil, err)
}

func TestPlayerManagerAddPlayerDuplicate(t *testing.T) {
	pm := NewPlayerManager()

	p1 := NewPlayer(1, 100, "TestPlayer1")
	err := pm.AddPlayer(p1)
	assert.NoError(t, err)

	p2 := NewPlayer(1, 200, "TestPlayer2")
	err = pm.AddPlayer(p2)
	assert.Equal(t, ErrPlayerAlreadyExists, err)
}

func TestPlayerManagerGetAllPlayers(t *testing.T) {
	pm := NewPlayerManager()

	_, _ = pm.CreatePlayer(1, 100, "Player1")
	_, _ = pm.CreatePlayer(2, 200, "Player2")
	_, _ = pm.CreatePlayer(3, 300, "Player3")

	players := pm.GetAllPlayers()
	assert.Len(t, players, 3)
}

func TestPlayerManagerClearAll(t *testing.T) {
	pm := NewPlayerManager()

	_, _ = pm.CreatePlayer(1, 100, "Player1")
	_, _ = pm.CreatePlayer(2, 200, "Player2")

	pm.ClearAll()
	assert.Equal(t, int64(0), pm.GetPlayerCount())
}

func TestPlayerManagerRouteMessage(t *testing.T) {
	pm := NewPlayerManager()

	_, _ = pm.CreatePlayer(1, 100, "TestPlayer")

	msg := NewPlayerMessage(1, SourceGateway, MsgNetEnterGame, nil)
	err := pm.RouteMessage(1, msg)
	assert.NoError(t, err)
}

func TestPlayerManagerRouteMessagePlayerNotFound(t *testing.T) {
	pm := NewPlayerManager()

	msg := NewPlayerMessage(999, SourceGateway, MsgNetEnterGame, nil)
	err := pm.RouteMessage(999, msg)
	assert.Equal(t, ErrPlayerNotFound, err)
}

func TestPlayerManagerSetMapOperator(t *testing.T) {
	pm := NewPlayerManager()

	op := &mockMapOperator{}
	pm.SetMapOperator(op)

	player, _ := pm.CreatePlayer(1, 100, "TestPlayer")
	assert.NotNil(t, player.mapOp)
}

type testMapOperator struct {
	entered bool
	left    bool
	moved   bool
}

func (m *testMapOperator) EnterMap(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	m.entered = true
	return nil
}

func (m *testMapOperator) LeaveMap(playerID id.PlayerIdType, mapID id.MapIdType) error {
	m.left = true
	return nil
}

func (m *testMapOperator) Move(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	m.moved = true
	return nil
}

func (m *testMapOperator) Attack(playerID id.PlayerIdType, mapID id.MapIdType, targetID id.ObjectIdType) (int64, int64, error) {
	return 100, 50, nil
}
