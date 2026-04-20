package dungeon

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/models"
	"github.com/pzqf/zCommon/config/tables"
	"go.uber.org/zap"
)

type DungeonStatus int

const (
	DungeonStatusNone DungeonStatus = iota
	DungeonStatusWaiting
	DungeonStatusInProgress
	DungeonStatusCompleted
	DungeonStatusFailed
	DungeonStatusClosed
)

func (s DungeonStatus) String() string {
	switch s {
	case DungeonStatusNone:
		return "none"
	case DungeonStatusWaiting:
		return "waiting"
	case DungeonStatusInProgress:
		return "in_progress"
	case DungeonStatusCompleted:
		return "completed"
	case DungeonStatusFailed:
		return "failed"
	case DungeonStatusClosed:
		return "closed"
	default:
		return "unknown"
	}
}

type DungeonInstance struct {
	mu             sync.RWMutex
	InstanceID     id.InstanceIdType
	DungeonID      id.DungeonIdType
	DungeonConfig  *models.Dungeon
	Waves          []*models.DungeonWave
	Status         DungeonStatus
	Players        map[id.PlayerIdType]*DungeonPlayer
	CurrentWave    int32
	StartTime      time.Time
	EndTime        time.Time
	KillCount      int32
	TotalKills     int32
	MapInstanceID  id.MapIdType
	IsSuccess      bool
	createTime     time.Time
}

type DungeonPlayer struct {
	PlayerID    id.PlayerIdType
	JoinTime    time.Time
	IsAlive     bool
	KillCount   int32
	DamageDealt int64
	DamageTaken int64
	HealAmount  int64
}

type PlayerDungeonRecord struct {
	PlayerID       id.PlayerIdType
	DungeonID      id.DungeonIdType
	CompletedCount int32
	BestTime       int32
	LastEnterTime  time.Time
	DailyCount     int32
	LastResetTime  time.Time
}

type DungeonManager struct {
	mu             sync.RWMutex
	instances      map[id.InstanceIdType]*DungeonInstance
	playerRecords  map[id.PlayerIdType]map[id.DungeonIdType]*PlayerDungeonRecord
	nextInstanceID int64
	tableManager   *tables.TableManager
}

func NewDungeonManager() *DungeonManager {
	return &DungeonManager{
		instances:     make(map[id.InstanceIdType]*DungeonInstance),
		playerRecords: make(map[id.PlayerIdType]map[id.DungeonIdType]*PlayerDungeonRecord),
	}
}

func (dm *DungeonManager) SetTableManager(tm *tables.TableManager) {
	dm.tableManager = tm
}

func (dm *DungeonManager) CreateInstance(dungeonID id.DungeonIdType) (*DungeonInstance, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.tableManager == nil {
		return nil, fmt.Errorf("table manager not set")
	}

	dungeonConfig, ok := dm.tableManager.GetDungeonLoader().GetDungeon(int32(dungeonID))
	if !ok {
		return nil, fmt.Errorf("dungeon config not found: %d", dungeonID)
	}

	if !dungeonConfig.IsOpen {
		return nil, fmt.Errorf("dungeon is not open: %d", dungeonID)
	}

	dm.nextInstanceID++
	instanceID := id.InstanceIdType(dm.nextInstanceID)

	waves := dm.tableManager.GetDungeonLoader().GetWavesByDungeonID(int32(dungeonID))

	instance := &DungeonInstance{
		InstanceID:    instanceID,
		DungeonID:     dungeonID,
		DungeonConfig: dungeonConfig,
		Waves:         waves,
		Status:        DungeonStatusWaiting,
		Players:       make(map[id.PlayerIdType]*DungeonPlayer),
		CurrentWave:   0,
		createTime:    time.Now(),
	}

	dm.instances[instanceID] = instance

	zLog.Info("Dungeon instance created",
		zap.Int64("instance_id", int64(instanceID)),
		zap.Int32("dungeon_id", int32(dungeonID)),
		zap.String("name", dungeonConfig.Name))

	return instance, nil
}

func (dm *DungeonManager) RemoveInstance(instanceID id.InstanceIdType) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.instances[instanceID]; !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	delete(dm.instances, instanceID)

	zLog.Info("Dungeon instance removed", zap.Int64("instance_id", int64(instanceID)))
	return nil
}

func (dm *DungeonManager) GetInstance(instanceID id.InstanceIdType) (*DungeonInstance, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	instance, ok := dm.instances[instanceID]
	return instance, ok
}

func (dm *DungeonManager) GetInstanceByMapID(mapID id.MapIdType) *DungeonInstance {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for _, instance := range dm.instances {
		if instance.MapInstanceID == mapID {
			return instance
		}
	}
	return nil
}

func (dm *DungeonManager) EnterDungeon(playerID id.PlayerIdType, instanceID id.InstanceIdType) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	instance, exists := dm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	if instance.Status != DungeonStatusWaiting {
		return fmt.Errorf("instance is not waiting for players")
	}

	if int32(len(instance.Players)) >= instance.DungeonConfig.MaxPlayers {
		return fmt.Errorf("instance is full")
	}

	if _, exists := instance.Players[playerID]; exists {
		return fmt.Errorf("player already in instance")
	}

	player := &DungeonPlayer{
		PlayerID:    playerID,
		JoinTime:    time.Now(),
		IsAlive:     true,
		KillCount:   0,
		DamageDealt: 0,
		DamageTaken: 0,
		HealAmount:  0,
	}

	instance.Players[playerID] = player

	zLog.Debug("Player entered dungeon",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("instance_id", int64(instanceID)))

	return nil
}

func (dm *DungeonManager) LeaveDungeon(playerID id.PlayerIdType, instanceID id.InstanceIdType) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	instance, exists := dm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	if _, exists := instance.Players[playerID]; !exists {
		return fmt.Errorf("player not in instance")
	}

	delete(instance.Players, playerID)

	zLog.Debug("Player left dungeon",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("instance_id", int64(instanceID)))

	if len(instance.Players) == 0 && instance.Status == DungeonStatusWaiting {
		delete(dm.instances, instanceID)
		zLog.Info("Empty instance removed", zap.Int64("instance_id", int64(instanceID)))
	}

	return nil
}

func (dm *DungeonManager) StartDungeon(instanceID id.InstanceIdType) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	instance, exists := dm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	if instance.Status != DungeonStatusWaiting {
		return fmt.Errorf("instance is not in waiting status")
	}

	if int32(len(instance.Players)) < instance.DungeonConfig.MinPlayers {
		return fmt.Errorf("not enough players: need %d, have %d",
			instance.DungeonConfig.MinPlayers, len(instance.Players))
	}

	instance.Status = DungeonStatusInProgress
	instance.StartTime = time.Now()
	instance.CurrentWave = 1

	zLog.Info("Dungeon started",
		zap.Int64("instance_id", int64(instanceID)),
		zap.Int32("dungeon_id", int32(instance.DungeonID)),
		zap.Int("player_count", len(instance.Players)))

	return nil
}

func (dm *DungeonManager) CompleteDungeon(instanceID id.InstanceIdType, isSuccess bool) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	instance, exists := dm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	if instance.Status != DungeonStatusInProgress {
		return fmt.Errorf("instance is not in progress")
	}

	instance.EndTime = time.Now()
	instance.IsSuccess = isSuccess

	if isSuccess {
		instance.Status = DungeonStatusCompleted
		for playerID := range instance.Players {
			dm.updatePlayerRecordLocked(playerID, instance.DungeonID, true,
				int32(time.Since(instance.StartTime).Seconds()))
		}
	} else {
		instance.Status = DungeonStatusFailed
		for playerID := range instance.Players {
			dm.updatePlayerRecordLocked(playerID, instance.DungeonID, false, 0)
		}
	}

	zLog.Info("Dungeon completed",
		zap.Int64("instance_id", int64(instanceID)),
		zap.Bool("success", isSuccess))

	return nil
}

func (dm *DungeonManager) PlayerDeath(playerID id.PlayerIdType, instanceID id.InstanceIdType) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	instance, exists := dm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	player, exists := instance.Players[playerID]
	if !exists {
		return fmt.Errorf("player not in instance")
	}

	player.IsAlive = false

	allDead := true
	for _, p := range instance.Players {
		if p.IsAlive {
			allDead = false
			break
		}
	}

	if allDead {
		instance.Status = DungeonStatusFailed
		instance.EndTime = time.Now()
		instance.IsSuccess = false

		zLog.Info("Dungeon failed - all players dead",
			zap.Int64("instance_id", int64(instanceID)))
	}

	return nil
}

func (dm *DungeonManager) MonsterKilled(instanceID id.InstanceIdType, killerID id.PlayerIdType, count int32) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	instance, exists := dm.instances[instanceID]
	if !exists {
		return
	}

	instance.KillCount += count
	instance.TotalKills += count

	if player, exists := instance.Players[killerID]; exists {
		player.KillCount += count
	}
}

func (dm *DungeonManager) AdvanceWave(instanceID id.InstanceIdType) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	instance, exists := dm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	if instance.Status != DungeonStatusInProgress {
		return fmt.Errorf("instance is not in progress")
	}

	instance.CurrentWave++
	instance.KillCount = 0

	totalWaves := int32(len(instance.Waves))
	if totalWaves == 0 {
		totalWaves = instance.DungeonConfig.WaveCount
	}

	if totalWaves > 0 && instance.CurrentWave > totalWaves {
		instance.Status = DungeonStatusCompleted
		instance.EndTime = time.Now()
		instance.IsSuccess = true

		for playerID := range instance.Players {
			dm.updatePlayerRecordLocked(playerID, instance.DungeonID, true,
				int32(time.Since(instance.StartTime).Seconds()))
		}

		zLog.Info("Dungeon completed - all waves cleared",
			zap.Int64("instance_id", int64(instanceID)))
	} else {
		zLog.Debug("Wave advanced",
			zap.Int64("instance_id", int64(instanceID)),
			zap.Int32("wave", instance.CurrentWave))
	}

	return nil
}

func (dm *DungeonManager) GetCurrentWaveMonsters(instanceID id.InstanceIdType) ([]int32, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	instance, exists := dm.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance not found: %d", instanceID)
	}

	if instance.CurrentWave < 1 || int(instance.CurrentWave) > len(instance.Waves) {
		return nil, nil
	}

	wave := instance.Waves[instance.CurrentWave-1]
	if dm.tableManager != nil {
		return dm.tableManager.GetDungeonLoader().ParseMonsterIDs(wave.MonsterIDs), nil
	}
	return nil, nil
}

func (dm *DungeonManager) CanEnterDungeon(playerID id.PlayerIdType, dungeonID id.DungeonIdType) (bool, string) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if dm.tableManager == nil {
		return false, "table manager not set"
	}

	dungeonConfig, ok := dm.tableManager.GetDungeonLoader().GetDungeon(int32(dungeonID))
	if !ok {
		return false, "dungeon not found"
	}

	if !dungeonConfig.IsOpen {
		return false, "dungeon is not open"
	}

	if records, exists := dm.playerRecords[playerID]; exists {
		if record, exists := records[dungeonID]; exists {
			now := time.Now()
			if now.Sub(record.LastResetTime) < 24*time.Hour {
				if dungeonConfig.DailyLimit > 0 && record.DailyCount >= dungeonConfig.DailyLimit {
					return false, "daily limit reached"
				}
			}
		}
	}

	return true, ""
}

func (dm *DungeonManager) GetPlayerRecords(playerID id.PlayerIdType) map[id.DungeonIdType]*PlayerDungeonRecord {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if records, exists := dm.playerRecords[playerID]; exists {
		result := make(map[id.DungeonIdType]*PlayerDungeonRecord)
		for k, v := range records {
			result[k] = v
		}
		return result
	}

	return make(map[id.DungeonIdType]*PlayerDungeonRecord)
}

func (dm *DungeonManager) GetActiveInstances() []*DungeonInstance {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	active := make([]*DungeonInstance, 0)
	for _, instance := range dm.instances {
		if instance.Status == DungeonStatusInProgress || instance.Status == DungeonStatusWaiting {
			active = append(active, instance)
		}
	}
	return active
}

func (dm *DungeonManager) Update(deltaTime time.Duration) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	now := time.Now()

	for _, instance := range dm.instances {
		if instance.Status == DungeonStatusInProgress {
			elapsed := now.Sub(instance.StartTime)
			if instance.DungeonConfig.TimeLimit > 0 &&
				elapsed.Seconds() >= float64(instance.DungeonConfig.TimeLimit) {
				instance.Status = DungeonStatusFailed
				instance.EndTime = now
				instance.IsSuccess = false

				zLog.Info("Dungeon failed - time limit exceeded",
					zap.Int64("instance_id", int64(instance.InstanceID)))
			}
		}

		if instance.Status == DungeonStatusWaiting {
			if now.Sub(instance.createTime) >= 10*time.Minute {
				instance.Status = DungeonStatusClosed
				zLog.Info("Dungeon instance closed - wait timeout",
					zap.Int64("instance_id", int64(instance.InstanceID)))
			}
		}
	}
}

func (dm *DungeonManager) updatePlayerRecordLocked(playerID id.PlayerIdType, dungeonID id.DungeonIdType, success bool, clearTime int32) {
	if _, exists := dm.playerRecords[playerID]; !exists {
		dm.playerRecords[playerID] = make(map[id.DungeonIdType]*PlayerDungeonRecord)
	}

	record, exists := dm.playerRecords[playerID][dungeonID]
	if !exists {
		record = &PlayerDungeonRecord{
			PlayerID:  playerID,
			DungeonID: dungeonID,
		}
		dm.playerRecords[playerID][dungeonID] = record
	}

	if success {
		record.CompletedCount++
		if record.BestTime == 0 || clearTime < record.BestTime {
			record.BestTime = clearTime
		}
	}

	record.LastEnterTime = time.Now()

	now := time.Now()
	if record.LastResetTime.IsZero() || now.Sub(record.LastResetTime) >= 24*time.Hour {
		record.DailyCount = 0
		record.LastResetTime = now
	}
	record.DailyCount++
}

func (di *DungeonInstance) GetRemainingTime() int32 {
	di.mu.RLock()
	defer di.mu.RUnlock()

	if di.Status != DungeonStatusInProgress {
		return 0
	}

	elapsed := time.Since(di.StartTime).Seconds()
	remaining := di.DungeonConfig.TimeLimit - int32(elapsed)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (di *DungeonInstance) GetPlayerCount() int {
	di.mu.RLock()
	defer di.mu.RUnlock()
	return len(di.Players)
}

func (di *DungeonInstance) GetAlivePlayerCount() int {
	di.mu.RLock()
	defer di.mu.RUnlock()

	count := 0
	for _, player := range di.Players {
		if player.IsAlive {
			count++
		}
	}
	return count
}

func (di *DungeonInstance) IsPlayerInInstance(playerID id.PlayerIdType) bool {
	di.mu.RLock()
	defer di.mu.RUnlock()
	_, exists := di.Players[playerID]
	return exists
}

func (di *DungeonInstance) GetProgress() float32 {
	di.mu.RLock()
	defer di.mu.RUnlock()

	totalWaves := int32(len(di.Waves))
	if totalWaves == 0 {
		totalWaves = di.DungeonConfig.WaveCount
	}
	if totalWaves == 0 {
		return 100.0
	}
	return float32(di.CurrentWave-1) / float32(totalWaves) * 100.0
}

func (di *DungeonInstance) SetMapInstanceID(mapID id.MapIdType) {
	di.mu.Lock()
	defer di.mu.Unlock()
	di.MapInstanceID = mapID
}

func (di *DungeonInstance) GetMapInstanceID() id.MapIdType {
	di.mu.RLock()
	defer di.mu.RUnlock()
	return di.MapInstanceID
}
