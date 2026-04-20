package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTaskManager(t *testing.T) {
	tm := NewTaskManager(20)

	assert.NotNil(t, tm)
	assert.Equal(t, int64(0), tm.Count())
}

func TestTaskManagerAcceptTask(t *testing.T) {
	tm := NewTaskManager(20)
	task := NewTask(1, 1001, "Kill Monsters", TaskTypeMain)

	err := tm.AcceptTask(task)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), tm.Count())
	assert.Equal(t, TaskStatusInProgress, task.Status)
}

func TestTaskManagerAcceptTaskDuplicate(t *testing.T) {
	tm := NewTaskManager(20)
	task1 := NewTask(1, 1001, "Kill Monsters", TaskTypeMain)
	_ = tm.AcceptTask(task1)

	task2 := NewTask(1, 1001, "Kill Monsters", TaskTypeMain)
	err := tm.AcceptTask(task2)

	assert.Error(t, err)
}

func TestTaskManagerAcceptTaskFull(t *testing.T) {
	tm := NewTaskManager(2)

	_ = tm.AcceptTask(NewTask(1, 1001, "Task1", TaskTypeMain))
	_ = tm.AcceptTask(NewTask(2, 1002, "Task2", TaskTypeMain))

	err := tm.AcceptTask(NewTask(3, 1003, "Task3", TaskTypeMain))
	assert.Error(t, err)
}

func TestTaskManagerUpdateProgress(t *testing.T) {
	tm := NewTaskManager(20)
	task := NewTask(1, 1001, "Kill Monsters", TaskTypeMain)
	task.Conditions = []*TaskCondition{
		{Type: TaskCondKillMonster, TargetID: 100, Current: 0, Required: 5},
	}
	_ = tm.AcceptTask(task)

	tm.UpdateProgress(1001, TaskCondKillMonster, 100, 3)

	stored, ok := tm.GetTask(1)
	assert.True(t, ok)
	assert.Equal(t, int32(3), stored.Conditions[0].Current)
}

func TestTaskManagerUpdateProgressComplete(t *testing.T) {
	tm := NewTaskManager(20)
	task := NewTask(1, 1001, "Kill Monsters", TaskTypeMain)
	task.Conditions = []*TaskCondition{
		{Type: TaskCondKillMonster, TargetID: 100, Current: 4, Required: 5},
	}
	_ = tm.AcceptTask(task)

	tm.UpdateProgress(1001, TaskCondKillMonster, 100, 1)

	stored, _ := tm.GetTask(1)
	assert.Equal(t, TaskStatusCompleted, stored.Status)
}

func TestTaskManagerCompleteTask(t *testing.T) {
	tm := NewTaskManager(20)
	task := NewTask(1, 1001, "Kill Monsters", TaskTypeMain)
	task.Conditions = []*TaskCondition{
		{Type: TaskCondKillMonster, TargetID: 100, Current: 0, Required: 5},
	}
	task.Rewards = []*TaskReward{
		{Type: TaskRewardGold, Amount: 1000},
	}
	_ = tm.AcceptTask(task)

	tm.UpdateProgress(1001, TaskCondKillMonster, 100, 5)

	rewards, err := tm.CompleteTask(1)

	assert.NoError(t, err)
	assert.Len(t, rewards, 1)
	assert.Equal(t, TaskRewardGold, rewards[0].Type)
	assert.Equal(t, int32(1000), rewards[0].Amount)
}

func TestTaskManagerCompleteTaskNotReady(t *testing.T) {
	tm := NewTaskManager(20)
	task := NewTask(1, 1001, "Kill Monsters", TaskTypeMain)
	task.Status = TaskStatusInProgress
	_ = tm.AcceptTask(task)

	_, err := tm.CompleteTask(1)
	assert.Error(t, err)
}

func TestTaskManagerAbandonTask(t *testing.T) {
	tm := NewTaskManager(20)
	task := NewTask(1, 1001, "Kill Monsters", TaskTypeMain)
	_ = tm.AcceptTask(task)

	err := tm.AbandonTask(1)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), tm.Count())
}

func TestTaskManagerAbandonRewardedTask(t *testing.T) {
	tm := NewTaskManager(20)
	task := NewTask(1, 1001, "Kill Monsters", TaskTypeMain)
	task.Conditions = []*TaskCondition{
		{Type: TaskCondKillMonster, TargetID: 100, Current: 0, Required: 5},
	}
	task.Rewards = []*TaskReward{
		{Type: TaskRewardGold, Amount: 1000},
	}
	_ = tm.AcceptTask(task)

	tm.UpdateProgress(1001, TaskCondKillMonster, 100, 5)
	_, _ = tm.CompleteTask(1)

	err := tm.AbandonTask(1)
	assert.Error(t, err)
}

func TestTaskManagerGetTasksByStatus(t *testing.T) {
	tm := NewTaskManager(20)

	task1 := NewTask(1, 1001, "Task1", TaskTypeMain)
	_ = tm.AcceptTask(task1)

	task2 := NewTask(2, 1002, "Task2", TaskTypeSide)
	task2.Conditions = []*TaskCondition{
		{Type: TaskCondKillMonster, TargetID: 100, Current: 0, Required: 1},
	}
	_ = tm.AcceptTask(task2)
	tm.UpdateProgress(1002, TaskCondKillMonster, 100, 1)

	inProgress := tm.GetTasksByStatus(TaskStatusInProgress)
	assert.Len(t, inProgress, 1)

	completed := tm.GetTasksByStatus(TaskStatusCompleted)
	assert.Len(t, completed, 1)
}

func TestTaskConditionIsComplete(t *testing.T) {
	cond := &TaskCondition{Type: TaskCondKillMonster, TargetID: 100, Current: 5, Required: 5}
	assert.True(t, cond.IsComplete())

	cond.Current = 3
	assert.False(t, cond.IsComplete())
}

func TestTaskConditionAddProgress(t *testing.T) {
	cond := &TaskCondition{Type: TaskCondKillMonster, TargetID: 100, Current: 3, Required: 5}

	cond.AddProgress(2)
	assert.Equal(t, int32(5), cond.Current)

	cond.AddProgress(10)
	assert.Equal(t, int32(5), cond.Current)
}
