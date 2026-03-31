package quest

import (
	"errors"
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

// QuestManager 任务管理器
type QuestManager struct {
	mu                sync.RWMutex
	playerID          id.PlayerIdType
	quests            *zMap.TypedMap[int32, *Quest]
	completedQuestIDs *zMap.TypedMap[int32, bool]
	maxAcceptCount    int32
	maxDailyCount     int32
	dailyAcceptCount  int32
	weeklyAcceptCount int32
}

// NewQuestManager 创建任务管理器
func NewQuestManager(playerID id.PlayerIdType, maxAcceptCount int32) *QuestManager {
	return &QuestManager{
		playerID:          playerID,
		quests:            zMap.NewTypedMap[int32, *Quest](),
		completedQuestIDs: zMap.NewTypedMap[int32, bool](),
		maxAcceptCount:    maxAcceptCount,
		maxDailyCount:     20,
		dailyAcceptCount:  0,
		weeklyAcceptCount: 0,
	}
}

// GetPlayerID 获取玩家ID
func (qm *QuestManager) GetPlayerID() id.PlayerIdType {
	return qm.playerID
}

// GetQuestCount 获取任务数量
func (qm *QuestManager) GetQuestCount() int32 {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	return int32(qm.quests.Len())
}

// GetMaxAcceptCount 获取最大接取数
func (qm *QuestManager) GetMaxAcceptCount() int32 {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	return qm.maxAcceptCount
}

// AddQuest 添加任务
func (qm *QuestManager) AddQuest(quest *Quest) error {
	if quest == nil {
		return errors.New("quest is nil")
	}

	qm.mu.Lock()
	defer qm.mu.Unlock()

	questConfigID := quest.GetQuestConfigID()

	if _, exists := qm.quests.Load(questConfigID); exists {
		return errors.New("quest already exists")
	}

	qm.quests.Store(questConfigID, quest)

	zLog.Debug("Quest added",
		zap.Int64("player_id", int64(qm.playerID)),
		zap.Int32("quest_config_id", questConfigID),
		zap.String("name", quest.GetName()))

	return nil
}

// RemoveQuest 移除任务
func (qm *QuestManager) RemoveQuest(questConfigID int32) (*Quest, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	quest, exists := qm.quests.Load(questConfigID)
	if !exists {
		return nil, errors.New("quest not found")
	}

	qm.quests.Delete(questConfigID)

	zLog.Debug("Quest removed",
		zap.Int64("player_id", int64(qm.playerID)),
		zap.Int32("quest_config_id", questConfigID))

	return quest, nil
}

// GetQuest 获取任务
func (qm *QuestManager) GetQuest(questConfigID int32) (*Quest, error) {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	quest, exists := qm.quests.Load(questConfigID)
	if !exists {
		return nil, errors.New("quest not found")
	}
	return quest, nil
}

// HasQuest 检查是否有指定任务
func (qm *QuestManager) HasQuest(questConfigID int32) bool {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	_, exists := qm.quests.Load(questConfigID)
	return exists
}

// GetAllQuests 获取所有任务
func (qm *QuestManager) GetAllQuests() map[int32]*Quest {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	result := make(map[int32]*Quest)
	qm.quests.Range(func(questConfigID int32, quest *Quest) bool {
		result[questConfigID] = quest
		return true
	})
	return result
}

// GetQuestsByType 获取指定类型的任务
func (qm *QuestManager) GetQuestsByType(questType QuestType) []*Quest {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	result := make([]*Quest, 0)
	qm.quests.Range(func(_ int32, quest *Quest) bool {
		if quest.GetQuestType() == questType {
			result = append(result, quest)
		}
		return true
	})
	return result
}

// GetQuestsByStatus 获取指定状态的任务
func (qm *QuestManager) GetQuestsByStatus(status QuestStatus) []*Quest {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	result := make([]*Quest, 0)
	qm.quests.Range(func(_ int32, quest *Quest) bool {
		if quest.GetStatus() == status {
			result = append(result, quest)
		}
		return true
	})
	return result
}

// GetInProgressQuests 获取进行中的任务
func (qm *QuestManager) GetInProgressQuests() []*Quest {
	return qm.GetQuestsByStatus(QuestStatusInProgress)
}

// GetCompletedQuests 获取已完成的任务
func (qm *QuestManager) GetCompletedQuests() []*Quest {
	return qm.GetQuestsByStatus(QuestStatusCompleted)
}

// AcceptQuest 接取任务
func (qm *QuestManager) AcceptQuest(quest *Quest, playerLevel int32) error {
	if quest == nil {
		return errors.New("quest is nil")
	}

	qm.mu.Lock()
	defer qm.mu.Unlock()

	questConfigID := quest.GetQuestConfigID()

	// 检查是否已存在
	if _, exists := qm.quests.Load(questConfigID); exists {
		return errors.New("quest already accepted")
	}

	// 检查接取数量限制
	count := 0
	qm.quests.Range(func(_ int32, _ *Quest) bool {
		count++
		return true
	})
	if int32(count) >= qm.maxAcceptCount {
		return errors.New("max accept count reached")
	}

	// 检查日常任务数限制
	if quest.IsDaily() && qm.dailyAcceptCount >= qm.maxDailyCount {
		return errors.New("max daily quest count reached")
	}

	// 检查是否可以接取
	completedIDs := make(map[int32]bool)
	qm.completedQuestIDs.Range(func(questID int32, _ bool) bool {
		completedIDs[questID] = true
		return true
	})
	if !quest.CanAccept(playerLevel, completedIDs) {
		return errors.New("cannot accept quest")
	}

	if !quest.Accept() {
		return errors.New("accept quest failed")
	}

	qm.quests.Store(questConfigID, quest)

	if quest.IsDaily() {
		qm.dailyAcceptCount++
	} else if quest.IsWeekly() {
		qm.weeklyAcceptCount++
	}

	qm.publishQuestAcceptEvent(quest)

	zLog.Info("Quest accepted",
		zap.Int64("player_id", int64(qm.playerID)),
		zap.Int32("quest_config_id", questConfigID),
		zap.String("name", quest.GetName()))

	return nil
}

// CompleteQuest 完成任务
func (qm *QuestManager) CompleteQuest(questConfigID int32) (*Quest, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	quest, exists := qm.quests.Load(questConfigID)
	if !exists {
		return nil, errors.New("quest not found")
	}

	if !quest.Complete() {
		return nil, errors.New("complete quest failed")
	}

	qm.publishQuestCompleteEvent(quest)

	zLog.Info("Quest completed",
		zap.Int64("player_id", int64(qm.playerID)),
		zap.Int32("quest_config_id", questConfigID),
		zap.String("name", quest.GetName()))

	return quest, nil
}

// SubmitQuest 提交任务
func (qm *QuestManager) SubmitQuest(questConfigID int32) (*Quest, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	quest, exists := qm.quests.Load(questConfigID)
	if !exists {
		return nil, errors.New("quest not found")
	}

	if !quest.Submit() {
		return nil, errors.New("submit quest failed")
	}

	qm.completedQuestIDs.Store(questConfigID, true)

	qm.publishQuestSubmitEvent(quest)

	zLog.Info("Quest submitted",
		zap.Int64("player_id", int64(qm.playerID)),
		zap.Int32("quest_config_id", questConfigID),
		zap.String("name", quest.GetName()))

	return quest, nil
}

// AbandonQuest 放弃任务
func (qm *QuestManager) AbandonQuest(questConfigID int32) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	quest, exists := qm.quests.Load(questConfigID)
	if !exists {
		return errors.New("quest not found")
	}

	// 已完成的任务不能放弃
	if quest.GetStatus() == QuestStatusCompleted || quest.GetStatus() == QuestStatusSubmitted {
		return errors.New("cannot abandon completed quest")
	}

	qm.quests.Delete(questConfigID)

	if quest.IsDaily() {
		qm.dailyAcceptCount--
	} else if quest.IsWeekly() {
		qm.weeklyAcceptCount--
	}

	zLog.Info("Quest abandoned",
		zap.Int64("player_id", int64(qm.playerID)),
		zap.Int32("quest_config_id", questConfigID),
		zap.String("name", quest.GetName()))

	return nil
}

// UpdateTargetProgress 更新任务目标进度
func (qm *QuestManager) UpdateTargetProgress(targetType QuestTargetType, targetID int32, count int32) int32 {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	updatedCount := int32(0)
	qm.quests.Range(func(_ int32, quest *Quest) bool {
		if quest.GetStatus() == QuestStatusInProgress {
			if quest.UpdateTargetProgress(targetType, targetID, count) {
				updatedCount++
				qm.publishQuestProgressEvent(quest)

				if quest.IsCompleted() {
					qm.publishQuestReadyToCompleteEvent(quest)
				}
			}
		}
		return true
	})

	return updatedCount
}

// OnKillMonster 杀怪事件处理
func (qm *QuestManager) OnKillMonster(monsterID int32) {
	qm.UpdateTargetProgress(QuestTargetTypeKill, monsterID, 1)
}

// OnCollectItem 收集物品事件处理
func (qm *QuestManager) OnCollectItem(itemConfigID int32, count int32) {
	qm.UpdateTargetProgress(QuestTargetTypeCollect, itemConfigID, count)
}

// OnTalkToNPC 对话NPC事件处理
func (qm *QuestManager) OnTalkToNPC(npcID int32) {
	qm.UpdateTargetProgress(QuestTargetTypeTalk, npcID, 1)
}

// OnReachLocation 到达地点事件处理
func (qm *QuestManager) OnReachLocation(locationID int32) {
	qm.UpdateTargetProgress(QuestTargetTypeReach, locationID, 1)
}

// OnLevelUp 升级事件处理
func (qm *QuestManager) OnLevelUp(level int32) {
	qm.UpdateTargetProgress(QuestTargetTypeLevel, level, 1)
}

// OnUseItem 使用物品事件处理
func (qm *QuestManager) OnUseItem(itemConfigID int32) {
	qm.UpdateTargetProgress(QuestTargetTypeUseItem, itemConfigID, 1)
}

// OnLearnSkill 学习技能事件处理
func (qm *QuestManager) OnLearnSkill(skillConfigID int32) {
	qm.UpdateTargetProgress(QuestTargetTypeSkill, skillConfigID, 1)
}

// OnExplore 探索事件处理
func (qm *QuestManager) OnExplore(areaID int32) {
	qm.UpdateTargetProgress(QuestTargetTypeExplore, areaID, 1)
}

// HasCompletedQuest 检查是否已完成指定任务
func (qm *QuestManager) HasCompletedQuest(questConfigID int32) bool {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	_, exists := qm.completedQuestIDs.Load(questConfigID)
	return exists
}

// GetCompletedQuestCount 获取已完成任务数
func (qm *QuestManager) GetCompletedQuestCount() int32 {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	count := int32(0)
	qm.completedQuestIDs.Range(func(_ int32, _ bool) bool {
		count++
		return true
	})
	return count
}

// RefreshDailyQuests 刷新日常任务
func (qm *QuestManager) RefreshDailyQuests() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	currentTime := time.Now().UnixMilli()
	refreshedCount := int32(0)

	qm.quests.Range(func(_ int32, quest *Quest) bool {
		if quest.IsDaily() && quest.NeedRefresh(currentTime) {
			quest.Reset()
			quest.SetRefreshTime(qm.getNextDailyRefreshTime())
			refreshedCount++
		}
		return true
	})

	qm.dailyAcceptCount = 0

	if refreshedCount > 0 {
		zLog.Info("Daily quests refreshed",
			zap.Int64("player_id", int64(qm.playerID)),
			zap.Int32("count", refreshedCount))
	}
}

// RefreshWeeklyQuests 刷新周常任务
func (qm *QuestManager) RefreshWeeklyQuests() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	currentTime := time.Now().UnixMilli()
	refreshedCount := int32(0)

	qm.quests.Range(func(_ int32, quest *Quest) bool {
		if quest.IsWeekly() && quest.NeedRefresh(currentTime) {
			quest.Reset()
			quest.SetRefreshTime(qm.getNextWeeklyRefreshTime())
			refreshedCount++
		}
		return true
	})

	qm.weeklyAcceptCount = 0

	if refreshedCount > 0 {
		zLog.Info("Weekly quests refreshed",
			zap.Int64("player_id", int64(qm.playerID)),
			zap.Int32("count", refreshedCount))
	}
}

// getNextDailyRefreshTime 获取下一次日常刷新时间
func (qm *QuestManager) getNextDailyRefreshTime() int64 {
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	tomorrow = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, now.Location())
	return tomorrow.UnixMilli()
}

// getNextWeeklyRefreshTime 获取下一次周常刷新时间
func (qm *QuestManager) getNextWeeklyRefreshTime() int64 {
	now := time.Now()
	weekday := now.Weekday()
	daysUntilMonday := (7 - int(weekday)) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	nextMonday := now.AddDate(0, 0, daysUntilMonday)
	nextMonday = time.Date(nextMonday.Year(), nextMonday.Month(), nextMonday.Day(), 0, 0, 0, 0, now.Location())
	return nextMonday.UnixMilli()
}

// Clear 清除所有任务
func (qm *QuestManager) Clear() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	qm.quests = zMap.NewTypedMap[int32, *Quest]()
	qm.completedQuestIDs = zMap.NewTypedMap[int32, bool]()
	qm.dailyAcceptCount = 0
	qm.weeklyAcceptCount = 0
}

// publishQuestAcceptEvent 发布任务接取事件
func (qm *QuestManager) publishQuestAcceptEvent(quest *Quest) {
	event.Publish(event.NewEvent(event.EventQuestAccept, qm, &event.QuestEventData{
		PlayerID: qm.playerID,
		QuestID:  quest.GetQuestConfigID(),
		Progress: 0,
	}))
}

// publishQuestProgressEvent 发布任务进度事件
func (qm *QuestManager) publishQuestProgressEvent(quest *Quest) {
	current, _ := quest.GetTotalProgress()
	event.Publish(event.NewEvent(event.EventQuestProgress, qm, &event.QuestEventData{
		PlayerID: qm.playerID,
		QuestID:  quest.GetQuestConfigID(),
		Progress: current,
	}))
}

// publishQuestCompleteEvent 发布任务完成事件
func (qm *QuestManager) publishQuestCompleteEvent(quest *Quest) {
	event.Publish(event.NewEvent(event.EventQuestComplete, qm, &event.QuestEventData{
		PlayerID: qm.playerID,
		QuestID:  quest.GetQuestConfigID(),
		Progress: 100,
	}))
}

// publishQuestReadyToCompleteEvent 发布任务可提交事件
func (qm *QuestManager) publishQuestReadyToCompleteEvent(quest *Quest) {
	event.Publish(event.NewEvent(event.EventQuestReadyToComplete, qm, &event.QuestEventData{
		PlayerID: qm.playerID,
		QuestID:  quest.GetQuestConfigID(),
		Progress: 100,
	}))
}

// publishQuestSubmitEvent 发布任务提交事件
func (qm *QuestManager) publishQuestSubmitEvent(quest *Quest) {
	event.Publish(event.NewEvent(event.EventQuestSubmit, qm, &event.QuestEventData{
		PlayerID: qm.playerID,
		QuestID:  quest.GetQuestConfigID(),
		Progress: 100,
	}))
}
