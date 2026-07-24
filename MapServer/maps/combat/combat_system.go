package combat

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

type DamageType int

const (
	DamageTypePhysical DamageType = iota
	DamageTypeMagical
	DamageTypeTrue
)

type DamageResult struct {
	AttackerID id.ObjectIdType
	TargetID   id.ObjectIdType
	Damage     int64
	DamageType DamageType
	IsCritical bool
	IsKill     bool
}

type CombatSystem struct {
	attackInterval time.Duration
}

func NewCombatSystem() *CombatSystem {
	return &CombatSystem{
		attackInterval: 1 * time.Second,
	}
}

func (cs *CombatSystem) CalculatePhysicalDamage(attacker *object.Player, target common.IGameObject) DamageResult {
	attackPower := cs.GetPlayerAttackPower(attacker)
	defense := cs.GetDefense(target)

	baseDamage := float64(attackPower) * (100.0 / (100.0 + float64(defense)))
	variance := 0.1 * baseDamage
	// MAP-9：伤害浮动用真随机（math/rand/v2，goroutine 安全）。此前用 time.Now().UnixNano()%1000
	// 当随机 → 可预测、且同毫秒内连续攻击取到相同值。
	damage := int64(baseDamage + variance*(0.5-rand.Float64()))

	if damage < 1 {
		damage = 1
	}

	isCritical := cs.IsCriticalHit(attacker)
	if isCritical {
		damage = int64(float64(damage) * 1.5)
	}

	return DamageResult{
		AttackerID: attacker.GetID(),
		TargetID:   target.GetID(),
		Damage:     damage,
		DamageType: DamageTypePhysical,
		IsCritical: isCritical,
	}
}

func (cs *CombatSystem) CalculateMonsterDamage(monster *object.Monster, target common.IGameObject) DamageResult {
	attackPower := cs.GetMonsterAttackPower(monster)
	defense := cs.GetDefense(target)

	baseDamage := float64(attackPower) * (100.0 / (100.0 + float64(defense)))
	damage := int64(baseDamage)

	if damage < 1 {
		damage = 1
	}

	return DamageResult{
		AttackerID: monster.GetID(),
		TargetID:   target.GetID(),
		Damage:     damage,
		DamageType: DamageTypePhysical,
		IsCritical: false,
	}
}

func (cs *CombatSystem) CalculateSkillDamage(caster common.IGameObject, target common.IGameObject, skillPower int32, skillMultiplier float64) DamageResult {
	attackPower := cs.GetAttackPower(caster)
	defense := cs.GetDefense(target)

	skillDamage := float64(skillPower) * skillMultiplier
	totalDamage := (skillDamage + float64(attackPower)*0.5) * (100.0 / (100.0 + float64(defense)))
	damage := int64(totalDamage)

	if damage < 1 {
		damage = 1
	}

	return DamageResult{
		AttackerID: caster.GetID(),
		TargetID:   target.GetID(),
		Damage:     damage,
		DamageType: DamageTypeMagical,
		IsCritical: false,
	}
}

func (cs *CombatSystem) ApplyDamage(target common.IGameObject, result DamageResult) bool {
	switch t := target.(type) {
	case *object.Player:
		newHP := t.GetHealth() - int32(result.Damage)
		if newHP < 0 {
			newHP = 0
		}
		t.SetHealth(newHP)
		result.IsKill = newHP <= 0
	case *object.Monster:
		newHP := t.GetHealth() - int32(result.Damage)
		if newHP < 0 {
			newHP = 0
		}
		t.SetHealth(newHP)
		result.IsKill = newHP <= 0
	default:
		zLog.Warn("Cannot apply damage to target type",
			zap.String("type", fmt.Sprintf("%T", target)))
		return false
	}

	return result.IsKill
}

func (cs *CombatSystem) IsInRange(attacker common.IGameObject, target common.IGameObject, range_ float32) bool {
	attackerPos := attacker.GetPosition()
	targetPos := target.GetPosition()
	// MAP-1：DistanceTo 返回真实距离(sqrt)，此前拿它比 range² → 射程被平方放大(range 3 实际允许 9)。
	// 改用 DistanceSquared 与 range² 比较（同量纲、且省一次 sqrt）。
	distSq := attackerPos.DistanceSquared(targetPos)
	return distSq <= range_*range_
}

func (cs *CombatSystem) GetPlayerAttackPower(player *object.Player) int32 {
	return player.GetAttack()
}

func (cs *CombatSystem) GetMonsterAttackPower(monster *object.Monster) int32 {
	return monster.GetAttack()
}

func (cs *CombatSystem) GetAttackPower(obj common.IGameObject) int32 {
	switch t := obj.(type) {
	case *object.Player:
		return cs.GetPlayerAttackPower(t)
	case *object.Monster:
		return cs.GetMonsterAttackPower(t)
	default:
		return 0
	}
}

func (cs *CombatSystem) GetDefense(obj common.IGameObject) int32 {
	switch t := obj.(type) {
	case *object.Player:
		return t.GetDefense()
	case *object.Monster:
		return t.GetDefense()
	default:
		return 0
	}
}

func (cs *CombatSystem) IsCriticalHit(player *object.Player) bool {
	critRate := float64(player.CalculateCriticalRate()) / 100.0
	if critRate > 0.5 {
		critRate = 0.5
	}
	// MAP-9：暴击判定用真随机（此前 time.Now()%1000 可被卡时序稳定触发暴击）。
	return rand.Float64() < critRate
}

func (cs *CombatSystem) CanAttack(attacker common.IGameObject, lastAttackTime time.Time) bool {
	return time.Since(lastAttackTime) >= cs.attackInterval
}

func DistanceBetween(a, b common.Vector3) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}
