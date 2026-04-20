package tables

import (
	"github.com/pzqf/zCommon/config/models"
)

// AITableLoader AI表加载器
type AITableLoader struct {
	ais map[int32]*models.AI // AI配置映射
}

// NewAITableLoader 创建AI表加载器
func NewAITableLoader() *AITableLoader {
	return &AITableLoader{
		ais: make(map[int32]*models.AI),
	}
}

// Load 加载AI配置表
func (atl *AITableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "ai.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 9,
		TableName:  "ais",
	}

	tempAIs := make(map[int32]*models.AI)

	err := ReadExcelFile(config, dir, func(row []string) error {
		ai := &models.AI{
			AIID:           StrToInt32(row[0]),
			Type:           row[1],
			DetectionRange: StrToFloat32(row[2]),
			AttackRange:    StrToFloat32(row[3]),
			ChaseRange:     StrToFloat32(row[4]),
			FleeHealth:     StrToFloat32(row[5]),
			PatrolPoints:   row[6],
			Behavior:       row[7],
			SkillIDs:       row[8],
		}

		tempAIs[ai.AIID] = ai
		return nil
	})

	if err == nil {
		atl.ais = tempAIs
	}

	return err
}

// GetTableName ��ȡ��������
func (atl *AITableLoader) GetTableName() string {
	return "ais"
}

// GetAI ����ID��ȡAI
func (atl *AITableLoader) GetAI(aiID int32) (*models.AI, bool) {
	ai, ok := atl.ais[aiID]
	return ai, ok
}

// GetAllAIs ��ȡ����AI
func (atl *AITableLoader) GetAllAIs() map[int32]*models.AI {
	aisCopy := make(map[int32]*models.AI, len(atl.ais))
	for id, ai := range atl.ais {
		aisCopy[id] = ai
	}
	return aisCopy
}

