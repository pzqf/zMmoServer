package task

import (
	"sync"
	"time"

	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"

	"github.com/pzqf/zEngine/zLog"
)

// PlayerTask 玩家任务
type PlayerTask struct {
	TaskID       int32      `json:"task_id"`
	Status       TaskStatus `json:"status"`
	Progress     []int32    `json:"progress"` // 每个条件的进度
	AcceptTime   int64      `json:"accept_time"`
	CompleteTime int64      `json:"complete_time"`
}

// TaskManager 任务管理器
type TaskManager struct {
	mu            sync.RWMutex
	playerTasks   map[id.PlayerIdType]map[int32]*PlayerTask // 玩家ID -> 任务ID -> 任务
	configManager *TaskConfigManager
}

// NewTaskManager 创建任务管理器
func NewTaskManager() *TaskManager {
	return &TaskManager{
		playerTasks:   make(map[id.PlayerIdType]map[int32]*PlayerTask),
		configManager: NewTaskConfigManager(),
	}
}

// LoadTaskConfig 加载任务配置
func (tm *TaskManager) LoadTaskConfig(filePath string) error {
	return tm.configManager.LoadConfig(filePath)
}

// GetTaskConfig 获取任务配置
func (tm *TaskManager) GetTaskConfig(taskID int32) *TaskConfig {
	return tm.configManager.GetConfig(taskID)
}

// GetPlayerTasks 获取玩家的任务列表
func (tm *TaskManager) GetPlayerTasks(playerID id.PlayerIdType) []*PlayerTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks, exists := tm.playerTasks[playerID]
	if !exists {
		return []*PlayerTask{}
	}

	playerTasks := make([]*PlayerTask, 0, len(tasks))
	for _, task := range tasks {
		playerTasks = append(playerTasks, task)
	}

	return playerTasks
}

// GetPlayerTask 获取玩家的特定任务
func (tm *TaskManager) GetPlayerTask(playerID id.PlayerIdType, taskID int32) *PlayerTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks, exists := tm.playerTasks[playerID]
	if !exists {
		return nil
	}

	return tasks[taskID]
}

// AcceptTask 玩家接取任务
func (tm *TaskManager) AcceptTask(playerID id.PlayerIdType, taskID int32) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查任务配置是否存在
	taskConfig := tm.configManager.GetConfig(taskID)
	if taskConfig == nil {
		return nil
	}

	// 初始化玩家任务映射
	if _, exists := tm.playerTasks[playerID]; !exists {
		tm.playerTasks[playerID] = make(map[int32]*PlayerTask)
	}

	// 检查任务是否已接取
	if _, exists := tm.playerTasks[playerID][taskID]; exists {
		return nil
	}

	// 创建新任务
	progress := make([]int32, len(taskConfig.Conditions))
	playerTask := &PlayerTask{
		TaskID:     taskID,
		Status:     TaskStatusInProgress,
		Progress:   progress,
		AcceptTime: GetCurrentTimestamp(),
	}

	tm.playerTasks[playerID][taskID] = playerTask
	zLog.Debug("Player accepted task",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("task_id", taskID))

	return nil
}

// CompleteTask 完成任务
func (tm *TaskManager) CompleteTask(playerID id.PlayerIdType, taskID int32) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查玩家任务是否存在
	tasks, exists := tm.playerTasks[playerID]
	if !exists {
		return nil
	}

	task, exists := tasks[taskID]
	if !exists {
		return nil
	}

	// 检查任务状态
	if task.Status != TaskStatusInProgress {
		return nil
	}

	// 更新任务状态
	task.Status = TaskStatusCompleted
	task.CompleteTime = GetCurrentTimestamp()

	zLog.Debug("Player completed task",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("task_id", taskID))

	return nil
}

// RewardTask 领取任务奖励
func (tm *TaskManager) RewardTask(playerID id.PlayerIdType, taskID int32) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查玩家任务是否存在
	tasks, exists := tm.playerTasks[playerID]
	if !exists {
		return nil
	}

	task, exists := tasks[taskID]
	if !exists {
		return nil
	}

	// 检查任务状态
	if task.Status != TaskStatusCompleted {
		return nil
	}

	// 更新任务状态
	task.Status = TaskStatusRewarded

	zLog.Debug("Player received task reward",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("task_id", taskID))

	return nil
}

// UpdateTaskProgress 更新任务进度
func (tm *TaskManager) UpdateTaskProgress(playerID id.PlayerIdType, conditionType string, target int32, count int32) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查玩家任务是否存在
	tasks, exists := tm.playerTasks[playerID]
	if !exists {
		return
	}

	// 遍历玩家的所有任务
	for taskID, task := range tasks {
		// 只处理进行中的任务
		if task.Status != TaskStatusInProgress {
			continue
		}

		// 获取任务配置
		taskConfig := tm.configManager.GetConfig(taskID)
		if taskConfig == nil {
			continue
		}

		// 检查任务条件
		allCompleted := true
		for i, condition := range taskConfig.Conditions {
			if condition.Type == conditionType && condition.Target == target {
				// 更新进度
				task.Progress[i] += count
				if task.Progress[i] > condition.Count {
					task.Progress[i] = condition.Count
				}
			}

			// 检查是否所有条件都已完成
			if task.Progress[i] < condition.Count {
				allCompleted = false
			}
		}

		// 如果所有条件都已完成，标记任务为已完成
		if allCompleted {
			task.Status = TaskStatusCompleted
			task.CompleteTime = GetCurrentTimestamp()

			zLog.Debug("Task completed automatically",
				zap.Int64("player_id", int64(playerID)),
				zap.Int32("task_id", taskID))
		}
	}
}

// GetAvailableTasks 获取玩家可接取的任务
func (tm *TaskManager) GetAvailableTasks(playerID id.PlayerIdType, playerLevel int32) []*TaskConfig {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 获取玩家已完成的任务
	completedTasks := make([]int32, 0)
	if tasks, exists := tm.playerTasks[playerID]; exists {
		for taskID, task := range tasks {
			if task.Status == TaskStatusCompleted || task.Status == TaskStatusRewarded {
				completedTasks = append(completedTasks, taskID)
			}
		}
	}

	// 获取可接取的任务
	return tm.configManager.GetAvailableTasks(playerLevel, completedTasks)
}

// GetCurrentTimestamp 获取当前时间戳
func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}
