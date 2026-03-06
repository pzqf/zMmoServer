package pet

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// PetType 宠物类型
type PetType int32

const (
	PetTypeCommon    PetType = 1 // 普通
	PetTypeElite     PetType = 2 // 精英
	PetTypeRare      PetType = 3 // 稀有
	PetTypeEpic      PetType = 4 // 史诗
	PetTypeLegendary PetType = 5 // 传说
)

// PetAttribute 宠物属性
type PetAttribute struct {
	Strength     int32 `json:"strength"`     // 力量
	Agility      int32 `json:"agility"`      // 敏捷
	Intelligence int32 `json:"intelligence"` // 智力
	Stamina      int32 `json:"stamina"`      // 耐力
}

// PetSkill 宠物技能
type PetSkill struct {
	SkillID int32 `json:"skill_id"`
	Level   int32 `json:"level"`
}

// PetConfig 宠物配置
type PetConfig struct {
	ID          int32        `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        PetType      `json:"type"`
	Level       int32        `json:"level"`
	Model       string       `json:"model"`        // 模型
	Icon        string       `json:"icon"`         // 图标
	Attribute   PetAttribute `json:"attribute"`    // 基础属性
	Skills      []PetSkill   `json:"skills"`       // 技能
	EvolutionID int32        `json:"evolution_id"` // 进化ID
	Price       int64        `json:"price"`        // 价格
}

// PlayerPet 玩家宠物
type PlayerPet struct {
	PetID       id.PetIdType `json:"pet_id"`
	ConfigID    int32        `json:"config_id"`
	Name        string       `json:"name"`
	Level       int32        `json:"level"`
	Exp         int64        `json:"exp"`
	ExpToNext   int64        `json:"exp_to_next"`
	Quality     int32        `json:"quality"`      // 品质（1-5）
	Attributes  PetAttribute `json:"attributes"`   // 属性
	Skills      []PetSkill   `json:"skills"`       // 技能
	Status      int32        `json:"status"`       // 0:休息 1:出战
	BattlePower int32        `json:"battle_power"` // 战斗力
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// PetConfigManager 宠物配置管理器
type PetConfigManager struct {
	mu      sync.RWMutex
	configs map[int32]*PetConfig
}

// NewPetConfigManager 创建宠物配置管理器
func NewPetConfigManager() *PetConfigManager {
	return &PetConfigManager{
		configs: make(map[int32]*PetConfig),
	}
}

// LoadConfig 加载宠物配置
func (pcm *PetConfigManager) LoadConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		zLog.Error("Failed to read pet config file", zap.Error(err))
		return err
	}

	var configs []*PetConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		zLog.Error("Failed to unmarshal pet config", zap.Error(err))
		return err
	}

	for _, config := range configs {
		pcm.configs[config.ID] = config
	}

	zLog.Info("Pet config loaded successfully", zap.Int("count", len(pcm.configs)))
	return nil
}

// GetConfig 获取宠物配置
func (pcm *PetConfigManager) GetConfig(configID int32) *PetConfig {
	pcm.mu.RLock()
	defer pcm.mu.RUnlock()
	return pcm.configs[configID]
}

// PetManager 宠物管理器
type PetManager struct {
	mu            sync.RWMutex
	configManager *PetConfigManager
	playerPets    map[id.PlayerIdType]map[id.PetIdType]*PlayerPet
	activePets    map[id.PlayerIdType]id.PetIdType // 出战宠物
}

// NewPetManager 创建宠物管理器
func NewPetManager() *PetManager {
	return &PetManager{
		configManager: NewPetConfigManager(),
		playerPets:    make(map[id.PlayerIdType]map[id.PetIdType]*PlayerPet),
		activePets:    make(map[id.PlayerIdType]id.PetIdType),
	}
}

// LoadConfig 加载宠物配置
func (pm *PetManager) LoadConfig(filePath string) error {
	return pm.configManager.LoadConfig(filePath)
}

// GetConfig 获取宠物配置
func (pm *PetManager) GetConfig(configID int32) *PetConfig {
	return pm.configManager.GetConfig(configID)
}

// AddPet 添加宠物
func (pm *PetManager) AddPet(playerID id.PlayerIdType, configID int32, name string) (*PlayerPet, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	config := pm.configManager.GetConfig(configID)
	if config == nil {
		return nil, nil
	}

	// 生成宠物ID
	petID := id.PetIdType(time.Now().UnixNano() % 1000000000)

	// 创建宠物
	pet := &PlayerPet{
		PetID:       petID,
		ConfigID:    configID,
		Name:        name,
		Level:       1,
		Exp:         0,
		ExpToNext:   100,
		Quality:     int32(config.Type),
		Attributes:  config.Attribute,
		Skills:      make([]PetSkill, len(config.Skills)),
		Status:      0,
		BattlePower: config.Attribute.Strength + config.Attribute.Agility + config.Attribute.Intelligence + config.Attribute.Stamina,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 复制技能
	copy(pet.Skills, config.Skills)

	// 保存宠物
	if _, ok := pm.playerPets[playerID]; !ok {
		pm.playerPets[playerID] = make(map[id.PetIdType]*PlayerPet)
	}
	pm.playerPets[playerID][petID] = pet

	zLog.Debug("Pet added",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("pet_id", int64(petID)),
		zap.Int32("config_id", configID),
		zap.String("name", name))

	return pet, nil
}

// RemovePet 移除宠物
func (pm *PetManager) RemovePet(playerID id.PlayerIdType, petID id.PetIdType) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pets, ok := pm.playerPets[playerID]; ok {
		if _, ok := pets[petID]; ok {
			// 如果是出战宠物，先取消
			if activePetID, ok := pm.activePets[playerID]; ok && activePetID == petID {
				delete(pm.activePets, playerID)
			}
			delete(pets, petID)

			zLog.Debug("Pet removed",
				zap.Int64("player_id", int64(playerID)),
				zap.Int64("pet_id", int64(petID)))
		}
	}

	return nil
}

// GetPet 获取宠物
func (pm *PetManager) GetPet(playerID id.PlayerIdType, petID id.PetIdType) *PlayerPet {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pets, ok := pm.playerPets[playerID]; ok {
		return pets[petID]
	}
	return nil
}

// GetAllPets 获取玩家所有宠物
func (pm *PetManager) GetAllPets(playerID id.PlayerIdType) []*PlayerPet {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	pets := make([]*PlayerPet, 0)
	if playerPets, ok := pm.playerPets[playerID]; ok {
		for _, pet := range playerPets {
			pets = append(pets, pet)
		}
	}
	return pets
}

// GetActivePet 获取玩家出战宠物
func (pm *PetManager) GetActivePet(playerID id.PlayerIdType) *PlayerPet {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	petID, ok := pm.activePets[playerID]
	if !ok {
		return nil
	}

	if pets, ok := pm.playerPets[playerID]; ok {
		return pets[petID]
	}
	return nil
}

// SetActivePet 设置出战宠物
func (pm *PetManager) SetActivePet(playerID id.PlayerIdType, petID id.PetIdType) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查宠物是否存在
	if pets, ok := pm.playerPets[playerID]; ok {
		if _, ok := pets[petID]; ok {
			pm.activePets[playerID] = petID

			zLog.Debug("Pet set active",
				zap.Int64("player_id", int64(playerID)),
				zap.Int64("pet_id", int64(petID)))

			return nil
		}
	}

	return nil
}

// UnsetActivePet 取消出战宠物
func (pm *PetManager) UnsetActivePet(playerID id.PlayerIdType) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, ok := pm.activePets[playerID]; ok {
		delete(pm.activePets, playerID)

		zLog.Debug("Pet unset active",
			zap.Int64("player_id", int64(playerID)))
	}

	return nil
}

// AddPetExp 增加宠物经验
func (pm *PetManager) AddPetExp(playerID id.PlayerIdType, petID id.PetIdType, exp int64) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pets, ok := pm.playerPets[playerID]; ok {
		if pet, ok := pets[petID]; ok {
			pet.Exp += exp
			pet.UpdatedAt = time.Now()

			// 检查升级
			for pet.Exp >= pet.ExpToNext {
				pet.Exp -= pet.ExpToNext
				pet.Level++
				pet.ExpToNext = int64(100 * pet.Level)

				// 属性成长
				pet.Attributes.Strength += 2
				pet.Attributes.Agility += 2
				pet.Attributes.Intelligence += 2
				pet.Attributes.Stamina += 2

				// 战斗力提升
				pet.BattlePower += 8
			}

			zLog.Debug("Pet exp added",
				zap.Int64("player_id", int64(playerID)),
				zap.Int64("pet_id", int64(petID)),
				zap.Int64("exp", exp),
				zap.Int32("new_level", pet.Level))

			return nil
		}
	}

	return nil
}

// RenamePet 重命名宠物
func (pm *PetManager) RenamePet(playerID id.PlayerIdType, petID id.PetIdType, newName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pets, ok := pm.playerPets[playerID]; ok {
		if pet, ok := pets[petID]; ok {
			pet.Name = newName
			pet.UpdatedAt = time.Now()

			zLog.Debug("Pet renamed",
				zap.Int64("player_id", int64(playerID)),
				zap.Int64("pet_id", int64(petID)),
				zap.String("new_name", newName))

			return nil
		}
	}

	return nil
}

// GetPetCount 获取宠物数量
func (pm *PetManager) GetPetCount(playerID id.PlayerIdType) int32 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pets, ok := pm.playerPets[playerID]; ok {
		return int32(len(pets))
	}
	return 0
}
