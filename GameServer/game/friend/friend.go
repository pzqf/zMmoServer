package friend

import (
	"sync"
	"time"

	"github.com/pzqf/zMmoShared/common/id"
)

// FriendStatus 好友状态
type FriendStatus int32

const (
	FriendStatusPending  FriendStatus = 0 // 待确认
	FriendStatusAccepted FriendStatus = 1 // 已接受
	FriendStatusBlocked  FriendStatus = 2 // 已拉黑
)

// Friend 好友
type Friend struct {
	mu           sync.RWMutex
	friendID     id.PlayerIdType
	friendName   string
	status       FriendStatus
	addTime      int64
	lastLoginTime int64
	isOnline     bool
}

// NewFriend 创建好友
func NewFriend(friendID id.PlayerIdType, friendName string) *Friend {
	return &Friend{
		friendID:     friendID,
		friendName:   friendName,
		status:       FriendStatusAccepted,
		addTime:      time.Now().UnixMilli(),
		lastLoginTime: 0,
		isOnline:     false,
	}
}

// NewFriendRequest 创建好友请求
func NewFriendRequest(friendID id.PlayerIdType, friendName string) *Friend {
	return &Friend{
		friendID:     friendID,
		friendName:   friendName,
		status:       FriendStatusPending,
		addTime:      time.Now().UnixMilli(),
		lastLoginTime: 0,
		isOnline:     false,
	}
}

// GetFriendID 获取好友ID
func (f *Friend) GetFriendID() id.PlayerIdType {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.friendID
}

// GetFriendName 获取好友名称
func (f *Friend) GetFriendName() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.friendName
}

// GetStatus 获取好友状态
func (f *Friend) GetStatus() FriendStatus {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.status
}

// SetStatus 设置好友状态
func (f *Friend) SetStatus(status FriendStatus) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = status
}

// GetAddTime 获取添加时间
func (f *Friend) GetAddTime() int64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.addTime
}

// GetLastLoginTime 获取最后登录时间
func (f *Friend) GetLastLoginTime() int64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.lastLoginTime
}

// UpdateLastLoginTime 更新最后登录时间
func (f *Friend) UpdateLastLoginTime() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastLoginTime = time.Now().UnixMilli()
}

// IsOnline 检查是否在线
func (f *Friend) IsOnline() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.isOnline
}

// SetOnline 设置在线状态
func (f *Friend) SetOnline(online bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.isOnline = online
}

// IsAccepted 检查是否已接受
func (f *Friend) IsAccepted() bool {
	return f.GetStatus() == FriendStatusAccepted
}

// IsPending 检查是否待确认
func (f *Friend) IsPending() bool {
	return f.GetStatus() == FriendStatusPending
}

// IsBlocked 检查是否已拉黑
func (f *Friend) IsBlocked() bool {
	return f.GetStatus() == FriendStatusBlocked
}

// Clone 克隆好友
func (f *Friend) Clone() *Friend {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return &Friend{
		friendID:     f.friendID,
		friendName:   f.friendName,
		status:       f.status,
		addTime:      f.addTime,
		lastLoginTime: f.lastLoginTime,
		isOnline:     f.isOnline,
	}
}
