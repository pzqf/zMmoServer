package object

import (
	"testing"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/MapServer/common"
)

func TestNewPlayer(t *testing.T) {
	pos := common.Vector3{X: 100, Y: 0, Z: 200}
	player := NewPlayer(id.ObjectIdType(1), id.PlayerIdType(10), "TestPlayer", pos, 1)

	if player.GetID() != id.ObjectIdType(1) {
		t.Errorf("expected object ID 1, got %d", player.GetID())
	}
	if player.GetPlayerID() != id.PlayerIdType(10) {
		t.Errorf("expected player ID 10, got %d", player.GetPlayerID())
	}
	if player.GetName() != "TestPlayer" {
		t.Errorf("expected name TestPlayer, got %s", player.GetName())
	}
	if player.GetType() != common.GameObjectTypePlayer {
		t.Errorf("expected player type, got %d", player.GetType())
	}
	if player.GetPosition().X != 100 || player.GetPosition().Z != 200 {
		t.Errorf("expected position (100, 200), got (%f, %f)", player.GetPosition().X, player.GetPosition().Z)
	}
}

func TestPlayer_Health(t *testing.T) {
	player := NewPlayer(id.ObjectIdType(1), id.PlayerIdType(10), "TestPlayer", common.Vector3{}, 1)

	if player.GetHealth() <= 0 {
		t.Error("expected positive initial health")
	}
	if player.GetMaxHealth() <= 0 {
		t.Error("expected positive max health")
	}

	player.SetHealth(50)
	if player.GetHealth() != 50 {
		t.Errorf("expected health 50, got %d", player.GetHealth())
	}
}

func TestPlayer_Mana(t *testing.T) {
	player := NewPlayer(id.ObjectIdType(1), id.PlayerIdType(10), "TestPlayer", common.Vector3{}, 1)

	if player.GetMana() <= 0 {
		t.Error("expected positive initial mana")
	}

	player.SetMana(30)
	if player.GetMana() != 30 {
		t.Errorf("expected mana 30, got %d", player.GetMana())
	}
}

func TestPlayer_Level(t *testing.T) {
	player := NewPlayer(id.ObjectIdType(1), id.PlayerIdType(10), "TestPlayer", common.Vector3{}, 1)

	if player.GetLevel() != 1 {
		t.Errorf("expected level 1, got %d", player.GetLevel())
	}

	player.SetLevel(10)
	if player.GetLevel() != 10 {
		t.Errorf("expected level 10, got %d", player.GetLevel())
	}
}

func TestPlayer_Experience(t *testing.T) {
	player := NewPlayer(id.ObjectIdType(1), id.PlayerIdType(10), "TestPlayer", common.Vector3{}, 1)

	player.AddExperience(500)
	if player.GetExperience() != 500 {
		t.Errorf("expected experience 500, got %d", player.GetExperience())
	}
}

func TestPlayer_AttackDefense(t *testing.T) {
	player := NewPlayer(id.ObjectIdType(1), id.PlayerIdType(10), "TestPlayer", common.Vector3{}, 1)

	attack := player.GetAttack()
	if attack <= 0 {
		t.Errorf("expected positive attack, got %d", attack)
	}

	defense := player.GetDefense()
	if defense < 0 {
		t.Errorf("expected non-negative defense, got %d", defense)
	}
}

func TestPlayer_Position(t *testing.T) {
	player := NewPlayer(id.ObjectIdType(1), id.PlayerIdType(10), "TestPlayer", common.Vector3{X: 10, Y: 0, Z: 20}, 1)

	newPos := common.Vector3{X: 30, Y: 0, Z: 40}
	player.SetPosition(newPos)

	if player.GetPosition().X != 30 || player.GetPosition().Z != 40 {
		t.Errorf("expected position (30, 40), got (%f, %f)", player.GetPosition().X, player.GetPosition().Z)
	}
}

func TestPlayer_SkillCooldown(t *testing.T) {
	player := NewPlayer(id.ObjectIdType(1), id.PlayerIdType(10), "TestPlayer", common.Vector3{}, 1)

	if player.IsSkillInCooldown(1) {
		t.Error("expected skill 1 not to be on cooldown initially")
	}

	player.SetSkillCooldown(1, 10*time.Second)
	if !player.IsSkillInCooldown(1) {
		t.Error("expected skill 1 to be on cooldown after setting")
	}
}

func TestPlayer_TakeDamage(t *testing.T) {
	player := NewPlayer(id.ObjectIdType(1), id.PlayerIdType(10), "TestPlayer", common.Vector3{}, 1)

	initialHP := player.GetHealth()
	player.TakeDamage(10)

	if player.GetHealth() != initialHP-10 {
		t.Errorf("expected HP %d, got %d", initialHP-10, player.GetHealth())
	}
}

func TestPlayer_AddExp(t *testing.T) {
	player := NewPlayer(id.ObjectIdType(1), id.PlayerIdType(10), "TestPlayer", common.Vector3{}, 1)

	player.AddExp(100)
	if player.GetExperience() != 100 {
		t.Errorf("expected exp 100, got %d", player.GetExperience())
	}
}

func TestNewMonster(t *testing.T) {
	pos := common.Vector3{X: 50, Y: 0, Z: 60}
	monster := NewMonster(id.ObjectIdType(2), 100, "TestMonster", pos, 1)

	if monster.GetID() != id.ObjectIdType(2) {
		t.Errorf("expected object ID 2, got %d", monster.GetID())
	}
	if monster.GetMonsterID() != 100 {
		t.Errorf("expected monster config ID 100, got %d", monster.GetMonsterID())
	}
	if monster.GetType() != common.GameObjectTypeMonster {
		t.Errorf("expected monster type, got %d", monster.GetType())
	}
}

func TestMonster_Health(t *testing.T) {
	monster := NewMonster(id.ObjectIdType(2), 100, "TestMonster", common.Vector3{}, 1)

	if monster.GetHealth() <= 0 {
		t.Error("expected positive initial health")
	}

	monster.TakeDamage(10)
	if monster.GetHealth() != monster.GetMaxHealth()-10 {
		t.Errorf("expected HP %d, got %d", monster.GetMaxHealth()-10, monster.GetHealth())
	}
}

func TestMonster_AttackDefense(t *testing.T) {
	monster := NewMonster(id.ObjectIdType(2), 100, "TestMonster", common.Vector3{}, 1)

	attack := monster.GetAttack()
	if attack <= 0 {
		t.Errorf("expected positive attack, got %d", attack)
	}

	defense := monster.GetDefense()
	if defense < 0 {
		t.Errorf("expected non-negative defense, got %d", defense)
	}
}

func TestNewNPC(t *testing.T) {
	pos := common.Vector3{X: 100, Y: 0, Z: 100}
	npc := NewNPC(id.ObjectIdType(3), 200, "TestNPC", pos, "Hello")

	if npc.GetID() != id.ObjectIdType(3) {
		t.Errorf("expected object ID 3, got %d", npc.GetID())
	}
	if npc.GetNPCID() != 200 {
		t.Errorf("expected NPC config ID 200, got %d", npc.GetNPCID())
	}
	if npc.GetType() != common.GameObjectTypeNPC {
		t.Errorf("expected NPC type, got %d", npc.GetType())
	}
}

func TestNewItem(t *testing.T) {
	pos := common.Vector3{X: 100, Y: 0, Z: 100}
	item := NewItem(id.ObjectIdType(4), 300, "TestItem", pos, 5, ItemTypeConsumable, ItemRarityCommon)

	if item.GetID() != id.ObjectIdType(4) {
		t.Errorf("expected object ID 4, got %d", item.GetID())
	}
	if item.GetItemID() != 300 {
		t.Errorf("expected item config ID 300, got %d", item.GetItemID())
	}
	if item.GetType() != common.GameObjectTypeItem {
		t.Errorf("expected item type, got %d", item.GetType())
	}
}
