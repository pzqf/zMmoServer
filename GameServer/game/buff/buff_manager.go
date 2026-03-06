package buff

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// BuffManager Buff管理器
type BuffManager struct {
	buffs         map[id.BuffIdType]*Buff
	instances     map[id.BuffInstanceIdType]*BuffInstance
	targetBuffs   map[id.ObjectIdType]map[id.BuffIdType]*BuffInstance // 目标 -> Buff列表
	mutex         sync.RWMutex
	instanceCounter int64
}

// NewBuffManager 创建Buff管理器
func NewBuffManager() *BuffManager {
	return &BuffManager{
		buffs:         make(map[id.BuffIdType]*Buff),
		instances:     make(map[id.BuffInstanceIdType]*BuffInstance),
		targetBuffs:   make(map[id.ObjectIdType]map[id.BuffIdType]*BuffInstance),
		instanceCounter: 0,
	}
}

// AddBuff 添加Buff定义
func (bm *BuffManager) AddBuff(buff *Buff) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.buffs[buff.BuffID] = buff
	zLog.Info("Buff added",
		zap.Uint64("buff_id", uint64(buff.BuffID)),
		zap.String("name", buff.Name),
		zap.Int("type", buff.Type))
}

// GetBuff 获取Buff定义
func (bm *BuffManager) GetBuff(buffID id.BuffIdType) *Buff {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	return bm.buffs[buffID]
}

// ApplyBuff 施加Buff
func (bm *BuffManager) ApplyBuff(buffID id.BuffIdType, casterID, targetID id.ObjectIdType) *BuffInstance {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	buff := bm.buffs[buffID]
	if buff == nil {
		return nil
	}

	// 检查目标是否已有相同Buff
	if targetBuffs, exists := bm.targetBuffs[targetID]; exists {
		if existingInstance, exists := targetBuffs[buffID]; exists && existingInstance.IsActive {
			// 可以堆叠
			if existingInstance.AddStack(buff.MaxStack) {
				existingInstance.Refresh(buff.Duration)
				zLog.Info("Buff stack added",
					zap.Uint64("target_id", uint64(targetID)),
					zap.Uint64("buff_id", uint64(buffID)),
					zap.Int("stack", existingInstance.StackCount))
				return existingInstance
			}
			// 无法堆叠，刷新时间
			existingInstance.Refresh(buff.Duration)
			return existingInstance
		}
	}

	// 创建新实例
	bm.instanceCounter++
	instanceID := id.BuffInstanceIdType(bm.instanceCounter)
	instance := NewBuffInstance(instanceID, buff, casterID, targetID)

	bm.instances[instanceID] = instance

	// 添加到目标的Buff列表
	if _, exists := bm.targetBuffs[targetID]; !exists {
		bm.targetBuffs[targetID] = make(map[id.BuffIdType]*BuffInstance)
	}
	bm.targetBuffs[targetID][buffID] = instance

	zLog.Info("Buff applied",
		zap.Uint64("target_id", uint64(targetID)),
		zap.Uint64("caster_id", uint64(casterID)),
		zap.Uint64("buff_id", uint64(buffID)),
		zap.String("buff_name", buff.Name))

	return instance
}

// RemoveBuff 移除Buff
func (bm *BuffManager) RemoveBuff(instanceID id.BuffInstanceIdType) bool {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	instance := bm.instances[instanceID]
	if instance == nil {
		return false
	}

	instance.IsActive = false
	delete(bm.instances, instanceID)

	// 从目标的Buff列表中移除
	if targetBuffs, exists := bm.targetBuffs[instance.TargetID]; exists {
		delete(targetBuffs, instance.BuffID)
		if len(targetBuffs) == 0 {
			delete(bm.targetBuffs, instance.TargetID)
		}
	}

	zLog.Info("Buff removed",
		zap.Uint64("instance_id", uint64(instanceID)),
		zap.Uint64("target_id", uint64(instance.TargetID)),
		zap.Uint64("buff_id", uint64(instance.BuffID)))

	return true
}

// RemoveBuffByID 根据BuffID移除
func (bm *BuffManager) RemoveBuffByID(targetID id.ObjectIdType, buffID id.BuffIdType) bool {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	if targetBuffs, exists := bm.targetBuffs[targetID]; exists {
		if instance, exists := targetBuffs[buffID]; exists {
			instance.IsActive = false
			delete(bm.instances, instance.InstanceID)
			delete(targetBuffs, buffID)
			return true
		}
	}

	return false
}

// GetTargetBuffs 获取目标的所有Buff
func (bm *BuffManager) GetTargetBuffs(targetID id.ObjectIdType) []*BuffInstance {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	instances := make([]*BuffInstance, 0)
	if targetBuffs, exists := bm.targetBuffs[targetID]; exists {
		for _, instance := range targetBuffs {
			if instance.IsActive && !instance.IsExpired() {
				instances = append(instances, instance)
			}
		}
	}

	return instances
}

// HasBuff 检查目标是否有指定Buff
func (bm *BuffManager) HasBuff(targetID id.ObjectIdType, buffID id.BuffIdType) bool {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	if targetBuffs, exists := bm.targetBuffs[targetID]; exists {
		if instance, exists := targetBuffs[buffID]; exists {
			return instance.IsActive && !instance.IsExpired()
		}
	}

	return false
}

// DispelBuffs 驱散Buff
func (bm *BuffManager) DispelBuffs(targetID id.ObjectIdType, count int, dispelDebuff bool) int {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	dispelled := 0
	if targetBuffs, exists := bm.targetBuffs[targetID]; exists {
		for buffID, instance := range targetBuffs {
			if dispelled >= count {
				break
			}

			buff := bm.buffs[buffID]
			if buff == nil || !buff.IsDispellable {
				continue
			}

			// 驱散Debuff或Buff
			if dispelDebuff && buff.IsDebuff {
				instance.IsActive = false
				delete(bm.instances, instance.InstanceID)
				delete(targetBuffs, buffID)
				dispelled++
			} else if !dispelDebuff && !buff.IsDebuff {
				instance.IsActive = false
				delete(bm.instances, instance.InstanceID)
				delete(targetBuffs, buffID)
				dispelled++
			}
		}
	}

	return dispelled
}

// Update 更新所有Buff
func (bm *BuffManager) Update() {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	now := time.Now()
	for instanceID, instance := range bm.instances {
		if !instance.IsActive {
			continue
		}

		// 检查过期
		if now.After(instance.EndTime) {
			instance.IsActive = false
			delete(bm.instances, instanceID)

			if targetBuffs, exists := bm.targetBuffs[instance.TargetID]; exists {
				delete(targetBuffs, instance.BuffID)
			}

			zLog.Debug("Buff expired",
				zap.Uint64("instance_id", uint64(instanceID)),
				zap.Uint64("buff_id", uint64(instance.BuffID)))
		}
	}
}

// GetBuffEffects 获取Buff效果总和
func (bm *BuffManager) GetBuffEffects(targetID id.ObjectIdType, effectType int) int64 {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	var total int64 = 0
	if targetBuffs, exists := bm.targetBuffs[targetID]; exists {
		for buffID, instance := range targetBuffs {
			if !instance.IsActive || instance.IsExpired() {
				continue
			}

			buff := bm.buffs[buffID]
			if buff != nil && buff.EffectType == effectType {
				total += buff.Value * int64(instance.StackCount)
			}
		}
	}

	return total
}

// ClearAllBuffs 清除目标所有Buff
func (bm *BuffManager) ClearAllBuffs(targetID id.ObjectIdType) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	if targetBuffs, exists := bm.targetBuffs[targetID]; exists {
		for buffID, instance := range targetBuffs {
			instance.IsActive = false
			delete(bm.instances, instance.InstanceID)
			delete(targetBuffs, buffID)
		}
		delete(bm.targetBuffs, targetID)
	}

	zLog.Info("All buffs cleared",
		zap.Uint64("target_id", uint64(targetID)))
}
