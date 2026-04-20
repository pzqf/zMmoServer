package task

import (
	"encoding/json"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/models"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type TaskType int32

const (
	TaskTypeMain        TaskType = 1
	TaskTypeSide        TaskType = 2
	TaskTypeDaily       TaskType = 3
	TaskTypeAchievement TaskType = 4
)

type TaskStatus int32

const (
	TaskStatusNotAccepted TaskStatus = 1
	TaskStatusInProgress  TaskStatus = 2
	TaskStatusCompleted   TaskStatus = 3
	TaskStatusRewarded    TaskStatus = 4
)

type TaskReward struct {
	Type   string `json:"type"`
	Target int32  `json:"target"`
	Count  int32  `json:"count"`
}

type PlayerTask struct {
	TaskID       int32      `json:"task_id"`
	Status       TaskStatus `json:"status"`
	Progress     []int32    `json:"progress"`
	AcceptTime   int64      `json:"accept_time"`
	CompleteTime int64      `json:"complete_time"`
}

type TaskManager struct {
	playerTasks  *zMap.TypedMap[id.PlayerIdType, *zMap.TypedMap[int32, *PlayerTask]]
	tableManager *tables.TableManager
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		playerTasks: zMap.NewTypedMap[id.PlayerIdType, *zMap.TypedMap[int32, *PlayerTask]](),
	}
}

func (tm *TaskManager) SetTableManager(t *tables.TableManager) {
	tm.tableManager = t
}

func (tm *TaskManager) GetTaskConfig(taskID int32) *models.Quest {
	if tm.tableManager == nil {
		return nil
	}
	quest, ok := tm.tableManager.GetQuestLoader().GetQuest(taskID)
	if !ok {
		return nil
	}
	return quest
}

func (tm *TaskManager) ParseTaskRewards(rewardsJSON string) []*TaskReward {
	if rewardsJSON == "" {
		return nil
	}
	var rewards []*TaskReward
	if err := json.Unmarshal([]byte(rewardsJSON), &rewards); err != nil {
		zLog.Warn("Failed to parse task rewards", zap.String("json", rewardsJSON), zap.Error(err))
		return nil
	}
	return rewards
}

func (tm *TaskManager) GetPlayerTasks(playerID id.PlayerIdType) []*PlayerTask {
	tasks, exists := tm.playerTasks.Load(playerID)
	if !exists {
		return []*PlayerTask{}
	}

	playerTasks := make([]*PlayerTask, 0)
	tasks.Range(func(_ int32, task *PlayerTask) bool {
		playerTasks = append(playerTasks, task)
		return true
	})

	return playerTasks
}

func (tm *TaskManager) GetPlayerTask(playerID id.PlayerIdType, taskID int32) *PlayerTask {
	tasks, exists := tm.playerTasks.Load(playerID)
	if !exists {
		return nil
	}

	task, _ := tasks.Load(taskID)
	return task
}

func (tm *TaskManager) AcceptTask(playerID id.PlayerIdType, taskID int32) error {
	taskConfig := tm.GetTaskConfig(taskID)
	if taskConfig == nil {
		return nil
	}

	tasks, exists := tm.playerTasks.Load(playerID)
	if !exists {
		tasks = zMap.NewTypedMap[int32, *PlayerTask]()
		tm.playerTasks.Store(playerID, tasks)
	}

	if _, exists := tasks.Load(taskID); exists {
		return nil
	}

	conditions := ParseTaskConditions(taskConfig.Objectives)
	progress := make([]int32, len(conditions))
	playerTask := &PlayerTask{
		TaskID:     taskID,
		Status:     TaskStatusInProgress,
		Progress:   progress,
		AcceptTime: GetCurrentTimestamp(),
	}

	tasks.Store(taskID, playerTask)
	zLog.Debug("Player accepted task",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("task_id", taskID))

	return nil
}

func (tm *TaskManager) CompleteTask(playerID id.PlayerIdType, taskID int32) error {
	tasks, exists := tm.playerTasks.Load(playerID)
	if !exists {
		return nil
	}

	task, exists := tasks.Load(taskID)
	if !exists {
		return nil
	}

	if task.Status != TaskStatusInProgress {
		return nil
	}

	task.Status = TaskStatusCompleted
	task.CompleteTime = GetCurrentTimestamp()

	zLog.Debug("Player completed task",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("task_id", taskID))

	return nil
}

func (tm *TaskManager) RewardTask(playerID id.PlayerIdType, taskID int32) error {
	tasks, exists := tm.playerTasks.Load(playerID)
	if !exists {
		return nil
	}

	task, exists := tasks.Load(taskID)
	if !exists {
		return nil
	}

	if task.Status != TaskStatusCompleted {
		return nil
	}

	task.Status = TaskStatusRewarded

	zLog.Debug("Player received task reward",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("task_id", taskID))

	return nil
}

func (tm *TaskManager) UpdateTaskProgress(playerID id.PlayerIdType, conditionType string, target int32, count int32) {
	if tasks, exists := tm.playerTasks.Load(playerID); exists {
		tasks.Range(func(taskID int32, task *PlayerTask) bool {
			if task.Status != TaskStatusInProgress {
				return true
			}

			taskConfig := tm.GetTaskConfig(taskID)
			if taskConfig == nil {
				return true
			}

			conditions := ParseTaskConditions(taskConfig.Objectives)
			allCompleted := true
			for i, condition := range conditions {
				if condition.Type == conditionType && condition.Target == target {
					task.Progress[i] += count
					if task.Progress[i] > condition.Count {
						task.Progress[i] = condition.Count
					}
				}

				if task.Progress[i] < condition.Count {
					allCompleted = false
				}
			}

			if allCompleted {
				task.Status = TaskStatusCompleted
				task.CompleteTime = GetCurrentTimestamp()

				zLog.Debug("Task completed automatically",
					zap.Int64("player_id", int64(playerID)),
					zap.Int32("task_id", taskID))
			}
			return true
		})
	}
}

func (tm *TaskManager) GetAvailableTasks(playerID id.PlayerIdType, playerLevel int32) []*models.Quest {
	if tm.tableManager == nil {
		return nil
	}

	completedTasks := make([]int32, 0)
	if tasks, exists := tm.playerTasks.Load(playerID); exists {
		tasks.Range(func(taskID int32, task *PlayerTask) bool {
			if task.Status == TaskStatusCompleted || task.Status == TaskStatusRewarded {
				completedTasks = append(completedTasks, taskID)
			}
			return true
		})
	}

	allQuests := tm.tableManager.GetQuestLoader().GetAllQuests()
	available := make([]*models.Quest, 0)

	for _, quest := range allQuests {
		if quest.Level > playerLevel {
			continue
		}

		isCompleted := false
		for _, tid := range completedTasks {
			if tid == quest.QuestID {
				isCompleted = true
				break
			}
		}
		if isCompleted {
			continue
		}

		if quest.PreQuestID > 0 {
			hasPrev := false
			for _, tid := range completedTasks {
				if tid == quest.PreQuestID {
					hasPrev = true
					break
				}
			}
			if !hasPrev {
				continue
			}
		}

		available = append(available, quest)
	}

	return available
}

type TaskCondition struct {
	Type   string `json:"type"`
	Target int32  `json:"target"`
	Count  int32  `json:"count"`
}

func ParseTaskConditions(objectivesJSON string) []TaskCondition {
	if objectivesJSON == "" {
		return nil
	}
	var conditions []TaskCondition
	if err := json.Unmarshal([]byte(objectivesJSON), &conditions); err != nil {
		return nil
	}
	return conditions
}

func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}
