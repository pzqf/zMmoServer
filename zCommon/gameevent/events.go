package gameevent

import "fmt"

const (
	EventPlayerLogin EventID = iota + 1000
	EventPlayerLogout
	EventPlayerLevelUp
	EventPlayerDie
	EventPlayerRespawn
	EventPlayerMove
	EventPlayerEnterMap
	EventPlayerLeaveMap
	EventPlayerChat
	EventPlayerTrade
	EventPlayerJoinTeam
	EventPlayerLeaveTeam
	EventPlayerJoinGuild
	EventPlayerLeaveGuild

	EventMonsterSpawn EventID = iota + 2000
	EventMonsterDie
	EventMonsterAggro
	EventMonsterDeaggro
	EventMonsterRespawn

	EventItemPickup EventID = iota + 3000
	EventItemDrop
	EventItemUse
	EventItemEquip
	EventItemUnequip

	EventSkillCast EventID = iota + 4000
	EventSkillHit
	EventSkillMiss
	EventSkillCooldown
	EventBuffAdd
	EventBuffRemove
	EventBuffExpire
	EventDebuffAdd
	EventDebuffRemove

	EventMapEnter EventID = iota + 5000
	EventMapLeave
	EventMapCreate
	EventMapDestroy
	EventObjectEnterAOI
	EventObjectLeaveAOI

	EventDungeonEnter EventID = iota + 6000
	EventDungeonLeave
	EventDungeonComplete
	EventDungeonFail

	EventServerStart EventID = iota + 9000
	EventServerStop
	EventServerHealthCheck
	EventServiceRegister
	EventServiceUnregister
)

var eventNames = map[EventID]string{
	EventPlayerLogin:       "PlayerLogin",
	EventPlayerLogout:      "PlayerLogout",
	EventPlayerLevelUp:     "PlayerLevelUp",
	EventPlayerDie:         "PlayerDie",
	EventPlayerRespawn:     "PlayerRespawn",
	EventPlayerMove:        "PlayerMove",
	EventPlayerEnterMap:    "PlayerEnterMap",
	EventPlayerLeaveMap:    "PlayerLeaveMap",
	EventPlayerChat:        "PlayerChat",
	EventPlayerTrade:       "PlayerTrade",
	EventPlayerJoinTeam:    "PlayerJoinTeam",
	EventPlayerLeaveTeam:   "PlayerLeaveTeam",
	EventPlayerJoinGuild:   "PlayerJoinGuild",
	EventPlayerLeaveGuild:  "PlayerLeaveGuild",
	EventMonsterSpawn:      "MonsterSpawn",
	EventMonsterDie:        "MonsterDie",
	EventMonsterAggro:      "MonsterAggro",
	EventMonsterDeaggro:    "MonsterDeaggro",
	EventMonsterRespawn:    "MonsterRespawn",
	EventItemPickup:        "ItemPickup",
	EventItemDrop:          "ItemDrop",
	EventItemUse:           "ItemUse",
	EventItemEquip:         "ItemEquip",
	EventItemUnequip:       "ItemUnequip",
	EventSkillCast:         "SkillCast",
	EventSkillHit:          "SkillHit",
	EventSkillMiss:         "SkillMiss",
	EventSkillCooldown:     "SkillCooldown",
	EventBuffAdd:           "BuffAdd",
	EventBuffRemove:        "BuffRemove",
	EventBuffExpire:        "BuffExpire",
	EventDebuffAdd:         "DebuffAdd",
	EventDebuffRemove:      "DebuffRemove",
	EventMapEnter:          "MapEnter",
	EventMapLeave:          "MapLeave",
	EventMapCreate:         "MapCreate",
	EventMapDestroy:        "MapDestroy",
	EventObjectEnterAOI:    "ObjectEnterAOI",
	EventObjectLeaveAOI:    "ObjectLeaveAOI",
	EventDungeonEnter:      "DungeonEnter",
	EventDungeonLeave:      "DungeonLeave",
	EventDungeonComplete:   "DungeonComplete",
	EventDungeonFail:       "DungeonFail",
	EventServerStart:       "ServerStart",
	EventServerStop:        "ServerStop",
	EventServerHealthCheck: "ServerHealthCheck",
	EventServiceRegister:   "ServiceRegister",
	EventServiceUnregister: "ServiceUnregister",
}

func EventName(id EventID) string {
	if name, ok := eventNames[id]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", id)
}
