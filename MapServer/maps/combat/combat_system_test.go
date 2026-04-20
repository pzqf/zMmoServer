package combat

import (
	"testing"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
)

func newTestCombatSystem() *CombatSystem {
	return NewCombatSystem()
}

func newTestPlayer() *object.Player {
	return object.NewPlayer(id.ObjectIdType(1), id.PlayerIdType(1), "TestPlayer", common.Vector3{X: 0, Y: 0, Z: 0}, 1)
}

func newTestMonster() *object.Monster {
	return object.NewMonster(id.ObjectIdType(2), 1, "TestMonster", common.Vector3{X: 5, Y: 0, Z: 5}, 1)
}

func TestCombatSystem_CalculatePhysicalDamage(t *testing.T) {
	cs := newTestCombatSystem()
	player := newTestPlayer()
	monster := newTestMonster()

	result := cs.CalculatePhysicalDamage(player, monster)

	if result.AttackerID != player.GetID() {
		t.Errorf("expected attacker ID %d, got %d", player.GetID(), result.AttackerID)
	}
	if result.TargetID != monster.GetID() {
		t.Errorf("expected target ID %d, got %d", monster.GetID(), result.TargetID)
	}
	if result.Damage < 1 {
		t.Errorf("expected damage >= 1, got %d", result.Damage)
	}
	if result.DamageType != DamageTypePhysical {
		t.Errorf("expected physical damage type, got %d", result.DamageType)
	}
}

func TestCombatSystem_CalculateMonsterDamage(t *testing.T) {
	cs := newTestCombatSystem()
	player := newTestPlayer()
	monster := newTestMonster()

	result := cs.CalculateMonsterDamage(monster, player)

	if result.AttackerID != monster.GetID() {
		t.Errorf("expected attacker ID %d, got %d", monster.GetID(), result.AttackerID)
	}
	if result.TargetID != player.GetID() {
		t.Errorf("expected target ID %d, got %d", player.GetID(), result.TargetID)
	}
	if result.Damage < 1 {
		t.Errorf("expected damage >= 1, got %d", result.Damage)
	}
}

func TestCombatSystem_ApplyDamage_Player(t *testing.T) {
	cs := newTestCombatSystem()
	player := newTestPlayer()
	monster := newTestMonster()

	initialHP := player.GetHealth()
	result := DamageResult{
		AttackerID: monster.GetID(),
		TargetID:   player.GetID(),
		Damage:     10,
		DamageType: DamageTypePhysical,
	}

	isKill := cs.ApplyDamage(player, result)
	if isKill {
		t.Error("expected player not to be killed by 10 damage")
	}
	if player.GetHealth() != initialHP-10 {
		t.Errorf("expected HP %d, got %d", initialHP-10, player.GetHealth())
	}
}

func TestCombatSystem_ApplyDamage_Kill(t *testing.T) {
	cs := newTestCombatSystem()
	player := newTestPlayer()

	player.SetHealth(5)

	result := DamageResult{
		AttackerID: id.ObjectIdType(2),
		TargetID:   player.GetID(),
		Damage:     10,
		DamageType: DamageTypePhysical,
	}

	isKill := cs.ApplyDamage(player, result)
	if !isKill {
		t.Error("expected player to be killed")
	}
	if player.GetHealth() != 0 {
		t.Errorf("expected HP 0, got %d", player.GetHealth())
	}
}

func TestCombatSystem_IsInRange_Close(t *testing.T) {
	cs := newTestCombatSystem()
	player := object.NewPlayer(id.ObjectIdType(1), id.PlayerIdType(1), "Player", common.Vector3{X: 0, Y: 0, Z: 0}, 1)
	monster := object.NewMonster(id.ObjectIdType(2), 1, "Monster", common.Vector3{X: 2, Y: 0, Z: 2}, 1)

	if !cs.IsInRange(player, monster, 5) {
		t.Error("expected objects to be in range")
	}
}

func TestCombatSystem_IsInRange_Far(t *testing.T) {
	cs := newTestCombatSystem()
	player := object.NewPlayer(id.ObjectIdType(1), id.PlayerIdType(1), "Player", common.Vector3{X: 0, Y: 0, Z: 0}, 1)
	monster := object.NewMonster(id.ObjectIdType(2), 1, "Monster", common.Vector3{X: 100, Y: 0, Z: 100}, 1)

	if cs.IsInRange(player, monster, 5) {
		t.Error("expected objects to be out of range")
	}
}

func TestCombatSystem_GetPlayerAttackPower(t *testing.T) {
	cs := newTestCombatSystem()
	player := newTestPlayer()

	attack := cs.GetPlayerAttackPower(player)
	if attack <= 0 {
		t.Errorf("expected positive attack power, got %d", attack)
	}
}

func TestCombatSystem_GetMonsterAttackPower(t *testing.T) {
	cs := newTestCombatSystem()
	monster := newTestMonster()

	attack := cs.GetMonsterAttackPower(monster)
	if attack <= 0 {
		t.Errorf("expected positive attack power, got %d", attack)
	}
}

func TestCombatSystem_GetDefense_Player(t *testing.T) {
	cs := newTestCombatSystem()
	player := newTestPlayer()

	defense := cs.GetDefense(player)
	if defense < 0 {
		t.Errorf("expected non-negative defense, got %d", defense)
	}
}

func TestCombatSystem_GetDefense_Monster(t *testing.T) {
	cs := newTestCombatSystem()
	monster := newTestMonster()

	defense := cs.GetDefense(monster)
	if defense < 0 {
		t.Errorf("expected non-negative defense, got %d", defense)
	}
}
