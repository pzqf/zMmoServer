package combat

import (
	"fmt"
	"math"
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
	damage := int64(baseDamage + variance*(0.5-float64(time.Now().UnixNano()%1000)/1000.0))

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
	distSq := attackerPos.DistanceTo(targetPos)
	return distSq <= range_*range_
}

func (cs *CombatSystem) GetPlayerAttackPower(player *object.Player) int32 {
	return player.GetStrength()*2 + player.GetLevel()*3
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
		return t.GetStamina() + t.GetLevel()*2
	case *object.Monster:
		return t.GetDefense()
	default:
		return 0
	}
}

func (cs *CombatSystem) IsCriticalHit(player *object.Player) bool {
	critRate := float64(player.GetAgility()) / 1000.0
	if critRate > 0.5 {
		critRate = 0.5
	}
	return float64(time.Now().UnixNano()%1000)/1000.0 < critRate
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
