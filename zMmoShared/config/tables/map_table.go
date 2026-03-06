package tables

import (
	"github.com/pzqf/zMmoShared/config/models"
)

// MapTableLoader 地图表加载器
type MapTableLoader struct {
	maps           map[int32]*models.Map           // 地图配置映射
	spawnPoints    map[int32]*models.MapSpawnPoint    // 生成点配置映射
	teleportPoints map[int32]*models.MapTeleportPoint // 传送点配置映射
	buildings      map[int32]*models.MapBuilding      // 建筑物配置映射
	events         map[int32]*models.MapEvent         // 事件配置映射
	resources      map[int32]*models.MapResource      // 资源配置映射
}

// NewMapTableLoader 创建地图表加载器
func NewMapTableLoader() *MapTableLoader {
	return &MapTableLoader{
		maps:           make(map[int32]*models.Map),
		spawnPoints:    make(map[int32]*models.MapSpawnPoint),
		teleportPoints: make(map[int32]*models.MapTeleportPoint),
		buildings:      make(map[int32]*models.MapBuilding),
		events:         make(map[int32]*models.MapEvent),
		resources:      make(map[int32]*models.MapResource),
	}
}

// Load 加载地图表数据
func (mtl *MapTableLoader) Load(dir string) error {
	if err := mtl.loadMaps(dir); err != nil {
		return err
	}

	if err := mtl.loadSpawnPoints(dir); err != nil {
		return err
	}

	if err := mtl.loadTeleportPoints(dir); err != nil {
		return err
	}

	if err := mtl.loadBuildings(dir); err != nil {
		return err
	}

	if err := mtl.loadEvents(dir); err != nil {
		return err
	}

	if err := mtl.loadResources(dir); err != nil {
		return err
	}

	return nil
}

// loadMaps 加载地图基本信息
func (mtl *MapTableLoader) loadMaps(dir string) error {
	config := ExcelConfig{
		FileName:   "map.xlsx",
		SheetName:  "Maps",
		MinColumns: 15,
		TableName:  "maps",
	}

	tempMaps := make(map[int32]*models.Map)

	err := ReadExcelFile(config, dir, func(row []string) error {
		m := &models.Map{
			MapID:       StrToInt32(row[0]),
			Name:        row[1],
			MapType:     StrToInt32(row[2]),
			Width:       StrToInt32(row[3]),
			Height:      StrToInt32(row[4]),
			RegionSize:  StrToInt32(row[5]),
			TileWidth:   StrToInt32(row[6]),
			TileHeight:  StrToInt32(row[7]),
			IsInstance:  StrToBool(row[8]),
			MaxPlayers:  StrToInt32(row[9]),
			Description: row[10],
			Background:  row[11],
			Music:       row[12],
			WeatherType: row[13],
			MinLevel:    StrToInt32(row[14]),
			MaxLevel:    StrToInt32(row[15]),
			RespawnRate: StrToFloat64(row[16]),
		}

		tempMaps[m.MapID] = m
		return nil
	})

	if err == nil {
		mtl.maps = tempMaps
	}

	return err
}

// loadSpawnPoints 加载生成点
func (mtl *MapTableLoader) loadSpawnPoints(dir string) error {
	config := ExcelConfig{
		FileName:   "map.xlsx",
		SheetName:  "SpawnPoints",
		MinColumns: 10,
		TableName:  "spawn_points",
	}

	tempSpawnPoints := make(map[int32]*models.MapSpawnPoint)

	err := ReadExcelFile(config, dir, func(row []string) error {
		sp := &models.MapSpawnPoint{
			ID:        StrToInt32(row[0]),
			MapID:     StrToInt32(row[1]),
			Type:      row[2],
			ObjectID:  StrToInt32(row[3]),
			X:         StrToFloat64(row[4]),
			Y:         StrToFloat64(row[5]),
			Z:         StrToFloat64(row[6]),
			Name:      row[7],
			Frequency: StrToInt32(row[8]),
			GroupID:   StrToInt32(row[9]),
		}

		tempSpawnPoints[sp.ID] = sp
		return nil
	})

	if err == nil {
		mtl.spawnPoints = tempSpawnPoints
	}

	return err
}

// loadTeleportPoints 加载传送点
func (mtl *MapTableLoader) loadTeleportPoints(dir string) error {
	config := ExcelConfig{
		FileName:   "map.xlsx",
		SheetName:  "TeleportPoints",
		MinColumns: 13,
		TableName:  "teleport_points",
	}

	tempTeleportPoints := make(map[int32]*models.MapTeleportPoint)

	err := ReadExcelFile(config, dir, func(row []string) error {
		tp := &models.MapTeleportPoint{
			ID:            StrToInt32(row[0]),
			MapID:         StrToInt32(row[1]),
			X:             StrToFloat64(row[2]),
			Y:             StrToFloat64(row[3]),
			Z:             StrToFloat64(row[4]),
			TargetMapID:   StrToInt32(row[5]),
			TargetX:       StrToFloat64(row[6]),
			TargetY:       StrToFloat64(row[7]),
			TargetZ:       StrToFloat64(row[8]),
			Name:          row[9],
			RequiredLevel: StrToInt32(row[10]),
			RequiredItem:  StrToInt32(row[11]),
			IsActive:      StrToBool(row[12]),
		}

		tempTeleportPoints[tp.ID] = tp
		return nil
	})

	if err == nil {
		mtl.teleportPoints = tempTeleportPoints
	}

	return err
}

// loadBuildings 加载建筑物
func (mtl *MapTableLoader) loadBuildings(dir string) error {
	config := ExcelConfig{
		FileName:   "map.xlsx",
		SheetName:  "Buildings",
		MinColumns: 11,
		TableName:  "buildings",
	}

	tempBuildings := make(map[int32]*models.MapBuilding)

	err := ReadExcelFile(config, dir, func(row []string) error {
		b := &models.MapBuilding{
			ID:      StrToInt32(row[0]),
			MapID:   StrToInt32(row[1]),
			X:       StrToFloat64(row[2]),
			Y:       StrToFloat64(row[3]),
			Z:       StrToFloat64(row[4]),
			Width:   StrToFloat64(row[5]),
			Height:  StrToFloat64(row[6]),
			Type:    row[7],
			Name:    row[8],
			Level:   StrToInt32(row[9]),
			HP:      StrToInt32(row[10]),
			Faction: StrToInt32(row[11]),
		}

		tempBuildings[b.ID] = b
		return nil
	})

	if err == nil {
		mtl.buildings = tempBuildings
	}

	return err
}

// loadEvents 加载事件
func (mtl *MapTableLoader) loadEvents(dir string) error {
	config := ExcelConfig{
		FileName:   "map.xlsx",
		SheetName:  "Events",
		MinColumns: 13,
		TableName:  "events",
	}

	tempEvents := make(map[int32]*models.MapEvent)

	err := ReadExcelFile(config, dir, func(row []string) error {
		e := &models.MapEvent{
			EventID:     StrToInt32(row[0]),
			MapID:       StrToInt32(row[1]),
			Type:        row[2],
			Name:        row[3],
			Description: row[4],
			X:           StrToFloat64(row[5]),
			Y:           StrToFloat64(row[6]),
			Z:           StrToFloat64(row[7]),
			Radius:      StrToFloat64(row[8]),
			StartTime:   row[9],
			EndTime:     row[10],
			Duration:    StrToInt32(row[11]),
			RewardID:    StrToInt32(row[12]),
			IsActive:    StrToBool(row[13]),
		}

		tempEvents[e.EventID] = e
		return nil
	})

	if err == nil {
		mtl.events = tempEvents
	}

	return err
}

// loadResources 加载资源
func (mtl *MapTableLoader) loadResources(dir string) error {
	config := ExcelConfig{
		FileName:   "map.xlsx",
		SheetName:  "Resources",
		MinColumns: 10,
		TableName:  "resources",
	}

	tempResources := make(map[int32]*models.MapResource)

	err := ReadExcelFile(config, dir, func(row []string) error {
		r := &models.MapResource{
			ResourceID:  StrToInt32(row[0]),
			MapID:       StrToInt32(row[1]),
			Type:        row[2],
			X:           StrToFloat64(row[3]),
			Y:           StrToFloat64(row[4]),
			Z:           StrToFloat64(row[5]),
			RespawnTime: StrToInt32(row[6]),
			ItemID:      StrToInt32(row[7]),
			Quantity:    StrToInt32(row[8]),
			Level:       StrToInt32(row[9]),
			IsGathering: StrToBool(row[10]),
		}

		tempResources[r.ResourceID] = r
		return nil
	})

	if err == nil {
		mtl.resources = tempResources
	}

	return err
}

// GetMap 根据ID获取地图
func (mtl *MapTableLoader) GetMap(mapID int32) (*models.Map, bool) {
	mapData, ok := mtl.maps[mapID]
	return mapData, ok
}

// GetAllMaps 获取所有地图
func (mtl *MapTableLoader) GetAllMaps() map[int32]*models.Map {
	return mtl.maps
}

// GetSpawnPointsByMapID 根据地图ID获取生成点
func (mtl *MapTableLoader) GetSpawnPointsByMapID(mapID int32) []*models.MapSpawnPoint {
	var spawnPoints []*models.MapSpawnPoint
	for _, sp := range mtl.spawnPoints {
		if sp.MapID == mapID {
			spawnPoints = append(spawnPoints, sp)
		}
	}
	return spawnPoints
}

// GetTeleportPointsByMapID 根据地图ID获取传送点
func (mtl *MapTableLoader) GetTeleportPointsByMapID(mapID int32) []*models.MapTeleportPoint {
	var teleportPoints []*models.MapTeleportPoint
	for _, tp := range mtl.teleportPoints {
		if tp.MapID == mapID {
			teleportPoints = append(teleportPoints, tp)
		}
	}
	return teleportPoints
}

// GetBuildingsByMapID 根据地图ID获取建筑物
func (mtl *MapTableLoader) GetBuildingsByMapID(mapID int32) []*models.MapBuilding {
	var buildings []*models.MapBuilding
	for _, b := range mtl.buildings {
		if b.MapID == mapID {
			buildings = append(buildings, b)
		}
	}
	return buildings
}

// GetEventsByMapID 根据地图ID获取事件
func (mtl *MapTableLoader) GetEventsByMapID(mapID int32) []*models.MapEvent {
	var events []*models.MapEvent
	for _, e := range mtl.events {
		if e.MapID == mapID {
			events = append(events, e)
		}
	}
	return events
}

// GetResourcesByMapID 根据地图ID获取资源
func (mtl *MapTableLoader) GetResourcesByMapID(mapID int32) []*models.MapResource {
	var resources []*models.MapResource
	for _, r := range mtl.resources {
		if r.MapID == mapID {
			resources = append(resources, r)
		}
	}
	return resources
}

// GetTableName 获取表格名称
func (mtl *MapTableLoader) GetTableName() string {
	return "map"
}
