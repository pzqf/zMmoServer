package achievement

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// AchievementType 成就类型
type AchievementType int32

const (
	AchievementTypeLevel      AchievementType = 1 // 等级成就
	AchievementTypeCombat     AchievementType = 2 // 战斗成就
	AchievementTypeCollection AchievementType = 3 // 收集成就
	AchievementTypeSocial     AchievementType = 4 // 社交成就
	AchievementTypeExplore    AchievementType = 5 // 探索成就
	AchievementTypeEconomy    AchievementType = 6 // 经济成就
)

// AchievementCondition 成就条件
type AchievementCondition struct {
	Type   string `json:"type"`   // 条件类型
	Target int32  `json:"target"` // 目标值
}

// AchievementReward 成就奖励
type AchievementReward struct {
	Type  string `json:"type"`  // 奖励类型：currency, item, skill
	ID    int32   `json:"id"`    // 奖励ID
	Count int32   `json:"count"` // 奖励数量
}

// AchievementConfig 成就配置
type AchievementConfig struct {
	ID          int32                  `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        AchievementType       `json:"type"`
	Level       int32                 `json:"level"` // 成就等级（1-3星）
	Conditions  []AchievementCondition `json:"conditions"`
	Rewards     []AchievementReward    `json:"rewards"`
	PrevAchievementID int32            `json:"prev_achievement_id"` // 前置成就ID
}

// PlayerAchievement 玩家成就进度
type PlayerAchievement struct {
	AchievementID int32   `json:"achievement_id"`
	Status        int32   `json:"status"` // 0:未达成 1:进行中 2:已完成 3:已领取
	Progress      int32   `json:"progress"`
	CompleteTime  int64   `json:"complete_time"`
	RewardTime    int64   `json:"reward_time"`
}

// AchievementConfigManager 成就配置管理器
type AchievementConfigManager struct {
	mu       sync.RWMutex
	configs  map[int32]*AchievementConfig
	typeMap  map[AchievementType][]*AchievementConfig
}

// NewAchievementConfigManager 创建成就配置管理器
func NewAchievementConfigManager() *AchievementConfigManager {
	return &AchievementConfigManager{
		configs: make(map[int32]*AchievementConfig),
		typeMap: make(map[AchievementType][]*AchievementConfig),
	}
}

// LoadConfig 加载成就配置
func (acm *AchievementConfigManager) LoadConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		zLog.Error("Failed to read achievement config file", zap.Error(err))
		return err
	}

	var configs []*AchievementConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		zLog.Error("Failed to unmarshal achievement config", zap.Error(err))
		return err
	}

	for _, config := range configs {
		acm.configs[config.ID] = config

		// 按类型分组
		acm.typeMap[config.Type] = append(acm.typeMap[config.Type], config)
	}

	zLog.Info("Achievement config loaded successfully", zap.Int("count", len(acm.configs)))
	return nil
}

// GetConfig 获取成就配置
func (acm *AchievementConfigManager) GetConfig(achievementID int32) *AchievementConfig {
	acm.mu.RLock()
	defer acm.mu.RUnlock()
	return acm.configs[achievementID]
}

// GetConfigsByType 获取指定类型的成就
func (acm *AchievementConfigManager) GetConfigsByType(achievementType AchievementType) []*AchievementConfig {
	acm.mu.RLock()
	defer acm.mu.RUnlock()
	configs := make([]*AchievementConfig, len(acm.typeMap[achievementType]))
	copy(configs, acm.typeMap[achievementType])
	return configs
}

// AchievementManager 成就管理器
type AchievementManager struct {
	mu            sync.RWMutex
	configManager *AchievementConfigManager
	playerAchievements map[id.PlayerIdType]map[int32]*PlayerAchievement
}

// NewAchievementManager 创建成就管理器
func NewAchievementManager() *AchievementManager {
	return &AchievementManager{
		configManager: NewAchievementConfigManager(),
		playerAchievements: make(map[id.PlayerIdType]map[int32]*PlayerAchievement),
	}
}

// LoadConfig 加载成就配置
func (am *AchievementManager) LoadConfig(filePath string) error {
	return am.configManager.LoadConfig(filePath)
}

// GetConfig 获取成就配置
func (am *AchievementManager) GetConfig(achievementID int32) *AchievementConfig {
	return am.configManager.GetConfig(achievementID)
}

// InitPlayerAchievement 初始化玩家成就
func (am *AchievementManager) InitPlayerAchievement(playerID id.PlayerIdType) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.playerAchievements[playerID]; !exists {
		am.playerAchievements[playerID] = make(map[int32]*PlayerAchievement)

		// 初始化所有成就为未达成状态
		for _, config := range am.configManager.configs {
			am.playerAchievements[playerID][config.ID] = &PlayerAchievement{
				AchievementID: config.ID,
				Status:        0,
				Progress:      0,
				CompleteTime:  0,
				RewardTime:    0,
			}
		}
	}
}

// GetPlayerAchievement 获取玩家成就进度
func (am *AchievementManager) GetPlayerAchievement(playerID id.PlayerIdType, achievementID int32) *PlayerAchievement {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if achievements, exists := am.playerAchievements[playerID]; exists {
		return achievements[achievementID]
	}
	return nil
}

// GetPlayerAllAchievements 获取玩家所有成就进度
func (am *AchievementManager) GetPlayerAllAchievements(playerID id.PlayerIdType) map[int32]*PlayerAchievement {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if achievements, exists := am.playerAchievements[playerID]; exists {
		result := make(map[int32]*PlayerAchievement)
		for k, v := range achievements {
			result[k] = v
		}
		return result
	}
	return make(map[int32]*PlayerAchievement)
}

// UpdateAchievementProgress 更新成就进度
func (am *AchievementManager) UpdateAchievementProgress(playerID id.PlayerIdType, conditionType string, target int32, value int32) {
	am.mu.Lock()
	defer am.mu.Unlock()

	achievements, exists := am.playerAchievements[playerID]
	if !exists {
		return
	}

	now := time.Now().Unix()

	// 遍历所有成就，检查条件
	for _, config := range am.configManager.configs {
		achievement, ok := achievements[config.ID]
		if !ok || achievement.Status == 2 || achievement.Status == 3 {
			continue // 已完成或已领取，跳过
		}

		// 检查是否匹配条件类型
		for _, condition := range config.Conditions {
			if condition.Type == conditionType {
				// 检查目标是否匹配
				if condition.Target == target {
					// 增加进度
					achievement.Progress += value
					if achievement.Status == 0 {
						achievement.Status = 1 // 开始进行
					}

					// 检查是否完成
					if achievement.Progress >= config.Conditions[0].Target {
						achievement.Status = 2       // 已完成
						achievement.CompleteTime = now
					}

					zLog.Debug("Achievement progress updated",
						zap.Int64("player_id", int64(playerID)),
						zap.Int32("achievement_id", config.ID),
						zap.Int32("progress", achievement.Progress),
						zap.Int32("status", achievement.Status))
				}
			}
		}
	}
}

// ClaimAchievementReward 领取成就奖励
func (am *AchievementManager) ClaimAchievementReward(playerID id.PlayerIdType, achievementID int32) ([]AchievementReward, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	achievements, exists := am.playerAchievements[playerID]
	if !exists {
		return nil, nil
	}

	achievement, ok := achievements[achievementID]
	if !ok {
		return nil, nil
	}

	// 检查成就是否已完成
	if achievement.Status != 2 {
		return nil, nil
	}

	// 检查奖励是否已领取
	if achievement.Status == 3 {
		return nil, nil
	}

	// 获取成就配置
	config := am.configManager.GetConfig(achievementID)
	if config == nil {
		return nil, nil
	}

	// 标记为已领取
	achievement.Status = 3
	achievement.RewardTime = time.Now().Unix()

	zLog.Debug("Achievement reward claimed",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("achievement_id", achievementID))

	return config.Rewards, nil
}

// GetCompletedCount 获取已完成成就数量
func (am *AchievementManager) GetCompletedCount(playerID id.PlayerIdType) int32 {
	am.mu.RLock()
	defer am.mu.RUnlock()

	count := int32(0)
	if achievements, exists := am.playerAchievements[playerID]; exists {
		for _, achievement := range achievements {
			if achievement.Status == 2 || achievement.Status == 3 {
				count++
			}
		}
	}
	return count
}

// GetTotalAchievementCount 获取总成就数量
func (am *AchievementManager) GetTotalAchievementCount() int32 {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return int32(len(am.configManager.configs))
}