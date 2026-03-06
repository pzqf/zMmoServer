package tables

import (
	"strconv"

	"github.com/pzqf/zMmoShared/config/models"
)

// StrToBool 字符串转布尔值
func StrToBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return b
}

// BuffTableLoader buff表加载器
type BuffTableLoader struct {
	buffs map[int32]*models.Buff // buff配置映射
}

// NewBuffTableLoader 创建buff表加载器
func NewBuffTableLoader() *BuffTableLoader {
	return &BuffTableLoader{
		buffs: make(map[int32]*models.Buff),
	}
}

// Load 加载buff表数据
func (btl *BuffTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "buff.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 9,
		TableName:  "buffs",
	}

	tempBuffs := make(map[int32]*models.Buff)

	err := ReadExcelFile(config, dir, func(row []string) error {
		buff := &models.Buff{
			BuffID:      StrToInt32(row[0]),
			Name:        row[1],
			Description: row[2],
			Type:        row[3],
			Duration:    StrToInt32(row[4]),
			Value:       StrToInt32(row[5]),
			Property:    row[6],
			IsPermanent: StrToBool(row[7]),
		}

		tempBuffs[buff.BuffID] = buff
		return nil
	})

	if err == nil {
		btl.buffs = tempBuffs
	}

	return err
}

// GetTableName 获取表格名称
func (btl *BuffTableLoader) GetTableName() string {
	return "buffs"
}

// GetBuff 根据ID获取buff
func (btl *BuffTableLoader) GetBuff(buffID int32) (*models.Buff, bool) {
	buff, ok := btl.buffs[buffID]
	return buff, ok
}

// GetAllBuffs 获取所有buff
func (btl *BuffTableLoader) GetAllBuffs() map[int32]*models.Buff {
	buffsCopy := make(map[int32]*models.Buff, len(btl.buffs))
	for id, buff := range btl.buffs {
		buffsCopy[id] = buff
	}
	return buffsCopy
}
