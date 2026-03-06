package tables

import (
	"github.com/pzqf/zMmoShared/config/models"
)

// QuestTableLoader 任务表格加载器
type QuestTableLoader struct {
	quests map[int32]*models.Quest // 任务配置映射
}

// NewQuestTableLoader 创建任务表格加载器
func NewQuestTableLoader() *QuestTableLoader {
	return &QuestTableLoader{
		quests: make(map[int32]*models.Quest),
	}
}

// Load 加载任务表数据
func (qtl *QuestTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "quest.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 9,
		TableName:  "quests",
	}

	tempQuests := make(map[int32]*models.Quest)

	err := ReadExcelFile(config, dir, func(row []string) error {
		quest := &models.Quest{
			QuestID:     StrToInt32(row[0]),
			Name:        row[1],
			Type:        StrToInt32(row[2]),
			Level:       StrToInt32(row[3]),
			Description: row[4],
			Objectives:  row[5],
			Rewards:     row[6],
			NextQuestID: StrToInt32(row[7]),
			PreQuestID:  StrToInt32(row[8]),
		}

		tempQuests[quest.QuestID] = quest
		return nil
	})

	if err == nil {
		qtl.quests = tempQuests
	}

	return err
}

// GetTableName 获取表格名称
func (qtl *QuestTableLoader) GetTableName() string {
	return "quests"
}

// GetQuest 根据ID获取任务
func (qtl *QuestTableLoader) GetQuest(questID int32) (*models.Quest, bool) {
	quest, ok := qtl.quests[questID]
	return quest, ok
}

// GetAllQuests 获取所有任务
func (qtl *QuestTableLoader) GetAllQuests() map[int32]*models.Quest {
	questsCopy := make(map[int32]*models.Quest, len(qtl.quests))
	for id, quest := range qtl.quests {
		questsCopy[id] = quest
	}
	return questsCopy
}
