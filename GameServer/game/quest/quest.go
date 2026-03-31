package quest

import (
	"sync"

	"github.com/pzqf/zCommon/common/id"
)

// QuestType 任务类型
type QuestType int32

const (
	QuestTypeMain        QuestType = 1 // 主线任务
	QuestTypeSide        QuestType = 2 // 支线任务
	QuestTypeDaily       QuestType = 3 // 日常任务
	QuestTypeWeekly      QuestType = 4 // 周常任务
	QuestTypeAchievement QuestType = 5 // 成就任务
	QuestTypeGuild       QuestType = 6 // 公会任务
)

// QuestStatus 任务状态
type QuestStatus int32

const (
	QuestStatusNotAccepted QuestStatus = 0 // 未接任务
	QuestStatusInProgress  QuestStatus = 1 // 进行中
	QuestStatusCompleted   QuestStatus = 2 // 已完成
	QuestStatusSubmitted   QuestStatus = 3 // 已提交
	QuestStatusFailed      QuestStatus = 4 // 已失败
)

// QuestTargetType 任务目标类型
type QuestTargetType int32

const (
	QuestTargetTypeKill    QuestTargetType = 1 // 击杀目标
	QuestTargetTypeCollect QuestTargetType = 2 // 收集物品
	QuestTargetTypeTalk    QuestTargetType = 3 // 对话
	QuestTargetTypeReach   QuestTargetType = 4 // 到达地点
	QuestTargetTypeLevel   QuestTargetType = 5 // 达到等级
	QuestTargetTypeUseItem QuestTargetType = 6 // 使用物品
	QuestTargetTypeSkill   QuestTargetType = 7 // 学习技能
	QuestTargetTypeExplore QuestTargetType = 8 // 探索
)

// QuestTarget 任务目标
type QuestTarget struct {
	TargetType QuestTargetType
	TargetID   int32
	TargetName string
	Count      int32
	Current    int32
}

// QuestReward 任务奖励
type QuestReward struct {
	Exp     int64
	Gold    int64
	Diamond int64
	Items   map[int32]int32 // itemConfigID -> count
}

// QuestCondition 任务条件
type QuestCondition struct {
	RequireLevel   int32
	RequireQuestID int32
	RequireGuildID id.GuildIdType
	MaxLevel       int32
	TimeLimit      int64 // 毫秒
}

// Quest 任务结构
type Quest struct {
	mu            sync.RWMutex
	questID       id.QuestIdType
	questConfigID int32
	name          string
	description   string
	questType     QuestType
	status        QuestStatus
	targets       []*QuestTarget
	rewards       *QuestReward
	conditions    *QuestCondition
	acceptTime    int64
	completeTime  int64
	submitTime    int64
	refreshTime   int64 // 刷新时间（日常和周常任务）
}

// NewQuest 创建新任务
func NewQuest(questConfigID int32, name string, questType QuestType) *Quest {
	return &Quest{
		questConfigID: questConfigID,
		name:          name,
		questType:     questType,
		status:        QuestStatusNotAccepted,
		targets:       make([]*QuestTarget, 0),
		rewards: &QuestReward{
			Items: make(map[int32]int32),
		},
		conditions: &QuestCondition{},
	}
}

// GetQuestID 获取任务ID
func (q *Quest) GetQuestID() id.QuestIdType {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.questID
}

// SetQuestID 设置任务ID
func (q *Quest) SetQuestID(questID id.QuestIdType) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.questID = questID
}

// GetQuestConfigID 获取任务配置ID
func (q *Quest) GetQuestConfigID() int32 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.questConfigID
}

// GetName 获取任务名称
func (q *Quest) GetName() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.name
}

// GetDescription 获取任务描述
func (q *Quest) GetDescription() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.description
}

// SetDescription 设置任务描述
func (q *Quest) SetDescription(description string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.description = description
}

// GetQuestType 获取任务类型
func (q *Quest) GetQuestType() QuestType {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.questType
}

// GetStatus 获取任务状态
func (q *Quest) GetStatus() QuestStatus {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.status
}

// SetStatus 设置任务状态
func (q *Quest) SetStatus(status QuestStatus) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.status = status
}

// GetTargets 获取任务目标
func (q *Quest) GetTargets() []*QuestTarget {
	q.mu.RLock()
	defer q.mu.RUnlock()
	result := make([]*QuestTarget, len(q.targets))
	copy(result, q.targets)
	return result
}

// AddTarget 添加任务目标
func (q *Quest) AddTarget(target *QuestTarget) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.targets = append(q.targets, target)
}

// GetRewards 获取任务奖励
func (q *Quest) GetRewards() *QuestReward {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.rewards
}

// GetConditions 获取任务条件
func (q *Quest) GetConditions() *QuestCondition {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.conditions
}

// GetAcceptTime 获取接取时间
func (q *Quest) GetAcceptTime() int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.acceptTime
}

// GetCompleteTime 获取完成时间
func (q *Quest) GetCompleteTime() int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.completeTime
}

// GetSubmitTime 获取提交时间
func (q *Quest) GetSubmitTime() int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.submitTime
}

// GetRefreshTime 获取刷新时间
func (q *Quest) GetRefreshTime() int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.refreshTime
}

// SetRefreshTime 设置刷新时间
func (q *Quest) SetRefreshTime(refreshTime int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.refreshTime = refreshTime
}

// CanAccept 检查是否可以接取任务
func (q *Quest) CanAccept(playerLevel int32, completedQuestIDs map[int32]bool) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.status != QuestStatusNotAccepted {
		return false
	}

	// 检查等级要求
	if playerLevel < q.conditions.RequireLevel {
		return false
	}

	// 检查等级上限
	if q.conditions.MaxLevel > 0 && playerLevel > q.conditions.MaxLevel {
		return false
	}

	// 检查前置任务
	if q.conditions.RequireQuestID > 0 {
		if !completedQuestIDs[q.conditions.RequireQuestID] {
			return false
		}
	}

	return true
}

// Accept 接取任务
func (q *Quest) Accept() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.status != QuestStatusNotAccepted {
		return false
	}

	q.status = QuestStatusInProgress
	q.acceptTime = 0

	// 重置目标进度
	for _, target := range q.targets {
		target.Current = 0
	}

	return true
}

// IsCompleted 检查是否已完成
func (q *Quest) IsCompleted() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.status != QuestStatusInProgress {
		return false
	}

	for _, target := range q.targets {
		if target.Current < target.Count {
			return false
		}
	}

	return true
}

// Complete 完成任务
func (q *Quest) Complete() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.status != QuestStatusInProgress {
		return false
	}

	if !q.isAllTargetsCompleted() {
		return false
	}

	q.status = QuestStatusCompleted
	q.completeTime = 0

	return true
}

// Submit 提交任务
func (q *Quest) Submit() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.status != QuestStatusCompleted {
		return false
	}

	q.status = QuestStatusSubmitted
	q.submitTime = 0

	return true
}

// UpdateTargetProgress 更新目标进度
func (q *Quest) UpdateTargetProgress(targetType QuestTargetType, targetID int32, count int32) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.status != QuestStatusInProgress {
		return false
	}

	for _, target := range q.targets {
		if target.TargetType == targetType && target.TargetID == targetID {
			target.Current += count
			if target.Current > target.Count {
				target.Current = target.Count
			}
			return true
		}
	}

	return false
}

// GetTargetProgress 获取目标进度
func (q *Quest) GetTargetProgress(targetType QuestTargetType, targetID int32) (current int32, total int32, found bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, target := range q.targets {
		if target.TargetType == targetType && target.TargetID == targetID {
			return target.Current, target.Count, true
		}
	}

	return 0, 0, false
}

// GetTotalProgress 获取总进度
func (q *Quest) GetTotalProgress() (current int32, total int32) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	total = int32(len(q.targets))
	if total == 0 {
		return 0, 0
	}

	completed := int32(0)
	for _, target := range q.targets {
		if target.Current >= target.Count {
			completed++
		}
	}

	return completed, total
}

// IsExpired 检查是否过期
func (q *Quest) IsExpired(currentTime int64) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.conditions.TimeLimit <= 0 {
		return false
	}

	if q.status == QuestStatusNotAccepted {
		return false
	}

	return currentTime-q.acceptTime > q.conditions.TimeLimit
}

// IsDaily 检查是否是日常任务
func (q *Quest) IsDaily() bool {
	return q.questType == QuestTypeDaily
}

// IsWeekly 检查是否是周常任务
func (q *Quest) IsWeekly() bool {
	return q.questType == QuestTypeWeekly
}

// NeedRefresh 检查是否需要刷新
func (q *Quest) NeedRefresh(currentTime int64) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if !q.IsDaily() && !q.IsWeekly() {
		return false
	}

	if q.refreshTime <= 0 {
		return false
	}

	return currentTime >= q.refreshTime
}

// Reset 重置任务（用于日常和周常任务）
func (q *Quest) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.status = QuestStatusNotAccepted
	q.acceptTime = 0
	q.completeTime = 0
	q.submitTime = 0

	for _, target := range q.targets {
		target.Current = 0
	}
}

// isAllTargetsCompleted 检查所有目标是否已完成
func (q *Quest) isAllTargetsCompleted() bool {
	for _, target := range q.targets {
		if target.Current < target.Count {
			return false
		}
	}
	return true
}

// Clone 克隆任务
func (q *Quest) Clone() *Quest {
	q.mu.RLock()
	defer q.mu.RUnlock()

	clone := &Quest{
		questConfigID: q.questConfigID,
		name:          q.name,
		description:   q.description,
		questType:     q.questType,
		status:        q.status,
		targets:       make([]*QuestTarget, len(q.targets)),
		rewards: &QuestReward{
			Exp:     q.rewards.Exp,
			Gold:    q.rewards.Gold,
			Diamond: q.rewards.Diamond,
			Items:   make(map[int32]int32),
		},
		conditions: &QuestCondition{
			RequireLevel:   q.conditions.RequireLevel,
			RequireQuestID: q.conditions.RequireQuestID,
			RequireGuildID: q.conditions.RequireGuildID,
			MaxLevel:       q.conditions.MaxLevel,
			TimeLimit:      q.conditions.TimeLimit,
		},
		acceptTime:   q.acceptTime,
		completeTime: q.completeTime,
		submitTime:   q.submitTime,
		refreshTime:  q.refreshTime,
	}

	copy(clone.targets, q.targets)
	for k, v := range q.rewards.Items {
		clone.rewards.Items[k] = v
	}

	return clone
}
