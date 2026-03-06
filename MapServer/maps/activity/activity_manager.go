package activity

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

type ActivityType int

const (
	ActivityTypeNone ActivityType = iota
	ActivityTypeDaily
	ActivityTypeWeekly
	ActivityTypeMonthly
	ActivityTypeSpecial
	ActivityTypeFestival
)

type ActivityStatus int

const (
	ActivityStatusNone ActivityStatus = iota
	ActivityStatusNotStarted
	ActivityStatusInProgress
	ActivityStatusEnded
)

type ActivityConfig struct {
	ActivityID      int32
	Name            string
	Description     string
	Type            ActivityType
	StartTime       time.Time
	EndTime         time.Time
	MinLevel        int32
	MaxLevel        int32
	MaxParticipants int32
	Rewards         []ActivityReward
	Conditions      []ActivityCondition
}

type ActivityReward struct {
	RewardID   int32
	Type       string
	ItemID     int32
	Count      int32
	Probability float32
}

type ActivityCondition struct {
	ConditionType string
	TargetID      int32
	RequiredCount int32
}

type PlayerActivity struct {
	ActivityID    int32
	PlayerID      id.PlayerIdType
	Progress      map[string]int32
	IsCompleted   bool
	IsRewarded    bool
	JoinTime      time.Time
	CompleteTime  time.Time
	LastUpdateTime time.Time
}

type Activity struct {
	Config       *ActivityConfig
	Status       ActivityStatus
	Participants map[id.PlayerIdType]*PlayerActivity
	StartTime    time.Time
	EndTime      time.Time
}

type ActivityManager struct {
	mu          sync.RWMutex
	activities  map[int32]*Activity
	playerActivities map[id.PlayerIdType]map[int32]*PlayerActivity
}

var globalActivityManager *ActivityManager
var activityOnce sync.Once

func NewActivityManager() *ActivityManager {
	return &ActivityManager{
		activities:       make(map[int32]*Activity),
		playerActivities: make(map[id.PlayerIdType]map[int32]*PlayerActivity),
	}
}

func GetActivityManager() *ActivityManager {
	if globalActivityManager == nil {
		activityOnce.Do(func() {
			globalActivityManager = NewActivityManager()
		})
	}
	return globalActivityManager
}

func (am *ActivityManager) CreateActivity(config *ActivityConfig) (*Activity, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.activities[config.ActivityID]; exists {
		return nil, fmt.Errorf("activity already exists: %d", config.ActivityID)
	}

	activity := &Activity{
		Config:       config,
		Status:       ActivityStatusNotStarted,
		Participants: make(map[id.PlayerIdType]*PlayerActivity),
		StartTime:    config.StartTime,
		EndTime:      config.EndTime,
	}

	am.activities[config.ActivityID] = activity

	zLog.Info("Activity created",
		zap.Int32("activity_id", config.ActivityID),
		zap.String("name", config.Name))

	return activity, nil
}

func (am *ActivityManager) RemoveActivity(activityID int32) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.activities[activityID]; !exists {
		return fmt.Errorf("activity not found: %d", activityID)
	}

	delete(am.activities, activityID)

	zLog.Info("Activity removed", zap.Int32("activity_id", activityID))
	return nil
}

func (am *ActivityManager) GetActivity(activityID int32) (*Activity, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	activity, ok := am.activities[activityID]
	return activity, ok
}

func (am *ActivityManager) GetActiveActivities() []*Activity {
	am.mu.RLock()
	defer am.mu.RUnlock()

	active := make([]*Activity, 0)
	for _, activity := range am.activities {
		if activity.Status == ActivityStatusInProgress {
			active = append(active, activity)
		}
	}
	return active
}

func (am *ActivityManager) StartActivity(activityID int32) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	activity, exists := am.activities[activityID]
	if !exists {
		return fmt.Errorf("activity not found: %d", activityID)
	}

	if activity.Status == ActivityStatusInProgress {
		return fmt.Errorf("activity already in progress: %d", activityID)
	}

	activity.Status = ActivityStatusInProgress
	activity.StartTime = time.Now()

	zLog.Info("Activity started",
		zap.Int32("activity_id", activityID),
		zap.String("name", activity.Config.Name))

	return nil
}

func (am *ActivityManager) EndActivity(activityID int32) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	activity, exists := am.activities[activityID]
	if !exists {
		return fmt.Errorf("activity not found: %d", activityID)
	}

	activity.Status = ActivityStatusEnded
	activity.EndTime = time.Now()

	zLog.Info("Activity ended",
		zap.Int32("activity_id", activityID),
		zap.String("name", activity.Config.Name))

	return nil
}

func (am *ActivityManager) JoinActivity(playerID id.PlayerIdType, activityID int32) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	activity, exists := am.activities[activityID]
	if !exists {
		return fmt.Errorf("activity not found: %d", activityID)
	}

	if activity.Status != ActivityStatusInProgress {
		return fmt.Errorf("activity is not in progress: %d", activityID)
	}

	if _, exists := activity.Participants[playerID]; exists {
		return fmt.Errorf("player already joined: %d", playerID)
	}

	if activity.Config.MaxParticipants > 0 && int32(len(activity.Participants)) >= activity.Config.MaxParticipants {
		return fmt.Errorf("activity is full: %d", activityID)
	}

	playerActivity := &PlayerActivity{
		ActivityID:     activityID,
		PlayerID:       playerID,
		Progress:       make(map[string]int32),
		IsCompleted:    false,
		IsRewarded:     false,
		JoinTime:       time.Now(),
		LastUpdateTime: time.Now(),
	}

	activity.Participants[playerID] = playerActivity

	if _, exists := am.playerActivities[playerID]; !exists {
		am.playerActivities[playerID] = make(map[int32]*PlayerActivity)
	}
	am.playerActivities[playerID][activityID] = playerActivity

	zLog.Debug("Player joined activity",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("activity_id", activityID))

	return nil
}

func (am *ActivityManager) LeaveActivity(playerID id.PlayerIdType, activityID int32) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	activity, exists := am.activities[activityID]
	if !exists {
		return fmt.Errorf("activity not found: %d", activityID)
	}

	if _, exists := activity.Participants[playerID]; !exists {
		return fmt.Errorf("player not in activity: %d", playerID)
	}

	delete(activity.Participants, playerID)

	if playerActivities, exists := am.playerActivities[playerID]; exists {
		delete(playerActivities, activityID)
	}

	zLog.Debug("Player left activity",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("activity_id", activityID))

	return nil
}

func (am *ActivityManager) UpdateProgress(playerID id.PlayerIdType, activityID int32, conditionType string, count int32) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	activity, exists := am.activities[activityID]
	if !exists {
		return fmt.Errorf("activity not found: %d", activityID)
	}

	playerActivity, exists := activity.Participants[playerID]
	if !exists {
		return fmt.Errorf("player not in activity: %d", playerID)
	}

	playerActivity.Progress[conditionType] += count
	playerActivity.LastUpdateTime = time.Now()

	am.checkCompletion(activity, playerActivity)

	zLog.Debug("Activity progress updated",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("activity_id", activityID),
		zap.String("condition", conditionType),
		zap.Int32("count", count))

	return nil
}

func (am *ActivityManager) checkCompletion(activity *Activity, playerActivity *PlayerActivity) {
	allConditionsMet := true

	for _, condition := range activity.Config.Conditions {
		progress, exists := playerActivity.Progress[condition.ConditionType]
		if !exists || progress < condition.RequiredCount {
			allConditionsMet = false
			break
		}
	}

	if allConditionsMet && !playerActivity.IsCompleted {
		playerActivity.IsCompleted = true
		playerActivity.CompleteTime = time.Now()

		zLog.Info("Activity completed by player",
			zap.Int64("player_id", int64(playerActivity.PlayerID)),
			zap.Int32("activity_id", activity.Config.ActivityID))
	}
}

func (am *ActivityManager) ClaimReward(playerID id.PlayerIdType, activityID int32) ([]ActivityReward, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	activity, exists := am.activities[activityID]
	if !exists {
		return nil, fmt.Errorf("activity not found: %d", activityID)
	}

	playerActivity, exists := activity.Participants[playerID]
	if !exists {
		return nil, fmt.Errorf("player not in activity: %d", playerID)
	}

	if !playerActivity.IsCompleted {
		return nil, fmt.Errorf("activity not completed")
	}

	if playerActivity.IsRewarded {
		return nil, fmt.Errorf("reward already claimed")
	}

	playerActivity.IsRewarded = true

	zLog.Info("Activity reward claimed",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("activity_id", activityID))

	return activity.Config.Rewards, nil
}

func (am *ActivityManager) GetPlayerActivities(playerID id.PlayerIdType) []*PlayerActivity {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if _, exists := am.playerActivities[playerID]; !exists {
		return []*PlayerActivity{}
	}

	activities := make([]*PlayerActivity, 0, len(am.playerActivities[playerID]))
	for _, pa := range am.playerActivities[playerID] {
		activities = append(activities, pa)
	}

	return activities
}

func (am *ActivityManager) GetPlayerActivity(playerID id.PlayerIdType, activityID int32) (*PlayerActivity, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if _, exists := am.playerActivities[playerID]; !exists {
		return nil, false
	}

	pa, exists := am.playerActivities[playerID][activityID]
	return pa, exists
}

func (am *ActivityManager) Update(deltaTime time.Duration) {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()

	for _, activity := range am.activities {
		switch activity.Status {
		case ActivityStatusNotStarted:
			if now.After(activity.Config.StartTime) || now.Equal(activity.Config.StartTime) {
				activity.Status = ActivityStatusInProgress
				activity.StartTime = now
				zLog.Info("Activity auto-started",
					zap.Int32("activity_id", activity.Config.ActivityID))
			}
		case ActivityStatusInProgress:
			if now.After(activity.Config.EndTime) {
				activity.Status = ActivityStatusEnded
				activity.EndTime = now
				zLog.Info("Activity auto-ended",
					zap.Int32("activity_id", activity.Config.ActivityID))
			}
		}
	}
}

func (am *ActivityManager) GetActivityStatus(activityID int32) ActivityStatus {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if activity, exists := am.activities[activityID]; exists {
		return activity.Status
	}
	return ActivityStatusNone
}

func (am *ActivityManager) GetParticipantCount(activityID int32) int {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if activity, exists := am.activities[activityID]; exists {
		return len(activity.Participants)
	}
	return 0
}

func (am *ActivityManager) IsPlayerInActivity(playerID id.PlayerIdType, activityID int32) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if activity, exists := am.activities[activityID]; exists {
		_, inActivity := activity.Participants[playerID]
		return inActivity
	}
	return false
}

func (a *Activity) GetProgress(playerID id.PlayerIdType) map[string]int32 {
	if pa, exists := a.Participants[playerID]; exists {
		return pa.Progress
	}
	return make(map[string]int32)
}

func (a *Activity) IsCompleted(playerID id.PlayerIdType) bool {
	if pa, exists := a.Participants[playerID]; exists {
		return pa.IsCompleted
	}
	return false
}

func (a *Activity) GetRemainingTime() time.Duration {
	if a.Status != ActivityStatusInProgress {
		return 0
	}

	remaining := time.Until(a.Config.EndTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (pa *PlayerActivity) GetProgressPercent(conditionType string, required int32) float32 {
	if required <= 0 {
		return 100.0
	}
	progress, exists := pa.Progress[conditionType]
	if !exists {
		return 0.0
	}
	percent := float32(progress) / float32(required) * 100.0
	if percent > 100.0 {
		percent = 100.0
	}
	return percent
}
