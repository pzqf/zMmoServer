package player

import (
	"errors"
	"sync"

	"github.com/pzqf/zUtil/zMap"
	"github.com/pzqf/zMmoShared/common/id"
)

// PlayerManager 玩家管理器
type PlayerManager struct {
	players         *zMap.Map
	playersMu       sync.RWMutex
	playersByAccount map[id.AccountIdType]*Player
}

// NewPlayerManager 创建玩家管理器
func NewPlayerManager() *PlayerManager {
	return &PlayerManager{
		players:         zMap.NewMap(),
		playersByAccount: make(map[id.AccountIdType]*Player),
	}
}

// AddPlayer 添加玩家
func (pm *PlayerManager) AddPlayer(player *Player) error {
	if player == nil {
		return errors.New("player can't be nil")
	}

	playerID := player.GetPlayerID()
	if playerID == 0 {
		return errors.New("player id can't be 0")
	}

	pm.playersMu.Lock()
	defer pm.playersMu.Unlock()

	_, exists := pm.players.Load(playerID)
	if exists {
		return errors.New("player already exists")
	}

	pm.players.Store(playerID, player)
	pm.playersByAccount[player.GetAccountID()] = player

	return nil
}

// GetPlayer 获取玩家
func (pm *PlayerManager) GetPlayer(playerID id.PlayerIdType) (*Player, error) {
	v, ok := pm.players.Load(playerID)
	if !ok {
		return nil, errors.New("player not found")
	}
	return v.(*Player), nil
}

// GetPlayerByAccount 通过账号ID获取玩家
func (pm *PlayerManager) GetPlayerByAccount(accountID id.AccountIdType) (*Player, error) {
	pm.playersMu.RLock()
	defer pm.playersMu.RUnlock()

	player, exists := pm.playersByAccount[accountID]
	if !exists {
		return nil, errors.New("player not found")
	}
	return player, nil
}

// RemovePlayer 移除玩家
func (pm *PlayerManager) RemovePlayer(playerID id.PlayerIdType) error {
	pm.playersMu.Lock()
	defer pm.playersMu.Unlock()

	v, ok := pm.players.Load(playerID)
	if !ok {
		return errors.New("player not found")
	}

	player := v.(*Player)
	pm.players.Delete(playerID)
	delete(pm.playersByAccount, player.GetAccountID())

	return nil
}

// GetAllPlayers 获取所有玩家
func (pm *PlayerManager) GetAllPlayers() []*Player {
	players := make([]*Player, 0)
	pm.players.Range(func(key, value interface{}) bool {
		players = append(players, value.(*Player))
		return true
	})
	return players
}

// GetPlayerCount 获取玩家数量
func (pm *PlayerManager) GetPlayerCount() int64 {
	return pm.players.Len()
}

// HasPlayer 检查玩家是否存在
func (pm *PlayerManager) HasPlayer(playerID id.PlayerIdType) bool {
	_, ok := pm.players.Load(playerID)
	return ok
}

// HasPlayerByAccount 检查账号是否在线
func (pm *PlayerManager) HasPlayerByAccount(accountID id.AccountIdType) bool {
	pm.playersMu.RLock()
	defer pm.playersMu.RUnlock()
	_, exists := pm.playersByAccount[accountID]
	return exists
}

// Range 遍历所有玩家
func (pm *PlayerManager) Range(f func(playerID id.PlayerIdType, player *Player) bool) {
	pm.players.Range(func(key, value interface{}) bool {
		return f(key.(id.PlayerIdType), value.(*Player))
	})
}

// ClearAll 清除所有玩家
func (pm *PlayerManager) ClearAll() {
	pm.playersMu.Lock()
	defer pm.playersMu.Unlock()

	pm.players.Range(func(key, value interface{}) bool {
		player := value.(*Player)
		player.Destroy()
		return true
	})

	pm.players.Clear()
	pm.playersByAccount = make(map[id.AccountIdType]*Player)
}
