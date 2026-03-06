package ranking

import (
	"sort"
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// RankingManager 排行榜管理器
type RankingManager struct {
	mu         sync.RWMutex
	rankType   RankType
	ranks      map[id.PlayerIdType]*RankItem
	rankList   []*RankItem
	maxRanks   int32
}

// NewRankingManager 创建排行榜管理器
func NewRankingManager(rankType RankType, maxRanks int32) *RankingManager {
	return &RankingManager{
		rankType: rankType,
		ranks:    make(map[id.PlayerIdType]*RankItem),
		rankList: make([]*RankItem, 0),
		maxRanks: maxRanks,
	}
}

// GetRankType 获取排行榜类型
func (rm *RankingManager) GetRankType() RankType {
	return rm.rankType
}

// GetRankCount 获取排行榜数量
func (rm *RankingManager) GetRankCount() int32 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return int32(len(rm.ranks))
}

// GetMaxRanks 获取最大排名数
func (rm *RankingManager) GetMaxRanks() int32 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.maxRanks
}

// IsFull 检查排行榜是否已满
func (rm *RankingManager) IsFull() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return int32(len(rm.ranks)) >= rm.maxRanks
}

// AddOrUpdatePlayer 添加或更新玩家
func (rm *RankingManager) AddOrUpdatePlayer(playerID id.PlayerIdType, playerName string, value int64, level int32) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	item, exists := rm.ranks[playerID]
	if exists {
		item.SetValue(value)
		item.SetLevel(level)
	} else {
		item = NewRankItem(0, playerID, playerName, value, level)
		rm.ranks[playerID] = item
	}

	rm.updateRankList()
	rm.updateRanks()
}

// RemovePlayer 移除玩家
func (rm *RankingManager) RemovePlayer(playerID id.PlayerIdType) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.ranks[playerID]; !exists {
		return
	}

	delete(rm.ranks, playerID)
	rm.updateRankList()
	rm.updateRanks()
}

// GetRank 获取玩家排名
func (rm *RankingManager) GetRank(playerID id.PlayerIdType) (int32, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	item, exists := rm.ranks[playerID]
	if !exists {
		return 0, ErrPlayerNotRanked
	}

	return item.GetRank(), nil
}

// GetRankItem 获取排行榜项
func (rm *RankingManager) GetRankItem(playerID id.PlayerIdType) (*RankItem, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	item, exists := rm.ranks[playerID]
	if !exists {
		return nil, ErrPlayerNotRanked
	}

	return item.Clone(), nil
}

// GetTopRanks 获取前N名
func (rm *RankingManager) GetTopRanks(n int32) []*RankItem {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if n > int32(len(rm.rankList)) {
		n = int32(len(rm.rankList))
	}

	result := make([]*RankItem, n)
	for i := int32(0); i < n; i++ {
		result[i] = rm.rankList[i].Clone()
	}

	return result
}

// GetAllRanks 获取所有排名
func (rm *RankingManager) GetAllRanks() []*RankItem {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	result := make([]*RankItem, len(rm.rankList))
	for i, item := range rm.rankList {
		result[i] = item.Clone()
	}

	return result
}

// GetRankRange 获取指定范围的排名
func (rm *RankingManager) GetRankRange(start, end int32) []*RankItem {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if start < 1 {
		start = 1
	}

	if end > int32(len(rm.rankList)) {
		end = int32(len(rm.rankList))
	}

	if start > end {
		return []*RankItem{}
	}

	result := make([]*RankItem, end-start+1)
	for i := start - 1; i < end; i++ {
		result[i-start+1] = rm.rankList[i].Clone()
	}

	return result
}

// Clear 清空排行榜
func (rm *RankingManager) Clear() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.ranks = make(map[id.PlayerIdType]*RankItem)
	rm.rankList = make([]*RankItem, 0)
}

// updateRankList 更新排序列表
func (rm *RankingManager) updateRankList() {
	rm.rankList = make([]*RankItem, 0, len(rm.ranks))
	for _, item := range rm.ranks {
		rm.rankList = append(rm.rankList, item)
	}

	sort.Slice(rm.rankList, func(i, j int) bool {
		if rm.rankList[i].GetValue() != rm.rankList[j].GetValue() {
			return rm.rankList[i].GetValue() > rm.rankList[j].GetValue()
		}
		if rm.rankList[i].GetLevel() != rm.rankList[j].GetLevel() {
			return rm.rankList[i].GetLevel() > rm.rankList[j].GetLevel()
		}
		return rm.rankList[i].GetPlayerID() < rm.rankList[j].GetPlayerID()
	})

	if int32(len(rm.rankList)) > rm.maxRanks {
		rm.rankList = rm.rankList[:rm.maxRanks]
	}
}

// updateRanks 更新排名
func (rm *RankingManager) updateRanks() {
	for i, item := range rm.rankList {
		item.SetRank(int32(i) + 1)
	}
}

// publishRankUpdateEvent 发布排名更新事件
func (rm *RankingManager) publishRankUpdateEvent(playerID id.PlayerIdType, rank int32) {
	event.Publish(event.NewEvent(event.EventRankUpdate, rm, &event.RankEventData{
		PlayerID: playerID,
		RankType: rm.rankType,
		Rank:     rank,
	}))
}

var (
	ErrPlayerNotRanked = ErrPlayerNotRankedError()
)

type ErrPlayerNotRankedError struct{}

func (e ErrPlayerNotRankedError) Error() string {
	return "player not ranked"
}

func ErrPlayerNotRankedError() ErrPlayerNotRankedError {
	return ErrPlayerNotRankedError{}
}

// GetRankTypeName 获取排行榜类型名称
func GetRankTypeName(rankType RankType) string {
	switch rankType {
	case RankTypeLevel:
		return "Level"
	case RankTypePower:
		return "Power"
	case RankTypeGold:
		return "Gold"
	case RankTypeDiamond:
		return "Diamond"
	default:
		return "Unknown"
	}
}

// LogTopRanks 记录前N名
func (rm *RankingManager) LogTopRanks(n int32) {
	topRanks := rm.GetTopRanks(n)
	zLog.Info("Top ranks",
		zap.String("type", GetRankTypeName(rm.rankType)),
		zap.Int32("count", int32(len(topRanks))))

	for _, item := range topRanks {
		zLog.Debug("Rank item",
			zap.Int32("rank", item.GetRank()),
			zap.Int64("player_id", int64(item.GetPlayerID())),
			zap.String("name", item.GetPlayerName()),
			zap.Int64("value", item.GetValue()),
			zap.Int32("level", item.GetLevel()))
	}
}
