package game

import (
	"fmt"
	"sync"

	"github.com/pzqf/zUtil/zMap"
)

type TaskType int

const (
	TaskTypeMain     TaskType = 1
	TaskTypeSide     TaskType = 2
	TaskTypeDaily    TaskType = 3
	TaskTypeWeekly   TaskType = 4
	TaskTypeActivity TaskType = 5
)

type TaskStatus int

const (
	TaskStatusNotAccepted TaskStatus = 0
	TaskStatusInProgress  TaskStatus = 1
	TaskStatusCompleted   TaskStatus = 2
	TaskStatusRewarded    TaskStatus = 3
)

type TaskConditionType int

const (
	TaskCondKillMonster  TaskConditionType = 1
	TaskCondCollectItem  TaskConditionType = 2
	TaskCondTalkNPC      TaskConditionType = 3
	TaskCondReachLevel   TaskConditionType = 4
	TaskCondCompleteTask TaskConditionType = 5
)

type TaskRewardType int

const (
	TaskRewardGold      TaskRewardType = 1
	TaskRewardExp       TaskRewardType = 2
	TaskRewardItem      TaskRewardType = 3
	TaskRewardSkillPoint TaskRewardType = 4
)

type TaskCondition struct {
	Type     TaskConditionType
	TargetID int32
	Current  int32
	Required int32
}

func (c *TaskCondition) IsComplete() bool {
	return c.Current >= c.Required
}

func (c *TaskCondition) AddProgress(count int32) {
	c.Current += count
	if c.Current > c.Required {
		c.Current = c.Required
	}
}

type TaskReward struct {
	Type     TaskRewardType
	Amount   int32
	ItemID   int32
	ItemCount int32
}

type Task struct {
	mu         sync.RWMutex
	TaskID     int64
	ConfigID   int32
	Name       string
	Type       TaskType
	Status     TaskStatus
	Conditions []*TaskCondition
	Rewards    []*TaskReward
}

func NewTask(taskID int64, configID int32, name string, taskType TaskType) *Task {
	return &Task{
		TaskID:   taskID,
		ConfigID: configID,
		Name:     name,
		Type:     taskType,
		Status:   TaskStatusNotAccepted,
	}
}

func (t *Task) IsComplete() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, cond := range t.Conditions {
		if !cond.IsComplete() {
			return false
		}
	}
	return true
}

type TaskManager struct {
	BaseComponent
	mu       sync.RWMutex
	tasks    *zMap.TypedMap[int64, *Task]
	maxCount int32
}

func NewTaskManager(maxCount int32) *TaskManager {
	return &TaskManager{
		BaseComponent: NewBaseComponent("tasks"),
		tasks:         zMap.NewTypedMap[int64, *Task](),
		maxCount:      maxCount,
	}
}

func (tm *TaskManager) AcceptTask(task *Task) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if int32(tm.tasks.Len()) >= tm.maxCount {
		return fmt.Errorf("task slots full")
	}

	if _, ok := tm.tasks.Load(task.TaskID); ok {
		return fmt.Errorf("task %d already accepted", task.TaskID)
	}

	task.Status = TaskStatusInProgress
	tm.tasks.Store(task.TaskID, task)
	return nil
}

func (tm *TaskManager) UpdateProgress(configID int32, condType TaskConditionType, targetID int32, count int32) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tm.tasks.Range(func(taskID int64, task *Task) bool {
		if task.Status != TaskStatusInProgress {
			return true
		}
		if task.ConfigID != configID && configID != 0 {
			return true
		}
		for _, cond := range task.Conditions {
			if cond.Type == condType && cond.TargetID == targetID {
				cond.AddProgress(count)
			}
		}
		if task.IsComplete() {
			task.Status = TaskStatusCompleted
		}
		return true
	})
}

func (tm *TaskManager) CompleteTask(taskID int64) ([]*TaskReward, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks.Load(taskID)
	if !ok {
		return nil, fmt.Errorf("task %d not found", taskID)
	}

	if task.Status != TaskStatusCompleted {
		return nil, fmt.Errorf("task %d not complete", taskID)
	}

	task.Status = TaskStatusRewarded
	rewards := task.Rewards
	return rewards, nil
}

func (tm *TaskManager) AbandonTask(taskID int64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks.Load(taskID)
	if !ok {
		return fmt.Errorf("task %d not found", taskID)
	}

	if task.Status == TaskStatusRewarded {
		return fmt.Errorf("cannot abandon rewarded task")
	}

	tm.tasks.Delete(taskID)
	return nil
}

func (tm *TaskManager) GetTask(taskID int64) (*Task, bool) {
	return tm.tasks.Load(taskID)
}

func (tm *TaskManager) GetTasksByStatus(status TaskStatus) []*Task {
	var result []*Task
	tm.tasks.Range(func(id int64, task *Task) bool {
		if task.Status == status {
			result = append(result, task)
		}
		return true
	})
	return result
}

func (tm *TaskManager) Count() int64 {
	return tm.tasks.Len()
}
