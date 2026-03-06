package social

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// GuildRank 公会职位
type GuildRank int32

const (
	GuildRankMember  GuildRank = 1 // 普通成员
	GuildRankOfficer GuildRank = 2 // 官员
	GuildRankVice    GuildRank = 3 // 副会长
	GuildRankLeader  GuildRank = 4 // 会长
)

// GuildStatus 公会状态
type GuildStatus int32

const (
	GuildStatusNormal   GuildStatus = 1 // 正常
	GuildStatusDisband  GuildStatus = 2 // 解散
	GuildStatusRecruiting GuildStatus = 3 // 招募中
)

// GuildMember 公会成员
type GuildMember struct {
	PlayerID   id.PlayerIdType `json:"player_id"`
	Name       string          `json:"name"`
	Level      int32           `json:"level"`
	Class      int32           `json:"class"`
	Rank       GuildRank       `json:"rank"`
	JoinTime   time.Time       `json:"join_time"`
	LastActive time.Time       `json:"last_active"`
	Online     bool            `json:"online"`
	Contribution int64         `json:"contribution"` // 贡献值
}

// Guild 公会
type Guild struct {
	GuildID     id.GuildIdType  `json:"guild_id"`
	Name        string          `json:"name"`
	LeaderID    id.PlayerIdType `json:"leader_id"`
	Level       int32           `json:"level"`
	Exp         int64           `json:"exp"`
	Status      GuildStatus     `json:"status"`
	Members     []*GuildMember  `json:"members"`
	CreatedAt   time.Time       `json:"created_at"`
	LastActive  time.Time       `json:"last_active"`
	MaxMembers  int32           `json:"max_members"`
	Notice      string          `json:"notice"` // 公会公告
}

// GuildManager 公会管理器
type GuildManager struct {
	mu         sync.RWMutex
	guilds     map[id.GuildIdType]*Guild
	playerGuilds map[id.PlayerIdType]id.GuildIdType
}

// NewGuildManager 创建公会管理器
func NewGuildManager() *GuildManager {
	return &GuildManager{
		guilds:       make(map[id.GuildIdType]*Guild),
		playerGuilds: make(map[id.PlayerIdType]id.GuildIdType),
	}
}

// CreateGuild 创建公会
func (gm *GuildManager) CreateGuild(leaderID id.PlayerIdType, guildName, leaderName string, leaderLevel, leaderClass int32) (*Guild, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 检查玩家是否已在公会中
	if _, exists := gm.playerGuilds[leaderID]; exists {
		return nil, nil
	}

	// 生成公会ID
	guildID := id.GuildIdType(time.Now().UnixNano() % 1000000000)

	// 创建公会
	guild := &Guild{
		GuildID:     guildID,
		Name:        guildName,
		LeaderID:    leaderID,
		Level:       1,
		Exp:         0,
		Status:      GuildStatusNormal,
		Members:     make([]*GuildMember, 0),
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
		MaxMembers:  20, // 默认最大20人
		Notice:      "欢迎加入公会！",
	}

	// 添加会长到公会
	leader := &GuildMember{
		PlayerID:     leaderID,
		Name:         leaderName,
		Level:        leaderLevel,
		Class:        leaderClass,
		Rank:         GuildRankLeader,
		JoinTime:     time.Now(),
		LastActive:   time.Now(),
		Online:       true,
		Contribution: 0,
	}
	guild.Members = append(guild.Members, leader)

	// 保存公会和玩家公会关系
	gm.guilds[guildID] = guild
	gm.playerGuilds[leaderID] = guildID

	zLog.Debug("Guild created",
		zap.Int64("guild_id", int64(guildID)),
		zap.String("guild_name", guildName),
		zap.Int64("leader_id", int64(leaderID)),
		zap.String("leader_name", leaderName))

	return guild, nil
}

// InviteToGuild 邀请玩家加入公会
func (gm *GuildManager) InviteToGuild(guildID id.GuildIdType, inviterID, targetID id.PlayerIdType) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 检查公会是否存在
	guild, exists := gm.guilds[guildID]
	if !exists {
		return nil
	}

	// 检查邀请者是否在公会中且有邀请权限
	inviterInGuild := false
	canInvite := false
	for _, member := range guild.Members {
		if member.PlayerID == inviterID {
			inviterInGuild = true
			// 会长、副会长和官员可以邀请
			if member.Rank >= GuildRankOfficer {
				canInvite = true
			}
			break
		}
	}
	if !inviterInGuild || !canInvite {
		return nil
	}

	// 检查目标玩家是否已在公会中
	if _, exists := gm.playerGuilds[targetID]; exists {
		return nil
	}

	// 检查公会是否已满
	if int32(len(guild.Members)) >= guild.MaxMembers {
		return nil
	}

	// 这里可以添加邀请逻辑，比如发送邀请消息给目标玩家

	zLog.Debug("Player invited to guild",
		zap.Int64("guild_id", int64(guildID)),
		zap.Int64("inviter_id", int64(inviterID)),
		zap.Int64("target_id", int64(targetID)))

	return nil
}

// AcceptGuildInvite 接受公会邀请
func (gm *GuildManager) AcceptGuildInvite(guildID id.GuildIdType, playerID id.PlayerIdType, playerName string, playerLevel, playerClass int32) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 检查公会是否存在
	guild, exists := gm.guilds[guildID]
	if !exists {
		return nil
	}

	// 检查玩家是否已在公会中
	if _, exists := gm.playerGuilds[playerID]; exists {
		return nil
	}

	// 检查公会是否已满
	if int32(len(guild.Members)) >= guild.MaxMembers {
		return nil
	}

	// 添加玩家到公会
	member := &GuildMember{
		PlayerID:     playerID,
		Name:         playerName,
		Level:        playerLevel,
		Class:        playerClass,
		Rank:         GuildRankMember,
		JoinTime:     time.Now(),
		LastActive:   time.Now(),
		Online:       true,
		Contribution: 0,
	}
	guild.Members = append(guild.Members, member)

	// 保存玩家公会关系
	gm.playerGuilds[playerID] = guildID

	// 更新公会最后活动时间
	guild.LastActive = time.Now()

	zLog.Debug("Player joined guild",
		zap.Int64("guild_id", int64(guildID)),
		zap.Int64("player_id", int64(playerID)),
		zap.String("player_name", playerName))

	return nil
}

// LeaveGuild 离开公会
func (gm *GuildManager) LeaveGuild(playerID id.PlayerIdType) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 检查玩家是否在公会中
	guildID, exists := gm.playerGuilds[playerID]
	if !exists {
		return nil
	}

	// 检查公会是否存在
	guild, exists := gm.guilds[guildID]
	if !exists {
		return nil
	}

	// 移除玩家从公会
	newMembers := make([]*GuildMember, 0)
	isLeader := false
	for _, member := range guild.Members {
		if member.PlayerID == playerID {
			isLeader = (member.Rank == GuildRankLeader)
			continue
		}
		newMembers = append(newMembers, member)
	}
	guild.Members = newMembers

	// 移除玩家公会关系
	delete(gm.playerGuilds, playerID)

	// 如果会长离开，重新分配会长
	if isLeader && len(guild.Members) > 0 {
		// 找到副会长或官员作为新会长
		newLeader := guild.Members[0]
		for _, member := range guild.Members {
			if member.Rank > newLeader.Rank {
				newLeader = member
			}
		}
		guild.LeaderID = newLeader.PlayerID
		newLeader.Rank = GuildRankLeader
	}

	// 如果公会为空，解散公会
	if len(guild.Members) == 0 {
		guild.Status = GuildStatusDisband
		delete(gm.guilds, guildID)
		zLog.Debug("Guild disbanded due to no members", zap.Int64("guild_id", int64(guildID)))
	} else {
		// 更新公会最后活动时间
		guild.LastActive = time.Now()
		zLog.Debug("Player left guild",
			zap.Int64("guild_id", int64(guildID)),
			zap.Int64("player_id", int64(playerID)))
	}

	return nil
}

// KickFromGuild 踢出公会
func (gm *GuildManager) KickFromGuild(guildID id.GuildIdType, operatorID, targetID id.PlayerIdType) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 检查公会是否存在
	guild, exists := gm.guilds[guildID]
	if !exists {
		return nil
	}

	// 检查操作者是否在公会中且有踢人权限
	operatorInGuild := false
	canKick := false
	for _, member := range guild.Members {
		if member.PlayerID == operatorID {
			operatorInGuild = true
			// 会长、副会长可以踢人
			if member.Rank >= GuildRankVice {
				canKick = true
			}
			break
		}
	}
	if !operatorInGuild || !canKick {
		return nil
	}

	// 检查目标玩家是否在公会中
	targetInGuild := false
	targetRank := GuildRankMember
	for _, member := range guild.Members {
		if member.PlayerID == targetID {
			targetInGuild = true
			targetRank = member.Rank
			break
		}
	}
	if !targetInGuild {
		return nil
	}

	// 不能踢会长
	if targetRank == GuildRankLeader {
		return nil
	}

	// 移除玩家从公会
	newMembers := make([]*GuildMember, 0)
	for _, member := range guild.Members {
		if member.PlayerID == targetID {
			continue
		}
		newMembers = append(newMembers, member)
	}
	guild.Members = newMembers

	// 移除玩家公会关系
	delete(gm.playerGuilds, targetID)

	// 更新公会最后活动时间
	guild.LastActive = time.Now()

	zLog.Debug("Player kicked from guild",
		zap.Int64("guild_id", int64(guildID)),
		zap.Int64("operator_id", int64(operatorID)),
		zap.Int64("target_id", int64(targetID)))

	return nil
}

// DisbandGuild 解散公会
func (gm *GuildManager) DisbandGuild(guildID id.GuildIdType, leaderID id.PlayerIdType) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 检查公会是否存在
	guild, exists := gm.guilds[guildID]
	if !exists {
		return nil
	}

	// 检查是否是会长
	if guild.LeaderID != leaderID {
		return nil
	}

	// 移除所有玩家的公会关系
	for _, member := range guild.Members {
		delete(gm.playerGuilds, member.PlayerID)
	}

	// 标记公会为解散状态
	guild.Status = GuildStatusDisband

	// 移除公会
	delete(gm.guilds, guildID)

	zLog.Debug("Guild disbanded",
		zap.Int64("guild_id", int64(guildID)),
		zap.Int64("leader_id", int64(leaderID)))

	return nil
}

// PromoteMember 提升公会成员职位
func (gm *GuildManager) PromoteMember(guildID id.GuildIdType, leaderID, targetID id.PlayerIdType) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 检查公会是否存在
	guild, exists := gm.guilds[guildID]
	if !exists {
		return nil
	}

	// 检查是否是会长
	if guild.LeaderID != leaderID {
		return nil
	}

	// 提升成员职位
	for _, member := range guild.Members {
		if member.PlayerID == targetID {
			// 只能提升到副会长
			if member.Rank < GuildRankVice {
				member.Rank++
				// 更新公会最后活动时间
				guild.LastActive = time.Now()
				zLog.Debug("Guild member promoted",
					zap.Int64("guild_id", int64(guildID)),
					zap.Int64("target_id", int64(targetID)),
					zap.Int32("new_rank", int32(member.Rank)))
			}
			break
		}
	}

	return nil
}

// DemoteMember 降低公会成员职位
func (gm *GuildManager) DemoteMember(guildID id.GuildIdType, leaderID, targetID id.PlayerIdType) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 检查公会是否存在
	guild, exists := gm.guilds[guildID]
	if !exists {
		return nil
	}

	// 检查是否是会长
	if guild.LeaderID != leaderID {
		return nil
	}

	// 降低成员职位
	for _, member := range guild.Members {
		if member.PlayerID == targetID {
			// 不能降低会长职位
			if member.Rank > GuildRankMember && member.Rank < GuildRankLeader {
				member.Rank--
				// 更新公会最后活动时间
				guild.LastActive = time.Now()
				zLog.Debug("Guild member demoted",
					zap.Int64("guild_id", int64(guildID)),
					zap.Int64("target_id", int64(targetID)),
					zap.Int32("new_rank", int32(member.Rank)))
			}
			break
		}
	}

	return nil
}

// UpdateGuildNotice 更新公会公告
func (gm *GuildManager) UpdateGuildNotice(guildID id.GuildIdType, leaderID id.PlayerIdType, notice string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 检查公会是否存在
	guild, exists := gm.guilds[guildID]
	if !exists {
		return nil
	}

	// 检查是否是会长
	if guild.LeaderID != leaderID {
		return nil
	}

	// 更新公告
	guild.Notice = notice

	// 更新公会最后活动时间
	guild.LastActive = time.Now()

	zLog.Debug("Guild notice updated",
		zap.Int64("guild_id", int64(guildID)),
		zap.Int64("leader_id", int64(leaderID)))

	return nil
}

// GetGuild 获取公会信息
func (gm *GuildManager) GetGuild(guildID id.GuildIdType) *Guild {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	return gm.guilds[guildID]
}

// GetPlayerGuild 获取玩家所在公会
func (gm *GuildManager) GetPlayerGuild(playerID id.PlayerIdType) *Guild {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	guildID, exists := gm.playerGuilds[playerID]
	if !exists {
		return nil
	}

	return gm.guilds[guildID]
}

// GetGuildMembers 获取公会成员列表
func (gm *GuildManager) GetGuildMembers(guildID id.GuildIdType) []*GuildMember {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	guild, exists := gm.guilds[guildID]
	if !exists {
		return []*GuildMember{}
	}

	// 复制成员列表
	members := make([]*GuildMember, len(guild.Members))
	copy(members, guild.Members)

	return members
}

// UpdatePlayerStatus 更新玩家状态
func (gm *GuildManager) UpdatePlayerStatus(playerID id.PlayerIdType, online bool) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 检查玩家是否在公会中
	guildID, exists := gm.playerGuilds[playerID]
	if !exists {
		return nil
	}

	// 检查公会是否存在
	guild, exists := gm.guilds[guildID]
	if !exists {
		return nil
	}

	// 更新玩家状态
	for _, member := range guild.Members {
		if member.PlayerID == playerID {
			member.Online = online
			if online {
				member.LastActive = time.Now()
			}
			break
		}
	}

	// 更新公会最后活动时间
	guild.LastActive = time.Now()

	return nil
}

// AddContribution 增加贡献值
func (gm *GuildManager) AddContribution(playerID id.PlayerIdType, amount int64) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 检查玩家是否在公会中
	guildID, exists := gm.playerGuilds[playerID]
	if !exists {
		return nil
	}

	// 检查公会是否存在
	guild, exists := gm.guilds[guildID]
	if !exists {
		return nil
	}

	// 增加贡献值
	for _, member := range guild.Members {
		if member.PlayerID == playerID {
			member.Contribution += amount
			// 可以在这里添加公会经验增加逻辑
			guild.Exp += amount / 10 // 每10点贡献增加1点公会经验
			break
		}
	}

	// 更新公会最后活动时间
	guild.LastActive = time.Now()

	return nil
}