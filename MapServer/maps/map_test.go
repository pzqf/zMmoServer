package maps

import (
	"testing"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
)

func newTestMap() *Map {
	return NewMap(id.MapIdType(1), 1, "TestMap", 1000, 1000, nil)
}

func TestNewMap(t *testing.T) {
	m := newTestMap()

	if m.GetID() != id.MapIdType(1) {
		t.Errorf("expected map ID 1, got %d", m.GetID())
	}
	if m.GetName() != "TestMap" {
		t.Errorf("expected map name TestMap, got %s", m.GetName())
	}
	if m.GetWidth() != 1000 {
		t.Errorf("expected width 1000, got %f", m.GetWidth())
	}
	if m.GetHeight() != 1000 {
		t.Errorf("expected height 1000, got %f", m.GetHeight())
	}
}

func TestMap_AddAndGetObject(t *testing.T) {
	m := newTestMap()

	player := object.NewPlayer(id.ObjectIdType(100), id.PlayerIdType(1), "TestPlayer", common.Vector3{X: 100, Y: 0, Z: 100}, 1)
	m.AddObject(player)

	retrieved := m.GetObject(id.ObjectIdType(100))
	if retrieved == nil {
		t.Fatal("expected to retrieve player, got nil")
	}
	if retrieved.GetID() != id.ObjectIdType(100) {
		t.Errorf("expected object ID 100, got %d", retrieved.GetID())
	}
}

func TestMap_RemoveObject(t *testing.T) {
	m := newTestMap()

	player := object.NewPlayer(id.ObjectIdType(100), id.PlayerIdType(1), "TestPlayer", common.Vector3{X: 100, Y: 0, Z: 100}, 1)
	m.AddObject(player)
	m.RemoveObject(id.ObjectIdType(100))

	retrieved := m.GetObject(id.ObjectIdType(100))
	if retrieved != nil {
		t.Error("expected nil after removal")
	}
}

func TestMap_MoveObject(t *testing.T) {
	m := newTestMap()

	player := object.NewPlayer(id.ObjectIdType(100), id.PlayerIdType(1), "TestPlayer", common.Vector3{X: 100, Y: 0, Z: 100}, 1)
	m.AddObject(player)

	newPos := common.Vector3{X: 200, Y: 0, Z: 200}
	err := m.MoveObject(id.ObjectIdType(100), newPos)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved := m.GetObject(id.ObjectIdType(100))
	if retrieved.GetPosition().X != 200 || retrieved.GetPosition().Z != 200 {
		t.Errorf("expected position (200, 200), got (%f, %f)", retrieved.GetPosition().X, retrieved.GetPosition().Z)
	}
}

func TestMap_MoveNonExistentObject(t *testing.T) {
	m := newTestMap()

	err := m.MoveObject(id.ObjectIdType(999), common.Vector3{X: 200, Y: 0, Z: 200})
	if err == nil {
		t.Error("expected error for non-existent object")
	}
}

func TestMap_GetObjectsInRange(t *testing.T) {
	m := newTestMap()

	player1 := object.NewPlayer(id.ObjectIdType(100), id.PlayerIdType(1), "Player1", common.Vector3{X: 100, Y: 0, Z: 100}, 1)
	player2 := object.NewPlayer(id.ObjectIdType(101), id.PlayerIdType(2), "Player2", common.Vector3{X: 110, Y: 0, Z: 110}, 1)
	player3 := object.NewPlayer(id.ObjectIdType(102), id.PlayerIdType(3), "Player3", common.Vector3{X: 500, Y: 0, Z: 500}, 1)

	m.AddObject(player1)
	m.AddObject(player2)
	m.AddObject(player3)

	objects := m.GetObjectsInRange(common.Vector3{X: 100, Y: 0, Z: 100}, 20)
	if len(objects) < 2 {
		t.Errorf("expected at least 2 objects in range, got %d", len(objects))
	}

	farObjects := m.GetObjectsInRange(common.Vector3{X: 100, Y: 0, Z: 100}, 15)
	if len(farObjects) < 1 {
		t.Errorf("expected at least 1 object in range, got %d", len(farObjects))
	}
}

func TestMap_GetPlayerCount(t *testing.T) {
	m := newTestMap()

	if m.GetPlayerCount() != 0 {
		t.Errorf("expected 0 players, got %d", m.GetPlayerCount())
	}

	player1 := object.NewPlayer(id.ObjectIdType(100), id.PlayerIdType(1), "Player1", common.Vector3{X: 100, Y: 0, Z: 100}, 1)
	m.AddObject(player1)

	if m.GetPlayerCount() != 1 {
		t.Errorf("expected 1 player, got %d", m.GetPlayerCount())
	}

	player2 := object.NewPlayer(id.ObjectIdType(101), id.PlayerIdType(2), "Player2", common.Vector3{X: 200, Y: 0, Z: 200}, 1)
	m.AddObject(player2)

	if m.GetPlayerCount() != 2 {
		t.Errorf("expected 2 players, got %d", m.GetPlayerCount())
	}

	m.RemoveObject(id.ObjectIdType(100))
	if m.GetPlayerCount() != 1 {
		t.Errorf("expected 1 player after removal, got %d", m.GetPlayerCount())
	}
}

func TestMap_IsPositionInMap(t *testing.T) {
	m := newTestMap()

	tests := []struct {
		name     string
		pos      common.Vector3
		expected bool
	}{
		{"origin", common.Vector3{X: 0, Y: 0, Z: 0}, true},
		{"center", common.Vector3{X: 500, Y: 0, Z: 500}, true},
		{"edge", common.Vector3{X: 1000, Y: 0, Z: 1000}, true},
		{"out_of_bounds_x", common.Vector3{X: 1001, Y: 0, Z: 500}, false},
		{"out_of_bounds_z", common.Vector3{X: 500, Y: 0, Z: 1001}, false},
		{"negative_x", common.Vector3{X: -1, Y: 0, Z: 500}, false},
		{"negative_z", common.Vector3{X: 500, Y: 0, Z: -1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.IsPositionInMap(tt.pos)
			if result != tt.expected {
				t.Errorf("IsPositionInMap(%v) = %v, want %v", tt.pos, result, tt.expected)
			}
		})
	}
}

func TestMap_CalculateDistance(t *testing.T) {
	m := newTestMap()

	tests := []struct {
		name     string
		pos1     common.Vector3
		pos2     common.Vector3
		expected float32
		delta    float32
	}{
		{"same_position", common.Vector3{X: 0, Y: 0, Z: 0}, common.Vector3{X: 0, Y: 0, Z: 0}, 0, 0.01},
		{"unit_x", common.Vector3{X: 0, Y: 0, Z: 0}, common.Vector3{X: 1, Y: 0, Z: 0}, 1, 0.01},
		{"unit_z", common.Vector3{X: 0, Y: 0, Z: 0}, common.Vector3{X: 0, Y: 0, Z: 1}, 1, 0.01},
		{"diagonal", common.Vector3{X: 0, Y: 0, Z: 0}, common.Vector3{X: 3, Y: 0, Z: 4}, 5, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.CalculateDistance(tt.pos1, tt.pos2)
			if result < tt.expected-tt.delta || result > tt.expected+tt.delta {
				t.Errorf("CalculateDistance(%v, %v) = %f, want %f", tt.pos1, tt.pos2, result, tt.expected)
			}
		})
	}
}

func TestMap_AddPlayer(t *testing.T) {
	m := newTestMap()

	err := m.AddPlayer(id.PlayerIdType(1), id.ObjectIdType(1), 100, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.GetPlayerCount() != 1 {
		t.Errorf("expected 1 player, got %d", m.GetPlayerCount())
	}
}

func TestMap_RemovePlayer(t *testing.T) {
	m := newTestMap()

	m.AddPlayer(id.PlayerIdType(1), id.ObjectIdType(1), 100, 0, 100)
	m.RemovePlayer(id.PlayerIdType(1))

	if m.GetPlayerCount() != 0 {
		t.Errorf("expected 0 players, got %d", m.GetPlayerCount())
	}
}

func TestMap_AddNilObject(t *testing.T) {
	m := newTestMap()

	m.AddObject(nil)

	if m.GetPlayerCount() != 0 {
		t.Error("expected 0 players after adding nil")
	}
}

func TestMap_GetObjectsByType(t *testing.T) {
	m := newTestMap()

	player := object.NewPlayer(id.ObjectIdType(100), id.PlayerIdType(1), "Player1", common.Vector3{X: 100, Y: 0, Z: 100}, 1)
	monster := object.NewMonster(id.ObjectIdType(200), 1, "Monster", common.Vector3{X: 200, Y: 0, Z: 200}, 1)

	m.AddObject(player)
	m.AddObject(monster)

	players := m.GetObjectsByType(common.GameObjectTypePlayer)
	if len(players) != 1 {
		t.Errorf("expected 1 player, got %d", len(players))
	}

	monsters := m.GetObjectsByType(common.GameObjectTypeMonster)
	if len(monsters) != 1 {
		t.Errorf("expected 1 monster, got %d", len(monsters))
	}
}

func TestMap_SpawnPoints(t *testing.T) {
	m := newTestMap()

	if len(m.GetSpawnPoints()) != 0 {
		t.Error("expected 0 spawn points initially")
	}
}

func TestMap_Cleanup(t *testing.T) {
	m := newTestMap()

	player := object.NewPlayer(id.ObjectIdType(100), id.PlayerIdType(1), "Player1", common.Vector3{X: 100, Y: 0, Z: 100}, 1)
	m.AddObject(player)

	m.Cleanup()

	if m.GetPlayerCount() != 0 {
		t.Errorf("expected 0 players after cleanup, got %d", m.GetPlayerCount())
	}
}
