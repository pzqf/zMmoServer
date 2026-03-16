package player

import (
	"github.com/pzqf/zMmoShared/common/id"
)

// GetAccountID 获取账号ID
func (p *Player) GetAccountID() id.AccountIdType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.accountID
}

// GetPlayerID 获取玩家ID
func (p *Player) GetPlayerID() id.PlayerIdType {
	return id.PlayerIdType(p.GetID())
}

// GetGold 获取金币
func (p *Player) GetGold() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.gold
}

// SetGold 设置金币
func (p *Player) SetGold(gold int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gold = gold
}

// AddGold 增加金币
func (p *Player) AddGold(amount int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gold += amount
}

// ReduceGold 减少金币
func (p *Player) ReduceGold(amount int64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.gold < amount {
		return false
	}
	p.gold -= amount
	return true
}

// GetDiamond 获取钻石
func (p *Player) GetDiamond() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.diamond
}

// SetDiamond 设置钻石
func (p *Player) SetDiamond(diamond int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.diamond = diamond
}

// AddDiamond 增加钻石
func (p *Player) AddDiamond(amount int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.diamond += amount
}

// ReduceDiamond 减少钻石
func (p *Player) ReduceDiamond(amount int64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.diamond < amount {
		return false
	}
	p.diamond -= amount
	return true
}

// GetVipLevel 获取VIP等级
func (p *Player) GetVipLevel() int32 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.vipLevel
}

// SetVipLevel 设置VIP等级
func (p *Player) SetVipLevel(level int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.vipLevel = level
}

// GetVipExp 获取VIP经验
func (p *Player) GetVipExp() int32 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.vipExp
}

// SetVipExp 设置VIP经验
func (p *Player) SetVipExp(exp int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.vipExp = exp
}

// GetGuildID 获取公会ID
func (p *Player) GetGuildID() id.GuildIdType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.guildID
}

// SetGuildID 设置公会ID
func (p *Player) SetGuildID(guildID id.GuildIdType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.guildID = guildID
}

// HasGuild 是否有公会
func (p *Player) HasGuild() bool {
	return p.GetGuildID() > 0
}

// GetTeamID 获取队伍ID
func (p *Player) GetTeamID() id.TeamIdType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.teamID
}

// SetTeamID 设置队伍ID
func (p *Player) SetTeamID(teamID id.TeamIdType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.teamID = teamID
}

// HasTeam 是否有队伍
func (p *Player) HasTeam() bool {
	return p.GetTeamID() > 0
}

// GetFriendList 获取好友列表
func (p *Player) GetFriendList() []id.PlayerIdType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]id.PlayerIdType, len(p.friendList))
	copy(result, p.friendList)
	return result
}

// AddFriend 添加好友
func (p *Player) AddFriend(friendID id.PlayerIdType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, id := range p.friendList {
		if id == friendID {
			return
		}
	}
	p.friendList = append(p.friendList, friendID)
}

// RemoveFriend 移除好友
func (p *Player) RemoveFriend(friendID id.PlayerIdType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, id := range p.friendList {
		if id == friendID {
			p.friendList = append(p.friendList[:i], p.friendList[i+1:]...)
			return
		}
	}
}

// IsFriend 是否是好友
func (p *Player) IsFriend(friendID id.PlayerIdType) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, id := range p.friendList {
		if id == friendID {
			return true
		}
	}
	return false
}

