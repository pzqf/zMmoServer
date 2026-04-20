package object

import (
	"math/rand"
	"time"

	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zCommon/common/id"
)

// Monster 怪物对象
type Monster struct {
	id              id.ObjectIdType
	monsterID       int32
	name            string
	position        common.Vector3
	level           int32
	health          int32
	maxHealth       int32
	mana            int32
	maxMana         int32
	attack          int32
	defense         int32
	speed           float32
	exp             int64
	lootItems       []int32
	state           MonsterState
	patrolPath      []common.Vector3
	patrolIndex     int
	target          id.ObjectIdType
	targetPosition  common.Vector3
	attackRange     float32
	visionRange     float32
	aggroRange      float32
	lastAttackTime  time.Time
	lastMoveTime    time.Time
	aiType          string
	difficulty      string
	faction         int32
	behaviorPattern string
}

// MonsterState 怪物状态

type MonsterState int

const (
	MonsterStateIdle MonsterState = iota
	MonsterStatePatrolling
	MonsterStateChasing
	MonsterStateAttacking
	MonsterStateDead
)

// NewMonster 创建新怪物
func NewMonster(objectID id.ObjectIdType, monsterID int32, name string, pos common.Vector3, level int32) *Monster {
	maxHealth := 100 + int32(level*20)
	maxMana := 50 + int32(level*10)

	return &Monster{
		id:              objectID,
		monsterID:       monsterID,
		name:            name,
		position:        pos,
		level:           level,
		health:          maxHealth,
		maxHealth:       maxHealth,
		mana:            maxMana,
		maxMana:         maxMana,
		attack:          10 + int32(level*5),
		defense:         5 + int32(level*2),
		speed:           1.0 + float32(level)*0.1,
		exp:             int64(100 * level),
		lootItems:       make([]int32, 0),
		state:           MonsterStateIdle,
		patrolPath:      []common.Vector3{},
		patrolIndex:     0,
		target:          0,
		targetPosition:  common.Vector3{X: pos.X, Y: pos.Y, Z: pos.Z},
		attackRange:     2.0,
		visionRange:     10.0,
		aggroRange:      8.0,
		lastAttackTime:  time.Now(),
		lastMoveTime:    time.Now(),
		aiType:          "normal",
		difficulty:      "normal",
		faction:         0,
		behaviorPattern: "patrol",
	}
}

// GetID 获取对象ID
func (m *Monster) GetID() id.ObjectIdType {
	return m.id
}

// GetType 获取对象类型
func (m *Monster) GetType() common.GameObjectType {
	return common.GameObjectTypeMonster
}

// GetPosition 获取位置
func (m *Monster) GetPosition() common.Vector3 {
	return m.position
}

// SetPosition 设置位置
func (m *Monster) SetPosition(pos common.Vector3) {
	m.position = pos
}

// GetMonsterID 获取怪物ID
func (m *Monster) GetMonsterID() int32 {
	return m.monsterID
}

// GetName 获取怪物名称
func (m *Monster) GetName() string {
	return m.name
}

// GetLevel 获取等级
func (m *Monster) GetLevel() int32 {
	return m.level
}

// GetHealth 获取生命值
func (m *Monster) GetHealth() int32 {
	return m.health
}

// SetHealth 设置生命值
func (m *Monster) SetHealth(health int32) {
	m.health = health
	if m.health <= 0 {
		m.health = 0
		m.state = MonsterStateDead
	}
}

// GetMana 获取魔法值
func (m *Monster) GetMana() int32 {
	return m.mana
}

// SetMana 设置魔法值
func (m *Monster) SetMana(mana int32) {
	m.mana = mana
}

// GetAttack 获取攻击力
func (m *Monster) GetAttack() int32 {
	return m.attack
}

func (m *Monster) GetExpReward() int64 {
	return m.exp
}

func (m *Monster) SetExpReward(exp int64) {
	m.exp = exp
}

// GetDefense 获取防御力
func (m *Monster) GetDefense() int32 {
	return m.defense
}

// GetSpeed 获取移动速度
func (m *Monster) GetSpeed() float32 {
	return m.speed
}

// GetExp 获取经验值
func (m *Monster) GetExp() int64 {
	return m.exp
}

// AddLootItem 添加掉落物品
func (m *Monster) AddLootItem(itemID int32) {
	m.lootItems = append(m.lootItems, itemID)
}

// GetLootItems 获取掉落物品列表
func (m *Monster) GetLootItems() []int32 {
	return m.lootItems
}

// GetState 获取怪物状态
func (m *Monster) GetState() MonsterState {
	return m.state
}

// SetState 设置怪物状态
func (m *Monster) SetState(state MonsterState) {
	m.state = state
}

// SetPatrolPath 设置巡逻路径
func (m *Monster) SetPatrolPath(path []common.Vector3) {
	m.patrolPath = path
	m.patrolIndex = 0
}

// GetNextPatrolPoint 获取下一个巡逻点
func (m *Monster) GetNextPatrolPoint() common.Vector3 {
	if len(m.patrolPath) == 0 {
		return m.position
	}
	point := m.patrolPath[m.patrolIndex]
	m.patrolIndex = (m.patrolIndex + 1) % len(m.patrolPath)
	return point
}

// IsDead 检查怪物是否死亡
func (m *Monster) IsDead() bool {
	return m.state == MonsterStateDead
}

// CanAttack 检查怪物是否可以攻击
func (m *Monster) CanAttack() bool {
	return m.state != MonsterStateDead && m.health > 0
}

// GetMaxHealth 获取最大生命值
func (m *Monster) GetMaxHealth() int32 {
	return m.maxHealth
}

// GetMaxMana 获取最大魔法值
func (m *Monster) GetMaxMana() int32 {
	return m.maxMana
}

// GetTarget 获取目标
func (m *Monster) GetTarget() id.ObjectIdType {
	return m.target
}

// SetTarget 设置目标
func (m *Monster) SetTarget(target id.ObjectIdType) {
	m.target = target
}

// GetAttackRange 获取攻击范围
func (m *Monster) GetAttackRange() float32 {
	return m.attackRange
}

// GetVisionRange 获取视野范围
func (m *Monster) GetVisionRange() float32 {
	return m.visionRange
}

// GetAggroRange 获取仇恨范围
func (m *Monster) GetAggroRange() float32 {
	return m.aggroRange
}

// GetAIType 获取AI类型
func (m *Monster) GetAIType() string {
	return m.aiType
}

// SetAIType 设置AI类型
func (m *Monster) SetAIType(aiType string) {
	m.aiType = aiType
}

// GetDifficulty 获取难度
func (m *Monster) GetDifficulty() string {
	return m.difficulty
}

// SetDifficulty 设置难度
func (m *Monster) SetDifficulty(difficulty string) {
	m.difficulty = difficulty
}

// GetFaction 获取阵营
func (m *Monster) GetFaction() int32 {
	return m.faction
}

// SetFaction 设置阵营
func (m *Monster) SetFaction(faction int32) {
	m.faction = faction
}

// GetBehaviorPattern 获取行为模式
func (m *Monster) GetBehaviorPattern() string {
	return m.behaviorPattern
}

// SetBehaviorPattern 设置行为模式
func (m *Monster) SetBehaviorPattern(pattern string) {
	m.behaviorPattern = pattern
}

// Move 移动怪物
func (m *Monster) Move(targetPos common.Vector3) {
	// 计算移动方向
	dx := targetPos.X - m.position.X
	dy := targetPos.Y - m.position.Y
	dz := targetPos.Z - m.position.Z

	// 计算距离
	distance := float32(dx*dx + dy*dy + dz*dz)
	if distance <= 0.1 {
		// 已经到达目标位置
		m.position = targetPos
		return
	}

	// 计算单位向量
	distSqrt := float32(1.0 / float64(distance))
	dx *= distSqrt
	dy *= distSqrt
	dz *= distSqrt

	// 计算移动距离
	moveDistance := m.speed * 0.1 // 假设每帧移动0.1秒的距离

	// 更新位置
	m.position.X += dx * moveDistance
	m.position.Y += dy * moveDistance
	m.position.Z += dz * moveDistance

	m.lastMoveTime = time.Now()
}

// Attack 攻击目标
func (m *Monster) Attack(target id.ObjectIdType) bool {
	if !m.CanAttack() {
		return false
	}

	// 检查攻击冷却
	if time.Since(m.lastAttackTime) < time.Second {
		return false
	}

	// 执行攻击逻辑
	m.state = MonsterStateAttacking
	m.lastAttackTime = time.Now()

	return true
}

// Patrol 巡逻
func (m *Monster) Patrol() common.Vector3 {
	if len(m.patrolPath) == 0 {
		return m.position
	}

	// 获取下一个巡逻点
	nextPoint := m.GetNextPatrolPoint()
	return nextPoint
}

// Chase 追击目标
func (m *Monster) Chase(targetPos common.Vector3) {
	m.state = MonsterStateChasing
	m.targetPosition = targetPos
	m.Move(targetPos)
}

// UpdateAI 更新AI状态
func (m *Monster) UpdateAI() {
	if m.IsDead() {
		return
	}

	switch m.behaviorPattern {
	case "patrol":
		m.updatePatrolBehavior()
	case "guard":
		m.updateGuardBehavior()
	case "wander":
		m.updateWanderBehavior()
	case "aggressive":
		m.updateAggressiveBehavior()
	}
}

// updatePatrolBehavior 更新巡逻行为
func (m *Monster) updatePatrolBehavior() {
	switch m.state {
	case MonsterStateIdle:
		// 开始巡逻
		m.state = MonsterStatePatrolling
	case MonsterStatePatrolling:
		// 移动到下一个巡逻点
		nextPoint := m.Patrol()
		m.Move(nextPoint)

		// 检查是否到达巡逻点
		if m.position.DistanceTo(nextPoint) <= 1.0 {
			// 到达巡逻点，短暂停留
			m.state = MonsterStateIdle
		}
	case MonsterStateChasing:
		// 继续追击目标
		m.Move(m.targetPosition)
	case MonsterStateAttacking:
		// 攻击目标
		m.Attack(m.target)
	}
}

// updateGuardBehavior 更新守卫行为
func (m *Monster) updateGuardBehavior() {
	// 守卫行为：在固定区域巡逻，对进入视野的目标发起攻击
	switch m.state {
	case MonsterStateIdle:
		// 保持静止
	case MonsterStateChasing:
		// 追击目标
		m.Move(m.targetPosition)
	case MonsterStateAttacking:
		// 攻击目标
		m.Attack(m.target)
	}
}

// updateWanderBehavior 更新漫游行为
func (m *Monster) updateWanderBehavior() {
	// 漫游行为：随机移动
	switch m.state {
	case MonsterStateIdle:
		// 生成随机目标位置
		randomX := m.position.X + (rand.Float32()*20 - 10)
		randomY := m.position.Y + (rand.Float32()*20 - 10)
		m.targetPosition = common.Vector3{X: randomX, Y: randomY, Z: m.position.Z}
		m.state = MonsterStatePatrolling
	case MonsterStatePatrolling:
		// 移动到随机目标位置
		m.Move(m.targetPosition)

		// 检查是否到达目标位置
		if m.position.DistanceTo(m.targetPosition) <= 1.0 {
			// 到达目标位置，短暂停留
			m.state = MonsterStateIdle
		}
	case MonsterStateChasing:
		// 追击目标
		m.Move(m.targetPosition)
	case MonsterStateAttacking:
		// 攻击目标
		m.Attack(m.target)
	}
}

// updateAggressiveBehavior 更新主动攻击行为
func (m *Monster) updateAggressiveBehavior() {
	// 主动攻击行为：主动寻找并攻击目标
	switch m.state {
	case MonsterStateIdle:
		// 寻找目标
		m.state = MonsterStatePatrolling
	case MonsterStatePatrolling:
		// 移动寻找目标
		m.Move(m.targetPosition)
	case MonsterStateChasing:
		// 追击目标
		m.Move(m.targetPosition)
	case MonsterStateAttacking:
		// 攻击目标
		m.Attack(m.target)
	}
}

// GenerateLoot 生成掉落物品
func (m *Monster) GenerateLoot() []int32 {
	// 根据怪物等级和难度生成掉落物品
	loot := make([]int32, 0)

	// 基础掉落
	baseItems := []int32{1001, 1002, 1003} // 假设这些是基础物品ID

	// 根据难度调整掉落数量
	lootCount := 1
	switch m.difficulty {
	case "easy":
		lootCount = 1
	case "normal":
		lootCount = 2
	case "hard":
		lootCount = 3
	case "elite":
		lootCount = 4
	case "boss":
		lootCount = 5
	}

	// 随机选择掉落物品
	for i := 0; i < lootCount; i++ {
		if len(baseItems) > 0 {
			index := rand.Intn(len(baseItems))
			loot = append(loot, baseItems[index])
		}
	}

	// 根据怪物等级添加特殊掉落
	if m.level >= 10 {
		specialItems := []int32{2001, 2002}                // 假设这些是特殊物品ID
		if len(specialItems) > 0 && rand.Float32() < 0.3 { // 30% 几率掉落特殊物品
			index := rand.Intn(len(specialItems))
			loot = append(loot, specialItems[index])
		}
	}

	return loot
}

// DropLoot 掉落物品
func (m *Monster) DropLoot() []int32 {
	if !m.IsDead() {
		return nil
	}

	// 生成掉落物品
	loot := m.GenerateLoot()
	m.lootItems = loot
	return loot
}

// CalculateDamage 计算伤害
func (m *Monster) CalculateDamage(targetDefense int32) int32 {
	// 基础伤害 = 攻击 - 目标防御
	damage := m.attack - targetDefense
	if damage < 1 {
		damage = 1 // 最低伤害为1
	}

	// 根据难度调整伤害
	switch m.difficulty {
	case "easy":
		damage = int32(float32(damage) * 0.8)
	case "normal":
		// 正常伤害
	case "hard":
		damage = int32(float32(damage) * 1.2)
	case "elite":
		damage = int32(float32(damage) * 1.5)
	case "boss":
		damage = int32(float32(damage) * 2.0)
	}

	return damage
}

// TakeDamage 受到伤害
func (m *Monster) TakeDamage(damage int32) {
	m.health -= damage
	if m.health <= 0 {
		m.health = 0
		m.state = MonsterStateDead
	}
}

// Reset 重置怪物状态
func (m *Monster) Reset() {
	m.health = m.maxHealth
	m.mana = m.maxMana
	m.state = MonsterStateIdle
	m.target = 0
	m.targetPosition = m.position
	m.lastAttackTime = time.Now()
	m.lastMoveTime = time.Now()
}

// SetVisionRange 设置视野范围
func (m *Monster) SetVisionRange(range_ float32) {
	m.visionRange = range_
}

// SetAttackRange 设置攻击范围
func (m *Monster) SetAttackRange(range_ float32) {
	m.attackRange = range_
}

// SetAggroRange 设置仇恨范围
func (m *Monster) SetAggroRange(range_ float32) {
	m.aggroRange = range_
}

// GetTargetPosition 获取目标位置
func (m *Monster) GetTargetPosition() common.Vector3 {
	return m.targetPosition
}

// SetTargetPosition 设置目标位置
func (m *Monster) SetTargetPosition(pos common.Vector3) {
	m.targetPosition = pos
}

// GetPatrolPath 获取巡逻路径
func (m *Monster) GetPatrolPath() []common.Vector3 {
	return m.patrolPath
}

