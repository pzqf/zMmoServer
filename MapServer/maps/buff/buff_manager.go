package buff

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/config/models"
	"github.com/pzqf/zMmoShared/config/tables"
	"go.uber.org/zap"
)

type BuffType int

const (
	BuffTypeNone BuffType = iota
	BuffTypeBuff
	BuffTypeDebuff
	BuffTypeNeutral
)

type BuffEffect int

const (
	BuffEffectNone BuffEffect = iota
	BuffEffectHP
	BuffEffectMP
	BuffEffectAttack
	BuffEffectDefense
	BuffEffectSpeed
	BuffEffectCritical
	BuffEffectDodge
	BuffEffectStun
	BuffEffectSilence
	BuffEffectPoison
	BuffEffectBurn
	BuffEffectFreeze
	BuffEffectSlow
	BuffEffectShield
	BuffEffectInvisible
	BuffEffectInvincible
)

type BuffManager struct {
	mu           sync.RWMutex
	tableManager *tables.TableManager
	playerBuffs  map[id.PlayerIdType]map[int32]*ActiveBuff
}

var globalBuffManager *BuffManager
var buffOnce sync.Once

func NewBuffManager() *BuffManager {
	return &BuffManager{
		playerBuffs: make(map[id.PlayerIdType]map[int32]*ActiveBuff),
	}
}

func GetBuffManager() *BuffManager {
	if globalBuffManager == nil {
		buffOnce.Do(func() {
			globalBuffManager = NewBuffManager()
		})
	}
	return globalBuffManager
}

func (bm *BuffManager) SetTableManager(tm *tables.TableManager) {
	bm.tableManager = tm
}

type ActiveBuff struct {
	BuffID       int32
	Config       *models.Buff
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	StackCount   int
	SourceID     id.ObjectIdType
	CurrentValue int32
	IsActive     bool
}

func (bm *BuffManager) AddBuff(playerID id.PlayerIdType, buffID int32, sourceID id.ObjectIdType) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.tableManager == nil {
		return fmt.Errorf("table manager not initialized")
	}

	buffConfig, ok := bm.tableManager.GetBuffLoader().GetBuff(buffID)
	if !ok {
		return fmt.Errorf("buff config not found: %d", buffID)
	}

	if _, exists := bm.playerBuffs[playerID]; !exists {
		bm.playerBuffs[playerID] = make(map[int32]*ActiveBuff)
	}

	if existingBuff, exists := bm.playerBuffs[playerID][buffID]; exists {
		existingBuff.StackCount++
		if !buffConfig.IsPermanent {
			existingBuff.EndTime = time.Now().Add(time.Duration(buffConfig.Duration) * time.Second)
		}
		zLog.Debug("Buff stacked",
			zap.Int64("player_id", int64(playerID)),
			zap.Int32("buff_id", buffID),
			zap.Int("stack_count", existingBuff.StackCount))
		return nil
	}

	duration := time.Duration(buffConfig.Duration) * time.Second
	if buffConfig.IsPermanent {
		duration = time.Duration(0)
	}

	activeBuff := &ActiveBuff{
		BuffID:     buffID,
		Config:     buffConfig,
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(duration),
		Duration:   duration,
		StackCount: 1,
		SourceID:   sourceID,
		IsActive:   true,
	}

	bm.playerBuffs[playerID][buffID] = activeBuff

	zLog.Debug("Buff added",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("buff_id", buffID),
		zap.String("name", buffConfig.Name),
		zap.Int32("duration", buffConfig.Duration))

	return nil
}

func (bm *BuffManager) RemoveBuff(playerID id.PlayerIdType, buffID int32) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, exists := bm.playerBuffs[playerID]; !exists {
		return fmt.Errorf("player has no buffs: %d", playerID)
	}

	if _, exists := bm.playerBuffs[playerID][buffID]; !exists {
		return fmt.Errorf("buff not found: %d", buffID)
	}

	delete(bm.playerBuffs[playerID], buffID)

	zLog.Debug("Buff removed",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("buff_id", buffID))

	return nil
}

func (bm *BuffManager) RemoveAllBuffs(playerID id.PlayerIdType) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	delete(bm.playerBuffs, playerID)

	zLog.Debug("All buffs removed", zap.Int64("player_id", int64(playerID)))
}

func (bm *BuffManager) GetActiveBuffs(playerID id.PlayerIdType) []*ActiveBuff {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	if _, exists := bm.playerBuffs[playerID]; !exists {
		return []*ActiveBuff{}
	}

	buffs := make([]*ActiveBuff, 0, len(bm.playerBuffs[playerID]))
	for _, buff := range bm.playerBuffs[playerID] {
		if buff.IsActive {
			buffs = append(buffs, buff)
		}
	}

	return buffs
}

func (bm *BuffManager) HasBuff(playerID id.PlayerIdType, buffID int32) bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	if _, exists := bm.playerBuffs[playerID]; !exists {
		return false
	}

	buff, exists := bm.playerBuffs[playerID][buffID]
	return exists && buff.IsActive
}

func (bm *BuffManager) GetBuffStackCount(playerID id.PlayerIdType, buffID int32) int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	if _, exists := bm.playerBuffs[playerID]; !exists {
		return 0
	}

	if buff, exists := bm.playerBuffs[playerID][buffID]; exists {
		return buff.StackCount
	}

	return 0
}

func (bm *BuffManager) Update(deltaTime time.Duration) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	now := time.Now()

	for playerID, buffs := range bm.playerBuffs {
		for buffID, buff := range buffs {
			if buff.Config.IsPermanent {
				continue
			}

			if now.After(buff.EndTime) {
				buff.IsActive = false
				delete(bm.playerBuffs[playerID], buffID)

				zLog.Debug("Buff expired",
					zap.Int64("player_id", int64(playerID)),
					zap.Int32("buff_id", buffID))
			}
		}
	}
}

func (bm *BuffManager) CalculateBuffEffect(playerID id.PlayerIdType, property string) int32 {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	if _, exists := bm.playerBuffs[playerID]; !exists {
		return 0
	}

	totalValue := int32(0)

	for _, buff := range bm.playerBuffs[playerID] {
		if !buff.IsActive {
			continue
		}

		if buff.Config.Property == property {
			value := buff.Config.Value * int32(buff.StackCount)
			if buff.Config.Type == "减益" {
				value = -value
			}
			totalValue += value
		}
	}

	return totalValue
}

func (bm *BuffManager) CalculateTotalHPMod(playerID id.PlayerIdType) int32 {
	return bm.CalculateBuffEffect(playerID, "hp")
}

func (bm *BuffManager) CalculateTotalMPMod(playerID id.PlayerIdType) int32 {
	return bm.CalculateBuffEffect(playerID, "mp")
}

func (bm *BuffManager) CalculateTotalAttackMod(playerID id.PlayerIdType) int32 {
	return bm.CalculateBuffEffect(playerID, "attack")
}

func (bm *BuffManager) CalculateTotalDefenseMod(playerID id.PlayerIdType) int32 {
	return bm.CalculateBuffEffect(playerID, "defense")
}

func (bm *BuffManager) CalculateTotalSpeedMod(playerID id.PlayerIdType) int32 {
	return bm.CalculateBuffEffect(playerID, "speed")
}

func (bm *BuffManager) IsStunned(playerID id.PlayerIdType) bool {
	return bm.HasBuffType(playerID, "stun")
}

func (bm *BuffManager) IsSilenced(playerID id.PlayerIdType) bool {
	return bm.HasBuffType(playerID, "silence")
}

func (bm *BuffManager) IsPoisoned(playerID id.PlayerIdType) bool {
	return bm.HasBuffType(playerID, "poison")
}

func (bm *BuffManager) IsBurning(playerID id.PlayerIdType) bool {
	return bm.HasBuffType(playerID, "burn")
}

func (bm *BuffManager) IsFrozen(playerID id.PlayerIdType) bool {
	return bm.HasBuffType(playerID, "freeze")
}

func (bm *BuffManager) IsSlowed(playerID id.PlayerIdType) bool {
	return bm.HasBuffType(playerID, "slow")
}

func (bm *BuffManager) IsShielded(playerID id.PlayerIdType) bool {
	return bm.HasBuffType(playerID, "shield")
}

func (bm *BuffManager) IsInvisible(playerID id.PlayerIdType) bool {
	return bm.HasBuffType(playerID, "invisible")
}

func (bm *BuffManager) IsInvincible(playerID id.PlayerIdType) bool {
	return bm.HasBuffType(playerID, "invincible")
}

func (bm *BuffManager) HasBuffType(playerID id.PlayerIdType, buffType string) bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	if _, exists := bm.playerBuffs[playerID]; !exists {
		return false
	}

	for _, buff := range bm.playerBuffs[playerID] {
		if buff.IsActive && buff.Config.Property == buffType {
			return true
		}
	}

	return false
}

func (bm *BuffManager) ProcessDotEffects(playerID id.PlayerIdType, deltaTime time.Duration) map[string]int32 {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	effects := make(map[string]int32)

	if _, exists := bm.playerBuffs[playerID]; !exists {
		return effects
	}

	for _, buff := range bm.playerBuffs[playerID] {
		if !buff.IsActive {
			continue
		}

		switch buff.Config.Property {
		case "poison", "burn":
			damage := buff.Config.Value * int32(buff.StackCount)
			effects[buff.Config.Property] += damage
		case "heal":
			heal := buff.Config.Value * int32(buff.StackCount)
			effects["heal"] += heal
		}
	}

	return effects
}

func (bm *BuffManager) DispelBuffsByType(playerID id.PlayerIdType, buffType string, count int) int {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, exists := bm.playerBuffs[playerID]; !exists {
		return 0
	}

	dispelled := 0
	toRemove := make([]int32, 0)

	for buffID, buff := range bm.playerBuffs[playerID] {
		if buff.Config.Type == buffType {
			toRemove = append(toRemove, buffID)
			dispelled++
			if dispelled >= count {
				break
			}
		}
	}

	for _, buffID := range toRemove {
		delete(bm.playerBuffs[playerID], buffID)
	}

	if dispelled > 0 {
		zLog.Debug("Buffs dispelled",
			zap.Int64("player_id", int64(playerID)),
			zap.String("type", buffType),
			zap.Int("count", dispelled))
	}

	return dispelled
}

func (bm *BuffManager) CleanseDebuffs(playerID id.PlayerIdType, count int) int {
	return bm.DispelBuffsByType(playerID, "减益", count)
}

func (bm *BuffManager) PurgeBuffs(playerID id.PlayerIdType, count int) int {
	return bm.DispelBuffsByType(playerID, "增益", count)
}

func (ab *ActiveBuff) GetRemainingTime() time.Duration {
	if ab.Config.IsPermanent {
		return time.Duration(0)
	}

	remaining := time.Until(ab.EndTime)
	if remaining < 0 {
		return 0
	}

	return remaining
}

func (ab *ActiveBuff) GetRemainingSeconds() int32 {
	return int32(ab.GetRemainingTime().Seconds())
}

func (ab *ActiveBuff) GetTotalValue() int32 {
	return ab.Config.Value * int32(ab.StackCount)
}

func (ab *ActiveBuff) IsExpired() bool {
	if ab.Config.IsPermanent {
		return false
	}
	return time.Now().After(ab.EndTime)
}

func (ab *ActiveBuff) Refresh() {
	if ab.Config.IsPermanent {
		return
	}
	ab.EndTime = time.Now().Add(ab.Duration)
	ab.StartTime = time.Now()
}
