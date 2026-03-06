package dungeon

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

type DungeonType int

const (
	DungeonTypeNone DungeonType = iota
	DungeonTypeNormal
	DungeonTypeElite
	DungeonTypeBoss
	DungeonTypeChallenge
	DungeonTypeTimeAttack
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

type DungeonConfig struct {
	DungeonID       int32
	Name            string
	Description     string
	Type            DungeonType
	MinLevel        int32
	MaxLevel        int32
	MinPlayers      int32
	MaxPlayers      int32
	TimeLimit       int32
	DailyLimit      int32
	Rewards         []DungeonReward
	Waves           []DungeonWave
	RequiredItems   []int32
	EntryCost       *DungeonCost
}

type DungeonReward struct {
	RewardID   int32
	Type       string
	ItemID     int32
	Count      int32
	Probability float32
}

type DungeonWave struct {
	WaveID      int32
	Monsters    []DungeonMonster
	SpawnDelay  int32
	CompleteCondition string
}

type DungeonMonster struct {
	MonsterID int32
	Count     int32
	Position  string
}

type DungeonCost struct {
	CurrencyType string
	Amount       int32
}

type DungeonInstance struct {
	InstanceID    int32
	DungeonID     int32
	Config        *DungeonConfig
	Status        DungeonStatus
	Players       map[id.PlayerIdType]*DungeonPlayer
	CurrentWave   int32
	StartTime     time.Time
	EndTime       time.Time
	KillCount     int32
	TotalKills    int32
	IsSuccess     bool
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
	DungeonID      int32
	CompletedCount int32
	BestTime       int32
	LastEnterTime  time.Time
	DailyCount     int32
	LastResetTime  time.Time
}

type DungeonManager struct {
	mu              sync.RWMutex
	instances       map[int32]*DungeonInstance
	playerRecords   map[id.PlayerIdType]map[int32]*PlayerDungeonRecord
	nextInstanceID  int32
}

var globalDungeonManager *DungeonManager
var dungeonOnce sync.Once

func NewDungeonManager() *DungeonManager {
	return &DungeonManager{
		instances:      make(map[int32]*DungeonInstance),
		playerRecords:  make(map[id.PlayerIdType]map[int32]*PlayerDungeonRecord),
		nextInstanceID: 1,
	}
}

func GetDungeonManager() *DungeonManager {
	if globalDungeonManager == nil {
		dungeonOnce.Do(func() {
			globalDungeonManager = NewDungeonManager()
		})
	}
	return globalDungeonManager
}

func (dm *DungeonManager) CreateInstance(config *DungeonConfig) (*DungeonInstance, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	instanceID := dm.nextInstanceID
	dm.nextInstanceID++

	instance := &DungeonInstance{
		InstanceID:  instanceID,
		DungeonID:   config.DungeonID,
		Config:      config,
		Status:      DungeonStatusWaiting,
		Players:     make(map[id.PlayerIdType]*DungeonPlayer),
		CurrentWave: 0,
	}

	dm.instances[instanceID] = instance

	zLog.Info("Dungeon instance created",
		zap.Int32("instance_id", instanceID),
		zap.Int32("dungeon_id", config.DungeonID),
		zap.String("name", config.Name))

	return instance, nil
}

func (dm *DungeonManager) RemoveInstance(instanceID int32) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.instances[instanceID]; !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	delete(dm.instances, instanceID)

	zLog.Info("Dungeon instance removed", zap.Int32("instance_id", instanceID))
	return nil
}

func (dm *DungeonManager) GetInstance(instanceID int32) (*DungeonInstance, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	instance, ok := dm.instances[instanceID]
	return instance, ok
}

func (dm *DungeonManager) EnterDungeon(playerID id.PlayerIdType, instanceID int32) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	instance, exists := dm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	if instance.Status != DungeonStatusWaiting {
		return fmt.Errorf("instance is not waiting for players")
	}

	if int32(len(instance.Players)) >= instance.Config.MaxPlayers {
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
		zap.Int32("instance_id", instanceID))

	return nil
}

func (dm *DungeonManager) LeaveDungeon(playerID id.PlayerIdType, instanceID int32) error {
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
		zap.Int32("instance_id", instanceID))

	if len(instance.Players) == 0 && instance.Status == DungeonStatusWaiting {
		delete(dm.instances, instanceID)
		zLog.Info("Empty instance removed", zap.Int32("instance_id", instanceID))
	}

	return nil
}

func (dm *DungeonManager) StartDungeon(instanceID int32) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	instance, exists := dm.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	if instance.Status != DungeonStatusWaiting {
		return fmt.Errorf("instance is not in waiting status")
	}

	if int32(len(instance.Players)) < instance.Config.MinPlayers {
		return fmt.Errorf("not enough players")
	}

	instance.Status = DungeonStatusInProgress
	instance.StartTime = time.Now()
	instance.CurrentWave = 1

	zLog.Info("Dungeon started",
		zap.Int32("instance_id", instanceID),
		zap.Int32("dungeon_id", instance.DungeonID),
		zap.Int("player_count", len(instance.Players)))

	return nil
}

func (dm *DungeonManager) CompleteDungeon(instanceID int32, isSuccess bool) error {
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
			dm.updatePlayerRecord(playerID, instance.DungeonID, true, int32(time.Since(instance.StartTime).Seconds()))
		}
	} else {
		instance.Status = DungeonStatusFailed
		for playerID := range instance.Players {
			dm.updatePlayerRecord(playerID, instance.DungeonID, false, 0)
		}
	}

	zLog.Info("Dungeon completed",
		zap.Int32("instance_id", instanceID),
		zap.Bool("success", isSuccess))

	return nil
}

func (dm *DungeonManager) updatePlayerRecord(playerID id.PlayerIdType, dungeonID int32, success bool, clearTime int32) {
	if _, exists := dm.playerRecords[playerID]; !exists {
		dm.playerRecords[playerID] = make(map[int32]*PlayerDungeonRecord)
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

func (dm *DungeonManager) PlayerDeath(playerID id.PlayerIdType, instanceID int32) error {
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
			zap.Int32("instance_id", instanceID))
	}

	return nil
}

func (dm *DungeonManager) MonsterKilled(instanceID int32, killerID id.PlayerIdType, count int32) {
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

func (dm *DungeonManager) AdvanceWave(instanceID int32) error {
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

	if instance.CurrentWave > int32(len(instance.Config.Waves)) {
		instance.Status = DungeonStatusCompleted
		instance.EndTime = time.Now()
		instance.IsSuccess = true

		for playerID := range instance.Players {
			dm.updatePlayerRecord(playerID, instance.DungeonID, true, int32(time.Since(instance.StartTime).Seconds()))
		}

		zLog.Info("Dungeon completed - all waves cleared",
			zap.Int32("instance_id", instanceID))
	} else {
		zLog.Debug("Wave advanced",
			zap.Int32("instance_id", instanceID),
			zap.Int32("wave", instance.CurrentWave))
	}

	return nil
}

func (dm *DungeonManager) GetPlayerRecords(playerID id.PlayerIdType) map[int32]*PlayerDungeonRecord {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if records, exists := dm.playerRecords[playerID]; exists {
		result := make(map[int32]*PlayerDungeonRecord)
		for k, v := range records {
			result[k] = v
		}
		return result
	}

	return make(map[int32]*PlayerDungeonRecord)
}

func (dm *DungeonManager) CanEnterDungeon(playerID id.PlayerIdType, config *DungeonConfig) (bool, string) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if records, exists := dm.playerRecords[playerID]; exists {
		if record, exists := records[config.DungeonID]; exists {
			now := time.Now()
			if now.Sub(record.LastResetTime) < 24*time.Hour {
				if record.DailyCount >= config.DailyLimit {
					return false, "daily limit reached"
				}
			}
		}
	}

	return true, ""
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
			if elapsed.Seconds() >= float64(instance.Config.TimeLimit) {
				instance.Status = DungeonStatusFailed
				instance.EndTime = now
				instance.IsSuccess = false

				zLog.Info("Dungeon failed - time limit exceeded",
					zap.Int32("instance_id", instance.InstanceID))
			}
		}
	}
}

func (di *DungeonInstance) GetRemainingTime() int32 {
	if di.Status != DungeonStatusInProgress {
		return 0
	}

	elapsed := time.Since(di.StartTime).Seconds()
	remaining := di.Config.TimeLimit - int32(elapsed)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (di *DungeonInstance) GetPlayerCount() int {
	return len(di.Players)
}

func (di *DungeonInstance) GetAlivePlayerCount() int {
	count := 0
	for _, player := range di.Players {
		if player.IsAlive {
			count++
		}
	}
	return count
}

func (di *DungeonInstance) IsPlayerInInstance(playerID id.PlayerIdType) bool {
	_, exists := di.Players[playerID]
	return exists
}

func (di *DungeonInstance) GetProgress() float32 {
	if len(di.Config.Waves) == 0 {
		return 100.0
	}
	return float32(di.CurrentWave-1) / float32(len(di.Config.Waves)) * 100.0
}
