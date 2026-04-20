package player

import (
	"github.com/pzqf/zCommon/common/id"
)

func (p *Player) GetAccountID() id.AccountIdType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.accountID
}

func (p *Player) GetPlayerID() id.PlayerIdType {
	return id.PlayerIdType(p.GetID())
}

func (p *Player) GetGold() int64 {
	return p.attrs.GetGold()
}

func (p *Player) SetGold(gold int64) {
	p.attrs.SetGold(gold)
}

func (p *Player) AddGold(amount int64) {
	p.attrs.AddGold(amount)
}

func (p *Player) ReduceGold(amount int64) bool {
	_, ok := p.attrs.AddGoldSafe(-amount)
	return ok
}

func (p *Player) GetDiamond() int64 {
	return p.attrs.GetDiamond()
}

func (p *Player) SetDiamond(diamond int64) {
	p.attrs.SetDiamond(diamond)
}

func (p *Player) AddDiamond(amount int64) {
	p.attrs.AddDiamond(amount)
}

func (p *Player) ReduceDiamond(amount int64) bool {
	_, ok := p.attrs.AddDiamondSafe(-amount)
	return ok
}

func (p *Player) GetVipLevel() int32 {
	return p.attrs.GetVipLevel()
}

func (p *Player) SetVipLevel(level int32) {
	p.attrs.SetVipLevel(level)
}

func (p *Player) GetVipExp() int32 {
	return p.attrs.GetVipExp()
}

func (p *Player) SetVipExp(exp int32) {
	p.attrs.SetVipExp(exp)
}

func (p *Player) GetCurrentMapID() id.MapIdType {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentMap
}

func (p *Player) SetCurrentMapID(mapID id.MapIdType) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentMap = mapID
}
