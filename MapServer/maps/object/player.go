package object

import (
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/buff"
	"go.uber.org/zap"
)

// PlayerStatus 玩家状态
type PlayerStatus int

const (
	PlayerStatusIdle PlayerStatus = iota
	PlayerStatusMoving
	PlayerStatusFighting
	PlayerStatusDead
	PlayerStatusTrading
	PlayerStatusTalking
)

// Buff 增益效果
type Buff struct {
	ID        int32
	Name      string
	Duration  time.Duration
	StartTime time.Time
	Effects   []BuffEffect
}

// BuffEffect 增益效果
type BuffEffect struct {
	Attribute string
	Value     int32
}

// Debuff 减益效果
type Debuff struct {
	ID        int32
	Name      string
	Duration  time.Duration
	StartTime time.Time
	Effects   []DebuffEffect
}

// DebuffEffect 减益效果
type DebuffEffect struct {
	Attribute string
	Value     int32
}

// Player 玩家对象
type Player struct {
	id               id.ObjectIdType
	playerID         id.PlayerIdType
	name             string
	position         common.Vector3
	level            int32
	class            int32
	exp              int64
	expToNext        int64
	health           int32
	maxHealth        int32
	mana             int32
	maxMana          int32
	strength         int32
	agility          int32
	intelligence     int32
	stamina          int32
	spirit           int32
	skillPoints      int32
	attributePoints  int32
	items            []int32
	skills           []int32
	buffs            []Buff
	debuffs          []Debuff
	status           PlayerStatus
	lastAction       time.Time
	skillCooldowns   map[int32]time.Time // 技能冷却时间
	skillHistory     []int32             // 技能释放历史
	skillHistoryTime time.Time           // 技能释放历史开始时间
	buffManager      *buff.BuffManager
	equipAttack      int32
	equipDefense     int32
	equipHP          int32
	equipMP          int32
	equipCritRate    float32
	equipSpeed       float32
}

// NewPlayer 创建新玩家
func NewPlayer(objectID id.ObjectIdType, playerID id.PlayerIdType, name string, pos common.Vector3, class int32) *Player {
	maxHealth := int32(100 + 10*10) // 基础生命值 + 耐力*10
	maxMana := int32(50 + 5*10)     // 基础魔法值 + 精神*5

	return &Player{
		id:               objectID,
		playerID:         playerID,
		name:             name,
		position:         pos,
		level:            1,
		class:            class,
		exp:              0,
		expToNext:        1000,
		health:           maxHealth,
		maxHealth:        maxHealth,
		mana:             maxMana,
		maxMana:          maxMana,
		strength:         10,
		agility:          10,
		intelligence:     10,
		stamina:          10,
		spirit:           10,
		skillPoints:      1,
		attributePoints:  5,
		items:            make([]int32, 0),
		skills:           make([]int32, 0),
		buffs:            make([]Buff, 0),
		debuffs:          make([]Debuff, 0),
		status:           PlayerStatusIdle,
		lastAction:       time.Now(),
		skillCooldowns:   make(map[int32]time.Time),
		skillHistory:     make([]int32, 0),
		skillHistoryTime: time.Now(),
	}
}

// GetID 获取对象ID
func (p *Player) GetID() id.ObjectIdType {
	return p.id
}

// GetType 获取对象类型
func (p *Player) GetType() common.GameObjectType {
	return common.GameObjectTypePlayer
}

// GetPosition 获取位置
func (p *Player) GetPosition() common.Vector3 {
	return p.position
}

// SetPosition 设置位置
func (p *Player) SetPosition(pos common.Vector3) {
	p.position = pos
}

// GetPlayerID 获取玩家ID
func (p *Player) GetPlayerID() id.PlayerIdType {
	return p.playerID
}

// GetName 获取玩家名称
func (p *Player) GetName() string {
	return p.name
}

// GetLevel 获取等级
func (p *Player) GetLevel() int32 {
	return p.level
}

// SetLevel 设置等级
func (p *Player) SetLevel(level int32) {
	p.level = level
}

// GetExp 获取经验
func (p *Player) GetExp() int64 {
	return p.exp
}

// AddExp 添加经验
func (p *Player) AddExp(exp int64) {
	p.exp += exp
	p.lastAction = time.Now()

	// 检查是否升级
	for p.exp >= p.expToNext {
		p.LevelUp()
	}
}

// LevelUp 玩家升级
func (p *Player) LevelUp() {
	p.level++
	p.exp -= p.expToNext
	p.expToNext = int64(1000 * p.level)

	// 增加基础属性
	p.strength += 1
	p.agility += 1
	p.intelligence += 1
	p.stamina += 1
	p.spirit += 1

	// 增加可分配的属性点和技能点
	p.attributePoints += 5
	p.skillPoints += 2

	// 更新生命值和魔法值
	p.maxHealth = int32(100 + p.stamina*10)
	p.maxMana = int32(50 + p.spirit*5)
	p.health = p.maxHealth
	p.mana = p.maxMana

	// 这里可以添加升级奖励逻辑
}

// GetExpToNext 获取升级所需经验
func (p *Player) GetExpToNext() int64 {
	return p.expToNext
}

// GetMaxHealth 获取最大生命值
func (p *Player) GetMaxHealth() int32 {
	return p.maxHealth
}

// GetMaxMana 获取最大魔法值
func (p *Player) GetMaxMana() int32 {
	return p.maxMana
}

// GetStamina 获取耐力
func (p *Player) GetStamina() int32 {
	return p.stamina
}

// GetSpirit 获取精神
func (p *Player) GetSpirit() int32 {
	return p.spirit
}

// GetStatus 获取玩家状态
func (p *Player) GetStatus() PlayerStatus {
	return p.status
}

// SetStatus 设置玩家状态
func (p *Player) SetStatus(status PlayerStatus) {
	p.status = status
	p.lastAction = time.Now()
}

// GetLastAction 获取最后行动时间
func (p *Player) GetLastAction() time.Time {
	return p.lastAction
}

// AddBuff 添加增益效果
func (p *Player) AddBuff(buff Buff) {
	p.buffs = append(p.buffs, buff)
}

// RemoveBuff 移除增益效果
func (p *Player) RemoveBuff(buffID int32) {
	for i, b := range p.buffs {
		if b.ID == buffID {
			p.buffs = append(p.buffs[:i], p.buffs[i+1:]...)
			break
		}
	}
}

// GetBuffs 获取增益效果列表
func (p *Player) GetBuffs() []Buff {
	return p.buffs
}

// AddDebuff 添加减益效果
func (p *Player) AddDebuff(debuff Debuff) {
	p.debuffs = append(p.debuffs, debuff)
}

// RemoveDebuff 移除减益效果
func (p *Player) RemoveDebuff(debuffID int32) {
	for i, d := range p.debuffs {
		if d.ID == debuffID {
			p.debuffs = append(p.debuffs[:i], p.debuffs[i+1:]...)
			break
		}
	}
}

// GetDebuffs 获取减益效果列表
func (p *Player) GetDebuffs() []Debuff {
	return p.debuffs
}

// IsAlive 检查玩家是否活着
func (p *Player) IsAlive() bool {
	return p.health > 0 && p.status != PlayerStatusDead
}

// CanMove 检查玩家是否可以移动
func (p *Player) CanMove() bool {
	return p.IsAlive() && p.status != PlayerStatusFighting && p.status != PlayerStatusTrading && p.status != PlayerStatusTalking
}

// CanFight 检查玩家是否可以战斗
func (p *Player) CanFight() bool {
	return p.IsAlive() && p.status != PlayerStatusTrading && p.status != PlayerStatusTalking
}

// GetHealth 获取生命值
func (p *Player) GetHealth() int32 {
	return p.health
}

// SetHealth 设置生命值
func (p *Player) SetHealth(health int32) {
	p.health = health
}

// AddExperience 添加经验值
func (p *Player) AddExperience(exp int64) {
	p.AddExp(exp)
}

func (p *Player) GetExperience() int64 {
	return p.GetExp()
}

func (p *Player) SetExperience(exp int64) {
	p.exp = exp
}

func (p *Player) GetAttackRange() float32 {
	return 3.0
}

// TakeDamage 受到伤害
func (p *Player) TakeDamage(damage int32) {
	if p.health > damage {
		p.health -= damage
	} else {
		p.health = 0
		p.status = PlayerStatusDead
	}
	p.lastAction = time.Now()
}

// GetMana 获取魔法值
func (p *Player) GetMana() int32 {
	return p.mana
}

// SetMana 设置魔法值
func (p *Player) SetMana(mana int32) {
	p.mana = mana
}

// GetAttack 获取攻击力
func (p *Player) GetAttack() int32 {
	attack := int32(10 + p.strength*2) + p.equipAttack
	if p.buffManager != nil {
		attack += p.buffManager.CalculateTotalAttackMod(p.playerID)
	}
	return attack
}

// GetDefense 获取防御力
func (p *Player) GetDefense() int32 {
	defense := int32(5 + p.stamina) + p.equipDefense
	if p.buffManager != nil {
		defense += p.buffManager.CalculateTotalDefenseMod(p.playerID)
	}
	return defense
}

// GetStrength 获取力量
func (p *Player) GetStrength() int32 {
	return p.strength
}

// GetAgility 获取敏捷
func (p *Player) GetAgility() int32 {
	return p.agility
}

// GetIntelligence 获取智力
func (p *Player) GetIntelligence() int32 {
	return p.intelligence
}

// GetClass 获取职业
func (p *Player) GetClass() int32 {
	return p.class
}

// AddItem 添加物品
func (p *Player) AddItem(itemID int32) {
	p.items = append(p.items, itemID)
	p.lastAction = time.Now()
}

// RemoveItem 移除物品
func (p *Player) RemoveItem(itemID int32) {
	for i, id := range p.items {
		if id == itemID {
			p.items = append(p.items[:i], p.items[i+1:]...)
			break
		}
	}
}

// GetItems 获取物品列表
func (p *Player) GetItems() []int32 {
	return p.items
}

// AddSkill 添加技能
func (p *Player) AddSkill(skillID int32) {
	p.skills = append(p.skills, skillID)
}

// RemoveSkill 移除技能
func (p *Player) RemoveSkill(skillID int32) {
	for i, id := range p.skills {
		if id == skillID {
			p.skills = append(p.skills[:i], p.skills[i+1:]...)
			break
		}
	}
}

// GetSkills 获取技能列表
func (p *Player) GetSkills() []int32 {
	return p.skills
}

// GetSkillPoints 获取技能点
func (p *Player) GetSkillPoints() int32 {
	return p.skillPoints
}

// AddSkillPoints 添加技能点
func (p *Player) AddSkillPoints(points int32) {
	p.skillPoints += points
}

// SpendSkillPoints 消耗技能点
func (p *Player) SpendSkillPoints(points int32) bool {
	if p.skillPoints >= points {
		p.skillPoints -= points
		return true
	}
	return false
}

// HasSkill 检查玩家是否拥有指定技能
func (p *Player) HasSkill(skillID int32) bool {
	for _, id := range p.skills {
		if id == skillID {
			return true
		}
	}
	return false
}

// IsSkillInCooldown 检查技能是否在冷却中
func (p *Player) IsSkillInCooldown(skillID int32) bool {
	if cooldownEnd, exists := p.skillCooldowns[skillID]; exists {
		return time.Now().Before(cooldownEnd)
	}
	return false
}

// SetSkillCooldown 设置技能冷却
func (p *Player) SetSkillCooldown(skillID int32, cooldown time.Duration) {
	p.skillCooldowns[skillID] = time.Now().Add(cooldown)
}

// GetSkillRemainingCooldown 获取技能剩余冷却时间
func (p *Player) GetSkillRemainingCooldown(skillID int32) time.Duration {
	if cooldownEnd, exists := p.skillCooldowns[skillID]; exists {
		remaining := time.Until(cooldownEnd)
		if remaining > 0 {
			return remaining
		}
		// 冷却时间已过，清理
		delete(p.skillCooldowns, skillID)
	}
	return 0
}

// ClearExpiredCooldowns 清理过期的冷却时间
func (p *Player) ClearExpiredCooldowns() {
	now := time.Now()
	for skillID, cooldownEnd := range p.skillCooldowns {
		if now.After(cooldownEnd) {
			delete(p.skillCooldowns, skillID)
		}
	}
}

// AddSkillToHistory 添加技能到释放历史
func (p *Player) AddSkillToHistory(skillID int32) {
	// 检查历史记录是否过期
	if time.Since(p.skillHistoryTime) > 5*time.Second {
		p.skillHistory = make([]int32, 0)
		p.skillHistoryTime = time.Now()
	}

	// 添加技能到历史记录
	p.skillHistory = append(p.skillHistory, skillID)

	// 限制历史记录长度为10
	if len(p.skillHistory) > 10 {
		p.skillHistory = p.skillHistory[len(p.skillHistory)-10:]
	}
}

// GetSkillHistory 获取技能释放历史
func (p *Player) GetSkillHistory() []int32 {
	return p.skillHistory
}

// GetSkillHistoryTime 获取技能释放历史开始时间
func (p *Player) GetSkillHistoryTime() time.Time {
	return p.skillHistoryTime
}

// ClearSkillHistory 清理技能释放历史
func (p *Player) ClearSkillHistory() {
	p.skillHistory = make([]int32, 0)
	p.skillHistoryTime = time.Now()
}

// GetAttributePoints 获取属性点
func (p *Player) GetAttributePoints() int32 {
	return p.attributePoints
}

// AddAttributePoints 添加属性点
func (p *Player) AddAttributePoints(points int32) {
	p.attributePoints += points
}

// SpendAttributePoints 消耗属性点
func (p *Player) SpendAttributePoints(points int32) bool {
	if p.attributePoints >= points {
		p.attributePoints -= points
		return true
	}
	return false
}

// AllocateAttribute 分配属性点
func (p *Player) AllocateAttribute(attribute string, points int32) bool {
	if !p.SpendAttributePoints(points) {
		return false
	}

	switch attribute {
	case "strength":
		p.strength += points
	case "agility":
		p.agility += points
	case "intelligence":
		p.intelligence += points
	case "stamina":
		p.stamina += points
	case "spirit":
		p.spirit += points
	default:
		p.attributePoints += points // 归还属性点
		return false
	}

	// 更新生命值和魔法值
	p.maxHealth = int32(100 + p.stamina*10)
	p.maxMana = int32(50 + p.spirit*5)

	return true
}

// CalculateAttack 计算攻击力
func (p *Player) CalculateAttack() int32 {
	attack := p.strength*2 + p.equipAttack
	if p.buffManager != nil {
		attack += p.buffManager.CalculateTotalAttackMod(p.playerID)
	}
	return attack
}

func (p *Player) CalculateDefense() int32 {
	defense := p.stamina*1 + p.agility/2 + p.equipDefense
	if p.buffManager != nil {
		defense += p.buffManager.CalculateTotalDefenseMod(p.playerID)
	}
	return defense
}

func (p *Player) CalculateCriticalRate() float32 {
	criticalRate := float32(p.agility)*0.1 + p.equipCritRate
	if criticalRate > 50 {
		criticalRate = 50
	}
	return criticalRate
}

// CalculateHitRate 计算命中率
func (p *Player) CalculateHitRate() float32 {
	// 基础命中率 = 90% + 敏捷 * 0.05%
	hitRate := 90.0 + float32(p.agility)*0.05

	// 可以添加装备和buff 加成

	// 上限 99%
	if hitRate > 99 {
		hitRate = 99
	}

	return hitRate
}

// CalculateAvoidRate 计算闪避率
func (p *Player) CalculateAvoidRate() float32 {
	// 基础闪避率 = 敏捷 * 0.1%
	avoidRate := float32(p.agility) * 0.1

	// 可以添加装备和buff 加成

	// 上限 30%
	if avoidRate > 30 {
		avoidRate = 30
	}

	return avoidRate
}

// CalculateMagicPower 计算魔法强度
func (p *Player) CalculateMagicPower() int32 {
	// 基础魔法强度 = 智力 * 2
	magicPower := p.intelligence * 2

	// 可以添加装备和buff 加成

	return magicPower
}

// CalculateManaRegen 计算魔法回复
func (p *Player) CalculateManaRegen() float32 {
	// 基础魔法回复 = 精神 * 0.5
	manaRegen := float32(p.spirit) * 0.5

	// 可以添加装备和buff 加成

	return manaRegen
}

// CalculateHealthRegen 计算生命回复
func (p *Player) CalculateHealthRegen() float32 {
	// 基础生命回复 = 耐力 * 0.3
	healthRegen := float32(p.stamina) * 0.3

	// 可以添加装备和buff 加成

	return healthRegen
}

// UpdateStats 更新玩家属性
func (p *Player) UpdateStats() {
	p.maxHealth = int32(100+p.stamina*10) + p.equipHP
	p.maxMana = int32(50+p.spirit*5) + p.equipMP

	if p.health > p.maxHealth {
		p.health = p.maxHealth
	}
	if p.mana > p.maxMana {
		p.mana = p.maxMana
	}
}

// ApplyBuffEffects 应用 buff 效果
func (p *Player) ApplyBuffEffects() {
	if p.buffManager == nil {
		return
	}

	attackMod := p.buffManager.CalculateTotalAttackMod(p.playerID)
	defenseMod := p.buffManager.CalculateTotalDefenseMod(p.playerID)
	hpMod := p.buffManager.CalculateTotalHPMod(p.playerID)
	speedMod := p.buffManager.CalculateTotalSpeedMod(p.playerID)

	if attackMod != 0 || defenseMod != 0 || hpMod != 0 || speedMod != 0 {
		zLog.Debug("Applying buff effects",
			zap.Int64("player_id", int64(p.playerID)),
			zap.Int32("attack_mod", attackMod),
			zap.Int32("defense_mod", defenseMod),
			zap.Int32("hp_mod", hpMod),
			zap.Int32("speed_mod", speedMod))
	}

	if hpMod > 0 {
		newHealth := p.health + hpMod
		if newHealth > p.maxHealth {
			newHealth = p.maxHealth
		}
		p.health = newHealth
	}

	dotEffects := p.buffManager.ProcessDotEffects(p.playerID, 0)
	for effectType, value := range dotEffects {
		switch effectType {
		case "poison", "burn":
			newHP := p.health - value
			if newHP < 0 {
				newHP = 0
			}
			p.health = newHP
		case "heal":
			newHP := p.health + value
			if newHP > p.maxHealth {
				newHP = p.maxHealth
			}
			p.health = newHP
		}
	}
}

// SetBuffManager 设置Buff管理器
func (p *Player) SetBuffManager(bm *buff.BuffManager) {
	p.buffManager = bm
}

func (p *Player) SetEquipStats(attack, defense, hp, mp int32, critRate, speed float32) {
	p.equipAttack = attack
	p.equipDefense = defense
	p.equipHP = hp
	p.equipMP = mp
	p.equipCritRate = critRate
	p.equipSpeed = speed
}

func (p *Player) GetEquipAttack() int32 {
	return p.equipAttack
}

func (p *Player) GetEquipDefense() int32 {
	return p.equipDefense
}

func (p *Player) GetEquipHP() int32 {
	return p.equipHP
}

func (p *Player) GetEquipCritRate() float32 {
	return p.equipCritRate
}

// RemoveExpiredBuffs 移除过期�?buff
func (p *Player) RemoveExpiredBuffs() {
	currentTime := time.Now()
	validBuffs := make([]Buff, 0)

	for _, buff := range p.buffs {
		if currentTime.Sub(buff.StartTime) < buff.Duration {
			validBuffs = append(validBuffs, buff)
		}
	}

	p.buffs = validBuffs
}

// RemoveExpiredDebuffs 移除过期�?debuff
func (p *Player) RemoveExpiredDebuffs() {
	currentTime := time.Now()
	validDebuffs := make([]Debuff, 0)

	for _, debuff := range p.debuffs {
		if currentTime.Sub(debuff.StartTime) < debuff.Duration {
			validDebuffs = append(validDebuffs, debuff)
		}
	}

	p.debuffs = validDebuffs
}

// UpdateStatus 更新玩家状态
func (p *Player) UpdateStatus() {
	// 移除过期的buff 和 debuff
	p.RemoveExpiredBuffs()
	p.RemoveExpiredDebuffs()

	// 应用 buff 效果
	p.ApplyBuffEffects()

	// 更新属性
	p.UpdateStats()
}
