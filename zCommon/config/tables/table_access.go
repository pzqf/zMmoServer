package tables

import (
	"github.com/pzqf/zCommon/config/models"
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

// GetMapByID ����ID��ȡ��ͼ����
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

// GetAllMaps ��ȡ���е�ͼ����
func GetAllMaps() map[int32]*models.Map {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetMapLoader().GetAllMaps()
}

// GetSkillByID ����ID��ȡ��������
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

// GetAllSkills ��ȡ���м�������
func GetAllSkills() map[int32]*models.Skill {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetSkillLoader().GetAllSkills()
}

// GetQuestByID ����ID��ȡ��������
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

// GetAllQuests ��ȡ������������
func GetAllQuests() map[int32]*models.Quest {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetQuestLoader().GetAllQuests()
}

// GetPlayerLevelByID ����ID��ȡ�ȼ�����
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

// GetAllPlayerLevels ��ȡ���еȼ�����
func GetAllPlayerLevels() map[int32]*models.PlayerLevel {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetPlayerLevelLoader().GetAllPlayerLevels()
}

// GetMonsterByID ����ID��ȡ��������
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

// GetAllMonsters ��ȡ���й�������
func GetAllMonsters() map[int32]*models.Monster {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetMonsterLoader().GetAllMonsters()
}



// GetBuffByID ����ID��ȡbuff����
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

// GetAllBuffs ��ȡ����buff����
func GetAllBuffs() map[int32]*models.Buff {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetBuffLoader().GetAllBuffs()
}

// GetAIByID ����ID��ȡAI����
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

// GetAllAIs ��ȡ����AI����
func GetAllAIs() map[int32]*models.AI {
	if GlobalTableManager == nil {
		return nil
	}

	return GlobalTableManager.GetAILoader().GetAllAIs()
}

