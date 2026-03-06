package tables

import (
	"github.com/pzqf/zMmoShared/config/models"
)

// GetItemByID 根据ID获取物品配置
func GetItemByID(itemID int32) *models.ItemBase {
	if GlobalTableManager == nil {
		return nil
	}

	item, ok := GlobalTableManager.GetItemLoader().GetItem(itemID)
	if !ok {
		return nil
	}
	return item
}

// GetAllItems 获取所有物品配置
func GetAllItems() map[int32]*models.ItemBase {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetItemLoader().GetAllItems()
}

// GetMapByID 根据ID获取地图配置
func GetMapByID(mapID int32) *models.Map {
	if GlobalTableManager == nil {
		return nil
	}

	mapData, ok := GlobalTableManager.GetMapLoader().GetMap(mapID)
	if !ok {
		return nil
	}
	return mapData
}

// GetAllMaps 获取所有地图配置
func GetAllMaps() map[int32]*models.Map {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetMapLoader().GetAllMaps()
}

// GetSkillByID 根据ID获取技能配置
func GetSkillByID(skillID int32) *models.Skill {
	if GlobalTableManager == nil {
		return nil
	}

	skill, ok := GlobalTableManager.GetSkillLoader().GetSkill(skillID)
	if !ok {
		return nil
	}
	return skill
}

// GetAllSkills 获取所有技能配置
func GetAllSkills() map[int32]*models.Skill {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetSkillLoader().GetAllSkills()
}

// GetQuestByID 根据ID获取任务配置
func GetQuestByID(questID int32) *models.Quest {
	if GlobalTableManager == nil {
		return nil
	}

	quest, ok := GlobalTableManager.GetQuestLoader().GetQuest(questID)
	if !ok {
		return nil
	}
	return quest
}

// GetAllQuests 获取所有任务配置
func GetAllQuests() map[int32]*models.Quest {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetQuestLoader().GetAllQuests()
}

// GetPlayerLevelByID 根据ID获取等级配置
func GetPlayerLevelByID(levelID int32) *models.PlayerLevel {
	if GlobalTableManager == nil {
		return nil
	}

	level, ok := GlobalTableManager.GetPlayerLevelLoader().GetPlayerLevel(levelID)
	if !ok {
		return nil
	}
	return level
}

// GetAllPlayerLevels 获取所有等级配置
func GetAllPlayerLevels() map[int32]*models.PlayerLevel {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetPlayerLevelLoader().GetAllPlayerLevels()
}

// GetMonsterByID 根据ID获取怪物配置
func GetMonsterByID(monsterID int32) *models.Monster {
	if GlobalTableManager == nil {
		return nil
	}

	monster, ok := GlobalTableManager.GetMonsterLoader().GetMonster(monsterID)
	if !ok {
		return nil
	}
	return monster
}

// GetAllMonsters 获取所有怪物配置
func GetAllMonsters() map[int32]*models.Monster {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetMonsterLoader().GetAllMonsters()
}

// GetShopByID 根据ID获取商店配置
func GetShopByID(shopID int32) *models.Shop {
	if GlobalTableManager == nil {
		return nil
	}

	shop, ok := GlobalTableManager.GetShopLoader().GetShop(shopID)
	if !ok {
		return nil
	}
	return shop
}

// GetAllShops 获取所有商店配置
func GetAllShops() map[int32]*models.Shop {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetShopLoader().GetAllShops()
}

// GetGuildByLevel 根据等级获取公会配置
func GetGuildByLevel(guildLevel int32) *models.Guild {
	if GlobalTableManager == nil {
		return nil
	}

	guild, ok := GlobalTableManager.GetGuildLoader().GetGuild(guildLevel)
	if !ok {
		return nil
	}
	return guild
}

// GetAllGuilds 获取所有公会配置
func GetAllGuilds() map[int32]*models.Guild {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetGuildLoader().GetAllGuilds()
}

// GetPetByID 根据ID获取宠物配置
func GetPetByID(petID int32) *models.Pet {
	if GlobalTableManager == nil {
		return nil
	}

	pet, ok := GlobalTableManager.GetPetLoader().GetPet(petID)
	if !ok {
		return nil
	}
	return pet
}

// GetAllPets 获取所有宠物配置
func GetAllPets() map[int32]*models.Pet {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetPetLoader().GetAllPets()
}

// GetBuffByID 根据ID获取buff配置
func GetBuffByID(buffID int32) *models.Buff {
	if GlobalTableManager == nil {
		return nil
	}

	buff, ok := GlobalTableManager.GetBuffLoader().GetBuff(buffID)
	if !ok {
		return nil
	}
	return buff
}

// GetAllBuffs 获取所有buff配置
func GetAllBuffs() map[int32]*models.Buff {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetBuffLoader().GetAllBuffs()
}

// GetAIByID 根据ID获取AI配置
func GetAIByID(aiID int32) *models.AI {
	if GlobalTableManager == nil {
		return nil
	}

	ai, ok := GlobalTableManager.GetAILoader().GetAI(aiID)
	if !ok {
		return nil
	}
	return ai
}

// GetAllAIs 获取所有AI配置
func GetAllAIs() map[int32]*models.AI {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetAILoader().GetAllAIs()
}
