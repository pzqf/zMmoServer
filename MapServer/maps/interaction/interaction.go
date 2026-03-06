package interaction

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"go.uber.org/zap"
)

// InteractionManager 交互管理器
type InteractionManager struct {
	mapInstance *maps.Map
}

// NewInteractionManager 创建新的交互管理器
func NewInteractionManager(mapInstance *maps.Map) *InteractionManager {
	return &InteractionManager{
		mapInstance: mapInstance,
	}
}

// HandlePlayerAttack 处理玩家攻击
func (im *InteractionManager) HandlePlayerAttack(player *object.Player, targetID int64) bool {
	// 查找目标对象
	target := im.mapInstance.GetObjectByID(int64(targetID))
	if target == nil {
		zLog.Warn("Target not found", zap.Int64("target_id", targetID))
		return false
	}

	// 检查目标是否是怪物
	if target.GetType() == common.GameObjectTypeMonster {
		monster, ok := target.(*object.Monster)
		if ok && monster.CanAttack() {
			// 计算伤害
			damage := player.GetStrength() * 2
			monster.SetHealth(monster.GetHealth() - damage)

			zLog.Info("Player attacked monster",
				zap.String("player", player.GetName()),
				zap.String("monster", monster.GetName()),
				zap.Int32("damage", damage),
				zap.Int32("monster_health", monster.GetHealth()))

			// 检查怪物是否死亡
			if monster.IsDead() {
				im.HandleMonsterDeath(monster, player)
			}

			return true
		}
	}

	return false
}

// HandleMonsterAttack 处理怪物攻击
func (im *InteractionManager) HandleMonsterAttack(monster *object.Monster, targetID int64) bool {
	// 查找目标对象
	target := im.mapInstance.GetObjectByID(int64(targetID))
	if target == nil {
		zLog.Warn("Target not found", zap.Int64("target_id", targetID))
		return false
	}

	// 检查目标是否是玩家
	if target.GetType() == common.GameObjectTypePlayer {
		player, ok := target.(*object.Player)
		if ok {
			// 计算伤害
			damage := monster.GetAttack() - (player.GetAgility() / 5)
			if damage < 1 {
				damage = 1
			}
			player.SetHealth(player.GetHealth() - damage)

			zLog.Info("Monster attacked player",
				zap.String("monster", monster.GetName()),
				zap.String("player", player.GetName()),
				zap.Int32("damage", damage),
				zap.Int32("player_health", player.GetHealth()))

			// 检查玩家是否死亡
			if player.GetHealth() <= 0 {
				im.HandlePlayerDeath(player)
			}

			return true
		}
	}

	return false
}

// HandleMonsterDeath 处理怪物死亡
func (im *InteractionManager) HandleMonsterDeath(monster *object.Monster, player *object.Player) {
	// 给玩家添加经验
	player.AddExp(monster.GetExp())

	// 处理掉落
	lootItems := monster.GetLootItems()
	for _, itemID := range lootItems {
		player.AddItem(itemID)
	}

	zLog.Info("Monster died",
		zap.String("monster", monster.GetName()),
		zap.String("killer", player.GetName()),
		zap.Int64("exp_gained", monster.GetExp()),
		zap.Int("items_dropped", len(lootItems)))

	// 从地图中移除怪物
	im.mapInstance.RemoveObject(monster.GetID())
}

// HandlePlayerDeath 处理玩家死亡
func (im *InteractionManager) HandlePlayerDeath(player *object.Player) {
	zLog.Info("Player died", zap.String("player", player.GetName()))

	// 可以添加死亡惩罚逻辑，比如掉落物品、减少经验等
	// 这里简单处理，只是记录日志
}

// HandlePlayerTalk 处理玩家与NPC对话
func (im *InteractionManager) HandlePlayerTalk(player *object.Player, npcID int64) bool {
	// 查找NPC
	npc := im.mapInstance.GetObjectByID(int64(npcID))
	if npc == nil {
		zLog.Warn("NPC not found", zap.Int64("npc_id", npcID))
		return false
	}

	// 检查是否是NPC
	if npc.GetType() == common.GameObjectTypeNPC {
		npcObj, ok := npc.(*object.NPC)
		if ok && npcObj.CanTalk() {
			zLog.Info("Player talked to NPC",
				zap.String("player", player.GetName()),
				zap.String("npc", npcObj.GetName()),
				zap.String("dialogue", npcObj.GetDialogue()))

			// 设置NPC状态为对话中
			npcObj.SetState(object.NPCStateTalking)

			return true
		}
	}

	return false
}

// HandlePlayerTrade 处理玩家与NPC交易
func (im *InteractionManager) HandlePlayerTrade(player *object.Player, npcID int64) bool {
	// 查找NPC
	npc := im.mapInstance.GetObjectByID(int64(npcID))
	if npc == nil {
		zLog.Warn("NPC not found", zap.Int64("npc_id", npcID))
		return false
	}

	// 检查是否是NPC
	if npc.GetType() == common.GameObjectTypeNPC {
		npcObj, ok := npc.(*object.NPC)
		if ok && npcObj.CanTrade() {
			zLog.Info("Player traded with NPC",
				zap.String("player", player.GetName()),
				zap.String("npc", npcObj.GetName()),
				zap.Int("shop_items", len(npcObj.GetShopItems())))

			// 设置NPC状态为交易中
			npcObj.SetState(object.NPCStateTrading)

			return true
		}
	}

	return false
}

// HandlePlayerPickup 处理玩家拾取物品
func (im *InteractionManager) HandlePlayerPickup(player *object.Player, itemID int64) bool {
	// 查找物品
	item := im.mapInstance.GetObjectByID(int64(itemID))
	if item == nil {
		zLog.Warn("Item not found", zap.Int64("item_id", itemID))
		return false
	}

	// 检查是否是物品
	if item.GetType() == common.GameObjectTypeItem {
		itemObj, ok := item.(*object.Item)
		if ok && itemObj.CanBePicked() {
			// 添加物品到玩家背包
			player.AddItem(itemObj.GetItemID())

			zLog.Info("Player picked up item",
				zap.String("player", player.GetName()),
				zap.String("item", itemObj.GetName()),
				zap.Int32("quantity", itemObj.GetQuantity()))

			// 设置物品为已拾取
			itemObj.SetPicked(true)

			// 从地图中移除物品
			im.mapInstance.RemoveObject(itemObj.GetID())

			return true
		}
	}

	return false
}

// HandlePlayerUseItem 处理玩家使用物品
func (im *InteractionManager) HandlePlayerUseItem(player *object.Player, itemID int32) bool {
	// 检查玩家是否拥有该物品
	hasItem := false
	for _, id := range player.GetItems() {
		if id == itemID {
			hasItem = true
			break
		}
	}

	if !hasItem {
		zLog.Warn("Player does not have item", zap.String("player", player.GetName()), zap.Int32("item_id", itemID))
		return false
	}

	// 这里可以根据物品类型执行不同的使用逻辑
	// 简单起见，这里只是记录日志
	zLog.Info("Player used item", zap.String("player", player.GetName()), zap.Int32("item_id", itemID))

	// 从玩家背包中移除物品
	player.RemoveItem(itemID)

	return true
}

// HandleMonsterPatrol 处理怪物巡逻
func (im *InteractionManager) HandleMonsterPatrol(monster *object.Monster) {
	if monster.GetState() == object.MonsterStatePatrolling {
		// 获取下一个巡逻点
		nextPoint := monster.GetNextPatrolPoint()
		// 移动怪物到下一个巡逻点
		monster.SetPosition(nextPoint)

		zLog.Debug("Monster patrolling",
			zap.String("monster", monster.GetName()),
			zap.Float32("x", nextPoint.X),
			zap.Float32("y", nextPoint.Y),
			zap.Float32("z", nextPoint.Z))
	}
}

// HandleMonsterAggro 处理怪物仇恨
func (im *InteractionManager) HandleMonsterAggro(monster *object.Monster, player *object.Player) {
	// 检查玩家是否在怪物的仇恨范围内
	distance := monster.GetPosition().DistanceTo(player.GetPosition())
	if distance <= 10000 { // 100^2
		// 设置怪物状态为追逐
		monster.SetState(object.MonsterStateChasing)
		zLog.Debug("Monster aggroed",
			zap.String("monster", monster.GetName()),
			zap.String("player", player.GetName()),
			zap.Float32("distance", distance))
	}
}
