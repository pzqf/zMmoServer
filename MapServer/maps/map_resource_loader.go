package maps

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// MapResource 地图资源结构
type MapResource struct {
	MapID         int32       `json:"map_id"`
	Name          string      `json:"name"`
	MapType       int32       `json:"map_type"`
	Width         float64     `json:"width"`
	Height        float64     `json:"height"`
	RegionSize    float64     `json:"region_size"`
	TileWidth     float64     `json:"tile_width"`
	TileHeight    float64     `json:"tile_height"`
	IsInstance    bool        `json:"is_instance"`
	MaxPlayers    int32       `json:"max_players"`
	Description   string      `json:"description"`
	Background    string      `json:"background"`
	Music         string      `json:"music"`
	WeatherType   string      `json:"weather_type"`
	MinLevel      int32       `json:"min_level"`
	MaxLevel      int32       `json:"max_level"`
	RespawnRate   float64     `json:"respawn_rate"`
	TileMap       [][]int32   `json:"tile_map"`
	SpawnPoints   []MapSpawnPoint `json:"spawn_points"`
	TeleportPoints []TeleportPoint `json:"teleport_points"`
	Buildings     []Building  `json:"buildings"`
	Resources     []Resource  `json:"resources"`
}

// MapSpawnPoint 地图出生点
type MapSpawnPoint struct {
	Type string  `json:"type"`
	ID   int32   `json:"id"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Z    float64 `json:"z"`
	Name string  `json:"name"`
}

// TeleportPoint 传送点
type TeleportPoint struct {
	ID             int32   `json:"id"`
	MapID          int32   `json:"map_id"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Z              float64 `json:"z"`
	TargetMapID    int32   `json:"target_map_id"`
	TargetX        float64 `json:"target_x"`
	TargetY        float64 `json:"target_y"`
	TargetZ        float64 `json:"target_z"`
	Name           string  `json:"name"`
	RequiredLevel  int32   `json:"required_level"`
	RequiredItem   int32   `json:"required_item"`
	IsActive       bool    `json:"is_active"`
}

// Building 建筑
type Building struct {
	ID       int32   `json:"id"`
	MapID    int32   `json:"map_id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Z        float64 `json:"z"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Type     string  `json:"type"`
	Name     string  `json:"name"`
	Level    int32   `json:"level"`
	HP       int32   `json:"hp"`
	Faction  int32   `json:"faction"`
}

// Resource 资源点
type Resource struct {
	ResourceID   int32   `json:"resource_id"`
	MapID        int32   `json:"map_id"`
	Type         string  `json:"type"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Z            float64 `json:"z"`
	RespawnTime  int32   `json:"respawn_time"`
	ItemID       int32   `json:"item_id"`
	Quantity     int32   `json:"quantity"`
	Level        int32   `json:"level"`
	IsGathering  bool    `json:"is_gathering"`
}

// MapResourceLoader 地图资源加载器
type MapResourceLoader struct {
	resourcesDir string
}

// NewMapResourceLoader 创建地图资源加载器
func NewMapResourceLoader(resourcesDir string) *MapResourceLoader {
	return &MapResourceLoader{
		resourcesDir: resourcesDir,
	}
}

// LoadMapFromFile 从文件加载地图
func (mrl *MapResourceLoader) LoadMapFromFile(filename string) (*MapResource, error) {
	// 构建完整路径
	filePath := filepath.Join(mrl.resourcesDir, filename)
	
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("map file not found: %s", filePath)
	}
	
	// 读取文件内容
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read map file: %w", err)
	}
	
	// 解析JSON
	var mapResource MapResource
	if err := json.Unmarshal(data, &mapResource); err != nil {
		return nil, fmt.Errorf("parse map file: %w", err)
	}
	
	return &mapResource, nil
}

// LoadMapByID 根据地图ID加载地图
func (mrl *MapResourceLoader) LoadMapByID(mapID id.MapIdType) (*MapResource, error) {
	// 构建文件名
	filename := fmt.Sprintf("%s.json", mrl.getMapFileName(mapID))
	return mrl.LoadMapFromFile(filename)
}

// GetAllMapFiles 获取所有地图文件
func (mrl *MapResourceLoader) GetAllMapFiles() ([]string, error) {
	// 检查目录是否存在
	if _, err := os.Stat(mrl.resourcesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("resources directory not found: %s", mrl.resourcesDir)
	}
	
	// 读取目录内容
	files, err := ioutil.ReadDir(mrl.resourcesDir)
	if err != nil {
		return nil, fmt.Errorf("read resources directory: %w", err)
	}
	
	// 过滤出JSON文件，排除非地图文件
	var mapFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			// 排除非地图文件
			filename := file.Name()
			if !strings.Contains(filename, "_monsters") && !strings.Contains(filename, "_npcs") && !strings.Contains(filename, "_quests") && !strings.Contains(filename, "_spawns") {
				mapFiles = append(mapFiles, filename)
			}
		}
	}
	
	return mapFiles, nil
}

// getMapFileName 根据地图ID获取文件名
func (mrl *MapResourceLoader) getMapFileName(mapID id.MapIdType) string {
	switch mapID {
	case 1001:
		return "novice_village"
	case 1002:
		return "city_map"
	case 2001:
		return "wilderness"
	case 2002:
		return "instance_dungeon"
	case 3001:
		return "world_map1"
	case 3002:
		return "world_map2"
	default:
		return fmt.Sprintf("map_%d", mapID)
	}
}

// LoadAllMaps 加载所有地图
func (mrl *MapResourceLoader) LoadAllMaps() (map[id.MapIdType]*MapResource, error) {
	// 获取所有地图文件
	mapFiles, err := mrl.GetAllMapFiles()
	if err != nil {
		return nil, err
	}
	
	// 加载每个地图文件
	maps := make(map[id.MapIdType]*MapResource)
	for _, file := range mapFiles {
		mapResource, err := mrl.LoadMapFromFile(file)
		if err != nil {
			zLog.Warn("Failed to load map file", zap.String("file", file), zap.Error(err))
			continue
		}
		
		maps[id.MapIdType(mapResource.MapID)] = mapResource
		zLog.Info("Loaded map from file", zap.Int32("map_id", mapResource.MapID), zap.String("name", mapResource.Name))
	}
	
	return maps, nil
}
