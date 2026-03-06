package player

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/mount"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// PlayerMount 玩家坐骑组件
type PlayerMount struct {
	playerID     id.PlayerIdType
	mounts       map[id.MountIdType]*mount.Mount
	activeMountID id.MountIdType
	maxMountCount int
}

// NewPlayerMount 创建玩家坐骑组件
func NewPlayerMount(playerID id.PlayerIdType) *PlayerMount {
	return &PlayerMount{
		playerID:      playerID,
		mounts:        make(map[id.MountIdType]*mount.Mount),
		activeMountID: 0,
		maxMountCount: 20, // 默认最多20个坐骑
	}
}

// GetMounts 获取所有坐骑
func (pm *PlayerMount) GetMounts() []*mount.Mount {
	result := make([]*mount.Mount, 0, len(pm.mounts))
	for _, m := range pm.mounts {
		result = append(result, m)
	}
	return result
}

// GetMount 获取指定坐骑
func (pm *PlayerMount) GetMount(mountID id.MountIdType) *mount.Mount {
	return pm.mounts[mountID]
}

// AddMount 添加坐骑
func (pm *PlayerMount) AddMount(m *mount.Mount) bool {
	if len(pm.mounts) >= pm.maxMountCount {
		zLog.Warn("Player mount count reached limit",
			zap.Uint64("player_id", uint64(pm.playerID)),
			zap.Int("max_count", pm.maxMountCount))
		return false
	}

	pm.mounts[m.MountID] = m
	zLog.Info("Player added mount",
		zap.Uint64("player_id", uint64(pm.playerID)),
		zap.Uint64("mount_id", uint64(m.MountID)),
		zap.String("mount_name", m.Name))
	return true
}

// RemoveMount 移除坐骑
func (pm *PlayerMount) RemoveMount(mountID id.MountIdType) bool {
	if _, exists := pm.mounts[mountID]; !exists {
		return false
	}

	// 如果是激活的坐骑，先下马
	if pm.activeMountID == mountID {
		pm.Dismount()
	}

	delete(pm.mounts, mountID)
	zLog.Info("Player removed mount",
		zap.Uint64("player_id", uint64(pm.playerID)),
		zap.Uint64("mount_id", uint64(mountID)))
	return true
}

// GetActiveMount 获取当前激活的坐骑
func (pm *PlayerMount) GetActiveMount() *mount.Mount {
	if pm.activeMountID == 0 {
		return nil
	}
	return pm.mounts[pm.activeMountID]
}

// GetActiveMountID 获取当前激活的坐骑ID
func (pm *PlayerMount) GetActiveMountID() id.MountIdType {
	return pm.activeMountID
}

// IsRiding 是否正在骑乘
func (pm *PlayerMount) IsRiding() bool {
	if pm.activeMountID == 0 {
		return false
	}
	m := pm.mounts[pm.activeMountID]
	if m == nil {
		return false
	}
	return m.IsRiding
}

// RideMount 骑乘坐骑
func (pm *PlayerMount) RideMount(mountID id.MountIdType) bool {
	m, exists := pm.mounts[mountID]
	if !exists {
		return false
	}

	// 先下马当前坐骑
	if pm.activeMountID > 0 && pm.activeMountID != mountID {
		pm.Dismount()
	}

	if m.Ride() {
		pm.activeMountID = mountID
		zLog.Info("Player rode mount",
			zap.Uint64("player_id", uint64(pm.playerID)),
			zap.Uint64("mount_id", uint64(mountID)),
			zap.String("mount_name", m.Name))
		return true
	}

	return false
}

// Dismount 下马
func (pm *PlayerMount) Dismount() bool {
	if pm.activeMountID == 0 {
		return false
	}

	m := pm.mounts[pm.activeMountID]
	if m == nil {
		pm.activeMountID = 0
		return false
	}

	if m.Dismount() {
		zLog.Info("Player dismounted",
			zap.Uint64("player_id", uint64(pm.playerID)),
			zap.Uint64("mount_id", uint64(pm.activeMountID)),
			zap.String("mount_name", m.Name))
		pm.activeMountID = 0
		return true
	}

	return false
}

// GetMountCount 获取坐骑数量
func (pm *PlayerMount) GetMountCount() int {
	return len(pm.mounts)
}

// CanAddMount 是否可以添加更多坐骑
func (pm *PlayerMount) CanAddMount() bool {
	return len(pm.mounts) < pm.maxMountCount
}

// GetMoveSpeed 获取当前移动速度
func (pm *PlayerMount) GetMoveSpeed() int {
	if pm.activeMountID == 0 {
		return 100 // 步行速度100%
	}
	m := pm.mounts[pm.activeMountID]
	if m == nil {
		return 100
	}
	return m.GetSpeed()
}
