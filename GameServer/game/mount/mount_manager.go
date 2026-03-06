package mount

import (
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// MountManager 坐骑管理器
type MountManager struct {
	mounts       map[id.MountIdType]*Mount
	playerMounts map[id.PlayerIdType][]id.MountIdType
	activeMounts map[id.PlayerIdType]id.MountIdType // 玩家当前激活的坐骑
	mutex        sync.RWMutex
}

// NewMountManager 创建坐骑管理器
func NewMountManager() *MountManager {
	return &MountManager{
		mounts:       make(map[id.MountIdType]*Mount),
		playerMounts: make(map[id.PlayerIdType][]id.MountIdType),
		activeMounts: make(map[id.PlayerIdType]id.MountIdType),
	}
}

// CreateMount 创建坐骑
func (mm *MountManager) CreateMount(playerID id.PlayerIdType, mountTypeID int, name string) *Mount {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mount := NewMount(playerID, mountTypeID, name)
	mm.mounts[mount.MountID] = mount
	mm.playerMounts[playerID] = append(mm.playerMounts[playerID], mount.MountID)

	zLog.Info("Mount created",
		zap.Uint64("mount_id", uint64(mount.MountID)),
		zap.Uint64("player_id", uint64(playerID)),
		zap.String("mount_name", name),
		zap.Int("mount_type", mountTypeID))

	return mount
}

// GetMount 获取坐骑
func (mm *MountManager) GetMount(mountID id.MountIdType) *Mount {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	return mm.mounts[mountID]
}

// GetPlayerMounts 获取玩家的所有坐骑
func (mm *MountManager) GetPlayerMounts(playerID id.PlayerIdType) []*Mount {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	mountIDs := mm.playerMounts[playerID]
	mounts := make([]*Mount, 0, len(mountIDs))

	for _, mountID := range mountIDs {
		if mount := mm.mounts[mountID]; mount != nil {
			mounts = append(mounts, mount)
		}
	}

	return mounts
}

// GetActiveMount 获取玩家当前激活的坐骑
func (mm *MountManager) GetActiveMount(playerID id.PlayerIdType) *Mount {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	mountID, exists := mm.activeMounts[playerID]
	if !exists {
		return nil
	}

	return mm.mounts[mountID]
}

// RideMount 骑乘坐骑
func (mm *MountManager) RideMount(mountID id.MountIdType) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mount := mm.mounts[mountID]
	if mount == nil {
		return false
	}

	playerID := mount.PlayerID

	// 先让玩家的其他坐骑下马
	for _, mID := range mm.playerMounts[playerID] {
		if m := mm.mounts[mID]; m != nil && m.IsRiding {
			m.Dismount()
		}
	}

	if mount.Ride() {
		mm.activeMounts[playerID] = mountID
		zLog.Info("Mount ridden",
			zap.Uint64("mount_id", uint64(mountID)),
			zap.Uint64("player_id", uint64(playerID)),
			zap.String("mount_name", mount.Name))
		return true
	}

	return false
}

// DismountMount 下马
func (mm *MountManager) DismountMount(playerID id.PlayerIdType) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mountID, exists := mm.activeMounts[playerID]
	if !exists {
		return false
	}

	mount := mm.mounts[mountID]
	if mount == nil {
		return false
	}

	if mount.Dismount() {
		delete(mm.activeMounts, playerID)
		zLog.Info("Mount dismounted",
			zap.Uint64("mount_id", uint64(mountID)),
			zap.Uint64("player_id", uint64(playerID)),
			zap.String("mount_name", mount.Name))
		return true
	}

	return false
}

// AddExpToMount 给坐骑添加经验值
func (mm *MountManager) AddExpToMount(mountID id.MountIdType, exp int64) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mount := mm.mounts[mountID]
	if mount == nil {
		return false
	}

	levelUp := mount.AddExp(exp)
	if levelUp {
		zLog.Info("Mount leveled up",
			zap.Uint64("mount_id", uint64(mountID)),
			zap.Uint64("player_id", uint64(mount.PlayerID)),
			zap.String("mount_name", mount.Name),
			zap.Int("new_level", mount.Level),
			zap.Int("new_speed", mount.Speed))
	}

	return true
}

// UpdateMounts 更新所有坐骑状态
func (mm *MountManager) UpdateMounts() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for _, mount := range mm.mounts {
		mount.Update()
	}
}

// DeleteMount 删除坐骑
func (mm *MountManager) DeleteMount(mountID id.MountIdType) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mount := mm.mounts[mountID]
	if mount == nil {
		return false
	}

	playerID := mount.PlayerID

	// 从玩家坐骑列表中移除
	mountIDs := mm.playerMounts[playerID]
	for i, id := range mountIDs {
		if id == mountID {
			mm.playerMounts[playerID] = append(mountIDs[:i], mountIDs[i+1:]...)
			break
		}
	}

	// 如果是当前激活的坐骑，清除激活状态
	if activeID, exists := mm.activeMounts[playerID]; exists && activeID == mountID {
		delete(mm.activeMounts, playerID)
	}

	// 删除坐骑
	delete(mm.mounts, mountID)
	zLog.Info("Mount deleted",
		zap.Uint64("mount_id", uint64(mountID)),
		zap.Uint64("player_id", uint64(playerID)),
		zap.String("mount_name", mount.Name))

	return true
}

// GetMountCount 获取坐骑总数
func (mm *MountManager) GetMountCount() int {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	return len(mm.mounts)
}

// GetPlayerMountCount 获取玩家坐骑数量
func (mm *MountManager) GetPlayerMountCount(playerID id.PlayerIdType) int {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	return len(mm.playerMounts[playerID])
}
