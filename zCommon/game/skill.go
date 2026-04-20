package game

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zUtil/zMap"
)

type SkillType int

const (
	SkillTypeActive   SkillType = 1
	SkillTypePassive  SkillType = 2
	SkillTypeUltimate SkillType = 3
)

type SkillStatus int

const (
	SkillStatusLocked   SkillStatus = 0
	SkillStatusUnlocked SkillStatus = 1
	SkillStatusUpgraded SkillStatus = 2
)

type Skill struct {
	mu          sync.RWMutex
	SkillID     int32
	Name        string
	Description string
	Type        SkillType
	Status      SkillStatus
	Level       int32
	MaxLevel    int32
	Cooldown    time.Duration
	LastUseTime time.Time
	ManaCost    int32
	Range       float32
	Damage      int32
}

func NewSkill(skillID int32, name string, skillType SkillType) *Skill {
	return &Skill{
		SkillID:  skillID,
		Name:     name,
		Type:     skillType,
		Status:   SkillStatusLocked,
		Level:    1,
		MaxLevel: 10,
		Cooldown: time.Second,
	}
}

func (s *Skill) CanUse() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status != SkillStatusLocked && time.Since(s.LastUseTime) >= s.Cooldown
}

func (s *Skill) Use() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastUseTime = time.Now()
}

func (s *Skill) IsOnCooldown() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.LastUseTime) < s.Cooldown
}

func (s *Skill) GetCooldownRemaining() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	remaining := s.Cooldown - time.Since(s.LastUseTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

type SkillManager struct {
	BaseComponent
	mu     sync.RWMutex
	skills *zMap.TypedMap[int32, *Skill]
	maxCount int32
}

func NewSkillManager(maxCount int32) *SkillManager {
	return &SkillManager{
		BaseComponent: NewBaseComponent("skills"),
		skills:        zMap.NewTypedMap[int32, *Skill](),
		maxCount:      maxCount,
	}
}

func (sm *SkillManager) LearnSkill(skill *Skill) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if int32(sm.skills.Len()) >= sm.maxCount {
		return fmt.Errorf("skill slots full")
	}

	if _, ok := sm.skills.Load(skill.SkillID); ok {
		return fmt.Errorf("skill %d already learned", skill.SkillID)
	}

	skill.Status = SkillStatusUnlocked
	sm.skills.Store(skill.SkillID, skill)
	return nil
}

func (sm *SkillManager) UseSkill(skillID int32) error {
	sm.mu.RLock()
	skill, ok := sm.skills.Load(skillID)
	sm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("skill %d not learned", skillID)
	}

	if !skill.CanUse() {
		return fmt.Errorf("skill %d on cooldown", skillID)
	}

	skill.Use()
	return nil
}

func (sm *SkillManager) UpgradeSkill(skillID int32) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	skill, ok := sm.skills.Load(skillID)
	if !ok {
		return fmt.Errorf("skill %d not learned", skillID)
	}

	if skill.Level >= skill.MaxLevel {
		return fmt.Errorf("skill %d already at max level", skillID)
	}

	skill.Level++
	skill.Status = SkillStatusUpgraded
	return nil
}

func (sm *SkillManager) GetSkill(skillID int32) (*Skill, bool) {
	return sm.skills.Load(skillID)
}

func (sm *SkillManager) GetAllSkills() map[int32]*Skill {
	result := make(map[int32]*Skill)
	sm.skills.Range(func(id int32, skill *Skill) bool {
		result[id] = skill
		return true
	})
	return result
}

func (sm *SkillManager) Count() int64 {
	return sm.skills.Len()
}
