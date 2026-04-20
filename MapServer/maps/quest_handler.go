package maps

import (
	"fmt"

	"github.com/pzqf/zCommon/config/models"
	"github.com/pzqf/zMmoServer/MapServer/maps/economy"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"github.com/pzqf/zMmoServer/MapServer/maps/task"
)

func (m *Map) HandleTaskAccept(player *object.Player, taskID int32) error {
	if m.taskManager == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return m.taskManager.AcceptTask(player.GetPlayerID(), taskID)
}

func (m *Map) HandleTaskReward(player *object.Player, taskID int32) error {
	if m.taskManager == nil {
		return fmt.Errorf("task manager not initialized")
	}

	err := m.taskManager.RewardTask(player.GetPlayerID(), taskID)
	if err != nil {
		return err
	}

	taskConfig := m.taskManager.GetTaskConfig(taskID)
	if taskConfig != nil {
		rewards := m.taskManager.ParseTaskRewards(taskConfig.Rewards)
		for _, reward := range rewards {
			switch reward.Type {
			case "exp":
				player.AddExp(int64(reward.Count))
			case "gold":
				if err := m.AddCurrency(player, economy.CurrencyTypeGold, int64(reward.Count)); err != nil {
					return err
				}
			case "item":
				if err := m.AddItem(player, reward.Target, reward.Count); err != nil {
					return err
				}
			case "skill_point":
			}
		}
	}

	return nil
}

func (m *Map) UpdateTaskProgress(player *object.Player, conditionType string, target int32, count int32) {
	if m.taskManager != nil {
		m.taskManager.UpdateTaskProgress(player.GetPlayerID(), conditionType, target, count)
	}
}

func (m *Map) GetPlayerTasks(player *object.Player) []*task.PlayerTask {
	if m.taskManager == nil {
		return []*task.PlayerTask{}
	}
	return m.taskManager.GetPlayerTasks(player.GetPlayerID())
}

func (m *Map) GetAvailableTasks(player *object.Player) []*models.Quest {
	if m.taskManager == nil {
		return []*models.Quest{}
	}
	return m.taskManager.GetAvailableTasks(player.GetPlayerID(), player.GetLevel())
}
