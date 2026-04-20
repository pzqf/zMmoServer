package dungeon

import (
	"testing"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/models"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/stretchr/testify/assert"
)

func setupDungeonManager(dungeons ...*models.Dungeon) *DungeonManager {
	dm := NewDungeonManager()
	tm := tables.NewTableManager()
	loader := tm.GetDungeonLoader()

	for _, d := range dungeons {
		loader.AddDungeonForTest(d)
	}

	dm.SetTableManager(tm)
	return dm
}

func TestDungeonManager_CreateInstance_NoTableManager(t *testing.T) {
	dm := NewDungeonManager()
	_, err := dm.CreateInstance(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "table manager not set")
}

func TestDungeonManager_CreateInstance_WithConfig(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
		WaveCount:  3,
		TimeLimit:  600,
	})

	instance, err := dm.CreateInstance(1)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	assert.Equal(t, id.DungeonIdType(1), instance.DungeonID)
	assert.Equal(t, DungeonStatusWaiting, instance.Status)
	assert.Equal(t, int32(0), instance.CurrentWave)
}

func TestDungeonManager_CreateInstance_ClosedDungeon(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  2,
		Name:       "Closed Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     false,
	})

	_, err := dm.CreateInstance(2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not open")
}

func TestDungeonManager_CreateInstance_NotFound(t *testing.T) {
	dm := setupDungeonManager()

	_, err := dm.CreateInstance(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDungeonManager_EnterAndLeave(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
		WaveCount:  3,
		TimeLimit:  600,
	})

	instance, _ := dm.CreateInstance(1)

	t.Run("enter dungeon", func(t *testing.T) {
		err := dm.EnterDungeon(1001, instance.InstanceID)
		assert.NoError(t, err)
		assert.Equal(t, 1, instance.GetPlayerCount())
		assert.True(t, instance.IsPlayerInInstance(1001))
	})

	t.Run("enter again should fail", func(t *testing.T) {
		err := dm.EnterDungeon(1001, instance.InstanceID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already in instance")
	})

	t.Run("second player enters", func(t *testing.T) {
		err := dm.EnterDungeon(1002, instance.InstanceID)
		assert.NoError(t, err)
		assert.Equal(t, 2, instance.GetPlayerCount())
	})

	t.Run("leave dungeon", func(t *testing.T) {
		err := dm.LeaveDungeon(1001, instance.InstanceID)
		assert.NoError(t, err)
		assert.Equal(t, 1, instance.GetPlayerCount())
		assert.False(t, instance.IsPlayerInInstance(1001))
	})

	t.Run("leave again should fail", func(t *testing.T) {
		err := dm.LeaveDungeon(1001, instance.InstanceID)
		assert.Error(t, err)
	})
}

func TestDungeonManager_FullInstance(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Small Dungeon",
		MaxPlayers: 2,
		MinPlayers: 1,
		IsOpen:     true,
	})

	instance, _ := dm.CreateInstance(1)
	dm.EnterDungeon(1001, instance.InstanceID)
	dm.EnterDungeon(1002, instance.InstanceID)

	err := dm.EnterDungeon(1003, instance.InstanceID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "full")
}

func TestDungeonManager_StartDungeon(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
		WaveCount:  3,
		TimeLimit:  600,
	})

	instance, _ := dm.CreateInstance(1)

	t.Run("start without players should fail", func(t *testing.T) {
		err := dm.StartDungeon(instance.InstanceID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enough players")
	})

	dm.EnterDungeon(1001, instance.InstanceID)

	t.Run("start with players", func(t *testing.T) {
		err := dm.StartDungeon(instance.InstanceID)
		assert.NoError(t, err)
		assert.Equal(t, DungeonStatusInProgress, instance.Status)
		assert.Equal(t, int32(1), instance.CurrentWave)
	})

	t.Run("start again should fail", func(t *testing.T) {
		err := dm.StartDungeon(instance.InstanceID)
		assert.Error(t, err)
	})
}

func TestDungeonManager_WaveProgression(t *testing.T) {
	dm := NewDungeonManager()
	tm := tables.NewTableManager()
	loader := tm.GetDungeonLoader()

	loader.AddDungeonForTest(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
		WaveCount:  3,
		TimeLimit:  600,
	})
	loader.AddWaveForTest(&models.DungeonWave{
		WaveID:     1,
		DungeonID:  1,
		WaveIndex:  1,
		MonsterIDs: "1001",
	})
	loader.AddWaveForTest(&models.DungeonWave{
		WaveID:     2,
		DungeonID:  1,
		WaveIndex:  2,
		MonsterIDs: "1002",
	})
	loader.AddWaveForTest(&models.DungeonWave{
		WaveID:     3,
		DungeonID:  1,
		WaveIndex:  3,
		MonsterIDs: "2001",
		IsBoss:     true,
	})

	dm.SetTableManager(tm)

	instance, _ := dm.CreateInstance(1)
	dm.EnterDungeon(1001, instance.InstanceID)
	dm.StartDungeon(instance.InstanceID)

	assert.Equal(t, int32(1), instance.CurrentWave)

	err := dm.AdvanceWave(instance.InstanceID)
	assert.NoError(t, err)
	assert.Equal(t, int32(2), instance.CurrentWave)

	err = dm.AdvanceWave(instance.InstanceID)
	assert.NoError(t, err)
	assert.Equal(t, int32(3), instance.CurrentWave)

	err = dm.AdvanceWave(instance.InstanceID)
	assert.NoError(t, err)
	assert.Equal(t, DungeonStatusCompleted, instance.Status)
	assert.True(t, instance.IsSuccess)
}

func TestDungeonManager_CompleteDungeon(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
		WaveCount:  1,
		TimeLimit:  600,
	})

	instance, _ := dm.CreateInstance(1)
	dm.EnterDungeon(1001, instance.InstanceID)
	dm.StartDungeon(instance.InstanceID)

	t.Run("complete with success", func(t *testing.T) {
		err := dm.CompleteDungeon(instance.InstanceID, true)
		assert.NoError(t, err)
		assert.Equal(t, DungeonStatusCompleted, instance.Status)
		assert.True(t, instance.IsSuccess)

		records := dm.GetPlayerRecords(1001)
		assert.NotNil(t, records)
		if _, ok := records[1]; ok {
			assert.Equal(t, int32(1), records[1].CompletedCount)
		}
	})

	t.Run("complete again should fail", func(t *testing.T) {
		err := dm.CompleteDungeon(instance.InstanceID, true)
		assert.Error(t, err)
	})
}

func TestDungeonManager_PlayerDeath(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
		WaveCount:  1,
		TimeLimit:  600,
	})

	instance, _ := dm.CreateInstance(1)
	dm.EnterDungeon(1001, instance.InstanceID)
	dm.StartDungeon(instance.InstanceID)

	assert.Equal(t, 1, instance.GetAlivePlayerCount())

	dm.PlayerDeath(1001, instance.InstanceID)
	assert.Equal(t, 0, instance.GetAlivePlayerCount())
	assert.Equal(t, DungeonStatusFailed, instance.Status)
}

func TestDungeonManager_PlayerDeath_MultiPlayer(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
		WaveCount:  1,
		TimeLimit:  600,
	})

	instance, _ := dm.CreateInstance(1)
	dm.EnterDungeon(1001, instance.InstanceID)
	dm.EnterDungeon(1002, instance.InstanceID)
	dm.StartDungeon(instance.InstanceID)

	dm.PlayerDeath(1001, instance.InstanceID)
	assert.Equal(t, DungeonStatusInProgress, instance.Status)
	assert.Equal(t, 1, instance.GetAlivePlayerCount())

	dm.PlayerDeath(1002, instance.InstanceID)
	assert.Equal(t, DungeonStatusFailed, instance.Status)
}

func TestDungeonManager_MonsterKilled(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
		WaveCount:  1,
		TimeLimit:  600,
	})

	instance, _ := dm.CreateInstance(1)
	dm.EnterDungeon(1001, instance.InstanceID)
	dm.StartDungeon(instance.InstanceID)

	dm.MonsterKilled(instance.InstanceID, 1001, 3)
	assert.Equal(t, int32(3), instance.KillCount)
	assert.Equal(t, int32(3), instance.TotalKills)

	player := instance.Players[1001]
	assert.Equal(t, int32(3), player.KillCount)
}

func TestDungeonInstance_Progress(t *testing.T) {
	instance := &DungeonInstance{
		DungeonConfig: &models.Dungeon{WaveCount: 4},
		Waves:         make([]*models.DungeonWave, 4),
		CurrentWave:   2,
	}

	progress := instance.GetProgress()
	assert.Equal(t, float32(25.0), progress)
}

func TestDungeonInstance_RemainingTime(t *testing.T) {
	instance := &DungeonInstance{
		Status:        DungeonStatusInProgress,
		StartTime:     time.Now().Add(-100 * time.Second),
		DungeonConfig: &models.Dungeon{TimeLimit: 600},
	}

	remaining := instance.GetRemainingTime()
	assert.Greater(t, remaining, int32(300))
	assert.LessOrEqual(t, remaining, int32(500))
}

func TestDungeonInstance_RemainingTime_NotInProgress(t *testing.T) {
	instance := &DungeonInstance{
		Status:        DungeonStatusWaiting,
		DungeonConfig: &models.Dungeon{TimeLimit: 600},
	}

	remaining := instance.GetRemainingTime()
	assert.Equal(t, int32(0), remaining)
}

func TestDungeonStatus_String(t *testing.T) {
	tests := []struct {
		status   DungeonStatus
		expected string
	}{
		{DungeonStatusNone, "none"},
		{DungeonStatusWaiting, "waiting"},
		{DungeonStatusInProgress, "in_progress"},
		{DungeonStatusCompleted, "completed"},
		{DungeonStatusFailed, "failed"},
		{DungeonStatusClosed, "closed"},
		{DungeonStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestDungeonManager_RemoveInstance(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
	})

	instance, _ := dm.CreateInstance(1)

	err := dm.RemoveInstance(instance.InstanceID)
	assert.NoError(t, err)

	_, exists := dm.GetInstance(instance.InstanceID)
	assert.False(t, exists)

	err = dm.RemoveInstance(instance.InstanceID)
	assert.Error(t, err)
}

func TestDungeonManager_GetActiveInstances(t *testing.T) {
	dm := setupDungeonManager(
		&models.Dungeon{
			DungeonID:  1,
			Name:       "Test Dungeon",
			MaxPlayers: 5,
			MinPlayers: 1,
			IsOpen:     true,
		},
		&models.Dungeon{
			DungeonID:  3,
			Name:       "Boss Dungeon",
			MaxPlayers: 10,
			MinPlayers: 2,
			IsOpen:     true,
		},
	)

	dm.CreateInstance(1)
	dm.CreateInstance(3)

	active := dm.GetActiveInstances()
	assert.Equal(t, 2, len(active))
}

func TestDungeonManager_EmptyInstanceCleanup(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
	})

	instance, _ := dm.CreateInstance(1)
	dm.EnterDungeon(1001, instance.InstanceID)

	dm.LeaveDungeon(1001, instance.InstanceID)

	_, exists := dm.GetInstance(instance.InstanceID)
	assert.False(t, exists)
}

func TestDungeonManager_GetInstanceByMapID(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
	})

	instance, _ := dm.CreateInstance(1)
	instance.SetMapInstanceID(5001)

	found := dm.GetInstanceByMapID(5001)
	assert.NotNil(t, found)
	assert.Equal(t, instance.InstanceID, found.InstanceID)

	notFound := dm.GetInstanceByMapID(9999)
	assert.Nil(t, notFound)
}

func TestDungeonManager_AdvanceWave_NotInProgress(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
	})

	instance, _ := dm.CreateInstance(1)
	err := dm.AdvanceWave(instance.InstanceID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in progress")
}

func TestDungeonManager_EnterDungeon_InvalidInstance(t *testing.T) {
	dm := NewDungeonManager()
	err := dm.EnterDungeon(1001, 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDungeonManager_LeaveDungeon_NotInInstance(t *testing.T) {
	dm := setupDungeonManager(&models.Dungeon{
		DungeonID:  1,
		Name:       "Test Dungeon",
		MaxPlayers: 5,
		MinPlayers: 1,
		IsOpen:     true,
	})

	instance, _ := dm.CreateInstance(1)
	err := dm.LeaveDungeon(1001, instance.InstanceID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in instance")
}
