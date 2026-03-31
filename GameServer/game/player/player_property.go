package player

import (
	"github.com/pzqf/zCommon/common/id"
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
