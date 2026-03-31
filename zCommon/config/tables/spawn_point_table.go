package tables

import (
	"github.com/pzqf/zCommon/config/models"
)

type SpawnPointTableLoader struct {
	spawnPoints map[int32]*models.SpawnPoint
	byMapID     map[int32][]*models.SpawnPoint
}

func NewSpawnPointTableLoader() *SpawnPointTableLoader {
	return &SpawnPointTableLoader{
		spawnPoints: make(map[int32]*models.SpawnPoint),
		byMapID:     make(map[int32][]*models.SpawnPoint),
	}
}

func (sptl *SpawnPointTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "spawn_point.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 11,
		TableName:  "spawn_points",
	}

	tempSpawnPoints := make(map[int32]*models.SpawnPoint)
	tempByMapID := make(map[int32][]*models.SpawnPoint)

	err := ReadExcelFile(config, dir, func(row []string) error {
		spawnPoint := &models.SpawnPoint{
			SpawnID:       StrToInt32(row[0]),
			MapID:         StrToInt32(row[1]),
			MonsterID:     StrToInt32(row[2]),
			SpawnType:     models.SpawnPointType(StrToInt32(row[3])),
			PosX:          StrToFloat32(row[4]),
			PosY:          StrToFloat32(row[5]),
			PosZ:          StrToFloat32(row[6]),
			MaxCount:      StrToInt32(row[7]),
			SpawnInterval: StrToInt32(row[8]),
			Radius:        StrToFloat32(row[9]),
			PatrolRange:   StrToFloat32(row[10]),
		}

		tempSpawnPoints[spawnPoint.SpawnID] = spawnPoint
		tempByMapID[spawnPoint.MapID] = append(tempByMapID[spawnPoint.MapID], spawnPoint)
		return nil
	})

	if err == nil {
		sptl.spawnPoints = tempSpawnPoints
		sptl.byMapID = tempByMapID
	}

	return err
}

func (sptl *SpawnPointTableLoader) GetTableName() string {
	return "spawn_points"
}

func (sptl *SpawnPointTableLoader) GetSpawnPoint(spawnID int32) (*models.SpawnPoint, bool) {
	spawnPoint, ok := sptl.spawnPoints[spawnID]
	return spawnPoint, ok
}

func (sptl *SpawnPointTableLoader) GetSpawnPointsByMap(mapID int32) []*models.SpawnPoint {
	return sptl.byMapID[mapID]
}

func (sptl *SpawnPointTableLoader) GetAllSpawnPoints() map[int32]*models.SpawnPoint {
	spawnPointsCopy := make(map[int32]*models.SpawnPoint, len(sptl.spawnPoints))
	for id, spawnPoint := range sptl.spawnPoints {
		spawnPointsCopy[id] = spawnPoint
	}
	return spawnPointsCopy
}

