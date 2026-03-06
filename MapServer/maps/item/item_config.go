package item

import (
	"encoding/json"
	"os"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// ItemType 物品类型
type ItemType int32

const (
	ItemTypeConsumable ItemType = 1 // 消耗品
	ItemTypeEquipment  ItemType = 2 // 装备
	ItemTypeMaterial   ItemType = 3 // 材料
	ItemTypeQuest      ItemType = 4 // 任务物品
	ItemTypeSpecial    ItemType = 5 // 特殊物品
)

// ItemRarity 物品稀有度
type ItemRarity int32

const (
	ItemRarityCommon    ItemRarity = 1 // 普通
	ItemRarityUncommon  ItemRarity = 2 // 优秀
	ItemRarityRare      ItemRarity = 3 // 稀有
	ItemRarityEpic      ItemRarity = 4 // 史诗
	ItemRarityLegendary ItemRarity = 5 // 传说
)

// ItemEffect 物品效果
type ItemEffect struct {
	Type     string `json:"type"`     // 效果类型：attack, defense, health, mana, speed
	Value    int32  `json:"value"`    // 效果值
	Duration int64  `json:"duration"` // 持续时间（毫秒）
}

// ItemConfig 物品配置
type ItemConfig struct {
	ID          int32        `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        ItemType     `json:"type"`
	Rarity      ItemRarity   `json:"rarity"`
	Level       int32        `json:"level"`
	Price       int64        `json:"price"`
	Weight      float32      `json:"weight"`
	Effects     []ItemEffect `json:"effects"`
	MaxStack    int32        `json:"max_stack"`
	Icon        string       `json:"icon"`
}

// ItemConfigManager 物品配置管理器
type ItemConfigManager struct {
	configs map[int32]*ItemConfig
}

// NewItemConfigManager 创建物品配置管理器
func NewItemConfigManager() *ItemConfigManager {
	return &ItemConfigManager{
		configs: make(map[int32]*ItemConfig),
	}
}

// LoadConfig 加载物品配置
func (icm *ItemConfigManager) LoadConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		zLog.Error("Failed to read item config file", zap.Error(err))
		return err
	}

	var configs []*ItemConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		zLog.Error("Failed to unmarshal item config", zap.Error(err))
		return err
	}

	for _, config := range configs {
		icm.configs[config.ID] = config
	}

	zLog.Info("Item config loaded successfully", zap.Int("count", len(icm.configs)))
	return nil
}

// GetConfig 获取物品配置
func (icm *ItemConfigManager) GetConfig(itemID int32) *ItemConfig {
	return icm.configs[itemID]
}

// GetAllConfigs 获取所有物品配置
func (icm *ItemConfigManager) GetAllConfigs() []*ItemConfig {
	configs := make([]*ItemConfig, 0, len(icm.configs))
	for _, config := range icm.configs {
		configs = append(configs, config)
	}
	return configs
}

// GetItemsByType 获取指定类型的物品
func (icm *ItemConfigManager) GetItemsByType(itemType ItemType) []*ItemConfig {
	items := make([]*ItemConfig, 0)
	for _, config := range icm.configs {
		if config.Type == itemType {
			items = append(items, config)
		}
	}
	return items
}

// GetItemsByLevel 获取指定等级范围内的物品
func (icm *ItemConfigManager) GetItemsByLevel(minLevel, maxLevel int32) []*ItemConfig {
	items := make([]*ItemConfig, 0)
	for _, config := range icm.configs {
		if config.Level >= minLevel && config.Level <= maxLevel {
			items = append(items, config)
		}
	}
	return items
}
