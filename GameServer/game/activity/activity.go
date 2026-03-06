package activity

import (
	"time"

	"github.com/pzqf/zMmoShared/common/id"
)

// Activity 活动结构
type Activity struct {
	ActivityID   id.ActivityIdType // 活动ID
	Name         string            // 活动名称
	Description  string            // 活动描述
	Type         int               // 活动类型
	Status       int               // 活动状态
	StartTime    time.Time         // 开始时间
	EndTime      time.Time         // 结束时间
	MinLevel     int               // 最低等级要求
	MaxLevel     int               // 最高等级限制
	MaxPlayers   int               // 最大参与人数
	RewardType   int               // 奖励类型
	RewardValue  int64             // 奖励数值
	IsRepeatable bool              // 是否可重复参与
	RepeatCount  int               // 已重复次数
	MaxRepeat    int               // 最大重复次数
}

// ActivityType 活动类型常量
const (
	ActivityTypeDaily   = 1 // 日常活动
	ActivityTypeWeekly  = 2 // 周常活动
	ActivityTypeMonthly = 3 // 月常活动
	ActivityTypeSpecial = 4 // 特殊活动
	ActivityTypeHoliday = 5 // 节日活动
	ActivityTypeLimited = 6 // 限时活动
)

// ActivityStatus 活动状态常量
const (
	ActivityStatusNotStarted = 0 // 未开始
	ActivityStatusRunning    = 1 // 进行中
	ActivityStatusPaused     = 2 // 暂停
	ActivityStatusEnded      = 3 // 已结束
)

// PlayerActivity 玩家活动参与记录
type PlayerActivity struct {
	PlayerID      id.PlayerIdType   // 玩家ID
	ActivityID    id.ActivityIdType // 活动ID
	JoinTime      time.Time         // 参与时间
	CompleteTime  time.Time         // 完成时间
	Progress      int               // 当前进度
	IsCompleted   bool              // 是否完成
	RewardClaimed bool              // 奖励是否领取
}

// NewActivity 创建新活动
func NewActivity(activityID id.ActivityIdType, name, description string, activityType int, startTime, endTime time.Time) *Activity {
	return &Activity{
		ActivityID:   activityID,
		Name:         name,
		Description:  description,
		Type:         activityType,
		Status:       ActivityStatusNotStarted,
		StartTime:    startTime,
		EndTime:      endTime,
		MinLevel:     1,
		MaxLevel:     999,
		MaxPlayers:   0, // 0表示无限制
		IsRepeatable: false,
		RepeatCount:  0,
		MaxRepeat:    1,
	}
}

// NewPlayerActivity 创建玩家活动记录
func NewPlayerActivity(playerID id.PlayerIdType, activityID id.ActivityIdType) *PlayerActivity {
	return &PlayerActivity{
		PlayerID:      playerID,
		ActivityID:    activityID,
		JoinTime:      time.Now(),
		Progress:      0,
		IsCompleted:   false,
		RewardClaimed: false,
	}
}

// IsActive 检查活动是否进行中
func (a *Activity) IsActive() bool {
	now := time.Now()
	return a.Status == ActivityStatusRunning && now.After(a.StartTime) && now.Before(a.EndTime)
}

// CanJoin 检查玩家是否可以参与活动
func (a *Activity) CanJoin(playerLevel int, currentPlayers int) bool {
	if !a.IsActive() {
		return false
	}

	if playerLevel < a.MinLevel || playerLevel > a.MaxLevel {
		return false
	}

	if a.MaxPlayers > 0 && currentPlayers >= a.MaxPlayers {
		return false
	}

	return true
}

// UpdateProgress 更新活动进度
func (pa *PlayerActivity) UpdateProgress(progress int) bool {
	if pa.IsCompleted {
		return false
	}

	pa.Progress += progress
	return true
}

// Complete 完成活动
func (pa *PlayerActivity) Complete() {
	pa.IsCompleted = true
	pa.CompleteTime = time.Now()
}

// ClaimReward 领取奖励
func (pa *PlayerActivity) ClaimReward() bool {
	if !pa.IsCompleted || pa.RewardClaimed {
		return false
	}

	pa.RewardClaimed = true
	return true
}
