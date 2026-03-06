package dungeon

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// DungeonManager 副本管理器
type DungeonManager struct {
	dungeons       map[id.DungeonIdType]*Dungeon
	instances      map[id.InstanceIdType]*DungeonInstance
	playerRecords  map[id.PlayerIdType]map[id.DungeonIdType]*PlayerDungeonRecord
	mutex          sync.RWMutex
	instanceCounter int64
}

// NewDungeonManager 创建副本管理器
func NewDungeonManager() *DungeonManager {
	return &DungeonManager{
		dungeons:       make(map[id.DungeonIdType]*Dungeon),
		instances:      make(map[id.InstanceIdType]*DungeonInstance),
		playerRecords:  make(map[id.PlayerIdType]map[id.DungeonIdType]*PlayerDungeonRecord),
		instanceCounter: 0,
	}
}

// AddDungeon 添加副本
func (dm *DungeonManager) AddDungeon(dungeon *Dungeon) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dm.dungeons[dungeon.DungeonID] = dungeon
	zLog.Info("Dungeon added",
		zap.Uint64("dungeon_id", uint64(dungeon.DungeonID)),
		zap.String("name", dungeon.Name),
		zap.Int("type", dungeon.Type))
}

// GetDungeon 获取副本
func (dm *DungeonManager) GetDungeon(dungeonID id.DungeonIdType) *Dungeon {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	return dm.dungeons[dungeonID]
}

// GetAllDungeons 获取所有副本
func (dm *DungeonManager) GetAllDungeons() []*Dungeon {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	dungeons := make([]*Dungeon, 0, len(dm.dungeons))
	for _, dungeon := range dm.dungeons {
		dungeons = append(dungeons, dungeon)
	}

	return dungeons
}

// GetDungeonsByType 获取指定类型的副本
func (dm *DungeonManager) GetDungeonsByType(dungeonType int) []*Dungeon {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	dungeons := make([]*Dungeon, 0)
	for _, dungeon := range dm.dungeons {
		if dungeon.Type == dungeonType {
			dungeons = append(dungeons, dungeon)
		}
	}

	return dungeons
}

// CreateInstance 创建副本实例
func (dm *DungeonManager) CreateInstance(dungeonID id.DungeonIdType, leaderID id.PlayerIdType) *DungeonInstance {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dungeon := dm.dungeons[dungeonID]
	if dungeon == nil {
		return nil
	}

	dm.instanceCounter++
	instanceID := id.InstanceIdType(dm.instanceCounter)

	instance := NewDungeonInstance(instanceID, dungeonID, leaderID)
	dm.instances[instanceID] = instance

	zLog.Info("Dungeon instance created",
		zap.Uint64("instance_id", uint64(instanceID)),
		zap.Uint64("dungeon_id", uint64(dungeonID)),
		zap.Uint64("leader_id", uint64(leaderID)))

	return instance
}

// GetInstance 获取副本实例
func (dm *DungeonManager) GetInstance(instanceID id.InstanceIdType) *DungeonInstance {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	return dm.instances[instanceID]
}

// StartInstance 开始副本
func (dm *DungeonManager) StartInstance(instanceID id.InstanceIdType) bool {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	instance := dm.instances[instanceID]
	if instance == nil {
		return false
	}

	instance.Start()
	zLog.Info("Dungeon instance started",
		zap.Uint64("instance_id", uint64(instanceID)),
		zap.Uint64("dungeon_id", uint64(instance.DungeonID)))

	return true
}

// CompleteInstance 完成副本
func (dm *DungeonManager) CompleteInstance(instanceID id.InstanceIdType) bool {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	instance := dm.instances[instanceID]
	if instance == nil {
		return false
	}

	instance.Complete()

	// 更新玩家记录
	dungeon := dm.dungeons[instance.DungeonID]
	for playerID := range instance.Players {
		dm.updatePlayerRecord(playerID, instance, dungeon)
	}

	zLog.Info("Dungeon instance completed",
		zap.Uint64("instance_id", uint64(instanceID)),
		zap.Uint64("dungeon_id", uint64(instance.DungeonID)),
		zap.Int("duration", instance.GetDuration()))

	return true
}

// FailInstance 副本失败
func (dm *DungeonManager) FailInstance(instanceID id.InstanceIdType) bool {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	instance := dm.instances[instanceID]
	if instance == nil {
		return false
	}

	instance.Fail()
	zLog.Info("Dungeon instance failed",
		zap.Uint64("instance_id", uint64(instanceID)),
		zap.Uint64("dungeon_id", uint64(instance.DungeonID)))

	return true
}

// RemoveInstance 移除副本实例
func (dm *DungeonManager) RemoveInstance(instanceID id.InstanceIdType) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	delete(dm.instances, instanceID)
}

// GetPlayerRecord 获取玩家副本记录
func (dm *DungeonManager) GetPlayerRecord(playerID id.PlayerIdType, dungeonID id.DungeonIdType) *PlayerDungeonRecord {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	if records, exists := dm.playerRecords[playerID]; exists {
		return records[dungeonID]
	}

	return nil
}

// CanEnterDungeon 检查玩家是否可以进入副本
func (dm *DungeonManager) CanEnterDungeon(playerID id.PlayerIdType, dungeonID id.DungeonIdType, playerLevel int) bool {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	dungeon := dm.dungeons[dungeonID]
	if dungeon == nil {
		return false
	}

	record := dm.GetPlayerRecord(playerID, dungeonID)
	dailyCount := 0
	if record != nil {
		dailyCount = record.DailyCount
	}

	return dungeon.CanEnter(playerLevel, dailyCount)
}

// updatePlayerRecord 更新玩家记录
func (dm *DungeonManager) updatePlayerRecord(playerID id.PlayerIdType, instance *DungeonInstance, dungeon *Dungeon) {
	if _, exists := dm.playerRecords[playerID]; !exists {
		dm.playerRecords[playerID] = make(map[id.DungeonIdType]*PlayerDungeonRecord)
	}

	record := dm.playerRecords[playerID][instance.DungeonID]
	if record == nil {
		record = &PlayerDungeonRecord{
			PlayerID:      playerID,
			DungeonID:     instance.DungeonID,
			CompleteCount: 0,
			BestTime:      0,
		}
		dm.playerRecords[playerID][instance.DungeonID] = record
	}

	record.CompleteCount++
	record.LastEnterTime = time.Now()
	record.DailyCount++
	record.WeeklyCount++

	duration := instance.GetDuration()
	if record.BestTime == 0 || duration < record.BestTime {
		record.BestTime = duration
	}
}

// ResetDailyCounts 重置每日次数
func (dm *DungeonManager) ResetDailyCounts() {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	for _, records := range dm.playerRecords {
		for _, record := range records {
			record.DailyCount = 0
		}
	}

	zLog.Info("All dungeon daily counts reset")
}

// ResetWeeklyCounts 重置每周次数
func (dm *DungeonManager) ResetWeeklyCounts() {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	for _, records := range dm.playerRecords {
		for _, record := range records {
			record.WeeklyCount = 0
		}
	}

	zLog.Info("All dungeon weekly counts reset")
}

// GetActiveInstances 获取进行中的副本实例
func (dm *DungeonManager) GetActiveInstances() []*DungeonInstance {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	instances := make([]*DungeonInstance, 0)
	for _, instance := range dm.instances {
		if instance.IsRunning() {
			instances = append(instances, instance)
		}
	}

	return instances
}
