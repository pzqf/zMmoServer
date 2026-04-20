package dungeon

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"go.uber.org/zap"
)

type MapFactoryFunc func(mapID id.MapIdType, mapConfigID int32, name string, width, height float32, connManager *connection.ConnectionManager) *MapInterface

type MapInterface interface {
	GetID() id.MapIdType
	GetName() string
	AddPlayer(playerID id.PlayerIdType, objectID id.ObjectIdType, x, y, z float32) error
	RemovePlayer(playerID id.PlayerIdType)
	AddObject(obj common.IGameObject)
	RemoveObject(objectID id.ObjectIdType)
	GetObject(objectID id.ObjectIdType) common.IGameObject
	GetObjectsByType(objectType common.GameObjectType) []common.IGameObject
}

type DungeonLifecycleManager struct {
	mu          sync.RWMutex
	dm          *DungeonManager
	instances   map[id.InstanceIdType]*DungeonLifecycle
	connManager *connection.ConnectionManager
}

type DungeonLifecycle struct {
	Instance    *DungeonInstance
	MapID       id.MapIdType
	CreatedAt   time.Time
	DestroyedAt time.Time
}

func NewDungeonLifecycleManager(dm *DungeonManager, connManager *connection.ConnectionManager) *DungeonLifecycleManager {
	return &DungeonLifecycleManager{
		dm:          dm,
		instances:   make(map[id.InstanceIdType]*DungeonLifecycle),
		connManager: connManager,
	}
}

func (dlm *DungeonLifecycleManager) CreateAndStartDungeon(dungeonID id.DungeonIdType, players []id.PlayerIdType, mapID id.MapIdType) (*DungeonInstance, error) {
	instance, err := dlm.dm.CreateInstance(dungeonID)
	if err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}

	instance.SetMapInstanceID(mapID)

	for _, playerID := range players {
		if err := dlm.dm.EnterDungeon(playerID, instance.InstanceID); err != nil {
			zLog.Warn("Player failed to enter dungeon",
				zap.Int64("player_id", int64(playerID)),
				zap.Int64("instance_id", int64(instance.InstanceID)),
				zap.String("error", err.Error()))
		}
	}

	if err := dlm.dm.StartDungeon(instance.InstanceID); err != nil {
		return nil, fmt.Errorf("start dungeon: %w", err)
	}

	dlm.mu.Lock()
	dlm.instances[instance.InstanceID] = &DungeonLifecycle{
		Instance:  instance,
		MapID:     mapID,
		CreatedAt: time.Now(),
	}
	dlm.mu.Unlock()

	zLog.Info("Dungeon lifecycle created",
		zap.Int64("instance_id", int64(instance.InstanceID)),
		zap.Int32("dungeon_id", int32(dungeonID)),
		zap.Int32("map_id", int32(mapID)),
		zap.Int("player_count", len(players)))

	return instance, nil
}

func (dlm *DungeonLifecycleManager) CompleteDungeon(instanceID id.InstanceIdType, isSuccess bool) error {
	if err := dlm.dm.CompleteDungeon(instanceID, isSuccess); err != nil {
		return err
	}

	dlm.mu.Lock()
	if lifecycle, exists := dlm.instances[instanceID]; exists {
		lifecycle.DestroyedAt = time.Now()
	}
	dlm.mu.Unlock()

	zLog.Info("Dungeon lifecycle completed",
		zap.Int64("instance_id", int64(instanceID)),
		zap.Bool("success", isSuccess))

	return nil
}

func (dlm *DungeonLifecycleManager) DestroyDungeon(instanceID id.InstanceIdType) error {
	instance, exists := dlm.dm.GetInstance(instanceID)
	if !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	if instance.Status == DungeonStatusInProgress {
		if err := dlm.dm.CompleteDungeon(instanceID, false); err != nil {
			zLog.Warn("Failed to complete dungeon on destroy",
				zap.Int64("instance_id", int64(instanceID)),
				zap.String("error", err.Error()))
		}
	}

	dlm.mu.Lock()
	delete(dlm.instances, instanceID)
	dlm.mu.Unlock()

	if err := dlm.dm.RemoveInstance(instanceID); err != nil {
		zLog.Warn("Failed to remove dungeon instance",
			zap.Int64("instance_id", int64(instanceID)),
			zap.String("error", err.Error()))
	}

	zLog.Info("Dungeon lifecycle destroyed",
		zap.Int64("instance_id", int64(instanceID)))

	return nil
}

func (dlm *DungeonLifecycleManager) SpawnWaveMonsters(instanceID id.InstanceIdType, spawnFn func(monsterIDs []int32, mapID id.MapIdType) error) error {
	monsterIDs, err := dlm.dm.GetCurrentWaveMonsters(instanceID)
	if err != nil {
		return fmt.Errorf("get wave monsters: %w", err)
	}

	if len(monsterIDs) == 0 {
		return nil
	}

	instance, exists := dlm.dm.GetInstance(instanceID)
	if !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	mapID := instance.GetMapInstanceID()

	if spawnFn != nil {
		if err := spawnFn(monsterIDs, mapID); err != nil {
			return fmt.Errorf("spawn monsters: %w", err)
		}
	}

	zLog.Debug("Wave monsters spawned",
		zap.Int64("instance_id", int64(instanceID)),
		zap.Int32("wave", instance.CurrentWave),
		zap.Int("monster_count", len(monsterIDs)))

	return nil
}

func (dlm *DungeonLifecycleManager) OnMonsterKilled(instanceID id.InstanceIdType, killerID id.PlayerIdType) error {
	instance, exists := dlm.dm.GetInstance(instanceID)
	if !exists {
		return fmt.Errorf("instance not found: %d", instanceID)
	}

	dlm.dm.MonsterKilled(instanceID, killerID, 1)

	waveMonsters, _ := dlm.dm.GetCurrentWaveMonsters(instanceID)
	if instance.KillCount >= int32(len(waveMonsters)) && len(waveMonsters) > 0 {
		return dlm.dm.AdvanceWave(instanceID)
	}

	return nil
}

func (dlm *DungeonLifecycleManager) GetDungeonRewards(instanceID id.InstanceIdType) (*DungeonRewards, error) {
	instance, exists := dlm.dm.GetInstance(instanceID)
	if !exists {
		return nil, fmt.Errorf("instance not found: %d", instanceID)
	}

	if instance.DungeonConfig == nil {
		return nil, fmt.Errorf("dungeon config not found")
	}

	rewards := &DungeonRewards{
		DungeonID:   instance.DungeonID,
		InstanceID:  instance.InstanceID,
		Exp:         instance.DungeonConfig.RewardExp,
		Gold:        instance.DungeonConfig.RewardGold,
		ClearTime:   int32(time.Since(instance.StartTime).Seconds()),
		PlayerStats: make(map[id.PlayerIdType]*PlayerDungeonStats),
	}

	for playerID, player := range instance.Players {
		rewards.PlayerStats[playerID] = &PlayerDungeonStats{
			PlayerID:    playerID,
			KillCount:   player.KillCount,
			DamageDealt: player.DamageDealt,
			DamageTaken: player.DamageTaken,
			HealAmount:  player.HealAmount,
		}
	}

	return rewards, nil
}

func (dlm *DungeonLifecycleManager) Update(deltaTime time.Duration) {
	dlm.dm.Update(deltaTime)

	dlm.mu.Lock()
	defer dlm.mu.Unlock()

	toDestroy := make([]id.InstanceIdType, 0)
	for instanceID, lifecycle := range dlm.instances {
		if lifecycle.Instance.Status == DungeonStatusCompleted ||
			lifecycle.Instance.Status == DungeonStatusFailed ||
			lifecycle.Instance.Status == DungeonStatusClosed {
			if lifecycle.DestroyedAt.IsZero() {
				lifecycle.DestroyedAt = time.Now()
			}
			if time.Since(lifecycle.DestroyedAt) >= 30*time.Second {
				toDestroy = append(toDestroy, instanceID)
			}
		}
	}

	for _, instanceID := range toDestroy {
		delete(dlm.instances, instanceID)
		dlm.dm.RemoveInstance(instanceID)
		zLog.Info("Dungeon lifecycle auto-destroyed",
			zap.Int64("instance_id", int64(instanceID)))
	}
}

func (dlm *DungeonLifecycleManager) GetLifecycle(instanceID id.InstanceIdType) (*DungeonLifecycle, bool) {
	dlm.mu.RLock()
	defer dlm.mu.RUnlock()
	lc, ok := dlm.instances[instanceID]
	return lc, ok
}

func (dlm *DungeonLifecycleManager) GetActiveLifecycles() []*DungeonLifecycle {
	dlm.mu.RLock()
	defer dlm.mu.RUnlock()

	result := make([]*DungeonLifecycle, 0)
	for _, lc := range dlm.instances {
		if lc.Instance.Status == DungeonStatusInProgress ||
			lc.Instance.Status == DungeonStatusWaiting {
			result = append(result, lc)
		}
	}
	return result
}

type DungeonRewards struct {
	DungeonID   id.DungeonIdType
	InstanceID  id.InstanceIdType
	Exp         int64
	Gold        int64
	ClearTime   int32
	PlayerStats map[id.PlayerIdType]*PlayerDungeonStats
}

type PlayerDungeonStats struct {
	PlayerID    id.PlayerIdType
	KillCount   int32
	DamageDealt int64
	DamageTaken int64
	HealAmount  int64
}

func CreateDungeonPlayer(objectID id.ObjectIdType, playerID id.PlayerIdType, name string, pos common.Vector3, level int32) *object.Player {
	return object.NewPlayer(objectID, playerID, name, pos, level)
}
