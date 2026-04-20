package skill

import (
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/models"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type SkillManager struct {
	skills       *zMap.TypedMap[id.ObjectIdType, *Skill]
	tableManager *tables.TableManager
}

func NewSkillManager() *SkillManager {
	return &SkillManager{
		skills: zMap.NewTypedMap[id.ObjectIdType, *Skill](),
	}
}

func (sm *SkillManager) SetTableManager(tm *tables.TableManager) {
	sm.tableManager = tm
}

func (sm *SkillManager) GetSkillConfig(skillID int32) *models.Skill {
	if sm.tableManager == nil {
		return nil
	}
	skill, ok := sm.tableManager.GetSkillLoader().GetSkill(skillID)
	if !ok {
		return nil
	}
	return skill
}

func (sm *SkillManager) AddSkill(skill *Skill) {
	sm.skills.Store(skill.GetID(), skill)
}

// RemoveSkill 移除技能
func (sm *SkillManager) RemoveSkill(skillID id.ObjectIdType) {
	sm.skills.Delete(skillID)
}

// GetSkill 获取技能
func (sm *SkillManager) GetSkill(skillID id.ObjectIdType) *Skill {
	skill, _ := sm.skills.Load(skillID)
	return skill
}

// Update 更新技能状态
func (sm *SkillManager) Update() {
	sm.skills.Range(func(skillID id.ObjectIdType, skill *Skill) bool {
		if skill.IsExpired() {
			sm.skills.Delete(skillID)
			zLog.Debug("Skill expired and removed", zap.Int64("skill_id", int64(skillID)))
		}
		return true
	})
}

// Skill 技能对象
type Skill struct {
	id            id.ObjectIdType
	skillConfigID int32
	casterID      id.ObjectIdType
	targetID      id.ObjectIdType
	position      common.Vector3
	damage        int32
	skillRange    float32
	effectType    int32
	startTime     time.Time
	duration      time.Duration
	cooldown      time.Duration
	lastUseTime   time.Time
	level         int32
	casterAttack  int32
	effectID      int32
	buffID        int32
}

// NewSkill 创建新技能
func NewSkill(id id.ObjectIdType, skillConfigID int32, casterID, targetID id.ObjectIdType, pos common.Vector3, damage int32, skillRange float32, effectType int32, buffID int32, duration, cooldown time.Duration) *Skill {
	return &Skill{
		id:            id,
		skillConfigID: skillConfigID,
		casterID:      casterID,
		targetID:      targetID,
		position:      pos,
		damage:        damage,
		skillRange:    skillRange,
		effectType:    effectType,
		buffID:        buffID,
		startTime:     time.Now(),
		duration:      duration,
		cooldown:      cooldown,
		lastUseTime:   time.Now(),
		level:         1,
		casterAttack:  0,
		effectID:      0,
	}
}

// GetID 获取技能ID
func (s *Skill) GetID() id.ObjectIdType {
	return s.id
}

// GetSkillConfigID 获取技能配置ID
func (s *Skill) GetSkillConfigID() int32 {
	return s.skillConfigID
}

// GetCasterID 获取施法者ID
func (s *Skill) GetCasterID() id.ObjectIdType {
	return s.casterID
}

// GetTargetID 获取目标ID
func (s *Skill) GetTargetID() id.ObjectIdType {
	return s.targetID
}

// GetPosition 获取技能位置
func (s *Skill) GetPosition() common.Vector3 {
	return s.position
}

// GetDamage 获取技能伤害
func (s *Skill) GetDamage() int32 {
	return s.damage
}

// GetRange 获取技能范围
func (s *Skill) GetRange() float32 {
	return s.skillRange
}

// GetEffectType 获取技能效果类型
func (s *Skill) GetEffectType() int32 {
	return s.effectType
}

// GetBuffID 获取技能附加的Buff ID
func (s *Skill) GetBuffID() int32 {
	return s.buffID
}

// SetLevel 设置技能等级
func (s *Skill) SetLevel(level int32) {
	s.level = level
}

// GetLevel 获取技能等级
func (s *Skill) GetLevel() int32 {
	return s.level
}

// SetCasterAttack 设置施法者攻击力
func (s *Skill) SetCasterAttack(attack int32) {
	s.casterAttack = attack
}

// GetCasterAttack 获取施法者攻击力
func (s *Skill) GetCasterAttack() int32 {
	return s.casterAttack
}

// SetEffectID 设置技能特效ID
func (s *Skill) SetEffectID(effectID int32) {
	s.effectID = effectID
}

// GetEffectID 获取技能特效ID
func (s *Skill) GetEffectID() int32 {
	return s.effectID
}

// IsInCooldown 检查是否在冷却中
func (s *Skill) IsInCooldown() bool {
	return time.Since(s.lastUseTime) < s.cooldown
}

// GetRemainingCooldown 获取剩余冷却时间
func (s *Skill) GetRemainingCooldown() time.Duration {
	elapsed := time.Since(s.lastUseTime)
	if elapsed < s.cooldown {
		return s.cooldown - elapsed
	}
	return 0
}

// IsExpired 检查技能是否过期
func (s *Skill) IsExpired() bool {
	return time.Since(s.startTime) > s.duration
}

// Use 使用技能
func (s *Skill) Use() {
	s.lastUseTime = time.Now()
}

// CalculateDamage 计算技能伤害
func (s *Skill) CalculateDamage(target *object.Player) int32 {
	// 基础伤害 + 施法者攻击力 * 技能系数 + 技能等级加成
	skillCoeff := 1.0 + float32(s.level)*0.1     // 每级增加10%伤害
	attackBonus := float32(s.casterAttack) * 0.5 // 攻击力的50%作为加成
	totalDamage := float32(s.damage) + attackBonus
	totalDamage *= skillCoeff
	return int32(totalDamage)
}

// CalculateDamageToMonster 计算对怪物的伤害
func (s *Skill) CalculateDamageToMonster(target *object.Monster) int32 {
	// 基础伤害 + 施法者攻击力 * 技能系数 + 技能等级加成
	skillCoeff := 1.0 + float32(s.level)*0.1     // 每级增加10%伤害
	attackBonus := float32(s.casterAttack) * 0.5 // 攻击力的50%作为加成
	totalDamage := float32(s.damage) + attackBonus
	totalDamage *= skillCoeff
	return int32(totalDamage)
}

// CheckHit 检查是否命中目标
func (s *Skill) CheckHit(targetPosition common.Vector3) bool {
	// 计算距离平方
	distSq := s.position.DistanceTo(targetPosition)
	rangeSq := s.skillRange * s.skillRange
	return distSq <= rangeSq
}
