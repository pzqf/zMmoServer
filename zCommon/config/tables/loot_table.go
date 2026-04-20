package tables

import (
	"encoding/json"

	"github.com/pzqf/zCommon/config/models"
)

type LootTableLoader struct {
	lootGroups map[int32]*models.LootGroup
}

func NewLootTableLoader() *LootTableLoader {
	return &LootTableLoader{
		lootGroups: make(map[int32]*models.LootGroup),
	}
}

func (ltl *LootTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "loot_group.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 8,
		TableName:  "loot_groups",
	}

	tempGroups := make(map[int32]*models.LootGroup)

	err := ReadExcelFile(config, dir, func(row []string) error {
		group := &models.LootGroup{
			LootGroupID:  StrToInt32(row[0]),
			Name:         row[1],
			Items:        row[2],
			DropRate:     StrToFloat32(row[3]),
			MinLevel:     StrToInt32(row[4]),
			MaxLevel:     StrToInt32(row[5]),
			Difficulty:   row[6],
			MaxDropCount: StrToInt32(row[7]),
		}

		tempGroups[group.LootGroupID] = group
		return nil
	})

	if err == nil {
		ltl.lootGroups = tempGroups
	}

	return err
}

func (ltl *LootTableLoader) GetTableName() string {
	return "loot_groups"
}

func (ltl *LootTableLoader) GetLootGroup(groupID int32) (*models.LootGroup, bool) {
	group, ok := ltl.lootGroups[groupID]
	return group, ok
}

func (ltl *LootTableLoader) GetAllLootGroups() map[int32]*models.LootGroup {
	result := make(map[int32]*models.LootGroup, len(ltl.lootGroups))
	for id, group := range ltl.lootGroups {
		result[id] = group
	}
	return result
}

func (ltl *LootTableLoader) ParseLootItems(itemsJSON string) []models.LootItem {
	var items []models.LootItem
	if itemsJSON == "" {
		return items
	}
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return items
	}
	return items
}
