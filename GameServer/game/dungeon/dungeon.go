package dungeon

import (
	"time"

	"github.com/pzqf/zCommon/common/id"
)

// Dungeon 副本结构
type Dungeon struct {
	DungeonID   id.DungeonIdType // 副本ID
	Name        string           // 副本名称
	Description string           // 副本描述
	Type        int              // 副本类型
	Difficulty  int              // 难度等级
	MinLevel    int              // 最低等级要求
	MaxLevel    int              // 最高等级限制
	MinPlayers  int              // 最少人数
	MaxPlayers  int              // 最多人数
	TimeLimit   int              // 时间限制（分钟）
	DailyLimit  int              // 每日次数限制
	RewardExp   int64            // 经验奖励
	RewardGold  int64            // 金币奖励
	RewardItems []int64          // 物品奖励ID列表
	BossCount   int              // Boss数量
	IsOpen      bool             // 是否开放
	OpenTime    time.Time        // 开放时间
	CloseTime   time.Time        // 关闭时间
}

// DungeonType 副本类型常量
const (
	DungeonTypeNormal    = 1 // 普通副本
	DungeonTypeElite     = 2 // 精英副本
	DungeonTypeBoss      = 3 // Boss副本
	DungeonTypeTeam      = 4 // 团队副本
	DungeonTypeChallenge = 5 // 挑战副本
	DungeonTypeEndless   = 6 // 无尽副本
)

// DungeonInstance 副本实例
type DungeonInstance struct {
	InstanceID   id.InstanceIdType                  // 实例ID
	DungeonID    id.DungeonIdType                   // 副本ID
	LeaderID     id.PlayerIdType                    // 队长ID
	Players      map[id.PlayerIdType]*DungeonPlayer // 参与玩家
	Status       int                                // 实例状态
	StartTime    time.Time                          // 开始时间
	EndTime      time.Time                          // 结束时间
	Progress     int                                // 当前进度（百分比）
	KilledBosses []int                              // 已击杀Boss列表
}

// DungeonPlayer 副本玩家信息
type DungeonPlayer struct {
	PlayerID    id.PlayerIdType // 玩家ID
	PlayerName  string          // 玩家名称
	Level       int             // 等级
	Class       int             // 职业
	JoinTime    time.Time       // 加入时间
	Damage      int64           // 造成伤害
	IsDead      bool            // 是否死亡
	ReviveCount int             // 复活次数
}

// PlayerDungeonRecord 玩家副本记录
type PlayerDungeonRecord struct {
	PlayerID      id.PlayerIdType  // 玩家ID
	DungeonID     id.DungeonIdType // 副本ID
	CompleteCount int              // 完成次数
	BestTime      int              // 最佳通关时间（秒�?
	LastEnterTime time.Time        // 上次进入时间
	DailyCount    int              // 今日次数
	WeeklyCount   int              // 本周次数
}

// InstanceStatus 副本实例状态常�?
const (
	InstanceStatusWaiting   = 0 // 等待�?
	InstanceStatusRunning   = 1 // 进行�?
	InstanceStatusPaused    = 2 // 暂停
	InstanceStatusCompleted = 3 // 已完�?
	InstanceStatusFailed    = 4 // 失败
)

// NewDungeon 创建新副�?
func NewDungeon(dungeonID id.DungeonIdType, name, description string, dungeonType int) *Dungeon {
	return &Dungeon{
		DungeonID:   dungeonID,
		Name:        name,
		Description: description,
		Type:        dungeonType,
		Difficulty:  1,
		MinLevel:    1,
		MaxLevel:    999,
		MinPlayers:  1,
		MaxPlayers:  5,
		TimeLimit:   30,
		DailyLimit:  3,
		RewardExp:   1000,
		RewardGold:  500,
		RewardItems: []int64{},
		BossCount:   1,
		IsOpen:      true,
	}
}

// NewDungeonInstance 创建副本实例
func NewDungeonInstance(instanceID id.InstanceIdType, dungeonID id.DungeonIdType, leaderID id.PlayerIdType) *DungeonInstance {
	return &DungeonInstance{
		InstanceID:   instanceID,
		DungeonID:    dungeonID,
		LeaderID:     leaderID,
		Players:      make(map[id.PlayerIdType]*DungeonPlayer),
		Status:       InstanceStatusWaiting,
		StartTime:    time.Time{},
		EndTime:      time.Time{},
		Progress:     0,
		KilledBosses: []int{},
	}
}

// AddPlayer 添加玩家
func (di *DungeonInstance) AddPlayer(playerID id.PlayerIdType, playerName string, level, class int) bool {
	if len(di.Players) >= 5 { // 假设最�?�?
		return false
	}

	di.Players[playerID] = &DungeonPlayer{
		PlayerID:    playerID,
		PlayerName:  playerName,
		Level:       level,
		Class:       class,
		JoinTime:    time.Now(),
		Damage:      0,
		IsDead:      false,
		ReviveCount: 0,
	}

	return true
}

// RemovePlayer 移除玩家
func (di *DungeonInstance) RemovePlayer(playerID id.PlayerIdType) {
	delete(di.Players, playerID)
}

// Start 开始副�?
func (di *DungeonInstance) Start() {
	di.Status = InstanceStatusRunning
	di.StartTime = time.Now()
}

// Complete 完成副本
func (di *DungeonInstance) Complete() {
	di.Status = InstanceStatusCompleted
	di.EndTime = time.Now()
	di.Progress = 100
}

// Fail 副本失败
func (di *DungeonInstance) Fail() {
	di.Status = InstanceStatusFailed
	di.EndTime = time.Now()
}

// IsRunning 是否进行�?
func (di *DungeonInstance) IsRunning() bool {
	return di.Status == InstanceStatusRunning
}

// GetDuration 获取副本持续时间（秒�?
func (di *DungeonInstance) GetDuration() int {
	if di.StartTime.IsZero() {
		return 0
	}
	endTime := di.EndTime
	if endTime.IsZero() {
		endTime = time.Now()
	}
	return int(endTime.Sub(di.StartTime).Seconds())
}

// CanEnter 检查玩家是否可以进入副�?
func (d *Dungeon) CanEnter(playerLevel int, dailyCount int) bool {
	if !d.IsOpen {
		return false
	}

	now := time.Now()
	if now.Before(d.OpenTime) || now.After(d.CloseTime) {
		return false
	}

	if playerLevel < d.MinLevel || playerLevel > d.MaxLevel {
		return false
	}

	if d.DailyLimit > 0 && dailyCount >= d.DailyLimit {
		return false
	}

	return true
}

