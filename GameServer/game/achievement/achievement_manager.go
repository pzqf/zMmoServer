package achievement

import (
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// AchievementManager 成就管理器
type AchievementManager struct {
	achievements     map[id.AchievementIdType]*Achievement
	playerAchievements map[id.PlayerIdType]map[id.AchievementIdType]*PlayerAchievement
	mutex            sync.RWMutex
}

// NewAchievementManager 创建成就管理器
func NewAchievementManager() *AchievementManager {
	return &AchievementManager{
		achievements:     make(map[id.AchievementIdType]*Achievement),
		playerAchievements: make(map[id.PlayerIdType]map[id.AchievementIdType]*PlayerAchievement),
	}
}

// AddAchievement 添加成就模板
func (am *AchievementManager) AddAchievement(achievement *Achievement) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.achievements[achievement.AchievementID] = achievement
	zLog.Info("Achievement template added",
		zap.Uint64("achievement_id", uint64(achievement.AchievementID)),
		zap.String("name", achievement.Name),
		zap.Int("type", achievement.Type),
		zap.Int("goal", achievement.Goal))
}

// GetAchievement 获取成就模板
func (am *AchievementManager) GetAchievement(achievementID id.AchievementIdType) *Achievement {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	return am.achievements[achievementID]
}

// GetAllAchievements 获取所有成就模板
func (am *AchievementManager) GetAllAchievements() []*Achievement {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	achievements := make([]*Achievement, 0, len(am.achievements))
	for _, achievement := range am.achievements {
		achievements = append(achievements, achievement)
	}

	return achievements
}

// GetPlayerAchievement 获取玩家成就进度
func (am *AchievementManager) GetPlayerAchievement(playerID id.PlayerIdType, achievementID id.AchievementIdType) *PlayerAchievement {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if playerAchievements, exists := am.playerAchievements[playerID]; exists {
		return playerAchievements[achievementID]
	}

	return nil
}

// GetPlayerAchievements 获取玩家所有成就进度
func (am *AchievementManager) GetPlayerAchievements(playerID id.PlayerIdType) []*PlayerAchievement {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if playerAchievements, exists := am.playerAchievements[playerID]; exists {
		achievements := make([]*PlayerAchievement, 0, len(playerAchievements))
		for _, achievement := range playerAchievements {
			achievements = append(achievements, achievement)
		}
		return achievements
	}

	return []*PlayerAchievement{}
}

// UpdateAchievementProgress 更新成就进度
func (am *AchievementManager) UpdateAchievementProgress(playerID id.PlayerIdType, achievementType int, progress int) []*Achievement {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// 初始化玩家成就数据
	if _, exists := am.playerAchievements[playerID]; !exists {
		am.playerAchievements[playerID] = make(map[id.AchievementIdType]*PlayerAchievement)
	}

	completedAchievements := []*Achievement{}

	// 更新对应类型的成就
	for achievementID, achievement := range am.achievements {
		if achievement.Type == achievementType {
			playerAchievement := am.playerAchievements[playerID][achievementID]
			if playerAchievement == nil {
				playerAchievement = NewPlayerAchievement(playerID, achievementID)
				am.playerAchievements[playerID][achievementID] = playerAchievement
			}

			if playerAchievement.UpdateProgress(achievement, progress) {
				completedAchievements = append(completedAchievements, achievement)
				zLog.Info("Player completed achievement",
					zap.Uint64("player_id", uint64(playerID)),
					zap.Uint64("achievement_id", uint64(achievementID)),
					zap.String("achievement_name", achievement.Name))
			}
		}
	}

	return completedAchievements
}

// ClaimAchievementReward 领取成就奖励
func (am *AchievementManager) ClaimAchievementReward(playerID id.PlayerIdType, achievementID id.AchievementIdType) bool {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if playerAchievements, exists := am.playerAchievements[playerID]; exists {
		if playerAchievement := playerAchievements[achievementID]; playerAchievement != nil {
			if playerAchievement.ClaimReward() {
				achievement := am.achievements[achievementID]
				zLog.Info("Player claimed achievement reward",
					zap.Uint64("player_id", uint64(playerID)),
					zap.Uint64("achievement_id", uint64(achievementID)),
					zap.String("achievement_name", achievement.Name),
					zap.Int("reward", achievement.Reward))
				return true
			}
		}
	}

	return false
}

// GetPlayerCompletedAchievements 获取玩家已完成的成就
func (am *AchievementManager) GetPlayerCompletedAchievements(playerID id.PlayerIdType) []*Achievement {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	completedAchievements := []*Achievement{}

	if playerAchievements, exists := am.playerAchievements[playerID]; exists {
		for achievementID, playerAchievement := range playerAchievements {
			if playerAchievement.IsCompleted {
				if achievement := am.achievements[achievementID]; achievement != nil {
					completedAchievements = append(completedAchievements, achievement)
				}
			}
		}
	}

	return completedAchievements
}

// GetPlayerUnclaimedRewards 获取玩家未领取的成就奖励
func (am *AchievementManager) GetPlayerUnclaimedRewards(playerID id.PlayerIdType) []*Achievement {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	unclaimedAchievements := []*Achievement{}

	if playerAchievements, exists := am.playerAchievements[playerID]; exists {
		for achievementID, playerAchievement := range playerAchievements {
			if playerAchievement.IsCompleted && !playerAchievement.RewardClaimed {
				if achievement := am.achievements[achievementID]; achievement != nil {
					unclaimedAchievements = append(unclaimedAchievements, achievement)
				}
			}
		}
	}

	return unclaimedAchievements
}

// GetAchievementCount 获取成就总数
func (am *AchievementManager) GetAchievementCount() int {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	return len(am.achievements)
}

// GetPlayerAchievementCount 获取玩家成就数量
func (am *AchievementManager) GetPlayerAchievementCount(playerID id.PlayerIdType) int {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if playerAchievements, exists := am.playerAchievements[playerID]; exists {
		return len(playerAchievements)
	}

	return 0
}
