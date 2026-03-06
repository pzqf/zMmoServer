package task

import (
	"encoding/json"
	"os"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// TaskType 任务类型
type TaskType int32

const (
	TaskTypeMain      TaskType = 1 // 主线任务
	TaskTypeSide      TaskType = 2 // 支线任务
	TaskTypeDaily     TaskType = 3 // 日常任务
	TaskTypeAchievement TaskType = 4 // 成就任务
)

// TaskStatus 任务状态
type TaskStatus int32

const (
	TaskStatusNotAccepted TaskStatus = 1 // 未接取
	TaskStatusInProgress  TaskStatus = 2 // 进行中
	TaskStatusCompleted   TaskStatus = 3 // 已完成
	TaskStatusRewarded    TaskStatus = 4 // 已领奖
)

// TaskCondition 任务条件
type TaskCondition struct {
	Type   string      `json:"type"` // 条件类型：kill_monster, collect_item, talk_npc, reach_level
	Target int32       `json:"target"` // 目标ID
	Count  int32       `json:"count"` // 目标数量
	Data   interface{} `json:"data"` // 额外数据
}

// TaskReward 任务奖励
type TaskReward struct {
	Type   string `json:"type"` // 奖励类型：exp, gold, item, skill_point
	Target int32  `json:"target"` // 目标ID（物品ID等）
	Count  int32  `json:"count"` // 奖励数量
}

// TaskConfig 任务配置
type TaskConfig struct {
	ID          int32           `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        TaskType        `json:"type"`
	Level       int32           `json:"level"`
	Conditions  []TaskCondition `json:"conditions"`
	Rewards     []TaskReward    `json:"rewards"`
	PrevTaskID  int32           `json:"prev_task_id"` // 前置任务ID
	NextTaskID  int32           `json:"next_task_id"` // 后置任务ID
}

// TaskConfigManager 任务配置管理器
type TaskConfigManager struct {
	configs map[int32]*TaskConfig
}

// NewTaskConfigManager 创建任务配置管理器
func NewTaskConfigManager() *TaskConfigManager {
	return &TaskConfigManager{
		configs: make(map[int32]*TaskConfig),
	}
}

// LoadConfig 加载任务配置
func (tcm *TaskConfigManager) LoadConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		zLog.Error("Failed to read task config file", zap.Error(err))
		return err
	}

	var configs []*TaskConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		zLog.Error("Failed to unmarshal task config", zap.Error(err))
		return err
	}

	for _, config := range configs {
		tcm.configs[config.ID] = config
	}

	zLog.Info("Task config loaded successfully", zap.Int("count", len(tcm.configs)))
	return nil
}

// GetConfig 获取任务配置
func (tcm *TaskConfigManager) GetConfig(taskID int32) *TaskConfig {
	return tcm.configs[taskID]
}

// GetAllConfigs 获取所有任务配置
func (tcm *TaskConfigManager) GetAllConfigs() []*TaskConfig {
	configs := make([]*TaskConfig, 0, len(tcm.configs))
	for _, config := range tcm.configs {
		configs = append(configs, config)
	}
	return configs
}

// GetAvailableTasks 获取玩家可接取的任务
func (tcm *TaskConfigManager) GetAvailableTasks(playerLevel int32, completedTasks []int32) []*TaskConfig {
	availableTasks := make([]*TaskConfig, 0)

	for _, config := range tcm.configs {
		// 检查等级要求
		if config.Level > playerLevel {
			continue
		}

		// 检查前置任务
		if config.PrevTaskID > 0 {
			hasCompleted := false
			for _, taskID := range completedTasks {
				if taskID == config.PrevTaskID {
					hasCompleted = true
					break
				}
			}
			if !hasCompleted {
				continue
			}
		}

		// 检查是否已完成
		hasCompleted := false
		for _, taskID := range completedTasks {
			if taskID == config.ID {
				hasCompleted = true
				break
			}
		}
		if hasCompleted {
			continue
		}

		availableTasks = append(availableTasks, config)
	}

	return availableTasks
}