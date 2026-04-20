package maps

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/combat"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"go.uber.org/zap"
)

func (m *Map) HandleObjectInteraction(player *object.Player, targetObject common.IGameObject) error {
	if targetObject == nil {
		return fmt.Errorf("target object not found")
	}

	switch targetObject.GetType() {
	case common.GameObjectTypeNPC:
		return m.handleNPCInteraction(player, targetObject)
	case common.GameObjectTypeMonster:
		return m.handleMonsterInteraction(player, targetObject)
	case common.GameObjectTypeItem:
		return m.handleItemInteraction(player, targetObject)
	default:
		return fmt.Errorf("unsupported object type")
	}
}

func (m *Map) handleNPCInteraction(player *object.Player, npc common.IGameObject) error {
	zLog.Info("Player interacted with NPC",
		zap.Int64("player_id", int64(player.GetPlayerID())),
		zap.Int64("npc_id", int64(npc.GetID())))
	return nil
}

func (m *Map) handleMonsterInteraction(player *object.Player, monster common.IGameObject) error {
	zLog.Info("Player attacked monster",
		zap.Int64("player_id", int64(player.GetPlayerID())),
		zap.Int64("monster_id", int64(monster.GetID())))
	return nil
}

func (m *Map) handleItemInteraction(player *object.Player, item common.IGameObject) error {
	zLog.Info("Player picked up item",
		zap.Int64("player_id", int64(player.GetPlayerID())),
		zap.Int64("item_id", int64(item.GetID())))

	m.RemoveObject(item.GetID())

	if inventoryItem, ok := item.(*object.Item); ok {
		return m.AddItem(player, inventoryItem.GetItemID(), 1)
	}

	return nil
}

func (m *Map) handlePlayerDeath(player *object.Player, killer common.IGameObject) {
	zLog.Info("Player died",
		zap.Int64("player_id", int64(player.GetPlayerID())),
		zap.Int64("killer_id", int64(killer.GetID())))

	player.SetHealth(0)

	expLoss := int64(player.GetLevel()) * 10
	player.AddExperience(-expLoss)
	if player.GetExperience() < 0 {
		player.SetExperience(0)
	}

	go func() {
		time.Sleep(5 * time.Second)
		player.SetHealth(player.GetMaxHealth())
		spawnPos := common.Vector3{X: m.width / 2, Y: 0, Z: m.height / 2}
		player.SetPosition(spawnPos)
		zLog.Info("Player auto-respawned",
			zap.Int64("player_id", int64(player.GetPlayerID())))
	}()
}

func (m *Map) handleMonsterDeath(monster *object.Monster, killer common.IGameObject) {
	zLog.Info("Monster killed",
		zap.Int64("monster_id", int64(monster.GetID())),
		zap.Int32("monster_config_id", monster.GetMonsterID()),
		zap.Int64("killer_id", int64(killer.GetID())))

	if player, ok := killer.(*object.Player); ok {
		exp := monster.GetExpReward()
		if exp <= 0 {
			exp = int64(monster.GetLevel() * 5)
		}
		player.AddExperience(exp)

		zLog.Debug("Exp awarded",
			zap.Int64("player_id", int64(player.GetPlayerID())),
			zap.Int64("exp", exp))
	}

	if m.lootSystem != nil {
		lootResults := m.lootSystem.GenerateLoot(monster.GetMonsterID(), monster.GetLevel(), monster.GetDifficulty())
		for i, lootItem := range lootResults {
			itemObjectID := id.ObjectIdType(time.Now().UnixNano()%1000000000 + int64(i)*10000 + int64(lootItem.ItemID))
			newItem := object.NewItem(itemObjectID, lootItem.ItemID, "Loot", monster.GetPosition(), lootItem.Count,
				object.ItemTypeConsumable, object.ItemRarityCommon)
			m.AddObject(newItem)

			zLog.Debug("Loot item dropped",
				zap.Int32("item_id", lootItem.ItemID),
				zap.Int32("count", lootItem.Count),
				zap.Float32("x", monster.GetPosition().X),
				zap.Float32("z", monster.GetPosition().Z))
		}
	}

	m.scheduleMonsterRespawn(monster)
	m.RemoveObject(monster.GetID())
}

func (m *Map) scheduleMonsterRespawn(monster *object.Monster) {
	respawnTime := int32(30)
	if m.lootSystem != nil {
		respawnTime = m.lootSystem.GetRespawnTime(monster.GetMonsterID())
	}

	monsterConfigID := monster.GetMonsterID()
	monsterPos := monster.GetPosition()

	go func() {
		time.Sleep(time.Duration(respawnTime) * time.Second)
		objectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)
		newMonster := object.NewMonster(objectID, monsterConfigID, "Monster", monsterPos, 1)

		if m.lootSystem != nil {
			config := m.lootSystem.GetMonsterConfig(monsterConfigID)
			if config != nil {
				newMonster.SetAIType(config.AIType)
				newMonster.SetDifficulty(config.Difficulty)
			}
		}

		m.AddObject(newMonster)
		zLog.Debug("Monster respawned",
			zap.Int32("monster_config_id", monsterConfigID),
			zap.Int64("object_id", int64(objectID)))
	}()
}

func (m *Map) dropItems(position common.Vector3, monsterLevel int32) {
	if rand.Float32() < 0.5 {
		itemID := int32(1)
		itemObjectID := id.ObjectIdType(time.Now().UnixNano() % 1000000000)
		newItem := object.NewItem(itemObjectID, itemID, "Test Item", position, 1, object.ItemTypeConsumable, object.ItemRarityCommon)
		m.AddObject(newItem)

		zLog.Debug("Item dropped", zap.Int32("item_id", itemID), zap.Float32("x", position.X), zap.Float32("y", position.Y))
	}
}

func (m *Map) AttackTarget(playerID id.PlayerIdType, objectID id.ObjectIdType, targetID id.ObjectIdType) (int64, int64, error) {
	attacker := m.GetObject(objectID)
	if attacker == nil {
		return 0, 0, fmt.Errorf("attacker not found")
	}

	target := m.GetObject(targetID)
	if target == nil {
		return 0, 0, fmt.Errorf("target not found")
	}

	attackRange := float32(3.0)
	if player, ok := attacker.(*object.Player); ok {
		attackRange = player.GetAttackRange()
		if attackRange <= 0 {
			attackRange = 3.0
		}
	}

	if !m.combatSystem.IsInRange(attacker, target, attackRange) {
		return 0, 0, fmt.Errorf("target out of range")
	}

	var result combat.DamageResult
	switch a := attacker.(type) {
	case *object.Player:
		result = m.combatSystem.CalculatePhysicalDamage(a, target)
	case *object.Monster:
		result = m.combatSystem.CalculateMonsterDamage(a, target)
	default:
		return 0, 0, fmt.Errorf("attacker type cannot attack")
	}

	isKill := m.combatSystem.ApplyDamage(target, result)

	if isKill {
		switch t := target.(type) {
		case *object.Player:
			m.handlePlayerDeath(t, attacker)
		case *object.Monster:
			m.handleMonsterDeath(t, attacker)
		}
	}

	return result.Damage, 0, nil
}

func (m *Map) GetTargetInRange(position common.Vector3, skillRange float32, casterID id.ObjectIdType, targetTypes []common.GameObjectType) []common.IGameObject {
	objects := m.GetObjectsInRange(position, skillRange)
	targets := make([]common.IGameObject, 0)

	for _, obj := range objects {
		if obj.GetID() == casterID {
			continue
		}

		objType := obj.GetType()
		for _, targetType := range targetTypes {
			if objType == targetType {
				targets = append(targets, obj)
				break
			}
		}
	}

	return targets
}

func (m *Map) GetNearestTarget(position common.Vector3, skillRange float32, casterID id.ObjectIdType, targetTypes []common.GameObjectType) common.IGameObject {
	targets := m.GetTargetInRange(position, skillRange, casterID, targetTypes)
	if len(targets) == 0 {
		return nil
	}

	var nearestTarget common.IGameObject
	minDistance := float32(math.MaxFloat32)

	for _, target := range targets {
		distance := position.DistanceTo(target.GetPosition())
		if distance < minDistance {
			minDistance = distance
			nearestTarget = target
		}
	}

	return nearestTarget
}

func (m *Map) IsPositionInMap(position common.Vector3) bool {
	return position.X >= 0 && position.X <= m.width &&
		position.Z >= 0 && position.Z <= m.height
}

func (m *Map) ValidateTarget(caster *object.Player, target common.IGameObject, skillRange float32) bool {
	distance := caster.GetPosition().DistanceTo(target.GetPosition())
	return distance <= skillRange
}

func (m *Map) CalculateDistance(pos1, pos2 common.Vector3) float32 {
	return pos1.DistanceTo(pos2)
}
