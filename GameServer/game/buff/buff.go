package buff

import (
	"time"

	"github.com/pzqf/zCommon/common/id"
)

// BuffType Buff类型常量
const (
	BuffTypeNone    = 0 // 无
	BuffTypeBuff    = 1 // 增益
	BuffTypeDebuff  = 2 // 减益
	BuffTypeControl = 3 // 控制
	BuffTypeDot     = 4 // 持续伤害
	BuffTypeHot     = 5 // 持续治疗
)

// BuffEffectType Buff效果类型
const (
	BuffEffectNone       = 0  // 无
	BuffEffectAttack     = 1  // 攻击力
	BuffEffectDefense    = 2  // 防御力
	BuffEffectSpeed      = 3  // 速度
	BuffEffectHP         = 4  // 生命值
	BuffEffectMP         = 5  // 魔法值
	BuffEffectCritRate   = 6  // 暴击率
	BuffEffectCritDamage = 7  // 暴击伤害
	BuffEffectDodge      = 8  // 闪避率
	BuffEffectHit        = 9  // 命中率
	BuffEffectStun       = 10 // 眩晕
	BuffEffectSilence    = 11 // 沉默
	BuffEffectRoot       = 12 // 定身
	BuffEffectDot        = 13 // 持续伤害
	BuffEffectHot        = 14 // 持续治疗
)

// Buff Buff结构
type Buff struct {
	BuffID        id.BuffIdType // Buff ID
	Name          string        // Buff名称
	Description   string        // Buff描述
	Icon          string        // Buff图标
	Type          int           // Buff类型
	EffectType    int           // 效果类型
	Value         int64         // 效果数值
	Percent       float64       // 百分比效果
	Duration      int           // 持续时间（秒）
	Interval      int           // 触发间隔（秒，用于Dot/Hot）
	MaxStack      int           // 最大堆叠层数
	IsDispellable bool          // 是否可驱散
	IsDebuff      bool          // 是否是Debuff
}

// BuffInstance Buff实例
type BuffInstance struct {
	InstanceID   id.BuffInstanceIdType // 实例ID
	BuffID       id.BuffIdType         // Buff ID
	CasterID     id.ObjectIdType       // 施法者ID
	TargetID     id.ObjectIdType       // 目标ID
	StartTime    time.Time             // 开始时�?
	EndTime      time.Time             // 结束时间
	StackCount   int                   // 当前层数
	LastTickTime time.Time             // 上次触发时间
	IsActive     bool                  // 是否激�?
}

// NewBuffInstance 创建Buff实例
func NewBuffInstance(instanceID id.BuffInstanceIdType, buff *Buff, casterID, targetID id.ObjectIdType) *BuffInstance {
	now := time.Now()
	return &BuffInstance{
		InstanceID:   instanceID,
		BuffID:       buff.BuffID,
		CasterID:     casterID,
		TargetID:     targetID,
		StartTime:    now,
		EndTime:      now.Add(time.Duration(buff.Duration) * time.Second),
		StackCount:   1,
		LastTickTime: now,
		IsActive:     true,
	}
}

// IsExpired 检查是否过�?
func (bi *BuffInstance) IsExpired() bool {
	return time.Now().After(bi.EndTime)
}

// ShouldTick 检查是否应该触发效�?
func (bi *BuffInstance) ShouldTick(interval int) bool {
	if interval <= 0 {
		return false
	}
	return time.Since(bi.LastTickTime) >= time.Duration(interval)*time.Second
}

// Tick 触发效果
func (bi *BuffInstance) Tick() {
	bi.LastTickTime = time.Now()
}

// AddStack 添加层数
func (bi *BuffInstance) AddStack(maxStack int) bool {
	if bi.StackCount >= maxStack {
		return false
	}
	bi.StackCount++
	return true
}

// RemoveStack 减少层数
func (bi *BuffInstance) RemoveStack() bool {
	if bi.StackCount <= 1 {
		return false
	}
	bi.StackCount--
	return true
}

// Refresh 刷新持续时间
func (bi *BuffInstance) Refresh(duration int) {
	bi.EndTime = time.Now().Add(time.Duration(duration) * time.Second)
}

// CalculateValue 计算实际效果�?
func (bi *BuffInstance) CalculateValue(baseValue int64, buff *Buff) int64 {
	if buff.Percent > 0 {
		return int64(float64(baseValue) * buff.Percent * float64(bi.StackCount))
	}
	return buff.Value * int64(bi.StackCount)
}

