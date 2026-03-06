package mount

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// MountType 坐骑类型
type MountType int32

const (
	MountTypeLand   MountType = 1 // 陆地
	MountTypeFlying MountType = 2 // 飞行
	MountTypeSwimming MountType = 3 // 水中
)

// MountAttribute 坐骑属性
type MountAttribute struct {
	Speed      float32 `json:"speed"`       // 移动速度加成
	Attack     int32   `json:"attack"`      // 攻击力加成
	Defense    int32   `json:"defense"`     // 防御力加成
	HP         int32   `json:"hp"`          // 生命值加成
}

// MountSkill 坐骑技能
type MountSkill struct {
	SkillID int32 `json:"skill_id"`
	Level   int32 `json:"level"`
}

// MountConfig 坐骑配置
type MountConfig struct {
	ID           int32           `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Type         MountType      `json:"type"`
	Level        int32           `json:"level"`
	Model        string          `json:"model"`
	Icon         string          `json:"icon"`
	Attribute    MountAttribute `json:"attribute"`
	Skills       []MountSkill   `json:"skills"`
	EvolutionID  int32           `json:"evolution_id"` // 进化ID
	Price        int64           `json:"price"`
	RequireLevel int32           `json:"require_level"`
}

// PlayerMount 玩家坐骑
type PlayerMount struct {
	MountID    id.MountIdType `json:"mount_id"`
	ConfigID   int32         `json:"config_id"`
	Name       string        `json:"name"`
	Level      int32         `json:"level"`
	Exp        int64         `json:"exp"`
	ExpToNext  int64         `json:"exp_to_next"`
	Quality    int32         `json:"quality"`
	Attributes MountAttribute `json:"attributes"`
	Skills     []MountSkill `json:"skills"`
	IsActive   bool          `json:"is_active"` // 是否激活
	BattlePower int32        `json:"battle_power"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

// MountConfigManager 坐骑配置管理器
type MountConfigManager struct {
	mu       sync.RWMutex
	configs  map[int32]*MountConfig
}

// NewMountConfigManager 创建坐骑配置管理器
func NewMountConfigManager() *MountConfigManager {
	return &MountConfigManager{
		configs: make(map[int32]*MountConfig),
	}
}

// LoadConfig 加载坐骑配置
func (mcm *MountConfigManager) LoadConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		zLog.Error("Failed to read mount config file", zap.Error(err))
		return err
	}

	var configs []*MountConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		zLog.Error("Failed to unmarshal mount config", zap.Error(err))
		return err
	}

	for _, config := range configs {
		mcm.configs[config.ID] = config
	}

	zLog.Info("Mount config loaded successfully", zap.Int("count", len(mcm.configs)))
	return nil
}

// GetConfig 获取坐骑配置
func (mcm *MountConfigManager) GetConfig(configID int32) *MountConfig {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()
	return mcm.configs[configID]
}

// MountManager 坐骑管理器
type MountManager struct {
	mu            sync.RWMutex
	configManager *MountConfigManager
	playerMounts  map[id.PlayerIdType]map[id.MountIdType]*PlayerMount
	activeMounts   map[id.PlayerIdType]id.MountIdType // 当前激活的坐骑
}

// NewMountManager 创建坐骑管理器
func NewMountManager() *MountManager {
	return &MountManager{
		configManager: NewMountConfigManager(),
		playerMounts: make(map[id.PlayerIdType]map[id.MountIdType]*PlayerMount),
		activeMounts: make(map[id.PlayerIdType]id.MountIdType),
	}
}

// LoadConfig 加载坐骑配置
func (mm *MountManager) LoadConfig(filePath string) error {
	return mm.configManager.LoadConfig(filePath)
}

// GetConfig 获取坐骑配置
func (mm *MountManager) GetConfig(configID int32) *MountConfig {
	return mm.configManager.GetConfig(configID)
}

// AddMount 添加坐骑
func (mm *MountManager) AddMount(playerID id.PlayerIdType, configID int32, name string) (*PlayerMount, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	config := mm.configManager.GetConfig(configID)
	if config == nil {
		return nil, nil
	}

	// 生成坐骑ID
	mountID := id.MountIdType(time.Now().UnixNano() % 1000000000)

	// 创建坐骑
	mount := &PlayerMount{
		MountID:    mountID,
		ConfigID:   configID,
		Name:       name,
		Level:      1,
		Exp:        0,
		ExpToNext:  100,
		Quality:    1,
		Attributes: config.Attribute,
		Skills:     make([]MountSkill, len(config.Skills)),
		IsActive:   false,
		BattlePower: config.Attribute.Attack + config.Attribute.Defense + config.Attribute.HP,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 复制技能
	copy(mount.Skills, config.Skills)

	// 保存坐骑
	if _, ok := mm.playerMounts[playerID]; !ok {
		mm.playerMounts[playerID] = make(map[id.MountIdType]*PlayerMount)
	}
	mm.playerMounts[playerID][mountID] = mount

	zLog.Debug("Mount added",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("mount_id", int64(mountID)),
		zap.Int32("config_id", configID),
		zap.String("name", name))

	return mount, nil
}

// RemoveMount 移除坐骑
func (mm *MountManager) RemoveMount(playerID id.PlayerIdType, mountID id.MountIdType) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mounts, ok := mm.playerMounts[playerID]; ok {
		if _, ok := mounts[mountID]; ok {
			// 如果是激活的坐骑，先取消
			if activeMountID, ok := mm.activeMounts[playerID]; ok && activeMountID == mountID {
				delete(mm.activeMounts, playerID)
			}
			delete(mounts, mountID)

			zLog.Debug("Mount removed",
				zap.Int64("player_id", int64(playerID)),
				zap.Int64("mount_id", int64(mountID)))
		}
	}

	return nil
}

// GetMount 获取坐骑
func (mm *MountManager) GetMount(playerID id.PlayerIdType, mountID id.MountIdType) *PlayerMount {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	if mounts, ok := mm.playerMounts[playerID]; ok {
		return mounts[mountID]
	}
	return nil
}

// GetAllMounts 获取玩家所有坐骑
func (mm *MountManager) GetAllMounts(playerID id.PlayerIdType) []*PlayerMount {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mounts := make([]*PlayerMount, 0)
	if playerMounts, ok := mm.playerMounts[playerID]; ok {
		for _, mount := range playerMounts {
			mounts = append(mounts, mount)
		}
	}
	return mounts
}

// GetActiveMount 获取玩家当前激活的坐骑
func (mm *MountManager) GetActiveMount(playerID id.PlayerIdType) *PlayerMount {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mountID, ok := mm.activeMounts[playerID]
	if !ok {
		return nil
	}

	if mounts, ok := mm.playerMounts[playerID]; ok {
		return mounts[mountID]
	}
	return nil
}

// SetActiveMount 激活坐骑
func (mm *MountManager) SetActiveMount(playerID id.PlayerIdType, mountID id.MountIdType) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// 检查坐骑是否存在
	if mounts, ok := mm.playerMounts[playerID]; ok {
		if _, ok := mounts[mountID]; ok {
			mm.activeMounts[playerID] = mountID

			zLog.Debug("Mount activated",
				zap.Int64("player_id", int64(playerID)),
				zap.Int64("mount_id", int64(mountID)))

			return nil
		}
	}

	return nil
}

// UnsetActiveMount 取消激活坐骑
func (mm *MountManager) UnsetActiveMount(playerID id.PlayerIdType) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, ok := mm.activeMounts[playerID]; ok {
		delete(mm.activeMounts, playerID)

		zLog.Debug("Mount deactivated",
			zap.Int64("player_id", int64(playerID)))
	}

	return nil
}

// AddMountExp 增加坐骑经验
func (mm *MountManager) AddMountExp(playerID id.PlayerIdType, mountID id.MountIdType, exp int64) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mounts, ok := mm.playerMounts[playerID]; ok {
		if mount, ok := mounts[mountID]; ok {
			mount.Exp += exp
			mount.UpdatedAt = time.Now()

			// 检查升级
			for mount.Exp >= mount.ExpToNext {
				mount.Exp -= mount.ExpToNext
				mount.Level++
				mount.ExpToNext = int64(100 * mount.Level)

				// 属性成长
				mount.Attributes.Speed += 0.1
				mount.Attributes.Attack += 3
				mount.Attributes.Defense += 3
				mount.Attributes.HP += 20

				// 战斗力提升
				mount.BattlePower += 10
			}

			zLog.Debug("Mount exp added",
				zap.Int64("player_id", int64(playerID)),
				zap.Int64("mount_id", int64(mountID)),
				zap.Int64("exp", exp),
				zap.Int32("new_level", mount.Level))

			return nil
		}
	}

	return nil
}

// RenameMount 重命名坐骑
func (mm *MountManager) RenameMount(playerID id.PlayerIdType, mountID id.MountIdType, newName string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mounts, ok := mm.playerMounts[playerID]; ok {
		if mount, ok := mounts[mountID]; ok {
			mount.Name = newName
			mount.UpdatedAt = time.Now()

			zLog.Debug("Mount renamed",
				zap.Int64("player_id", int64(playerID)),
				zap.Int64("mount_id", int64(mountID)),
				zap.String("new_name", newName))

			return nil
		}
	}

	return nil
}

// GetMountCount 获取坐骑数量
func (mm *MountManager) GetMountCount(playerID id.PlayerIdType) int32 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	if mounts, ok := mm.playerMounts[playerID]; ok {
		return int32(len(mounts))
	}
	return 0
}

// GetMountSpeedBonus 获取坐骑速度加成
func (mm *MountManager) GetMountSpeedBonus(playerID id.PlayerIdType) float32 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mountID, ok := mm.activeMounts[playerID]
	if !ok {
		return 0
	}

	if mounts, ok := mm.playerMounts[playerID]; ok {
		if mount, ok := mounts[mountID]; ok {
			return mount.Attributes.Speed
		}
	}

	return 0
}