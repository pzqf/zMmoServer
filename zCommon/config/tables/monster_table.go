package tables

import (
	"github.com/pzqf/zCommon/config/models"
)

// MonsterTableLoader 怪物表加载器
// 从Excel表加载怪物配置数据
type MonsterTableLoader struct {
	monsters map[int32]*models.Monster // 怪物配置映射（怪物ID -> 配置）
}

// NewMonsterTableLoader 创建怪物表加载器
// 功能: 初始化怪物表加载器实例
func NewMonsterTableLoader() *MonsterTableLoader {
	return &MonsterTableLoader{
		monsters: make(map[int32]*models.Monster),
	}
}

// Load ���ع��������
// ��monster.xlsx�ļ���ȡ��������
// ����:
//   - dir: Excel�ļ�����Ŀ¼
//
// ����: ���ش���
func (mtl *MonsterTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "monster.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 11,
		TableName:  "monsters",
	}

	// ʹ����ʱmap������������
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

		if len(row) > 11 {
			monster.LootGroupID = StrToInt32(row[11])
		}
		if len(row) > 12 {
			monster.RespawnTime = StrToInt32(row[12])
		}
		if len(row) > 13 {
			monster.AIType = row[13]
		}
		if len(row) > 14 {
			monster.Difficulty = row[14]
		}

		tempMonsters[monster.MonsterID] = monster
		return nil
	})

	// ������ɺ�һ���Ը�ֵ
	if err == nil {
		mtl.monsters = tempMonsters
	}

	return err
}

// GetTableName ��ȡ��������
// ����: ��������"monsters"
func (mtl *MonsterTableLoader) GetTableName() string {
	return "monsters"
}

// GetMonster ����ID��ȡ��������
// ����:
//   - monsterID: ����ID
//
// ����: �������ú��Ƿ����
func (mtl *MonsterTableLoader) GetMonster(monsterID int32) (*models.Monster, bool) {
	monster, ok := mtl.monsters[monsterID]
	return monster, ok
}

// GetAllMonsters ��ȡ���й�������
// �������õĸ���map�������ⲿ�޸��ڲ�����
// ����: ��������ӳ�丱��
func (mtl *MonsterTableLoader) GetAllMonsters() map[int32]*models.Monster {
	// ����һ�������������ⲿ�޸��ڲ�����
	monstersCopy := make(map[int32]*models.Monster, len(mtl.monsters))
	for id, monster := range mtl.monsters {
		monstersCopy[id] = monster
	}
	return monstersCopy
}

