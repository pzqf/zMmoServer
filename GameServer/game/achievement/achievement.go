package achievement

import (
	"github.com/pzqf/zMmoShared/common/id"
	"time"
)

// Achievement 成就结构
type Achievement struct {
	AchievementID id.AchievementIdType // 成就ID
	Name          string               // 成就名称
	Description   string               // 成就描述
	Type          int                  // 成就类型
	Goal          int                  // 目标值
	Reward        int                  // 奖励
	Icon          string               // 图标
	Rarity        int                  // 稀有度
	Category      string               // 分类
}

// PlayerAchievement 玩家成就进度
type PlayerAchievement struct {
	PlayerID       id.PlayerIdType       // 玩家ID
	AchievementID  id.AchievementIdType  // 成就ID
	Progress       int                   // 当前进度
	IsCompleted    bool                  // 是否完成
	CompleteTime   time.Time             // 完成时间
	RewardClaimed  bool                  // 奖励是否领取
}

// AchievementType 成就类型常量
const (
	AchievementTypeLevel      = 1 // 等级成就
	AchievementTypeKill       = 2 // 击杀成就
	AchievementTypeQuest      = 3 // 任务成就
	AchievementTypeCollection = 4 // 收集成就
	AchievementTypeSocial     = 5 // 社交成就
	AchievementTypePvP        = 6 // PVP成就
	AchievementTypeCraft      = 7 //  crafting成就
	AchievementTypeExplore    = 8 // 探索成就
)

// NewAchievement 创建新成就
func NewAchievement(id id.AchievementIdType, name, description string, achievementType, goal, reward int, icon string, rarity int, category string) *Achievement {
	return &Achievement{
		AchievementID: id,
		Name:          name,
		Description:   description,
		Type:          achievementType,
		Goal:          goal,
		Reward:        reward,
		Icon:          icon,
		Rarity:        rarity,
		Category:      category,
	}
}

// NewPlayerAchievement 创建玩家成就进度
func NewPlayerAchievement(playerID id.PlayerIdType, achievementID id.AchievementIdType) *PlayerAchievement {
	return &PlayerAchievement{
		PlayerID:       playerID,
		AchievementID:  achievementID,
		Progress:       0,
		IsCompleted:    false,
		CompleteTime:   time.Time{},
		RewardClaimed:  false,
	}
}

// UpdateProgress 更新成就进度
func (pa *PlayerAchievement) UpdateProgress(achievement *Achievement, progress int) bool {
	if pa.IsCompleted {
		return false
	}

	pa.Progress = min(pa.Progress+progress, achievement.Goal)

	if pa.Progress >= achievement.Goal {
		pa.IsCompleted = true
		pa.CompleteTime = time.Now()
		return true
	}

	return false
}

// ClaimReward 领取奖励
func (pa *PlayerAchievement) ClaimReward() bool {
	if !pa.IsCompleted || pa.RewardClaimed {
		return false
	}

	pa.RewardClaimed = true
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
