package maps

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
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
		// MAP-2: 复活改动玩家状态，串行回地图 goroutine。
		m.Do(func() {
			// MAP-10: 存活校验——玩家可能在这 5s 内已离场（RemoveObject）。若已不在本地图则不复活，
			// 避免复活一个已离场的对象（其可能已在别处重新进入）。
			if m.GetObject(id.ObjectIdType(player.GetPlayerID())) == nil {
				return
			}
			player.SetHealth(player.GetMaxHealth())
			spawnPos := common.Vector3{X: m.width / 2, Y: 0, Z: m.height / 2}
			player.SetPosition(spawnPos)
			zLog.Info("Player auto-respawned",
				zap.Int64("player_id", int64(player.GetPlayerID())))
		})
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
		player.AddExperience(exp) // MapServer 战斗对象本地经验（战斗态显示用）
		// 回写给 GameServer 持久化 actor（修 F-2）：此前战斗经验只加在地图对象、跨登录全丢。
		if m.aoiNotifier != nil {
			m.aoiNotifier.NotifyExpGrant(int64(player.GetPlayerID()), exp)
		}

		zLog.Debug("Exp awarded",
			zap.Int64("player_id", int64(player.GetPlayerID())),
			zap.Int64("exp", exp))
	}

	if m.lootSystem != nil {
		lootResults := m.lootSystem.GenerateLoot(monster.GetMonsterID(), monster.GetLevel(), monster.GetDifficulty())
		for _, lootItem := range lootResults {
			itemObjectID := nextMapObjectID()
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
		// MAP-2: 重生怪进地图，串行回地图 goroutine。
		m.Do(func() {
			objectID := nextMapObjectID()
			// OPT-7: 重生怪等级从配置读，不再硬编码 1。
			newMonster := object.NewMonster(objectID, monsterConfigID, "Monster", monsterPos, m.monsterConfigLevel(monsterConfigID))

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
		})
	}()
}

func (m *Map) dropItems(position common.Vector3, monsterLevel int32) {
	if rand.Float32() < 0.5 {
		itemID := int32(1)
		itemObjectID := nextMapObjectID()
		newItem := object.NewItem(itemObjectID, itemID, "Test Item", position, 1, object.ItemTypeConsumable, object.ItemRarityCommon)
		m.AddObject(newItem)

		zLog.Debug("Item dropped", zap.Int32("item_id", itemID), zap.Float32("x", position.X), zap.Float32("y", position.Y))
	}
}

// AttackTarget 玩家攻击（网络入口）。经 Do 串行到地图 goroutine，与 AI/帧更新互斥（MAP-2）。
func (m *Map) AttackTarget(playerID id.PlayerIdType, objectID id.ObjectIdType, targetID id.ObjectIdType) (int64, int64, error) {
	var dmg, extra int64
	var err error
	m.Do(func() {
		dmg, extra, err = m.attackTargetLocked(playerID, objectID, targetID)
	})
	return dmg, extra, err
}

func (m *Map) attackTargetLocked(playerID id.PlayerIdType, objectID id.ObjectIdType, targetID id.ObjectIdType) (int64, int64, error) {
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

	// 场景同步增强：把 target 的最新血量广播给视野内玩家（不止攻击者自己）。
	m.broadcastEntityAttr(target)

	if isKill {
		m.broadcastEntityDeath(target, attacker)
		switch t := target.(type) {
		case *object.Player:
			m.handlePlayerDeath(t, attacker)
		case *object.Monster:
			m.handleMonsterDeath(t, attacker)
		}
	}

	return result.Damage, 0, nil
}

// aoiBroadcastRadius 状态变更广播半径。取 AOI 网格边长的 2 倍，覆盖 target 所在格及相邻格
// 里的观察者（近似 AOI 视野范围）。见 defaultGridSize。
const aoiBroadcastRadius = defaultGridSize * 2

// broadcastEntityAttr 把 target 的最新血量广播给视野内所有玩家(watcher)。
// 由地图 actor goroutine 在战斗结算后同步调用；NotifyAOI 仅把值化事件入队，不阻塞 actor（MAP-7）。
// 血量经 pos 载荷透传：X=当前血量, Y=最大血量（对端 aoi_push.go case AOIAttr 还原）。
func (m *Map) broadcastEntityAttr(target common.IGameObject) {
	if m.aoiNotifier == nil || target == nil {
		return
	}
	var curHP, maxHP int32
	switch t := target.(type) {
	case *object.Player:
		curHP, maxHP = t.GetHealth(), t.GetMaxHealth()
	case *object.Monster:
		curHP, maxHP = t.GetHealth(), t.GetMaxHealth()
	default:
		return
	}
	pos := target.GetPosition()
	payload := common.Vector3{X: float32(curHP), Y: float32(maxHP)}
	for _, w := range m.GetPlayersInRange(pos, aoiBroadcastRadius) {
		wp, ok := w.(*object.Player)
		if !ok {
			continue
		}
		// watcher 用玩家 ID（GameServer 侧按 playerID 查会话；见 aoi_push.sendAOINotify）。
		m.aoiNotifier.NotifyAOI(int64(wp.GetPlayerID()), int32(m.mapID),
			crossserver.MsgInternalAOIAttr, int64(target.GetID()), payload)
	}
}

// broadcastEntityDeath 把 target 死亡广播给视野内所有玩家(watcher)。
// killer 对象 ID 经 pos.X 载荷透传（对端 aoi_push.go case AOIDeath 还原）。
func (m *Map) broadcastEntityDeath(target, killer common.IGameObject) {
	if m.aoiNotifier == nil || target == nil {
		return
	}
	pos := target.GetPosition()
	var killerID int64
	if killer != nil {
		killerID = int64(killer.GetID())
	}
	payload := common.Vector3{X: float32(killerID)}
	for _, w := range m.GetPlayersInRange(pos, aoiBroadcastRadius) {
		wp, ok := w.(*object.Player)
		if !ok {
			continue
		}
		m.aoiNotifier.NotifyAOI(int64(wp.GetPlayerID()), int32(m.mapID),
			crossserver.MsgInternalAOIDeath, int64(target.GetID()), payload)
	}
}

// buffRemainingSec 返回玩家某 buff 的剩余秒数（buff 增广播载荷用）；查不到返回 0。
func (m *Map) buffRemainingSec(playerID id.PlayerIdType, buffID int32) float32 {
	if m.buffManager == nil {
		return 0
	}
	for _, ab := range m.buffManager.GetActiveBuffs(playerID) {
		if ab.BuffID == buffID {
			return float32(ab.GetRemainingTime().Seconds())
		}
	}
	return 0
}

// broadcastEntityBuff 把 target 的 buff 增/删广播给视野内所有玩家(watcher)。
// added=true 为获得(remainingSec 为剩余秒数),false 为移除(remainingSec 忽略)。
// buff 载荷经 pos 透传：X=buffID, Y=added?1:0, Z=剩余秒数（对端 aoi_push case AOIBuff 还原）。
func (m *Map) broadcastEntityBuff(target common.IGameObject, buffID int32, added bool, remainingSec float32) {
	if m.aoiNotifier == nil || target == nil {
		return
	}
	var addedF float32
	if added {
		addedF = 1
	}
	pos := target.GetPosition()
	payload := common.Vector3{X: float32(buffID), Y: addedF, Z: remainingSec}
	for _, w := range m.GetPlayersInRange(pos, aoiBroadcastRadius) {
		wp, ok := w.(*object.Player)
		if !ok {
			continue
		}
		m.aoiNotifier.NotifyAOI(int64(wp.GetPlayerID()), int32(m.mapID),
			crossserver.MsgInternalAOIBuff, int64(target.GetID()), payload)
	}
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
