package guild

import (
	"sync"

	"github.com/pzqf/zMmoShared/common/id"
)

// Guild 公会结构
type Guild struct {
	mu              sync.RWMutex
	guildID         id.GuildIdType
	name            string
	level           int32
	exp             int64
	money           int64
	presidentID     id.PlayerIdType
	presidentName   string
	notice          string
	maxMembers      int32
	members         map[id.PlayerIdType]*GuildMember
	createdAt       int64
}

// NewGuild 创建新公会
func NewGuild(guildID id.GuildIdType, name string, presidentID id.PlayerIdType, presidentName string) *Guild {
	return &Guild{
		guildID:       guildID,
		name:          name,
		level:         GuildLevel1,
		exp:           0,
		money:         0,
		presidentID:   presidentID,
		presidentName: presidentName,
		notice:        "",
		maxMembers:    50,
		members:       make(map[id.PlayerIdType]*GuildMember),
		createdAt:     0,
	}
}

// GetGuildID 获取公会ID
func (g *Guild) GetGuildID() id.GuildIdType {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.guildID
}

// GetName 获取公会名称
func (g *Guild) GetName() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.name
}

// GetLevel 获取公会等级
func (g *Guild) GetLevel() int32 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.level
}

// GetExp 获取公会经验
func (g *Guild) GetExp() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.exp
}

// AddExp 增加公会经验
func (g *Guild) AddExp(exp int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.exp += exp
}

// GetMoney 获取公会资金
func (g *Guild) GetMoney() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.money
}

// AddMoney 增加公会资金
func (g *Guild) AddMoney(money int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.money += money
}

// ReduceMoney 减少公会资金
func (g *Guild) ReduceMoney(money int64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.money < money {
		return false
	}

	g.money -= money
	return true
}

// GetPresidentID 获取会长ID
func (g *Guild) GetPresidentID() id.PlayerIdType {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.presidentID
}

// GetPresidentName 获取会长名称
func (g *Guild) GetPresidentName() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.presidentName
}

// SetPresident 设置会长
func (g *Guild) SetPresident(playerID id.PlayerIdType, playerName string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.presidentID = playerID
	g.presidentName = playerName
}

// GetNotice 获取公会公告
func (g *Guild) GetNotice() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.notice
}

// SetNotice 设置公会公告
func (g *Guild) SetNotice(notice string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.notice = notice
}

// GetMaxMembers 获取最大成员数
func (g *Guild) GetMaxMembers() int32 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.maxMembers
}

// GetMemberCount 获取成员数量
func (g *Guild) GetMemberCount() int32 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return int32(len(g.members))
}

// GetOnlineMemberCount 获取在线成员数量
func (g *Guild) GetOnlineMemberCount() int32 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	count := int32(0)
	for _, member := range g.members {
		if member.IsOnline() {
			count++
		}
	}
	return count
}

// AddMember 添加成员
func (g *Guild) AddMember(member *GuildMember) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if int32(len(g.members)) >= g.maxMembers {
		return false
	}

	playerID := member.GetPlayerID()
	if _, exists := g.members[playerID]; exists {
		return false
	}

	g.members[playerID] = member
	return true
}

// RemoveMember 移除成员
func (g *Guild) RemoveMember(playerID id.PlayerIdType) *GuildMember {
	g.mu.Lock()
	defer g.mu.Unlock()

	member, exists := g.members[playerID]
	if !exists {
		return nil
	}

	delete(g.members, playerID)
	return member
}

// GetMember 获取成员
func (g *Guild) GetMember(playerID id.PlayerIdType) *GuildMember {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.members[playerID]
}

// HasMember 检查是否有指定成员
func (g *Guild) HasMember(playerID id.PlayerIdType) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, exists := g.members[playerID]
	return exists
}

// GetAllMembers 获取所有成员
func (g *Guild) GetAllMembers() []*GuildMember {
	g.mu.RLock()
	defer g.mu.RUnlock()

	members := make([]*GuildMember, 0, len(g.members))
	for _, member := range g.members {
		members = append(members, member)
	}
	return members
}

// GetOnlineMembers 获取在线成员
func (g *Guild) GetOnlineMembers() []*GuildMember {
	g.mu.RLock()
	defer g.mu.RUnlock()

	members := make([]*GuildMember, 0)
	for _, member := range g.members {
		if member.IsOnline() {
			members = append(members, member)
		}
	}
	return members
}

// GetMembersByLevel 获取指定等级的成员
func (g *Guild) GetMembersByLevel(level GuildMemberLevel) []*GuildMember {
	g.mu.RLock()
	defer g.mu.RUnlock()

	members := make([]*GuildMember, 0)
	for _, member := range g.members {
		if member.GetLevel() == level {
			members = append(members, member)
		}
	}
	return members
}

// SetMemberLevel 设置成员等级
func (g *Guild) SetMemberLevel(playerID id.PlayerIdType, level GuildMemberLevel) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	member, exists := g.members[playerID]
	if !exists {
		return false
	}

	member.SetLevel(level)
	return true
}

// IsFull 检查公会是否已满
func (g *Guild) IsFull() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return int32(len(g.members)) >= g.maxMembers
}

// CanUpgrade 检查是否可以升级
func (g *Guild) CanUpgrade() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.level >= GuildLevel5 {
		return false
	}

	return true
}

// Upgrade 升级公会
func (g *Guild) Upgrade() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.level >= GuildLevel5 {
		return false
	}

	g.level++
	g.maxMembers = g.getMaxMembersByLevel(g.level)
	return true
}

// getMaxMembersByLevel 根据等级获取最大成员数
func (g *Guild) getMaxMembersByLevel(level int32) int32 {
	switch level {
	case GuildLevel1:
		return 50
	case GuildLevel2:
		return 100
	case GuildLevel3:
		return 150
	case GuildLevel4:
		return 200
	case GuildLevel5:
		return 300
	default:
		return 50
	}
}

// Clone 克隆公会
func (g *Guild) Clone() *Guild {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clone := &Guild{
		guildID:       g.guildID,
		name:          g.name,
		level:         g.level,
		exp:           g.exp,
		money:         g.money,
		presidentID:   g.presidentID,
		presidentName: g.presidentName,
		notice:        g.notice,
		maxMembers:    g.maxMembers,
		members:       make(map[id.PlayerIdType]*GuildMember),
		createdAt:     g.createdAt,
	}

	for playerID, member := range g.members {
		clone.members[playerID] = member.Clone()
	}

	return clone
}
