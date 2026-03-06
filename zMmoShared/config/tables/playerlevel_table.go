package tables

import (
	"github.com/pzqf/zMmoShared/config/models"
)

// PlayerLevelTableLoader 人物等级表加载器
type PlayerLevelTableLoader struct {
	playerLevels map[int32]*models.PlayerLevel // 等级配置映射
}

// NewPlayerLevelTableLoader 创建人物等级表加载器
func NewPlayerLevelTableLoader() *PlayerLevelTableLoader {
	return &PlayerLevelTableLoader{
		playerLevels: make(map[int32]*models.PlayerLevel),
	}
}

// Load 加载人物等级表数据
func (plt *PlayerLevelTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "player_level.xlsx",
		SheetName:  "level",
		MinColumns: 8,
		TableName:  "playerLevels",
	}

	tempPlayerLevels := make(map[int32]*models.PlayerLevel)

	err := ReadExcelFile(config, dir, func(row []string) error {
		level := &models.PlayerLevel{
			LevelID:      StrToInt32(row[0]),
			RequiredExp:  StrToInt64(row[1]),
			HP:           StrToInt32(row[2]),
			MP:           StrToInt32(row[3]),
			Attack:       StrToInt32(row[4]),
			Defense:      StrToInt32(row[5]),
			CriticalRate: StrToFloat32(row[6]),
			SkillPoints:  StrToInt32(row[7]),
		}

		tempPlayerLevels[level.LevelID] = level
		return nil
	})

	if err == nil {
		plt.playerLevels = tempPlayerLevels
	}

	return err
}

// GetTableName 获取表格名称
func (plt *PlayerLevelTableLoader) GetTableName() string {
	return "playerLevels"
}

// GetPlayerLevel 根据ID获取等级配置
func (plt *PlayerLevelTableLoader) GetPlayerLevel(levelID int32) (*models.PlayerLevel, bool) {
	level, ok := plt.playerLevels[levelID]
	return level, ok
}

// GetAllPlayerLevels 获取所有等级配置
func (plt *PlayerLevelTableLoader) GetAllPlayerLevels() map[int32]*models.PlayerLevel {
	playerLevelsCopy := make(map[int32]*models.PlayerLevel, len(plt.playerLevels))
	for id, level := range plt.playerLevels {
		playerLevelsCopy[id] = level
	}
	return playerLevelsCopy
}
