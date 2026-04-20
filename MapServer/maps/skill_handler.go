package maps

import (
	"fmt"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"github.com/pzqf/zMmoServer/MapServer/maps/skill"
	"go.uber.org/zap"
)

func (m *Map) HandleSkillUse(caster *object.Player, skillID int32, targetID id.ObjectIdType, targetPos common.Vector3) error {
	skillConfig := m.skillManager.GetSkillConfig(skillID)
	if skillConfig == nil {
		return fmt.Errorf("skill not found")
	}

	if caster.IsSkillInCooldown(skillID) {
		return fmt.Errorf("skill is on cooldown")
	}

	if caster.GetMana() < skillConfig.ManaCost {
		return fmt.Errorf("not enough mana")
	}

	var target common.IGameObject
	if targetID != 0 {
		target = m.GetObject(targetID)
		if target == nil {
			return fmt.Errorf("target not found")
		}
		if !m.ValidateTarget(caster, target, skillConfig.Range) {
			return fmt.Errorf("target out of range")
		}
	} else {
		if !m.IsPositionInMap(targetPos) {
			return fmt.Errorf("invalid skill position")
		}
	}

	caster.SetMana(caster.GetMana() - skillConfig.ManaCost)
	caster.SetSkillCooldown(skillID, time.Duration(skillConfig.Cooldown*1000)*time.Millisecond)

	skillObjectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)
	skillDuration := time.Duration(skillConfig.SkillCastTime*1000) * time.Millisecond
	if skillDuration == 0 {
		skillDuration = 500 * time.Millisecond
	}
	newSkill := skill.NewSkill(
		skillObjectID,
		skillID,
		caster.GetID(),
		targetID,
		targetPos,
		skillConfig.Damage,
		skillConfig.Range,
		skillConfig.Type,
		skillConfig.BuffID,
		skillDuration,
		time.Duration(skillConfig.Cooldown*1000)*time.Millisecond,
	)

	newSkill.SetLevel(1)
	newSkill.SetCasterAttack(caster.GetAttack())
	m.skillManager.AddSkill(newSkill)
	m.handleSkillEffect(newSkill)

	zLog.Debug("Skill used",
		zap.Int64("caster_id", int64(caster.GetID())),
		zap.Int32("skill_id", skillID),
		zap.Int64("target_id", int64(targetID)),
		zap.Float32("x", targetPos.X),
		zap.Float32("y", targetPos.Y))

	return nil
}

func (m *Map) handleSkillEffect(skill *skill.Skill) {
	objects := m.GetObjectsInRange(skill.GetPosition(), skill.GetRange())

	for _, obj := range objects {
		if obj.GetID() == skill.GetCasterID() {
			continue
		}

		targetType := obj.GetType()
		validTarget := false

		switch skill.GetEffectType() {
		case 1:
			validTarget = targetType == common.GameObjectTypeMonster || targetType == common.GameObjectTypePlayer
		case 2:
			validTarget = targetType == common.GameObjectTypePlayer
		case 3:
			validTarget = targetType == common.GameObjectTypePlayer
		case 4:
			validTarget = targetType == common.GameObjectTypeMonster || targetType == common.GameObjectTypePlayer
		}

		if !validTarget {
			continue
		}

		switch skill.GetEffectType() {
		case 1:
			if obj.GetType() == common.GameObjectTypeMonster {
				if monster, ok := obj.(*object.Monster); ok {
					damage := skill.CalculateDamageToMonster(monster)
					monster.TakeDamage(damage)
					if monster.GetHealth() <= 0 {
						casterObj := m.GetObject(skill.GetCasterID())
						if casterObj == nil {
							casterObj = obj
						}
						m.handleMonsterDeath(monster, casterObj)
					}
				}
			} else if obj.GetType() == common.GameObjectTypePlayer {
				if player, ok := obj.(*object.Player); ok {
					damage := skill.CalculateDamage(player)
					player.TakeDamage(damage)
					if player.GetHealth() <= 0 {
						casterObj := m.GetObject(skill.GetCasterID())
						m.handlePlayerDeath(player, casterObj)
					}
				}
			}
		case 2:
			if player, ok := obj.(*object.Player); ok {
				healAmount := skill.GetDamage()
				newHealth := player.GetHealth() + healAmount
				if newHealth > player.GetMaxHealth() {
					newHealth = player.GetMaxHealth()
				}
				player.SetHealth(newHealth)
			}
		case 3:
			if player, ok := obj.(*object.Player); ok {
				buffID := skill.GetBuffID()
				if buffID > 0 {
					if err := m.buffManager.AddBuff(player.GetPlayerID(), buffID, skill.GetCasterID()); err != nil {
						zLog.Warn("Failed to apply buff",
							zap.Int64("player_id", int64(player.GetPlayerID())),
							zap.Int32("buff_id", buffID),
							zap.Error(err))
					} else {
						zLog.Debug("Buff applied by skill",
							zap.Int64("player_id", int64(player.GetPlayerID())),
							zap.Int32("buff_id", buffID),
							zap.Int32("skill_id", skill.GetSkillConfigID()))
					}
				}
			}
		case 4:
			buffID := skill.GetBuffID()
			if buffID > 0 {
				if player, ok := obj.(*object.Player); ok {
					if err := m.buffManager.AddBuff(player.GetPlayerID(), buffID, skill.GetCasterID()); err != nil {
						zLog.Warn("Failed to apply control debuff",
							zap.Int64("player_id", int64(player.GetPlayerID())),
							zap.Int32("buff_id", buffID),
							zap.Error(err))
					} else {
						zLog.Debug("Control debuff applied by skill",
							zap.Int64("player_id", int64(player.GetPlayerID())),
							zap.Int32("buff_id", buffID),
							zap.Int32("skill_id", skill.GetSkillConfigID()))
					}
				} else if monster, ok := obj.(*object.Monster); ok {
					zLog.Debug("Control effect applied to monster",
						zap.Int64("monster_id", int64(monster.GetID())),
						zap.Int32("buff_id", buffID),
						zap.Int32("skill_id", skill.GetSkillConfigID()))
				}
			}
		}
	}
}

func (m *Map) UpdateSkills() {
	if m.skillManager != nil {
		m.skillManager.Update()
	}
}
