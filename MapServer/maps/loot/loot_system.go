package loot

import (
	"math/rand"

	"github.com/pzqf/zCommon/config/models"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

type LootSystem struct {
	tableManager *tables.TableManager
}

func NewLootSystem(tm *tables.TableManager) *LootSystem {
	return &LootSystem{
		tableManager: tm,
	}
}

type LootResult struct {
	ItemID int32
	Count  int32
}

func (ls *LootSystem) GenerateLoot(monsterConfigID int32, monsterLevel int32, monsterDifficulty string) []LootResult {
	if ls.tableManager == nil {
		return nil
	}

	monsterConfig, ok := ls.tableManager.GetMonsterLoader().GetMonster(monsterConfigID)
	if !ok {
		zLog.Debug("Monster config not found for loot generation",
			zap.Int32("monster_config_id", monsterConfigID))
		return nil
	}

	results := make([]LootResult, 0)

	if monsterConfig.LootGroupID > 0 {
		results = append(results, ls.generateFromLootGroup(monsterConfig.LootGroupID, monsterLevel, monsterDifficulty)...)
	}

	if monsterConfig.DropItems != "" {
		items := ls.tableManager.GetLootLoader().ParseLootItems(monsterConfig.DropItems)
		for _, item := range items {
			if rand.Float32() <= item.DropRate {
				count := item.CountMin
				if item.CountMax > item.CountMin {
					count = item.CountMin + rand.Int31n(item.CountMax-item.CountMin+1)
				}
				results = append(results, LootResult{
					ItemID: item.ItemID,
					Count:  count,
				})
			}
		}
	}

	if monsterConfig.DropItemRate > 0 && len(results) == 0 {
		if rand.Float32() <= monsterConfig.DropItemRate {
			results = append(results, LootResult{
				ItemID: 1001,
				Count:  1,
			})
		}
	}

	return results
}

func (ls *LootSystem) generateFromLootGroup(groupID int32, monsterLevel int32, monsterDifficulty string) []LootResult {
	group, ok := ls.tableManager.GetLootLoader().GetLootGroup(groupID)
	if !ok {
		return nil
	}

	if group.MinLevel > 0 && monsterLevel < group.MinLevel {
		return nil
	}
	if group.MaxLevel > 0 && monsterLevel > group.MaxLevel {
		return nil
	}
	if group.Difficulty != "" && group.Difficulty != "any" && group.Difficulty != monsterDifficulty {
		return nil
	}

	items := ls.tableManager.GetLootLoader().ParseLootItems(group.Items)
	if len(items) == 0 {
		return nil
	}

	results := make([]LootResult, 0)
	dropped := int32(0)

	for _, item := range items {
		if group.MaxDropCount > 0 && dropped >= group.MaxDropCount {
			break
		}

		effectiveRate := item.DropRate * group.DropRate
		if rand.Float32() <= effectiveRate {
			count := item.CountMin
			if item.CountMax > item.CountMin {
				count = item.CountMin + rand.Int31n(item.CountMax-item.CountMin+1)
			}
			results = append(results, LootResult{
				ItemID: item.ItemID,
				Count:  count,
			})
			dropped++
		}
	}

	return results
}

func (ls *LootSystem) GetRespawnTime(monsterConfigID int32) int32 {
	if ls.tableManager == nil {
		return 30
	}

	monsterConfig, ok := ls.tableManager.GetMonsterLoader().GetMonster(monsterConfigID)
	if !ok {
		return 30
	}

	if monsterConfig.RespawnTime > 0 {
		return monsterConfig.RespawnTime
	}

	return 30
}

func (ls *LootSystem) GetMonsterConfig(monsterConfigID int32) *models.Monster {
	if ls.tableManager == nil {
		return nil
	}
	config, _ := ls.tableManager.GetMonsterLoader().GetMonster(monsterConfigID)
	return config
}
