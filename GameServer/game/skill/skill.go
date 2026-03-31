package skill

import (
	"sync"
	"time"
)

// SkillType 技能类型
type SkillType int32

const (
	SkillTypeActive   SkillType = 1 // 主动技能
	SkillTypePassive  SkillType = 2 // 被动技能
	SkillTypeUltimate SkillType = 3 // 终极技能
)

// SkillStatus 技能状态
type SkillStatus int32

const (
	SkillStatusLocked   SkillStatus = 1 // 未解锁
	SkillStatusUnlocked SkillStatus = 2 // 已解锁
	SkillStatusMaxLevel SkillStatus = 3 // 已满等级
)

// SkillEffectType 技能效果类型
type SkillEffectType int32

const (
	SkillEffectTypeDamage   SkillEffectType = 1 // 伤害
	SkillEffectTypeHeal     SkillEffectType = 2 // 治疗
	SkillEffectTypeBuff     SkillEffectType = 3 // 增益
	SkillEffectTypeDebuff   SkillEffectType = 4 // 减益
	SkillEffectTypeControl  SkillEffectType = 5 // 控制
	SkillEffectTypeSummon   SkillEffectType = 6 // 召唤
	SkillEffectTypeTeleport SkillEffectType = 7 // 传送
)

// SkillTargetType 技能目标类型
type SkillTargetType int32

const (
	SkillTargetTypeSelf   SkillTargetType = 0 // 自身
	SkillTargetTypeSingle SkillTargetType = 1 // 单个目标
	SkillTargetTypeAoE    SkillTargetType = 2 // 范围目标
	SkillTargetTypeLine   SkillTargetType = 3 // 直线目标
	SkillTargetTypeSector SkillTargetType = 4 // 扇形目标
)

// SkillEffect 技能效果
type SkillEffect struct {
	EffectType SkillEffectType
	Value      float64
	Duration   int32 // 毫秒
	Range      float32
	TargetType SkillTargetType
}

// Skill 技能结构
type Skill struct {
	mu            sync.RWMutex
	skillID       int64
	skillConfigID int32
	name          string
	description   string
	skillType     SkillType
	status        SkillStatus
	level         int32
	maxLevel      int32
	requireLevel  int32
	expCost       int64
	goldCost      int64
	effects       []*SkillEffect
	cooldown      int32 // 毫秒
	lastUseTime   int64
	mpCost        int32
	castTime      int32 // 毫秒
	castRange     float32
}

// NewSkill 创建新技能
func NewSkill(skillConfigID int32, name string, skillType SkillType) *Skill {
	return &Skill{
		skillConfigID: skillConfigID,
		name:          name,
		skillType:     skillType,
		status:        SkillStatusLocked,
		level:         0,
		maxLevel:      10,
		requireLevel:  1,
		expCost:       0,
		goldCost:      0,
		effects:       make([]*SkillEffect, 0),
		cooldown:      0,
		lastUseTime:   0,
		mpCost:        0,
		castTime:      0,
		castRange:     0,
	}
}

// GetSkillID 获取技能ID
func (s *Skill) GetSkillID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.skillID
}

// SetSkillID 设置技能ID
func (s *Skill) SetSkillID(skillID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.skillID = skillID
}

// GetSkillConfigID 获取技能配置ID
func (s *Skill) GetSkillConfigID() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.skillConfigID
}

// GetName 获取技能名称
func (s *Skill) GetName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.name
}

// GetDescription 获取技能描述
func (s *Skill) GetDescription() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.description
}

// SetDescription 设置技能描述
func (s *Skill) SetDescription(description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.description = description
}

// GetSkillType 获取技能类型
func (s *Skill) GetSkillType() SkillType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.skillType
}

// GetStatus 获取技能状态
func (s *Skill) GetStatus() SkillStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// SetStatus 设置技能状态
func (s *Skill) SetStatus(status SkillStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
}

// GetLevel 获取技能等级
func (s *Skill) GetLevel() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.level
}

// GetMaxLevel 获取最大等级
func (s *Skill) GetMaxLevel() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxLevel
}

// SetMaxLevel 设置最大等级
func (s *Skill) SetMaxLevel(maxLevel int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxLevel = maxLevel
}

// GetRequireLevel 获取需求等级
func (s *Skill) GetRequireLevel() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requireLevel
}

// SetRequireLevel 设置需求等级
func (s *Skill) SetRequireLevel(requireLevel int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requireLevel = requireLevel
}

// GetExpCost 获取经验消耗
func (s *Skill) GetExpCost() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.expCost
}

// SetExpCost 设置经验消耗
func (s *Skill) SetExpCost(expCost int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expCost = expCost
}

// GetGoldCost 获取金币消耗
func (s *Skill) GetGoldCost() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.goldCost
}

// SetGoldCost 设置金币消耗
func (s *Skill) SetGoldCost(goldCost int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.goldCost = goldCost
}

// GetCooldown 获取冷却时间
func (s *Skill) GetCooldown() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cooldown
}

// SetCooldown 设置冷却时间
func (s *Skill) SetCooldown(cooldown int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cooldown = cooldown
}

// GetMPCost 获取魔法消耗
func (s *Skill) GetMPCost() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mpCost
}

// SetMPCost 设置魔法消耗
func (s *Skill) SetMPCost(mpCost int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mpCost = mpCost
}

// GetCastTime 获取施法时间
func (s *Skill) GetCastTime() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.castTime
}

// SetCastTime 设置施法时间
func (s *Skill) SetCastTime(castTime int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.castTime = castTime
}

// GetCastRange 获取施法范围
func (s *Skill) GetCastRange() float32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.castRange
}

// SetCastRange 设置施法范围
func (s *Skill) SetCastRange(castRange float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.castRange = castRange
}

// GetEffects 获取技能效果
func (s *Skill) GetEffects() []*SkillEffect {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*SkillEffect, len(s.effects))
	copy(result, s.effects)
	return result
}

// AddEffect 添加技能效果
func (s *Skill) AddEffect(effect *SkillEffect) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.effects = append(s.effects, effect)
}

// ClearEffects 清除所有效果
func (s *Skill) ClearEffects() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.effects = make([]*SkillEffect, 0)
}

// IsUnlocked 检查是否已解锁
func (s *Skill) IsUnlocked() bool {
	return s.GetStatus() != SkillStatusLocked
}

// IsMaxLevel 检查是否已满级
func (s *Skill) IsMaxLevel() bool {
	return s.GetLevel() >= s.GetMaxLevel()
}

// CanUpgrade 检查是否可以升级
func (s *Skill) CanUpgrade(playerLevel int32) bool {
	if !s.IsUnlocked() {
		return false
	}
	if s.IsMaxLevel() {
		return false
	}
	return playerLevel >= s.GetRequireLevel()
}

// Upgrade 升级技能
func (s *Skill) Upgrade() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.level >= s.maxLevel {
		return false
	}

	s.level++
	if s.level >= s.maxLevel {
		s.status = SkillStatusMaxLevel
	}

	return true
}

// Unlock 解锁技能
func (s *Skill) Unlock() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != SkillStatusLocked {
		return false
	}

	s.status = SkillStatusUnlocked
	s.level = 1
	return true
}

// IsInCooldown 检查是否在冷却中
func (s *Skill) IsInCooldown() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cooldown <= 0 {
		return false
	}

	currentTime := time.Now().UnixMilli()
	return currentTime-s.lastUseTime < int64(s.cooldown)
}

// GetRemainingCooldown 获取剩余冷却时间
func (s *Skill) GetRemainingCooldown() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cooldown <= 0 {
		return 0
	}

	currentTime := time.Now().UnixMilli()
	elapsed := int32(currentTime - s.lastUseTime)
	if elapsed >= s.cooldown {
		return 0
	}
	return s.cooldown - elapsed
}

// Use 使用技能
func (s *Skill) Use() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == SkillStatusLocked {
		return false
	}

	if s.IsInCooldown() {
		return false
	}

	s.lastUseTime = time.Now().UnixMilli()
	return true
}

// CalculateDamage 计算伤害
func (s *Skill) CalculateDamage(baseAttack float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	damage := baseAttack
	for _, effect := range s.effects {
		if effect.EffectType == SkillEffectTypeDamage {
			damage += effect.Value * float64(s.level)
		}
	}
	return damage
}

// CalculateHeal 计算治疗
func (s *Skill) CalculateHeal(baseHeal float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	heal := baseHeal
	for _, effect := range s.effects {
		if effect.EffectType == SkillEffectTypeHeal {
			heal += effect.Value * float64(s.level)
		}
	}
	return heal
}

// Clone 克隆技能
func (s *Skill) Clone() *Skill {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &Skill{
		skillConfigID: s.skillConfigID,
		name:          s.name,
		description:   s.description,
		skillType:     s.skillType,
		status:        s.status,
		level:         s.level,
		maxLevel:      s.maxLevel,
		requireLevel:  s.requireLevel,
		expCost:       s.expCost,
		goldCost:      s.goldCost,
		effects:       make([]*SkillEffect, len(s.effects)),
		cooldown:      s.cooldown,
		lastUseTime:   s.lastUseTime,
		mpCost:        s.mpCost,
		castTime:      s.castTime,
		castRange:     s.castRange,
	}

	copy(clone.effects, s.effects)
	return clone
}
