package object

import (
	"sync"

	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoShared/common/id"
)

// LivingObject 生物对象（玩家、NPC、怪物等的基类）
type LivingObject struct {
	*GameObject
	mu         sync.RWMutex
	hp         int32
	maxHp      int32
	mp         int32
	maxMp      int32
	level      int32
	exp        int64
	attributes map[string]int32
}

// NewLivingObject 创建新的生物对象
func NewLivingObject(objectID id.ObjectIdType, name string, objectType common.GameObjectType) *LivingObject {
	return &LivingObject{
		GameObject:  NewGameObjectWithType(objectID, name, objectType),
		hp:          100,
		maxHp:       100,
		mp:          100,
		maxMp:       100,
		level:       1,
		exp:         0,
		attributes:  make(map[string]int32),
	}
}

// GetHP 获取当前生命值
func (lo *LivingObject) GetHP() int32 {
	lo.mu.RLock()
	defer lo.mu.RUnlock()
	return lo.hp
}

// SetHP 设置当前生命值
func (lo *LivingObject) SetHP(hp int32) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	if hp > lo.maxHp {
		lo.hp = lo.maxHp
	} else if hp < 0 {
		lo.hp = 0
	} else {
		lo.hp = hp
	}
}

// GetMaxHP 获取最大生命值
func (lo *LivingObject) GetMaxHP() int32 {
	lo.mu.RLock()
	defer lo.mu.RUnlock()
	return lo.maxHp
}

// SetMaxHP 设置最大生命值
func (lo *LivingObject) SetMaxHP(maxHp int32) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	lo.maxHp = maxHp
	if lo.hp > lo.maxHp {
		lo.hp = lo.maxHp
	}
}

// GetMP 获取当前魔法值
func (lo *LivingObject) GetMP() int32 {
	lo.mu.RLock()
	defer lo.mu.RUnlock()
	return lo.mp
}

// SetMP 设置当前魔法值
func (lo *LivingObject) SetMP(mp int32) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	if mp > lo.maxMp {
		lo.mp = lo.maxMp
	} else if mp < 0 {
		lo.mp = 0
	} else {
		lo.mp = mp
	}
}

// GetMaxMP 获取最大魔法值
func (lo *LivingObject) GetMaxMP() int32 {
	lo.mu.RLock()
	defer lo.mu.RUnlock()
	return lo.maxMp
}

// SetMaxMP 设置最大魔法值
func (lo *LivingObject) SetMaxMP(maxMp int32) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	lo.maxMp = maxMp
	if lo.mp > lo.maxMp {
		lo.mp = lo.maxMp
	}
}

// GetLevel 获取等级
func (lo *LivingObject) GetLevel() int32 {
	lo.mu.RLock()
	defer lo.mu.RUnlock()
	return lo.level
}

// SetLevel 设置等级
func (lo *LivingObject) SetLevel(level int32) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	lo.level = level
}

// GetExp 获取经验值
func (lo *LivingObject) GetExp() int64 {
	lo.mu.RLock()
	defer lo.mu.RUnlock()
	return lo.exp
}

// SetExp 设置经验值
func (lo *LivingObject) SetExp(exp int64) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	lo.exp = exp
}

// AddExp 增加经验值
func (lo *LivingObject) AddExp(exp int64) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	lo.exp += exp
}

// GetAttribute 获取属性值
func (lo *LivingObject) GetAttribute(name string) int32 {
	lo.mu.RLock()
	defer lo.mu.RUnlock()
	return lo.attributes[name]
}

// SetAttribute 设置属性值
func (lo *LivingObject) SetAttribute(name string, value int32) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	lo.attributes[name] = value
}

// GetAttributes 获取所有属性
func (lo *LivingObject) GetAttributes() map[string]int32 {
	lo.mu.RLock()
	defer lo.mu.RUnlock()
	result := make(map[string]int32)
	for k, v := range lo.attributes {
		result[k] = v
	}
	return result
}

// IsAlive 检查是否存活
func (lo *LivingObject) IsAlive() bool {
	return lo.GetHP() > 0
}

// TakeDamage 受到伤害
func (lo *LivingObject) TakeDamage(damage int32) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	lo.hp -= damage
	if lo.hp < 0 {
		lo.hp = 0
	}
}

// Heal 恢复生命值
func (lo *LivingObject) Heal(amount int32) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	lo.hp += amount
	if lo.hp > lo.maxHp {
		lo.hp = lo.maxHp
	}
}

// RestoreMP 恢复魔法值
func (lo *LivingObject) RestoreMP(amount int32) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	lo.mp += amount
	if lo.mp > lo.maxMp {
		lo.mp = lo.maxMp
	}
}

// ConsumeMP 消耗魔法值
func (lo *LivingObject) ConsumeMP(amount int32) bool {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	if lo.mp < amount {
		return false
	}
	lo.mp -= amount
	return true
}
