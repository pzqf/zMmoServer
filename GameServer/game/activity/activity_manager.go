package activity

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// ActivityManager 活动管理器
type ActivityManager struct {
	activities       map[id.ActivityIdType]*Activity
	playerActivities map[id.PlayerIdType]map[id.ActivityIdType]*PlayerActivity
	mutex            sync.RWMutex
}

// NewActivityManager 创建活动管理器
func NewActivityManager() *ActivityManager {
	return &ActivityManager{
		activities:       make(map[id.ActivityIdType]*Activity),
		playerActivities: make(map[id.PlayerIdType]map[id.ActivityIdType]*PlayerActivity),
	}
}

// AddActivity 添加活动
func (am *ActivityManager) AddActivity(activity *Activity) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.activities[activity.ActivityID] = activity
	zLog.Info("Activity added",
		zap.Uint64("activity_id", uint64(activity.ActivityID)),
		zap.String("name", activity.Name),
		zap.Int("type", activity.Type))
}

// GetActivity 获取活动
func (am *ActivityManager) GetActivity(activityID id.ActivityIdType) *Activity {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	return am.activities[activityID]
}

// GetAllActivities 获取所有活动
func (am *ActivityManager) GetAllActivities() []*Activity {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	activities := make([]*Activity, 0, len(am.activities))
	for _, activity := range am.activities {
		activities = append(activities, activity)
	}

	return activities
}

// GetActiveActivities 获取当前进行中的活动
func (am *ActivityManager) GetActiveActivities() []*Activity {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	activities := make([]*Activity, 0)
	for _, activity := range am.activities {
		if activity.IsActive() {
			activities = append(activities, activity)
		}
	}

	return activities
}

// JoinActivity 玩家参与活动
func (am *ActivityManager) JoinActivity(playerID id.PlayerIdType, activityID id.ActivityIdType, playerLevel int) bool {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	activity := am.activities[activityID]
	if activity == nil {
		return false
	}

	if !activity.CanJoin(playerLevel, am.getActivityPlayerCount(activityID)) {
		return false
	}

	// 初始化玩家活动记录
	if _, exists := am.playerActivities[playerID]; !exists {
		am.playerActivities[playerID] = make(map[id.ActivityIdType]*PlayerActivity)
	}

	// 检查是否已参与
	if _, exists := am.playerActivities[playerID][activityID]; exists {
		return false
	}

	playerActivity := NewPlayerActivity(playerID, activityID)
	am.playerActivities[playerID][activityID] = playerActivity

	zLog.Info("Player joined activity",
		zap.Uint64("player_id", uint64(playerID)),
		zap.Uint64("activity_id", uint64(activityID)),
		zap.String("activity_name", activity.Name))

	return true
}

// UpdateActivityProgress 更新活动进度
func (am *ActivityManager) UpdateActivityProgress(playerID id.PlayerIdType, activityID id.ActivityIdType, progress int) bool {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if playerActivities, exists := am.playerActivities[playerID]; exists {
		if playerActivity, exists := playerActivities[activityID]; exists {
			if playerActivity.UpdateProgress(progress) {
				activity := am.activities[activityID]
				if activity != nil && playerActivity.Progress >= 100 {
					playerActivity.Complete()
					zLog.Info("Player completed activity",
						zap.Uint64("player_id", uint64(playerID)),
						zap.Uint64("activity_id", uint64(activityID)),
						zap.String("activity_name", activity.Name))
				}
				return true
			}
		}
	}

	return false
}

// ClaimActivityReward 领取活动奖励
func (am *ActivityManager) ClaimActivityReward(playerID id.PlayerIdType, activityID id.ActivityIdType) bool {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if playerActivities, exists := am.playerActivities[playerID]; exists {
		if playerActivity, exists := playerActivities[activityID]; exists {
			if playerActivity.ClaimReward() {
				activity := am.activities[activityID]
				zLog.Info("Player claimed activity reward",
					zap.Uint64("player_id", uint64(playerID)),
					zap.Uint64("activity_id", uint64(activityID)),
					zap.String("activity_name", activity.Name))
				return true
			}
		}
	}

	return false
}

// GetPlayerActivities 获取玩家活动记录
func (am *ActivityManager) GetPlayerActivities(playerID id.PlayerIdType) []*PlayerActivity {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if playerActivities, exists := am.playerActivities[playerID]; exists {
		activities := make([]*PlayerActivity, 0, len(playerActivities))
		for _, activity := range playerActivities {
			activities = append(activities, activity)
		}
		return activities
	}

	return []*PlayerActivity{}
}

// GetPlayerActivity 获取玩家指定活动记录
func (am *ActivityManager) GetPlayerActivity(playerID id.PlayerIdType, activityID id.ActivityIdType) *PlayerActivity {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if playerActivities, exists := am.playerActivities[playerID]; exists {
		return playerActivities[activityID]
	}

	return nil
}

// UpdateActivityStatus 更新活动状态
func (am *ActivityManager) UpdateActivityStatus(activityID id.ActivityIdType, status int) bool {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if activity, exists := am.activities[activityID]; exists {
		activity.Status = status
		zLog.Info("Activity status updated",
			zap.Uint64("activity_id", uint64(activityID)),
			zap.Int("status", status))
		return true
	}

	return false
}

// getActivityPlayerCount 获取活动参与人数
func (am *ActivityManager) getActivityPlayerCount(activityID id.ActivityIdType) int {
	count := 0
	for _, playerActivities := range am.playerActivities {
		if _, exists := playerActivities[activityID]; exists {
			count++
		}
	}
	return count
}

// CheckAndUpdateActivities 检查并更新活动状态
func (am *ActivityManager) CheckAndUpdateActivities() {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	now := time.Now()
	for _, activity := range am.activities {
		if activity.Status == ActivityStatusNotStarted && now.After(activity.StartTime) {
			activity.Status = ActivityStatusRunning
			zLog.Info("Activity started",
				zap.Uint64("activity_id", uint64(activity.ActivityID)),
				zap.String("name", activity.Name))
		} else if activity.Status == ActivityStatusRunning && now.After(activity.EndTime) {
			activity.Status = ActivityStatusEnded
			zLog.Info("Activity ended",
				zap.Uint64("activity_id", uint64(activity.ActivityID)),
				zap.String("name", activity.Name))
		}
	}
}
