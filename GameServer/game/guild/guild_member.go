package guild

import (
	"sync"
	"time"

	"github.com/pzqf/zMmoShared/common/id"
)

// GuildLevel 公会等级
const (
	GuildLevel1  = 1
	GuildLevel2  = 2
	GuildLevel3  = 3
	GuildLevel4  = 4
	GuildLevel5  = 5
)

// GuildMemberLevel 公会成员等级
const (
	GuildMemberLevelPresident GuildMemberLevel = 1 // 会长
	GuildMemberLevelVice      GuildMemberLevel = 2 // 副会长
	GuildMemberLevelOfficer   GuildMemberLevel = 3 // 官员
	GuildMemberLevelElite     GuildMemberLevel = 4 // 精英
	GuildMemberLevelMember    GuildMemberLevel = 5 // 成员
)

// GuildMemberLevel 公会成员等级类型
type GuildMemberLevel int32

// GuildMember 公会成员
type GuildMember struct {
	mu            sync.RWMutex
	playerID      id.PlayerIdType
	playerName    string
	level         GuildMemberLevel
	contribution  int64
	joinTime      int64
	lastLoginTime int64
	isOnline      bool
}

// NewGuildMember 创建公会成员
func NewGuildMember(playerID id.PlayerIdType, playerName string, level GuildMemberLevel) *GuildMember {
	return &GuildMember{
		playerID:      playerID,
		playerName:    playerName,
		level:         level,
		contribution:  0,
		joinTime:      time.Now().UnixMilli(),
		lastLoginTime: time.Now().UnixMilli(),
		isOnline:      false,
	}
}

// GetPlayerID 获取玩家ID
func (gm *GuildMember) GetPlayerID() id.PlayerIdType {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.playerID
}

// GetPlayerName 获取玩家名称
func (gm *GuildMember) GetPlayerName() string {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.playerName
}

// GetLevel 获取成员等级
func (gm *GuildMember) GetLevel() GuildMemberLevel {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.level
}

// SetLevel 设置成员等级
func (gm *GuildMember) SetLevel(level GuildMemberLevel) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.level = level
}

// GetContribution 获取贡献值
func (gm *GuildMember) GetContribution() int64 {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.contribution
}

// AddContribution 增加贡献值
func (gm *GuildMember) AddContribution(contribution int64) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.contribution += contribution
}

// GetJoinTime 获取加入时间
func (gm *GuildMember) GetJoinTime() int64 {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.joinTime
}

// GetLastLoginTime 获取最后登录时间
func (gm *GuildMember) GetLastLoginTime() int64 {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.lastLoginTime
}

// UpdateLastLoginTime 更新最后登录时间
func (gm *GuildMember) UpdateLastLoginTime() {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.lastLoginTime = time.Now().UnixMilli()
}

// IsOnline 检查是否在线
func (gm *GuildMember) IsOnline() bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.isOnline
}

// SetOnline 设置在线状态
func (gm *GuildMember) SetOnline(online bool) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.isOnline = online
}

// IsPresident 检查是否是会长
func (gm *GuildMember) IsPresident() bool {
	return gm.GetLevel() == GuildMemberLevelPresident
}

// IsVice 检查是否是副会长
func (gm *GuildMember) IsVice() bool {
	return gm.GetLevel() == GuildMemberLevelVice
}

// IsOfficer 检查是否是官员
func (gm *GuildMember) IsOfficer() bool {
	level := gm.GetLevel()
	return level == GuildMemberLevelVice || level == GuildMemberLevelOfficer
}

// CanManage 检查是否有管理权限
func (gm *GuildMember) CanManage() bool {
	level := gm.GetLevel()
	return level == GuildMemberLevelPresident ||
		level == GuildMemberLevelVice ||
		level == GuildMemberLevelOfficer
}

// CanKick 检查是否有踢人权限
func (gm *GuildMember) CanKick() bool {
	level := gm.GetLevel()
	return level == GuildMemberLevelPresident || level == GuildMemberLevelVice
}

// CanInvite 检查是否有邀请权限
func (gm *GuildMember) CanInvite() bool {
	level := gm.GetLevel()
	return level == GuildMemberLevelPresident ||
		level == GuildMemberLevelVice ||
		level == GuildMemberLevelOfficer ||
		level == GuildMemberLevelElite
}

// Clone 克隆成员
func (gm *GuildMember) Clone() *GuildMember {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	return &GuildMember{
		playerID:      gm.playerID,
		playerName:    gm.playerName,
		level:         gm.level,
		contribution:  gm.contribution,
		joinTime:      gm.joinTime,
		lastLoginTime: gm.lastLoginTime,
		isOnline:      gm.isOnline,
	}
}
