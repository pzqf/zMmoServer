package game

import (
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// ServiceStatus 服务状态枚举
type ServiceStatus int

const (
	// ServiceStatusUnknown 未知状态
	ServiceStatusUnknown ServiceStatus = iota
	// ServiceStatusStarting 启动中
	ServiceStatusStarting
	// ServiceStatusRunning 运行中
	ServiceStatusRunning
	// ServiceStatusMaintenance 维护中
	ServiceStatusMaintenance
	// ServiceStatusStopping 停止中
	ServiceStatusStopping
	// ServiceStatusStopped 已停止
	ServiceStatusStopped
)

// ServiceStatusManager 服务状态管理器
type ServiceStatusManager struct {
	status     ServiceStatus
	statusMu   sync.RWMutex
	statusChan chan ServiceStatus
}

// NewServiceStatusManager 创建服务状态管理器
func NewServiceStatusManager() *ServiceStatusManager {
	return &ServiceStatusManager{
		status:     ServiceStatusStarting,
		statusChan: make(chan ServiceStatus, 10),
	}
}

// GetStatus 获取服务状态
func (sm *ServiceStatusManager) GetStatus() ServiceStatus {
	sm.statusMu.RLock()
	defer sm.statusMu.RUnlock()
	return sm.status
}

// SetStatus 设置服务状态
func (sm *ServiceStatusManager) SetStatus(status ServiceStatus) {
	sm.statusMu.Lock()
	oldStatus := sm.status
	sm.status = status
	sm.statusMu.Unlock()

	if oldStatus != status {
		zLog.Info("Service status changed", zap.String("old_status", sm.statusToString(oldStatus)), zap.String("new_status", sm.statusToString(status)))
		// 通知状态变更
		sm.statusChan <- status
	}
}

// GetStatusChan 获取状态变更通道
func (sm *ServiceStatusManager) GetStatusChan() <-chan ServiceStatus {
	return sm.statusChan
}

// statusToString 将状态转换为字符串
func (sm *ServiceStatusManager) statusToString(status ServiceStatus) string {
	switch status {
	case ServiceStatusUnknown:
		return "Unknown"
	case ServiceStatusStarting:
		return "Starting"
	case ServiceStatusRunning:
		return "Running"
	case ServiceStatusMaintenance:
		return "Maintenance"
	case ServiceStatusStopping:
		return "Stopping"
	case ServiceStatusStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}
