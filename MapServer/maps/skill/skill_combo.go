package skill

import (
	"encoding/json"
	"os"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// SkillCombo 技能组合
type SkillCombo struct {
	ID             int32   `json:"id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	SkillSequence  []int32 `json:"skill_sequence"`
	RequiredTime   int64   `json:"required_time"`
	BonusDamage    int32   `json:"bonus_damage"`
	EffectID       int32   `json:"effect_id"`
}

// SkillComboManager 技能组合管理器
type SkillComboManager struct {
	combos map[int32]*SkillCombo
}

// NewSkillComboManager 创建技能组合管理器
func NewSkillComboManager() *SkillComboManager {
	return &SkillComboManager{
		combos: make(map[int32]*SkillCombo),
	}
}

// LoadComboConfig 加载技能组合配置
func (scm *SkillComboManager) LoadComboConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		zLog.Error("Failed to read skill combo config file", zap.Error(err))
		return err
	}

	var combos []*SkillCombo
	if err := json.Unmarshal(data, &combos); err != nil {
		zLog.Error("Failed to unmarshal skill combo config", zap.Error(err))
		return err
	}

	for _, combo := range combos {
		scm.combos[combo.ID] = combo
	}

	zLog.Info("Skill combo config loaded successfully", zap.Int("count", len(scm.combos)))
	return nil
}

// GetCombo 获取技能组合
func (scm *SkillComboManager) GetCombo(comboID int32) *SkillCombo {
	return scm.combos[comboID]
}

// GetAllCombos 获取所有技能组合
func (scm *SkillComboManager) GetAllCombos() []*SkillCombo {
	combos := make([]*SkillCombo, 0, len(scm.combos))
	for _, combo := range scm.combos {
		combos = append(combos, combo)
	}
	return combos
}

// CheckSkillCombo 检查技能组合
func (scm *SkillComboManager) CheckSkillCombo(skillSequence []int32, startTime time.Time) *SkillCombo {
	currentTime := time.Now()
	elapsed := currentTime.Sub(startTime).Milliseconds()

	for _, combo := range scm.combos {
		if len(combo.SkillSequence) != len(skillSequence) {
			continue
		}

		// 检查技能序列是否匹配
		match := true
		for i, skillID := range combo.SkillSequence {
			if skillID != skillSequence[i] {
				match = false
				break
			}
		}

		if match && elapsed <= combo.RequiredTime {
			return combo
		}
	}

	return nil
}