package pet

import (
	"time"

	"github.com/pzqf/zMmoShared/common/id"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Pet 宠物结构
type Pet struct {
	PetID        id.PetIdType    // 宠物ID
	PlayerID     id.PlayerIdType // 主人ID
	PetTypeID    int             // 宠物类型ID
	Name         string          // 宠物名称
	Level        int             // 等级
	Exp          int64           // 经验值
	MaxExp       int64           // 最大经验值
	HP           int             // 生命值
	MaxHP        int             // 最大生命值
	MP           int             // 魔法值
	MaxMP        int             // 最大魔法值
	Attack       int             // 攻击力
	Defense      int             // 防御力
	Speed        int             // 速度
	Loyalty      int             // 忠诚度
	Happiness    int             // 快乐度
	Skills       []int           // 技能列表
	IsSummoned   bool            // 是否召唤中
	CreateTime   time.Time       // 创建时间
	LastFeedTime time.Time       // 最后喂食时间
}

// PetSkill 宠物技能
type PetSkill struct {
	SkillID  int       // 技能ID
	Level    int       // 技能等级
	CoolDown int       // 冷却时间(秒)
	LastUsed time.Time // 上次使用时间
}

// NewPet 创建新宠物
func NewPet(playerID id.PlayerIdType, petTypeID int, name string) *Pet {
	pet := &Pet{
		PetID:        id.PetIdType(id.GenerateId()),
		PlayerID:     playerID,
		PetTypeID:    petTypeID,
		Name:         name,
		Level:        1,
		Exp:          0,
		MaxExp:       100,
		HP:           100,
		MaxHP:        100,
		MP:           50,
		MaxMP:        50,
		Attack:       10,
		Defense:      5,
		Speed:        8,
		Loyalty:      100,
		Happiness:    100,
		Skills:       []int{},
		IsSummoned:   false,
		CreateTime:   time.Now(),
		LastFeedTime: time.Now(),
	}

	return pet
}

// AddExp 添加经验值
func (p *Pet) AddExp(exp int64) bool {
	p.Exp += exp
	levelUp := false

	for p.Exp >= p.MaxExp {
		p.Exp -= p.MaxExp
		p.Level++
		p.MaxExp = int64(100 + p.Level*50)
		p.MaxHP += 10
		p.HP = p.MaxHP
		p.MaxMP += 5
		p.MP = p.MaxMP
		p.Attack += 2
		p.Defense += 1
		p.Speed += 1
		levelUp = true
	}

	return levelUp
}

// Feed 喂食宠物
func (p *Pet) Feed(foodType int) bool {
	p.Happiness = min(p.Happiness+20, 100)
	p.Loyalty = min(p.Loyalty+10, 100)
	p.LastFeedTime = time.Now()
	return true
}

// Summon 召唤宠物
func (p *Pet) Summon() bool {
	if p.IsSummoned {
		return false
	}

	p.IsSummoned = true
	p.HP = p.MaxHP
	p.MP = p.MaxMP
	return true
}

// Dismiss 收回宠物
func (p *Pet) Dismiss() bool {
	if !p.IsSummoned {
		return false
	}

	p.IsSummoned = false
	return true
}

// AddSkill 添加技能
func (p *Pet) AddSkill(skillID int) bool {
	for _, s := range p.Skills {
		if s == skillID {
			return false
		}
	}

	p.Skills = append(p.Skills, skillID)
	return true
}

// RemoveSkill 移除技能
func (p *Pet) RemoveSkill(skillID int) bool {
	for i, s := range p.Skills {
		if s == skillID {
			p.Skills = append(p.Skills[:i], p.Skills[i+1:]...)
			return true
		}
	}

	return false
}

// UpdateStats 更新宠物状态
func (p *Pet) UpdateStats() {
	// 随时间减少快乐度和忠诚度
	elapsed := time.Since(p.LastFeedTime)
	if elapsed.Hours() > 1 {
		hours := int(elapsed.Hours())
		p.Happiness = max(p.Happiness-hours*2, 0)
		p.Loyalty = max(p.Loyalty-hours*1, 0)
	}
}

// GetCombatPower 获取战斗力
func (p *Pet) GetCombatPower() int {
	return p.Attack*3 + p.Defense*2 + p.MaxHP + p.MaxMP/2 + p.Level*10
}
