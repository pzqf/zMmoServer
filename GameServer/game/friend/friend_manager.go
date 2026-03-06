package friend

import (
	"errors"
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// FriendManager 好友管理器
type FriendManager struct {
	mu          sync.RWMutex
	playerID    id.PlayerIdType
	playerName  string
	friends     map[id.PlayerIdType]*Friend
	maxFriends  int32
}

// NewFriendManager 创建好友管理器
func NewFriendManager(playerID id.PlayerIdType, playerName string, maxFriends int32) *FriendManager {
	return &FriendManager{
		playerID:   playerID,
		playerName: playerName,
		friends:    make(map[id.PlayerIdType]*Friend),
		maxFriends: maxFriends,
	}
}

// GetPlayerID 获取玩家ID
func (fm *FriendManager) GetPlayerID() id.PlayerIdType {
	return fm.playerID
}

// GetPlayerName 获取玩家名称
func (fm *FriendManager) GetPlayerName() string {
	return fm.playerName
}

// GetFriendCount 获取好友数量
func (fm *FriendManager) GetFriendCount() int32 {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return int32(len(fm.friends))
}

// GetMaxFriends 获取最大好友数
func (fm *FriendManager) GetMaxFriends() int32 {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.maxFriends
}

// IsFull 检查好友列表是否已满
func (fm *FriendManager) IsFull() bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return int32(len(fm.friends)) >= fm.maxFriends
}

// AddFriend 添加好友
func (fm *FriendManager) AddFriend(friend *Friend) error {
	if friend == nil {
		return errors.New("friend is nil")
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	friendID := friend.GetFriendID()

	if _, exists := fm.friends[friendID]; exists {
		return errors.New("friend already exists")
	}

	if int32(len(fm.friends)) >= fm.maxFriends {
		return errors.New("friend list is full")
	}

	fm.friends[friendID] = friend

	fm.publishFriendAddEvent(friendID)

	zLog.Debug("Friend added",
		zap.Int64("player_id", int64(fm.playerID)),
		zap.Int64("friend_id", int64(friendID)),
		zap.String("name", friend.GetFriendName()))

	return nil
}

// RemoveFriend 移除好友
func (fm *FriendManager) RemoveFriend(friendID id.PlayerIdType) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	friend, exists := fm.friends[friendID]
	if !exists {
		return errors.New("friend not found")
	}

	delete(fm.friends, friendID)

	fm.publishFriendRemoveEvent(friendID)

	zLog.Debug("Friend removed",
		zap.Int64("player_id", int64(fm.playerID)),
		zap.Int64("friend_id", int64(friendID)))

	return nil
}

// GetFriend 获取好友
func (fm *FriendManager) GetFriend(friendID id.PlayerIdType) (*Friend, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	friend, exists := fm.friends[friendID]
	if !exists {
		return nil, errors.New("friend not found")
	}
	return friend, nil
}

// HasFriend 检查是否有指定好友
func (fm *FriendManager) HasFriend(friendID id.PlayerIdType) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	_, exists := fm.friends[friendID]
	return exists
}

// GetAllFriends 获取所有好友
func (fm *FriendManager) GetAllFriends() []*Friend {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	friends := make([]*Friend, 0, len(fm.friends))
	for _, friend := range fm.friends {
		friends = append(friends, friend)
	}
	return friends
}

// GetOnlineFriends 获取在线好友
func (fm *FriendManager) GetOnlineFriends() []*Friend {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	friends := make([]*Friend, 0)
	for _, friend := range fm.friends {
		if friend.IsOnline() {
			friends = append(friends, friend)
		}
	}
	return friends
}

// GetOfflineFriends 获取离线好友
func (fm *FriendManager) GetOfflineFriends() []*Friend {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	friends := make([]*Friend, 0)
	for _, friend := range fm.friends {
		if !friend.IsOnline() {
			friends = append(friends, friend)
		}
	}
	return friends
}

// GetOnlineFriendCount 获取在线好友数量
func (fm *FriendManager) GetOnlineFriendCount() int32 {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	count := int32(0)
	for _, friend := range fm.friends {
		if friend.IsOnline() {
			count++
		}
	}
	return count
}

// SetFriendOnline 设置好友在线状态
func (fm *FriendManager) SetFriendOnline(friendID id.PlayerIdType, online bool) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	friend, exists := fm.friends[friendID]
	if !exists {
		return errors.New("friend not found")
	}

	friend.SetOnline(online)
	if online {
		friend.UpdateLastLoginTime()
	}

	return nil
}

// BlockFriend 拉黑好友
func (fm *FriendManager) BlockFriend(friendID id.PlayerIdType) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	friend, exists := fm.friends[friendID]
	if !exists {
		return errors.New("friend not found")
	}

	friend.SetStatus(FriendStatusBlocked)

	zLog.Debug("Friend blocked",
		zap.Int64("player_id", int64(fm.playerID)),
		zap.Int64("friend_id", int64(friendID)))

	return nil
}

// UnblockFriend 解除拉黑
func (fm *FriendManager) UnblockFriend(friendID id.PlayerIdType) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	friend, exists := fm.friends[friendID]
	if !exists {
		return errors.New("friend not found")
	}

	friend.SetStatus(FriendStatusAccepted)

	zLog.Debug("Friend unblocked",
		zap.Int64("player_id", int64(fm.playerID)),
		zap.Int64("friend_id", int64(friendID)))

	return nil
}

// GetBlockedFriends 获取已拉黑的好友
func (fm *FriendManager) GetBlockedFriends() []*Friend {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	friends := make([]*Friend, 0)
	for _, friend := range fm.friends {
		if friend.IsBlocked() {
			friends = append(friends, friend)
		}
	}
	return friends
}

// Clear 清空好友列表
func (fm *FriendManager) Clear() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.friends = make(map[id.PlayerIdType]*Friend)
}

// publishFriendAddEvent 发布好友添加事件
func (fm *FriendManager) publishFriendAddEvent(friendID id.PlayerIdType) {
	event.Publish(event.NewEvent(event.EventPlayerAddFriend, fm, &event.FriendEventData{
		PlayerID: fm.playerID,
		FriendID: friendID,
	}))
}

// publishFriendRemoveEvent 发布好友移除事件
func (fm *FriendManager) publishFriendRemoveEvent(friendID id.PlayerIdType) {
	event.Publish(event.NewEvent(event.EventPlayerRemoveFriend, fm, &event.FriendEventData{
		PlayerID: fm.playerID,
		FriendID: friendID,
	}))
}
