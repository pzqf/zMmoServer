package skill

import (
	"errors"
	"sync"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
	"go.uber.org/zap"
)

// SkillManager 技能管理器
type SkillManager struct {
	mu       sync.RWMutex
	playerID id.PlayerIdType
	skills   map[int32]*Skill
	maxCount int32
}

// NewSkillManager 创建技能管理器
func NewSkillManager(playerID id.PlayerIdType, maxCount int32) *SkillManager {
	return &SkillManager{
		playerID: playerID,
		skills:   make(map[int32]*Skill),
		maxCount: maxCount,
	}
}

// GetPlayerID 获取玩家ID
func (sm *SkillManager) GetPlayerID() id.PlayerIdType {
	return sm.playerID
}

// GetSkillCount 获取技能数量
func (sm *SkillManager) GetSkillCount() int32 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return int32(len(sm.skills))
}

// GetMaxCount 获取最大技能数
func (sm *SkillManager) GetMaxCount() int32 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.maxCount
}

// AddSkill 添加技能
func (sm *SkillManager) AddSkill(skill *Skill) error {
	if skill == nil {
		return errors.New("skill is nil")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	skillConfigID := skill.GetSkillConfigID()

	// 检查是否已存在
	if _, exists := sm.skills[skillConfigID]; exists {
		return errors.New("skill already exists")
	}

	// 检查数量限制
	if int32(len(sm.skills)) >= sm.maxCount {
		return errors.New("skill count limit reached")
	}

	sm.skills[skillConfigID] = skill

	sm.publishSkillUnlockEvent(skill)

	zLog.Debug("Skill added",
		zap.Int64("player_id", int64(sm.playerID)),
		zap.Int32("skill_config_id", skillConfigID),
		zap.String("name", skill.GetName()))

	return nil
}

// RemoveSkill 移除技能
func (sm *SkillManager) RemoveSkill(skillConfigID int32) (*Skill, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[skillConfigID]
	if !exists {
		return nil, errors.New("skill not found")
	}

	delete(sm.skills, skillConfigID)

	zLog.Debug("Skill removed",
		zap.Int64("player_id", int64(sm.playerID)),
		zap.Int32("skill_config_id", skillConfigID))

	return skill, nil
}

// GetSkill 获取技能
func (sm *SkillManager) GetSkill(skillConfigID int32) (*Skill, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skill, exists := sm.skills[skillConfigID]
	if !exists {
		return nil, errors.New("skill not found")
	}
	return skill, nil
}

// HasSkill 检查是否有指定技能
func (sm *SkillManager) HasSkill(skillConfigID int32) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, exists := sm.skills[skillConfigID]
	return exists
}

// GetAllSkills 获取所有技能
func (sm *SkillManager) GetAllSkills() map[int32]*Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[int32]*Skill)
	for skillConfigID, skill := range sm.skills {
		result[skillConfigID] = skill
	}
	return result
}

// GetSkillsByType 获取指定类型的技能
func (sm *SkillManager) GetSkillsByType(skillType SkillType) []*Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]*Skill, 0)
	for _, skill := range sm.skills {
		if skill.GetSkillType() == skillType {
			result = append(result, skill)
		}
	}
	return result
}

// GetUnlockedSkills 获取已解锁的技能
func (sm *SkillManager) GetUnlockedSkills() []*Skill {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]*Skill, 0)
	for _, skill := range sm.skills {
		if skill.IsUnlocked() {
			result = append(result, skill)
		}
	}
	return result
}

// UnlockSkill 解锁技能
func (sm *SkillManager) UnlockSkill(skillConfigID int32) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[skillConfigID]
	if !exists {
		return errors.New("skill not found")
	}

	if !skill.Unlock() {
		return errors.New("skill already unlocked")
	}

	sm.publishSkillUnlockEvent(skill)

	zLog.Info("Skill unlocked",
		zap.Int64("player_id", int64(sm.playerID)),
		zap.Int32("skill_config_id", skillConfigID),
		zap.String("name", skill.GetName()))

	return nil
}

// UpgradeSkill 升级技能
func (sm *SkillManager) UpgradeSkill(skillConfigID int32, playerLevel int32, exp int64, gold int64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[skillConfigID]
	if !exists {
		return errors.New("skill not found")
	}

	if !skill.CanUpgrade(playerLevel) {
		return errors.New("cannot upgrade skill")
	}

	if exp < skill.GetExpCost() {
		return errors.New("not enough exp")
	}

	if gold < skill.GetGoldCost() {
		return errors.New("not enough gold")
	}

	oldLevel := skill.GetLevel()
	if !skill.Upgrade() {
		return errors.New("upgrade failed")
	}

	sm.publishSkillUpgradeEvent(skill, oldLevel)

	zLog.Info("Skill upgraded",
		zap.Int64("player_id", int64(sm.playerID)),
		zap.Int32("skill_config_id", skillConfigID),
		zap.String("name", skill.GetName()),
		zap.Int32("level", skill.GetLevel()))

	return nil
}

// UseSkill 使用技能
func (sm *SkillManager) UseSkill(skillConfigID int32, targetID id.ObjectIdType, mp int32) (*Skill, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[skillConfigID]
	if !exists {
		return nil, errors.New("skill not found")
	}

	if skill.GetStatus() == SkillStatusLocked {
		return nil, errors.New("skill is locked")
	}

	if skill.IsInCooldown() {
		return nil, errors.New("skill is in cooldown")
	}

	if mp < skill.GetMPCost() {
		return nil, errors.New("not enough mp")
	}

	if !skill.Use() {
		return nil, errors.New("skill use failed")
	}

	sm.publishSkillUseEvent(skill, targetID)

	zLog.Debug("Skill used",
		zap.Int64("player_id", int64(sm.playerID)),
		zap.Int32("skill_config_id", skillConfigID),
		zap.String("name", skill.GetName()),
		zap.Int64("target_id", int64(targetID)))

	return skill, nil
}

// CheckCooldown 检查技能冷却
func (sm *SkillManager) CheckCooldown(skillConfigID int32) (bool, int32, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	skill, exists := sm.skills[skillConfigID]
	if !exists {
		return false, 0, errors.New("skill not found")
	}

	isInCooldown := skill.IsInCooldown()
	remainingCooldown := skill.GetRemainingCooldown()

	return isInCooldown, remainingCooldown, nil
}

// ResetCooldown 重置技能冷却
func (sm *SkillManager) ResetCooldown(skillConfigID int32) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, exists := sm.skills[skillConfigID]
	if !exists {
		return errors.New("skill not found")
	}

	skill.lastUseTime = 0
	return nil
}

// ResetAllCooldowns 重置所有技能冷却
func (sm *SkillManager) ResetAllCooldowns() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, skill := range sm.skills {
		skill.lastUseTime = 0
	}
}

// GetTotalSkillLevel 获取技能总等级
func (sm *SkillManager) GetTotalSkillLevel() int32 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	total := int32(0)
	for _, skill := range sm.skills {
		total += skill.GetLevel()
	}
	return total
}

// GetSkillPower 获取技能总战斗力
func (sm *SkillManager) GetSkillPower() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	power := 0.0
	for _, skill := range sm.skills {
		if skill.IsUnlocked() {
			power += float64(skill.GetLevel()) * 100
			for _, effect := range skill.GetEffects() {
				power += effect.Value * float64(skill.GetLevel())
			}
		}
	}
	return power
}

// Clear 清除所有技能
func (sm *SkillManager) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.skills = make(map[int32]*Skill)
}

// publishSkillUnlockEvent 发布技能解锁事件
func (sm *SkillManager) publishSkillUnlockEvent(skill *Skill) {
	event.Publish(event.NewEvent(event.EventPlayerSkillUnlock, sm, &event.PlayerSkillEventData{
		PlayerID:      sm.playerID,
		SkillConfigID: skill.GetSkillConfigID(),
		SkillName:     skill.GetName(),
		Level:         skill.GetLevel(),
	}))
}

// publishSkillUpgradeEvent 发布技能升级事件
func (sm *SkillManager) publishSkillUpgradeEvent(skill *Skill, oldLevel int32) {
	event.Publish(event.NewEvent(event.EventPlayerSkillUpgrade, sm, &event.PlayerSkillEventData{
		PlayerID:      sm.playerID,
		SkillConfigID: skill.GetSkillConfigID(),
		SkillName:     skill.GetName(),
		Level:         skill.GetLevel(),
		OldLevel:      oldLevel,
	}))
}

// publishSkillUseEvent 发布技能使用事件
func (sm *SkillManager) publishSkillUseEvent(skill *Skill, targetID id.ObjectIdType) {
	event.Publish(event.NewEvent(event.EventPlayerSkillUse, sm, &event.PlayerSkillEventData{
		PlayerID:      sm.playerID,
		SkillConfigID: skill.GetSkillConfigID(),
		SkillName:     skill.GetName(),
		Level:         skill.GetLevel(),
		TargetID:      targetID,
	}))
}
