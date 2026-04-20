package tables

import (
	"strconv"
	"strings"

	"github.com/pzqf/zCommon/config/models"
)

type DungeonTableLoader struct {
	dungeons map[int32]*models.Dungeon
	waves    map[int32][]*models.DungeonWave
}

func NewDungeonTableLoader() *DungeonTableLoader {
	return &DungeonTableLoader{
		dungeons: make(map[int32]*models.Dungeon),
		waves:    make(map[int32][]*models.DungeonWave),
	}
}

func (dtl *DungeonTableLoader) Load(dir string) error {
	if err := dtl.loadDungeons(dir); err != nil {
		return err
	}
	if err := dtl.loadWaves(dir); err != nil {
		return err
	}
	return nil
}

func (dtl *DungeonTableLoader) loadDungeons(dir string) error {
	config := ExcelConfig{
		FileName:   "dungeon.xlsx",
		SheetName:  "Dungeons",
		MinColumns: 18,
		TableName:  "dungeons",
	}

	tempDungeons := make(map[int32]*models.Dungeon)

	err := ReadExcelFile(config, dir, func(row []string) error {
		d := &models.Dungeon{
			DungeonID:       StrToInt32(row[0]),
			Name:            row[1],
			Description:     row[2],
			Type:            StrToInt32(row[3]),
			Difficulty:      StrToInt32(row[4]),
			MinLevel:        StrToInt32(row[5]),
			MaxLevel:        StrToInt32(row[6]),
			MinPlayers:      StrToInt32(row[7]),
			MaxPlayers:      StrToInt32(row[8]),
			TimeLimit:       StrToInt32(row[9]),
			DailyLimit:      StrToInt32(row[10]),
			MapID:           StrToInt32(row[11]),
			RewardExp:       StrToInt64(row[12]),
			RewardGold:      StrToInt64(row[13]),
			RewardItemIDs:   row[14],
			EntryCostType:   StrToInt32(row[15]),
			EntryCostAmount: StrToInt32(row[16]),
			WaveCount:       StrToInt32(row[17]),
			BossID:          StrToInt32(safeGet(row, 18)),
			IsOpen:          StrToBool(safeGet(row, 19)),
		}

		tempDungeons[d.DungeonID] = d
		return nil
	})

	if err == nil {
		dtl.dungeons = tempDungeons
	}

	return err
}

func (dtl *DungeonTableLoader) loadWaves(dir string) error {
	config := ExcelConfig{
		FileName:   "dungeon.xlsx",
		SheetName:  "Waves",
		MinColumns: 7,
		TableName:  "dungeon_waves",
	}

	tempWaves := make(map[int32][]*models.DungeonWave)

	err := ReadExcelFile(config, dir, func(row []string) error {
		w := &models.DungeonWave{
			WaveID:       StrToInt32(row[0]),
			DungeonID:    StrToInt32(row[1]),
			WaveIndex:    StrToInt32(row[2]),
			MonsterIDs:   row[3],
			MonsterCount: StrToInt32(row[4]),
			SpawnDelay:   StrToInt32(row[5]),
			IsBoss:       StrToBool(row[6]),
		}

		tempWaves[w.DungeonID] = append(tempWaves[w.DungeonID], w)
		return nil
	})

	if err == nil {
		dtl.waves = tempWaves
	}

	return err
}

func (dtl *DungeonTableLoader) GetDungeon(dungeonID int32) (*models.Dungeon, bool) {
	d, ok := dtl.dungeons[dungeonID]
	return d, ok
}

func (dtl *DungeonTableLoader) GetAllDungeons() map[int32]*models.Dungeon {
	return dtl.dungeons
}

func (dtl *DungeonTableLoader) GetWavesByDungeonID(dungeonID int32) []*models.DungeonWave {
	return dtl.waves[dungeonID]
}

func (dtl *DungeonTableLoader) GetDungeonsByType(dungeonType int32) []*models.Dungeon {
	var result []*models.Dungeon
	for _, d := range dtl.dungeons {
		if d.Type == dungeonType {
			result = append(result, d)
		}
	}
	return result
}

func (dtl *DungeonTableLoader) ParseMonsterIDs(monsterIDsStr string) []int32 {
	if monsterIDsStr == "" {
		return nil
	}
	parts := strings.Split(monsterIDsStr, ",")
	ids := make([]int32, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 32)
		if err == nil {
			ids = append(ids, int32(id))
		}
	}
	return ids
}

func (dtl *DungeonTableLoader) AddDungeonForTest(d *models.Dungeon) {
	dtl.dungeons[d.DungeonID] = d
}

func (dtl *DungeonTableLoader) AddWaveForTest(w *models.DungeonWave) {
	dtl.waves[w.DungeonID] = append(dtl.waves[w.DungeonID], w)
}

func (dtl *DungeonTableLoader) GetTableName() string {
	return "dungeon"
}

func safeGet(row []string, index int) string {
	if index < len(row) {
		return row[index]
	}
	return ""
}
