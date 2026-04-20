package game

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zUtil/zMap"
)

type BuffType int

const (
	BuffTypePositive BuffType = 1
	BuffTypeNegative BuffType = 2
	BuffTypeNeutral  BuffType = 3
)

type Buff struct {
	mu          sync.RWMutex
	ID          int32
	Name        string
	Type        BuffType
	Duration    time.Duration
	Value       float32
	Property    string
	IsPermanent bool
	StartTime   time.Time
	EndTime     time.Time
}

func NewBuff(id int32, name string, buffType BuffType, duration time.Duration) *Buff {
	now := time.Now()
	return &Buff{
		ID:        id,
		Name:      name,
		Type:      buffType,
		Duration:  duration,
		StartTime: now,
		EndTime:   now.Add(duration),
	}
}

func (b *Buff) IsExpired() bool {
	if b.IsPermanent {
		return false
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return time.Now().After(b.EndTime)
}

func (b *Buff) GetRemaining() time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()
	remaining := time.Until(b.EndTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (b *Buff) Refresh() {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.StartTime = now
	b.EndTime = now.Add(b.Duration)
}

type BuffManager struct {
	BaseComponent
	mu    sync.RWMutex
	buffMap *zMap.TypedMap[int32, *Buff]
}

func NewBuffManager() *BuffManager {
	return &BuffManager{
		BaseComponent: NewBaseComponent("buffs"),
		buffMap:       zMap.NewTypedMap[int32, *Buff](),
	}
}

func (bm *BuffManager) AddBuff(buff *Buff) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if existing, ok := bm.buffMap.Load(buff.ID); ok {
		existing.Refresh()
		return
	}

	bm.buffMap.Store(buff.ID, buff)
}

func (bm *BuffManager) RemoveBuff(buffID int32) (*Buff, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	buff, ok := bm.buffMap.Load(buffID)
	if !ok {
		return nil, fmt.Errorf("buff %d not found", buffID)
	}

	bm.buffMap.Delete(buffID)
	return buff, nil
}

func (bm *BuffManager) GetBuff(buffID int32) (*Buff, bool) {
	return bm.buffMap.Load(buffID)
}

func (bm *BuffManager) GetAllBuffs() map[int32]*Buff {
	result := make(map[int32]*Buff)
	bm.buffMap.Range(func(id int32, buff *Buff) bool {
		result[id] = buff
		return true
	})
	return result
}

func (bm *BuffManager) Update(deltaTime float64) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	var expired []int32
	bm.buffMap.Range(func(id int32, buff *Buff) bool {
		if buff.IsExpired() {
			expired = append(expired, id)
		}
		return true
	})

	for _, id := range expired {
		bm.buffMap.Delete(id)
	}
}

func (bm *BuffManager) HasBuff(buffID int32) bool {
	_, ok := bm.buffMap.Load(buffID)
	return ok
}

func (bm *BuffManager) Count() int64 {
	return bm.buffMap.Len()
}
