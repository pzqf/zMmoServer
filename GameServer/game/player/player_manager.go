package player

import (
	"errors"
	"sync"

	"github.com/pzqf/zEngine/zActor"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zUtil/zMap"
)

// PlayerManager 玩家管理器
type PlayerManager struct {
	players          *zMap.Map
	playersMu        sync.RWMutex
	playersByAccount map[id.AccountIdType]*Player
}

// NewPlayerManager 创建玩家管理器
func NewPlayerManager() *PlayerManager {
	return &PlayerManager{
		players:          zMap.NewMap(),
		playersByAccount: make(map[id.AccountIdType]*Player),
	}
}

// CreatePlayer 创建并启动玩家Actor
func (pm *PlayerManager) CreatePlayer(playerID id.PlayerIdType, accountID id.AccountIdType, name string) (*Player, error) {
	pm.playersMu.Lock()
	defer pm.playersMu.Unlock()

	// 检查是否已存在
	_, exists := pm.players.Load(playerID)
	if exists {
		return nil, errors.New("player already exists")
	}

	// 创建Player
	player := NewPlayer(playerID, accountID, name)

	// 启动Actor
	if err := player.Start(); err != nil {
		return nil, err
	}

	// 注册到管理器
	pm.players.Store(playerID, player)
	pm.playersByAccount[accountID] = player

	return player, nil
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

	// 启动Actor
	if err := player.Start(); err != nil {
		return err
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

// RemovePlayer 停止并移除玩家Actor
func (pm *PlayerManager) RemovePlayer(playerID id.PlayerIdType) error {
	pm.playersMu.Lock()
	defer pm.playersMu.Unlock()

	v, ok := pm.players.Load(playerID)
	if !ok {
		return errors.New("player not found")
	}

	player := v.(*Player)

	// 停止Actor
	if err := player.Stop(); err != nil {
		// 记录错误但继续移除
	}

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
	var count int64
	pm.players.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// RouteMessage 路由消息到指定玩家
func (pm *PlayerManager) RouteMessage(playerID id.PlayerIdType, msg *PlayerMessage) error {
	player, err := pm.GetPlayer(playerID)
	if err != nil {
		return err
	}

	// 确保设置ActorID
	msg.ActorID = int64(playerID)

	// 发送消息
	player.SendMessage(msg)
	return nil
}

// BroadcastMessage 广播消息给所有玩家
func (pm *PlayerManager) BroadcastMessage(msg *PlayerMessage) {
	pm.players.Range(func(key, value interface{}) bool {
		player := value.(*Player)
		// 为每个玩家创建消息副本并设置ActorID
		msgCopy := &PlayerMessage{
			BaseActorMessage: zActor.BaseActorMessage{ActorID: int64(player.GetPlayerID())},
			Source:           msg.Source,
			Type:             msg.Type,
			Data:             msg.Data,
			// 回调通道不复制，因为广播不需要回调
		}
		player.SendMessage(msgCopy)
		return true
	})
}

// BroadcastMessageToPlayers 广播消息给指定玩家列表
func (pm *PlayerManager) BroadcastMessageToPlayers(playerIDs []id.PlayerIdType, msg *PlayerMessage) {
	for _, playerID := range playerIDs {
		pm.RouteMessage(playerID, msg)
	}
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
		player.Stop()
		return true
	})

	pm.players.Clear()
	pm.playersByAccount = make(map[id.AccountIdType]*Player)
}
