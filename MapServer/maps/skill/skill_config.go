package skill

import (
	"encoding/json"
	"os"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// SkillConfig 技能配置
type SkillConfig struct {
	ID          int32   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        int32   `json:"type"`
	Level       int32   `json:"level"`
	Damage      int32   `json:"damage"`
	Range       float32 `json:"range"`
	Cooldown    int64   `json:"cooldown"`
	Duration    int64   `json:"duration"`
	ManaCost    int32   `json:"mana_cost"`
	EffectID    int32   `json:"effect_id"`
}

// SkillConfigManager 技能配置管理器
type SkillConfigManager struct {
	configs map[int32]*SkillConfig
}

// NewSkillConfigManager 创建技能配置管理器
func NewSkillConfigManager() *SkillConfigManager {
	return &SkillConfigManager{
		configs: make(map[int32]*SkillConfig),
	}
}

// LoadConfig 加载技能配置
func (scm *SkillConfigManager) LoadConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		zLog.Error("Failed to read skill config file", zap.Error(err))
		return err
	}

	var configs []*SkillConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		zLog.Error("Failed to unmarshal skill config", zap.Error(err))
		return err
	}

	for _, config := range configs {
		scm.configs[config.ID] = config
	}

	zLog.Info("Skill config loaded successfully", zap.Int("count", len(scm.configs)))
	return nil
}

// GetConfig 获取技能配置
func (scm *SkillConfigManager) GetConfig(skillID int32) *SkillConfig {
	return scm.configs[skillID]
}

// GetAllConfigs 获取所有技能配置
func (scm *SkillConfigManager) GetAllConfigs() []*SkillConfig {
	configs := make([]*SkillConfig, 0, len(scm.configs))
	for _, config := range scm.configs {
		configs = append(configs, config)
	}
	return configs
}

// GetConfigCount 获取技能配置数量
func (scm *SkillConfigManager) GetConfigCount() int {
	return len(scm.configs)
}