package game

import "sync/atomic"

type PlayerAttributes struct {
	level       atomic.Int32
	exp         atomic.Int64
	gold        atomic.Int64
	diamond     atomic.Int64
	vipLevel    atomic.Int32
	vipExp      atomic.Int32
	hp          atomic.Int64
	maxHP       atomic.Int64
	mp          atomic.Int64
	maxMP       atomic.Int64
	strength    atomic.Int32
	agility     atomic.Int32
	intelligence atomic.Int32
	stamina     atomic.Int32
	spirit      atomic.Int32
}

func NewPlayerAttributes() *PlayerAttributes {
	return &PlayerAttributes{}
}

func (a *PlayerAttributes) GetLevel() int32        { return a.level.Load() }
func (a *PlayerAttributes) SetLevel(v int32)       { a.level.Store(v) }
func (a *PlayerAttributes) GetExp() int64          { return a.exp.Load() }
func (a *PlayerAttributes) SetExp(v int64)         { a.exp.Store(v) }
func (a *PlayerAttributes) AddExp(v int64) int64   { return a.exp.Add(v) }
func (a *PlayerAttributes) GetGold() int64         { return a.gold.Load() }
func (a *PlayerAttributes) SetGold(v int64)        { a.gold.Store(v) }
func (a *PlayerAttributes) AddGold(v int64) int64  { return a.gold.Add(v) }
func (a *PlayerAttributes) GetDiamond() int64      { return a.diamond.Load() }
func (a *PlayerAttributes) SetDiamond(v int64)     { a.diamond.Store(v) }
func (a *PlayerAttributes) AddDiamond(v int64) int64 { return a.diamond.Add(v) }
func (a *PlayerAttributes) GetVipLevel() int32     { return a.vipLevel.Load() }
func (a *PlayerAttributes) SetVipLevel(v int32)    { a.vipLevel.Store(v) }
func (a *PlayerAttributes) GetVipExp() int32       { return a.vipExp.Load() }
func (a *PlayerAttributes) SetVipExp(v int32)      { a.vipExp.Store(v) }
func (a *PlayerAttributes) GetHP() int64           { return a.hp.Load() }
func (a *PlayerAttributes) SetHP(v int64)          { a.hp.Store(v) }
func (a *PlayerAttributes) GetMaxHP() int64        { return a.maxHP.Load() }
func (a *PlayerAttributes) SetMaxHP(v int64)       { a.maxHP.Store(v) }
func (a *PlayerAttributes) GetMP() int64           { return a.mp.Load() }
func (a *PlayerAttributes) SetMP(v int64)          { a.mp.Store(v) }
func (a *PlayerAttributes) GetMaxMP() int64        { return a.maxMP.Load() }
func (a *PlayerAttributes) SetMaxMP(v int64)       { a.maxMP.Store(v) }
func (a *PlayerAttributes) GetStrength() int32     { return a.strength.Load() }
func (a *PlayerAttributes) SetStrength(v int32)    { a.strength.Store(v) }
func (a *PlayerAttributes) GetAgility() int32      { return a.agility.Load() }
func (a *PlayerAttributes) SetAgility(v int32)     { a.agility.Store(v) }
func (a *PlayerAttributes) GetIntelligence() int32 { return a.intelligence.Load() }
func (a *PlayerAttributes) SetIntelligence(v int32) { a.intelligence.Store(v) }
func (a *PlayerAttributes) GetStamina() int32      { return a.stamina.Load() }
func (a *PlayerAttributes) SetStamina(v int32)     { a.stamina.Store(v) }
func (a *PlayerAttributes) GetSpirit() int32       { return a.spirit.Load() }
func (a *PlayerAttributes) SetSpirit(v int32)      { a.spirit.Store(v) }

func (a *PlayerAttributes) AddGoldSafe(amount int64) (int64, bool) {
	for {
		current := a.gold.Load()
		if current+amount < 0 {
			return current, false
		}
		if a.gold.CompareAndSwap(current, current+amount) {
			return current + amount, true
		}
	}
}

func (a *PlayerAttributes) AddDiamondSafe(amount int64) (int64, bool) {
	for {
		current := a.diamond.Load()
		if current+amount < 0 {
			return current, false
		}
		if a.diamond.CompareAndSwap(current, current+amount) {
			return current + amount, true
		}
	}
}
