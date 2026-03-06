package ranking

import (
	"sync"

	"github.com/pzqf/zMmoShared/common/id"
)

// RankType 排行榜类型
type RankType int32

const (
	RankTypeLevel  RankType = 1 // 等级排行
	RankTypePower  RankType = 2 // 战力排行
	RankTypeGold   RankType = 3 // 财富排行（金币）
	RankTypeDiamond RankType = 4 // 钻石排行
)

// RankItem 排行榜项
type RankItem struct {
	mu        sync.RWMutex
	rank      int32
	playerID  id.PlayerIdType
	playerName string
	value     int64
	level     int32
}

// NewRankItem 创建排行榜项
func NewRankItem(rank int32, playerID id.PlayerIdType, playerName string, value int64, level int32) *RankItem {
	return &RankItem{
		rank:      rank,
		playerID:  playerID,
		playerName: playerName,
		value:     value,
		level:     level,
	}
}

// GetRank 获取排名
func (ri *RankItem) GetRank() int32 {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.rank
}

// SetRank 设置排名
func (ri *RankItem) SetRank(rank int32) {
	ri.mu.Lock()
	defer ri.mu.Unlock()
	ri.rank = rank
}

// GetPlayerID 获取玩家ID
func (ri *RankItem) GetPlayerID() id.PlayerIdType {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.playerID
}

// GetPlayerName 获取玩家名称
func (ri *RankItem) GetPlayerName() string {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.playerName
}

// GetValue 获取值
func (ri *RankItem) GetValue() int64 {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.value
}

// SetValue 设置值
func (ri *RankItem) SetValue(value int64) {
	ri.mu.Lock()
	defer ri.mu.Unlock()
	ri.value = value
}

// GetLevel 获取等级
func (ri *RankItem) GetLevel() int32 {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.level
}

// SetLevel 设置等级
func (ri *RankItem) SetLevel(level int32) {
	ri.mu.Lock()
	defer ri.mu.Unlock()
	ri.level = level
}

// Clone 克隆排行榜项
func (ri *RankItem) Clone() *RankItem {
	ri.mu.RLock()
	defer ri.mu.RUnlock()

	return &RankItem{
		rank:      ri.rank,
		playerID:  ri.playerID,
		playerName: ri.playerName,
		value:     ri.value,
		level:     ri.level,
	}
}
