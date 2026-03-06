package guild

import (
	"errors"
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// GuildManager 公会管理器
type GuildManager struct {
	mu            sync.RWMutex
	playerID      id.PlayerIdType
	playerName    string
	guild         *Guild
	memberLevel   GuildMemberLevel
}

// NewGuildManager 创建公会管理器
func NewGuildManager(playerID id.PlayerIdType, playerName string) *GuildManager {
	return &GuildManager{
		playerID:   playerID,
		playerName: playerName,
		guild:      nil,
		memberLevel: 0,
	}
}

// GetPlayerID 获取玩家ID
func (gm *GuildManager) GetPlayerID() id.PlayerIdType {
	return gm.playerID
}

// GetPlayerName 获取玩家名称
func (gm *GuildManager) GetPlayerName() string {
	return gm.playerName
}

// GetGuild 获取公会
func (gm *GuildManager) GetGuild() *Guild {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.guild
}

// SetGuild 设置公会
func (gm *GuildManager) SetGuild(guild *Guild) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.guild = guild
	if guild != nil {
		member := guild.GetMember(gm.playerID)
		if member != nil {
			gm.memberLevel = member.GetLevel()
		}
	} else {
		gm.memberLevel = 0
	}
}

// GetMemberLevel 获取成员等级
func (gm *GuildManager) GetMemberLevel() GuildMemberLevel {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.memberLevel
}

// HasGuild 检查是否有公会
func (gm *GuildManager) HasGuild() bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.guild != nil
}

// IsPresident 检查是否是会长
func (gm *GuildManager) IsPresident() bool {
	return gm.GetMemberLevel() == GuildMemberLevelPresident
}

// IsVice 检查是否是副会长
func (gm *GuildManager) IsVice() bool {
	return gm.GetMemberLevel() == GuildMemberLevelVice
}

// IsOfficer 检查是否是官员
func (gm *GuildManager) IsOfficer() bool {
	level := gm.GetMemberLevel()
	return level == GuildMemberLevelVice || level == GuildMemberLevelOfficer
}

// CanManage 检查是否有管理权限
func (gm *GuildManager) CanManage() bool {
	level := gm.GetMemberLevel()
	return level == GuildMemberLevelPresident ||
		level == GuildMemberLevelVice ||
		level == GuildMemberLevelOfficer
}

// CanKick 检查是否有踢人权限
func (gm *GuildManager) CanKick() bool {
	level := gm.GetMemberLevel()
	return level == GuildMemberLevelPresident || level == GuildMemberLevelVice
}

// CanInvite 检查是否有邀请权限
func (gm *GuildManager) CanInvite() bool {
	level := gm.GetMemberLevel()
	return level == GuildMemberLevelPresident ||
		level == GuildMemberLevelVice ||
		level == GuildMemberLevelOfficer ||
		level == GuildMemberLevelElite
}

// CreateGuild 创建公会
func (gm *GuildManager) CreateGuild(guildID id.GuildIdType, guildName string) error {
	if gm.HasGuild() {
		return errors.New("already in guild")
	}

	guild := NewGuild(guildID, guildName, gm.playerID, gm.playerName)
	president := NewGuildMember(gm.playerID, gm.playerName, GuildMemberLevelPresident)
	guild.AddMember(president)

	gm.SetGuild(guild)

	gm.publishGuildCreateEvent(guild)

	zLog.Info("Guild created",
		zap.Int64("player_id", int64(gm.playerID)),
		zap.Int64("guild_id", int64(guildID)),
		zap.String("name", guildName))

	return nil
}

// JoinGuild 加入公会
func (gm *GuildManager) JoinGuild(guild *Guild) error {
	if gm.HasGuild() {
		return errors.New("already in guild")
	}

	if guild.IsFull() {
		return errors.New("guild is full")
	}

	member := NewGuildMember(gm.playerID, gm.playerName, GuildMemberLevelMember)
	if !guild.AddMember(member) {
		return errors.New("join guild failed")
	}

	gm.SetGuild(guild)

	gm.publishGuildJoinEvent(guild)

	zLog.Info("Guild joined",
		zap.Int64("player_id", int64(gm.playerID)),
		zap.Int64("guild_id", int64(guild.GetGuildID())),
		zap.String("name", guild.GetName()))

	return nil
}

// LeaveGuild 离开公会
func (gm *GuildManager) LeaveGuild() error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.guild == nil {
		return errors.New("not in guild")
	}

	if gm.memberLevel == GuildMemberLevelPresident {
		return errors.New("president cannot leave guild")
	}

	guildID := gm.guild.GetGuildID()
	guildName := gm.guild.GetName()

	gm.guild.RemoveMember(gm.playerID)
	gm.guild = nil
	gm.memberLevel = 0

	gm.publishGuildLeaveEvent(guildID, guildName)

	zLog.Info("Guild left",
		zap.Int64("player_id", int64(gm.playerID)),
		zap.Int64("guild_id", int64(guildID)),
		zap.String("name", guildName))

	return nil
}

// DissolveGuild 解散公会（会长专用）
func (gm *GuildManager) DissolveGuild() error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.guild == nil {
		return errors.New("not in guild")
	}

	if gm.memberLevel != GuildMemberLevelPresident {
		return errors.New("only president can dissolve guild")
	}

	guildID := gm.guild.GetGuildID()
	guildName := gm.guild.GetName()

	gm.guild = nil
	gm.memberLevel = 0

	gm.publishGuildDissolveEvent(guildID, guildName)

	zLog.Info("Guild dissolved",
		zap.Int64("player_id", int64(gm.playerID)),
		zap.Int64("guild_id", int64(guildID)),
		zap.String("name", guildName))

	return nil
}

// KickMember 踢出成员
func (gm *GuildManager) KickMember(targetPlayerID id.PlayerIdType) error {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	if gm.guild == nil {
		return errors.New("not in guild")
	}

	if !gm.CanKick() {
		return errors.New("no permission")
	}

	if targetPlayerID == gm.playerID {
		return errors.New("cannot kick yourself")
	}

	member := gm.guild.GetMember(targetPlayerID)
	if member == nil {
		return errors.New("member not found")
	}

	if member.GetLevel() >= gm.memberLevel {
		return errors.New("cannot kick higher level member")
	}

	gm.guild.RemoveMember(targetPlayerID)

	gm.publishGuildKickEvent(targetPlayerID)

	zLog.Info("Guild member kicked",
		zap.Int64("player_id", int64(gm.playerID)),
		zap.Int64("target_id", int64(targetPlayerID)),
		zap.Int64("guild_id", int64(gm.guild.GetGuildID())))

	return nil
}

// PromoteMember 提升成员
func (gm *GuildManager) PromoteMember(targetPlayerID id.PlayerIdType, newLevel GuildMemberLevel) error {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	if gm.guild == nil {
		return errors.New("not in guild")
	}

	if !gm.CanManage() {
		return errors.New("no permission")
	}

	if newLevel >= gm.memberLevel {
		return errors.New("cannot promote to higher or equal level")
	}

	if !gm.guild.SetMemberLevel(targetPlayerID, newLevel) {
		return errors.New("member not found")
	}

	gm.publishGuildPromoteEvent(targetPlayerID, newLevel)

	zLog.Info("Guild member promoted",
		zap.Int64("player_id", int64(gm.playerID)),
		zap.Int64("target_id", int64(targetPlayerID)),
		zap.Int32("level", int32(newLevel)))

	return nil
}

// SetNotice 设置公会公告
func (gm *GuildManager) SetNotice(notice string) error {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	if gm.guild == nil {
		return errors.New("not in guild")
	}

	if !gm.CanManage() {
		return errors.New("no permission")
	}

	gm.guild.SetNotice(notice)

	gm.publishGuildNoticeEvent(notice)

	zLog.Debug("Guild notice updated",
		zap.Int64("player_id", int64(gm.playerID)),
		zap.Int64("guild_id", int64(gm.guild.GetGuildID())),
		zap.String("notice", notice))

	return nil
}

// AddContribution 增加贡献值
func (gm *GuildManager) AddContribution(contribution int64) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	if gm.guild == nil {
		return
	}

	member := gm.guild.GetMember(gm.playerID)
	if member != nil {
		member.AddContribution(contribution)
	}
}

// GetMemberInfo 获取成员信息
func (gm *GuildManager) GetMemberInfo() *GuildMember {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	if gm.guild == nil {
		return nil
	}

	return gm.guild.GetMember(gm.playerID)
}

// GetGuildInfo 获取公会信息
func (gm *GuildManager) GetGuildInfo() (id.GuildIdType, string, int32) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	if gm.guild == nil {
		return 0, "", 0
	}

	return gm.guild.GetGuildID(), gm.guild.GetName(), gm.guild.GetLevel()
}

// publishGuildCreateEvent 发布公会创建事件
func (gm *GuildManager) publishGuildCreateEvent(guild *Guild) {
	event.Publish(event.NewEvent(event.EventGuildCreate, gm, &event.GuildEventData{
		PlayerID: gm.playerID,
		GuildID:  guild.GetGuildID(),
		GuildName: guild.GetName(),
	}))
}

// publishGuildJoinEvent 发布公会加入事件
func (gm *GuildManager) publishGuildJoinEvent(guild *Guild) {
	event.Publish(event.NewEvent(event.EventPlayerJoinGuild, gm, &event.GuildEventData{
		PlayerID: gm.playerID,
		GuildID:  guild.GetGuildID(),
		GuildName: guild.GetName(),
	}))
}

// publishGuildLeaveEvent 发布公会离开事件
func (gm *GuildManager) publishGuildLeaveEvent(guildID id.GuildIdType, guildName string) {
	event.Publish(event.NewEvent(event.EventPlayerLeaveGuild, gm, &event.GuildEventData{
		PlayerID: gm.playerID,
		GuildID:  guildID,
		GuildName: guildName,
	}))
}

// publishGuildDissolveEvent 发布公会解散事件
func (gm *GuildManager) publishGuildDissolveEvent(guildID id.GuildIdType, guildName string) {
	event.Publish(event.NewEvent(event.EventGuildDissolve, gm, &event.GuildEventData{
		PlayerID: gm.playerID,
		GuildID:  guildID,
		GuildName: guildName,
	}))
}

// publishGuildKickEvent 发布公会踢人事件
func (gm *GuildManager) publishGuildKickEvent(targetPlayerID id.PlayerIdType) {
	event.Publish(event.NewEvent(event.EventGuildKick, gm, &event.GuildEventData{
		PlayerID: gm.playerID,
		GuildID:  gm.guild.GetGuildID(),
	}))
}

// publishGuildPromoteEvent 发布公会提升事件
func (gm *GuildManager) publishGuildPromoteEvent(targetPlayerID id.PlayerIdType, newLevel GuildMemberLevel) {
	event.Publish(event.NewEvent(event.EventGuildPromote, gm, &event.GuildEventData{
		PlayerID: gm.playerID,
		GuildID:  gm.guild.GetGuildID(),
	}))
}

// publishGuildNoticeEvent 发布公会公告事件
func (gm *GuildManager) publishGuildNoticeEvent(notice string) {
	event.Publish(event.NewEvent(event.EventGuildNotice, gm, &event.GuildEventData{
		PlayerID: gm.playerID,
		GuildID:  gm.guild.GetGuildID(),
	}))
}
