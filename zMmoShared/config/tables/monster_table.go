package tables

import (
	"github.com/pzqf/zMmoShared/config/models"
)

// MonsterTableLoader 怪物表加载器
// 负责从Excel加载怪物配置数据
type MonsterTableLoader struct {
	monsters map[int32]*models.Monster // 怪物配置映射（怪物ID -> 配置）
}

// NewMonsterTableLoader 创建怪物表加载器
// 返回: 初始化后的怪物表加载器实例
func NewMonsterTableLoader() *MonsterTableLoader {
	return &MonsterTableLoader{
		monsters: make(map[int32]*models.Monster),
	}
}

// Load 加载怪物表数据
// 从monster.xlsx文件读取怪物配置
// 参数:
//   - dir: Excel文件所在目录
//
// 返回: 加载错误
func (mtl *MonsterTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "monster.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 11,
		TableName:  "monsters",
	}

	// 使用临时map批量加载数据
	tempMonsters := make(map[int32]*models.Monster)

	err := ReadExcelFile(config, dir, func(row []string) error {
		monster := &models.Monster{
			MonsterID:    StrToInt32(row[0]),
			Name:         row[1],
			Level:        StrToInt32(row[2]),
			HP:           StrToInt32(row[3]),
			MP:           StrToInt32(row[4]),
			Attack:       StrToInt32(row[5]),
			Defense:      StrToInt32(row[6]),
			Speed:        StrToInt32(row[7]),
			Exp:          StrToInt32(row[8]),
			DropItemRate: StrToFloat32(row[9]),
			DropItems:    row[10],
		}

		tempMonsters[monster.MonsterID] = monster
		return nil
	})

	// 加载完成后一次性赋值
	if err == nil {
		mtl.monsters = tempMonsters
	}

	return err
}

// GetTableName 获取表格名称
// 返回: 表格名称"monsters"
func (mtl *MonsterTableLoader) GetTableName() string {
	return "monsters"
}

// GetMonster 根据ID获取怪物配置
// 参数:
//   - monsterID: 怪物ID
//
// 返回: 怪物配置和是否存在
func (mtl *MonsterTableLoader) GetMonster(monsterID int32) (*models.Monster, bool) {
	monster, ok := mtl.monsters[monsterID]
	return monster, ok
}

// GetAllMonsters 获取所有怪物配置
// 返回配置的副本map，避免外部修改内部数据
// 返回: 怪物配置映射副本
func (mtl *MonsterTableLoader) GetAllMonsters() map[int32]*models.Monster {
	// 创建一个副本，避免外部修改内部数据
	monstersCopy := make(map[int32]*models.Monster, len(mtl.monsters))
	for id, monster := range mtl.monsters {
		monstersCopy[id] = monster
	}
	return monstersCopy
}
