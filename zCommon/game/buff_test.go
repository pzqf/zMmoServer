package game

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBuffManager(t *testing.T) {
	bm := NewBuffManager()

	assert.NotNil(t, bm)
	assert.Equal(t, int64(0), bm.Count())
}

func TestBuffManagerAddBuff(t *testing.T) {
	bm := NewBuffManager()
	buff := NewBuff(1, "Speed Boost", BuffTypePositive, 10*time.Second)

	bm.AddBuff(buff)

	assert.Equal(t, int64(1), bm.Count())
	assert.True(t, bm.HasBuff(1))
}

func TestBuffManagerAddBuffRefresh(t *testing.T) {
	bm := NewBuffManager()
	buff1 := NewBuff(1, "Speed Boost", BuffTypePositive, 10*time.Second)
	bm.AddBuff(buff1)

	originalEnd := buff1.EndTime
	time.Sleep(10 * time.Millisecond)

	buff2 := NewBuff(1, "Speed Boost", BuffTypePositive, 10*time.Second)
	bm.AddBuff(buff2)

	assert.Equal(t, int64(1), bm.Count())
	stored, ok := bm.GetBuff(1)
	assert.True(t, ok)
	assert.True(t, stored.EndTime.After(originalEnd))
}

func TestBuffManagerRemoveBuff(t *testing.T) {
	bm := NewBuffManager()
	buff := NewBuff(1, "Speed Boost", BuffTypePositive, 10*time.Second)
	bm.AddBuff(buff)

	removed, err := bm.RemoveBuff(1)

	assert.NoError(t, err)
	assert.Equal(t, buff, removed)
	assert.Equal(t, int64(0), bm.Count())
	assert.False(t, bm.HasBuff(1))
}

func TestBuffManagerRemoveBuffNotFound(t *testing.T) {
	bm := NewBuffManager()

	_, err := bm.RemoveBuff(999)
	assert.Error(t, err)
}

func TestBuffManagerGetBuff(t *testing.T) {
	bm := NewBuffManager()
	buff := NewBuff(1, "Speed Boost", BuffTypePositive, 10*time.Second)
	bm.AddBuff(buff)

	stored, ok := bm.GetBuff(1)
	assert.True(t, ok)
	assert.Equal(t, int32(1), stored.ID)
	assert.Equal(t, "Speed Boost", stored.Name)
}

func TestBuffManagerGetBuffNotFound(t *testing.T) {
	bm := NewBuffManager()

	_, ok := bm.GetBuff(999)
	assert.False(t, ok)
}

func TestBuffManagerGetAllBuffs(t *testing.T) {
	bm := NewBuffManager()
	bm.AddBuff(NewBuff(1, "Buff1", BuffTypePositive, 10*time.Second))
	bm.AddBuff(NewBuff(2, "Buff2", BuffTypeNegative, 5*time.Second))

	all := bm.GetAllBuffs()
	assert.Len(t, all, 2)
}

func TestBuffExpired(t *testing.T) {
	buff := NewBuff(1, "Short Buff", BuffTypePositive, 1*time.Nanosecond)
	time.Sleep(1 * time.Millisecond)

	assert.True(t, buff.IsExpired())
}

func TestBuffPermanent(t *testing.T) {
	buff := NewBuff(1, "Permanent Buff", BuffTypePositive, 0)
	buff.IsPermanent = true

	assert.False(t, buff.IsExpired())
}

func TestBuffGetRemaining(t *testing.T) {
	buff := NewBuff(1, "Buff", BuffTypePositive, 10*time.Second)

	remaining := buff.GetRemaining()
	assert.True(t, remaining > 9*time.Second)
	assert.True(t, remaining <= 10*time.Second)
}

func TestBuffRefresh(t *testing.T) {
	buff := NewBuff(1, "Buff", BuffTypePositive, 10*time.Second)
	originalEnd := buff.EndTime

	time.Sleep(10 * time.Millisecond)
	buff.Refresh()

	assert.True(t, buff.EndTime.After(originalEnd))
}

func TestBuffManagerUpdateRemovesExpired(t *testing.T) {
	bm := NewBuffManager()
	buff := NewBuff(1, "Short Buff", BuffTypePositive, 1*time.Nanosecond)
	bm.AddBuff(buff)

	time.Sleep(1 * time.Millisecond)
	bm.Update(0.016)

	assert.Equal(t, int64(0), bm.Count())
	assert.False(t, bm.HasBuff(1))
}

func TestBuffManagerUpdateKeepsActive(t *testing.T) {
	bm := NewBuffManager()
	buff := NewBuff(1, "Long Buff", BuffTypePositive, 10*time.Second)
	bm.AddBuff(buff)

	bm.Update(0.016)

	assert.Equal(t, int64(1), bm.Count())
	assert.True(t, bm.HasBuff(1))
}
