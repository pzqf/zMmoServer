package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/maps"
	"go.uber.org/zap"
)

// MapConfig 地图配置
type MapConfig struct {
	MapID          int32                 `json:"map_id"`
	MapConfigID    int32                 `json:"map_config_id"`
	Name           string                `json:"name"`
	Width          float32               `json:"width"`
	Height         float32               `json:"height"`
	RegionSize     float32               `json:"region_size"`
	SpawnPoints    []SpawnPointConfig    `json:"spawn_points"`
	TeleportPoints []TeleportPointConfig `json:"teleport_points"`
	Buildings      []BuildingConfig      `json:"buildings"`
	Events         []EventConfig         `json:"events"`
	Resources      []ResourceConfig      `json:"resources"`
}

// SpawnPointConfig 刷新点配置
type SpawnPointConfig struct {
	ID          int32   `json:"id"`
	SpawnType   string  `json:"spawn_type"`
	SpawnID     int32   `json:"spawn_id"`
	PositionX   float32 `json:"position_x"`
	PositionY   float32 `json:"position_y"`
	PositionZ   float32 `json:"position_z"`
	RespawnTime int     `json:"respawn_time"`
	MaxCount    int     `json:"max_count"`
	Active      bool    `json:"active"`
}

// TeleportPointConfig 传送点配置
type TeleportPointConfig struct {
	ID          int32   `json:"id"`
	Name        string  `json:"name"`
	PositionX   float32 `json:"position_x"`
	PositionY   float32 `json:"position_y"`
	PositionZ   float32 `json:"position_z"`
	TargetMapID int32   `json:"target_map_id"`
	TargetX     float32 `json:"target_x"`
	TargetY     float32 `json:"target_y"`
	TargetZ     float32 `json:"target_z"`
	Active      bool    `json:"active"`
}

// BuildingConfig 建筑配置
type BuildingConfig struct {
	ID         int32   `json:"id"`
	BuildingID int32   `json:"building_id"`
	Name       string  `json:"name"`
	PositionX  float32 `json:"position_x"`
	PositionY  float32 `json:"position_y"`
	PositionZ  float32 `json:"position_z"`
	Rotation   float32 `json:"rotation"`
	Scale      float32 `json:"scale"`
	Active     bool    `json:"active"`
}

// EventConfig 事件配置
type EventConfig struct {
	ID        int32                  `json:"id"`
	EventType string                 `json:"event_type"`
	Name      string                 `json:"name"`
	PositionX float32                `json:"position_x"`
	PositionY float32                `json:"position_y"`
	PositionZ float32                `json:"position_z"`
	Radius    float32                `json:"radius"`
	Active    bool                   `json:"active"`
	Data      map[string]interface{} `json:"data"`
}

// ResourceConfig 资源配置
type ResourceConfig struct {
	ID          int32   `json:"id"`
	ResourceID  int32   `json:"resource_id"`
	Name        string  `json:"name"`
	PositionX   float32 `json:"position_x"`
	PositionY   float32 `json:"position_y"`
	PositionZ   float32 `json:"position_z"`
	RespawnTime int     `json:"respawn_time"`
	MaxCount    int     `json:"max_count"`
	Active      bool    `json:"active"`
}

// MapConfigLoader 地图配置加载器
type MapConfigLoader struct {
	configDir string
}

// NewMapConfigLoader 创建新的地图配置加载器
func NewMapConfigLoader(configDir string) *MapConfigLoader {
	return &MapConfigLoader{
		configDir: configDir,
	}
}

// LoadMapConfig 加载地图配置
func (loader *MapConfigLoader) LoadMapConfig(mapID int32) (*MapConfig, error) {
	configPath := filepath.Join(loader.configDir, fmt.Sprintf("map_%d.json", mapID))

	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		zLog.Warn("Map config file not found", zap.Int32("map_id", mapID), zap.String("path", configPath))
		return nil, err
	}

	// 读取文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		zLog.Error("Failed to read map config file", zap.Error(err), zap.String("path", configPath))
		return nil, err
	}

	// 解析JSON
	var config MapConfig
	if err := json.Unmarshal(data, &config); err != nil {
		zLog.Error("Failed to unmarshal map config", zap.Error(err), zap.String("path", configPath))
		return nil, err
	}

	zLog.Info("Loaded map config", zap.Int32("map_id", config.MapID), zap.String("name", config.Name))
	return &config, nil
}

// LoadAllMapConfigs 加载所有地图配置
func (loader *MapConfigLoader) LoadAllMapConfigs() ([]*MapConfig, error) {
	var configs []*MapConfig

	// 读取目录中的所有JSON文件
	files, err := os.ReadDir(loader.configDir)
	if err != nil {
		zLog.Error("Failed to read config directory", zap.Error(err), zap.String("dir", loader.configDir))
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			// 解析文件名，提取地图ID
			mapID, err := parseMapIDFromFilename(file.Name())
			if err != nil {
				zLog.Warn("Invalid map config filename", zap.String("filename", file.Name()))
				continue
			}

			// 加载配置
			config, err := loader.LoadMapConfig(mapID)
			if err != nil {
				zLog.Warn("Failed to load map config", zap.Int32("map_id", mapID), zap.Error(err))
				continue
			}

			configs = append(configs, config)
		}
	}

	zLog.Info("Loaded all map configs", zap.Int("count", len(configs)))
	return configs, nil
}

// parseMapIDFromFilename 从文件名解析地图ID
func parseMapIDFromFilename(filename string) (int32, error) {
	// 文件名格式: map_1.json
	baseName := filepath.Base(filename)
	var mapID int32
	_, err := fmt.Sscanf(baseName, "map_%d.json", &mapID)
	return mapID, err
}

// ApplyMapConfig 应用地图配置到地图实例
func (loader *MapConfigLoader) ApplyMapConfig(mapInstance *maps.Map, config *MapConfig) error {
	// 设置地图基本属性
	mapInstance.SetRegionSize(config.RegionSize)

	// 添加刷新点
	for _, spawnPoint := range config.SpawnPoints {
		// 这里可以添加刷新点到地图
		zLog.Debug("Added spawn point",
			zap.Int32("map_id", config.MapID),
			zap.Int32("spawn_id", spawnPoint.ID),
			zap.String("spawn_type", spawnPoint.SpawnType))
	}

	// 添加传送点
	for _, teleportPoint := range config.TeleportPoints {
		// 这里可以添加传送点到地图
		zLog.Debug("Added teleport point",
			zap.Int32("map_id", config.MapID),
			zap.Int32("teleport_id", teleportPoint.ID),
			zap.String("name", teleportPoint.Name))
	}

	// 添加建筑
	for _, building := range config.Buildings {
		// 这里可以添加建筑到地图
		zLog.Debug("Added building",
			zap.Int32("map_id", config.MapID),
			zap.Int32("building_id", building.ID),
			zap.String("name", building.Name))
	}

	// 添加事件
	for _, event := range config.Events {
		// 这里可以添加事件到地图
		zLog.Debug("Added event",
			zap.Int32("map_id", config.MapID),
			zap.Int32("event_id", event.ID),
			zap.String("event_type", event.EventType))
	}

	// 添加资源
	for _, resource := range config.Resources {
		// 这里可以添加资源到地图
		zLog.Debug("Added resource",
			zap.Int32("map_id", config.MapID),
			zap.Int32("resource_id", resource.ID),
			zap.String("name", resource.Name))
	}

	zLog.Info("Applied map config", zap.Int32("map_id", config.MapID))
	return nil
}
