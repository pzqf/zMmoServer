package player

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/achievement"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// PlayerAchievement 玩家成就组件
type PlayerAchievement struct {
	playerID      id.PlayerIdType
	achievements  map[id.AchievementIdType]*achievement.PlayerAchievement
	completedCount int
	totalPoints   int
}

// NewPlayerAchievement 创建玩家成就组件
func NewPlayerAchievement(playerID id.PlayerIdType) *PlayerAchievement {
	return &PlayerAchievement{
		playerID:       playerID,
		achievements:   make(map[id.AchievementIdType]*achievement.PlayerAchievement),
		completedCount: 0,
		totalPoints:    0,
	}
}

// GetAchievement 获取指定成就进度
func (pa *PlayerAchievement) GetAchievement(achievementID id.AchievementIdType) *achievement.PlayerAchievement {
	return pa.achievements[achievementID]
}

// GetAllAchievements 获取所有成就进度
func (pa *PlayerAchievement) GetAllAchievements() []*achievement.PlayerAchievement {
	result := make([]*achievement.PlayerAchievement, 0, len(pa.achievements))
	for _, a := range pa.achievements {
		result = append(result, a)
	}
	return result
}

// AddAchievement 添加成就进度
func (pa *PlayerAchievement) AddAchievement(playerAchievement *achievement.PlayerAchievement) {
	pa.achievements[playerAchievement.AchievementID] = playerAchievement

	if playerAchievement.IsCompleted {
		pa.completedCount++
	}
}

// UpdateProgress 更新成就进度
func (pa *PlayerAchievement) UpdateProgress(achievementID id.AchievementIdType, achievement *achievement.Achievement, progress int) bool {
	playerAchievement := pa.achievements[achievementID]
	if playerAchievement == nil {
		playerAchievement = achievement.NewPlayerAchievement(pa.playerID, achievementID)
		pa.achievements[achievementID] = playerAchievement
	}

	wasCompleted := playerAchievement.IsCompleted
	if playerAchievement.UpdateProgress(achievement, progress) {
		if !wasCompleted {
			pa.completedCount++
			pa.totalPoints += achievement.Reward
			zLog.Info("Player completed achievement",
				zap.Uint64("player_id", uint64(pa.playerID)),
				zap.Uint64("achievement_id", uint64(achievementID)),
				zap.String("achievement_name", achievement.Name),
				zap.Int("points", achievement.Reward))
		}
		return true
	}

	return false
}

// ClaimReward 领取成就奖励
func (pa *PlayerAchievement) ClaimReward(achievementID id.AchievementIdType) bool {
	playerAchievement := pa.achievements[achievementID]
	if playerAchievement == nil {
		return false
	}

	if playerAchievement.ClaimReward() {
		zLog.Info("Player claimed achievement reward",
			zap.Uint64("player_id", uint64(pa.playerID)),
			zap.Uint64("achievement_id", uint64(achievementID)))
		return true
	}

	return false
}

// GetCompletedCount 获取已完成成就数量
func (pa *PlayerAchievement) GetCompletedCount() int {
	return pa.completedCount
}

// GetTotalPoints 获取总成就点数
func (pa *PlayerAchievement) GetTotalPoints() int {
	return pa.totalPoints
}

// GetUnclaimedRewards 获取未领取奖励的成就
func (pa *PlayerAchievement) GetUnclaimedRewards() []*achievement.PlayerAchievement {
	result := make([]*achievement.PlayerAchievement, 0)
	for _, a := range pa.achievements {
		if a.IsCompleted && !a.RewardClaimed {
			result = append(result, a)
		}
	}
	return result
}

// HasAchievement 是否拥有指定成就
func (pa *PlayerAchievement) HasAchievement(achievementID id.AchievementIdType) bool {
	_, exists := pa.achievements[achievementID]
	return exists
}

// IsAchievementCompleted 指定成就是否已完成
func (pa *PlayerAchievement) IsAchievementCompleted(achievementID id.AchievementIdType) bool {
	if a, exists := pa.achievements[achievementID]; exists {
		return a.IsCompleted
	}
	return false
}
