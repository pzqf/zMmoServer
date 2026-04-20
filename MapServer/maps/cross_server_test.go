package maps

import (
	"testing"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/stretchr/testify/assert"
)

func TestMirrorMapManager_GetOrCreateMirrorMap(t *testing.T) {
	mm := NewMapManager()
	config := MirrorMapConfig{
		SourceMapConfigID:     100,
		MaxPlayersPerInstance: 50,
		MinPlayersForCreate:   1,
		IdleTimeout:           30 * time.Second,
		MaxInstances:          10,
	}

	mgr := NewMirrorMapManager(mm, nil, config)

	mapID, err := mgr.GetOrCreateMirrorMap(100, 1)
	assert.NoError(t, err)
	assert.NotEqual(t, id.MapIdType(0), mapID)
	assert.Equal(t, 1, mgr.GetInstanceCount())

	mapID2, err := mgr.GetOrCreateMirrorMap(100, 1)
	assert.NoError(t, err)
	assert.Equal(t, mapID, mapID2)

	instances := mgr.GetInstancesBySource(100)
	assert.Equal(t, 1, len(instances))
	assert.Equal(t, int32(2), instances[0].PlayerCount)
}

func TestMirrorMapManager_DifferentServerGroups(t *testing.T) {
	mm := NewMapManager()
	config := MirrorMapConfig{
		SourceMapConfigID:     100,
		MaxPlayersPerInstance: 50,
		MinPlayersForCreate:   1,
		IdleTimeout:           30 * time.Second,
		MaxInstances:          10,
	}

	mgr := NewMirrorMapManager(mm, nil, config)

	mapID1, err := mgr.GetOrCreateMirrorMap(100, 1)
	assert.NoError(t, err)

	mapID2, err := mgr.GetOrCreateMirrorMap(100, 2)
	assert.NoError(t, err)

	assert.NotEqual(t, mapID1, mapID2)
	assert.Equal(t, 2, mgr.GetInstanceCount())
}

func TestMirrorMapManager_MaxInstances(t *testing.T) {
	mm := NewMapManager()
	config := MirrorMapConfig{
		SourceMapConfigID:     100,
		MaxPlayersPerInstance: 50,
		MinPlayersForCreate:   1,
		IdleTimeout:           30 * time.Second,
		MaxInstances:          2,
	}

	mgr := NewMirrorMapManager(mm, nil, config)

	_, err := mgr.GetOrCreateMirrorMap(100, 1)
	assert.NoError(t, err)

	_, err = mgr.GetOrCreateMirrorMap(100, 2)
	assert.NoError(t, err)

	_, err = mgr.GetOrCreateMirrorMap(100, 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max mirror instances")
}

func TestMirrorMapManager_PlayerLeave(t *testing.T) {
	mm := NewMapManager()
	config := MirrorMapConfig{
		SourceMapConfigID:     100,
		MaxPlayersPerInstance: 50,
		MinPlayersForCreate:   1,
		IdleTimeout:           30 * time.Second,
		MaxInstances:          10,
	}

	mgr := NewMirrorMapManager(mm, nil, config)

	mapID, _ := mgr.GetOrCreateMirrorMap(100, 1)
	assert.Equal(t, int32(1), mgr.GetTotalPlayerCount())

	mgr.PlayerLeave(mapID)
	assert.Equal(t, int32(0), mgr.GetTotalPlayerCount())

	instances := mgr.GetInstancesBySource(100)
	assert.Equal(t, 0, len(instances))
}

func TestMirrorMapManager_Update_IdleTimeout(t *testing.T) {
	mm := NewMapManager()
	config := MirrorMapConfig{
		SourceMapConfigID:     100,
		MaxPlayersPerInstance: 50,
		MinPlayersForCreate:   1,
		IdleTimeout:           100 * time.Millisecond,
		MaxInstances:          10,
	}

	mgr := NewMirrorMapManager(mm, nil, config)

	mapID, _ := mgr.GetOrCreateMirrorMap(100, 1)

	mgr.mu.Lock()
	if inst, ok := mgr.instances.Load(mapID); ok {
		inst.PlayerCount = 0
		inst.LastActiveAt = time.Now().Add(-200 * time.Millisecond)
	}
	mgr.mu.Unlock()

	assert.Equal(t, 1, mgr.GetInstanceCount())

	mgr.Update(0)

	assert.Equal(t, 0, mgr.GetInstanceCount())
}

func TestMirrorMapManager_GetTotalPlayerCount(t *testing.T) {
	mm := NewMapManager()
	config := MirrorMapConfig{
		SourceMapConfigID:     100,
		MaxPlayersPerInstance: 50,
		MinPlayersForCreate:   1,
		IdleTimeout:           30 * time.Second,
		MaxInstances:          10,
	}

	mgr := NewMirrorMapManager(mm, nil, config)

	mgr.GetOrCreateMirrorMap(100, 1)
	mgr.GetOrCreateMirrorMap(100, 1)

	assert.Equal(t, int32(2), mgr.GetTotalPlayerCount())
}

func TestMapMode_IsCrossServer(t *testing.T) {
	m := NewMap(1, 100, "Test", 500, 500, nil)

	m.SetMapMode(MapModeSingleServer)
	assert.False(t, m.IsCrossServer())

	m.SetMapMode(MapModeCrossGroup)
	assert.True(t, m.IsCrossServer())

	m.SetMapMode(MapModeMirror)
	assert.True(t, m.IsCrossServer())
}

func TestMap_DungeonFields(t *testing.T) {
	m := NewMap(1, 100, "Test", 500, 500, nil)

	assert.False(t, m.IsDungeon())

	m.SetIsDungeon(true)
	assert.True(t, m.IsDungeon())

	m.SetDungeonInstanceID(42)
	assert.Equal(t, id.InstanceIdType(42), m.GetDungeonInstanceID())
}

func TestMap_ServerGroupID(t *testing.T) {
	m := NewMap(1, 100, "Test", 500, 500, nil)

	m.SetServerGroupID(5)
	assert.Equal(t, int32(5), m.GetServerGroupID())
}
