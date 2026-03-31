package health

import (
	"fmt"

	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zUtil/zMap"
)

type StatusManager struct {
	statuses *zMap.TypedMap[string, discovery.ServerStatus]
	watchers *zMap.TypedMap[string, []chan discovery.ServerStatus]
}

func NewStatusManager() *StatusManager {
	return &StatusManager{
		statuses: zMap.NewTypedMap[string, discovery.ServerStatus](),
		watchers: zMap.NewTypedMap[string, []chan discovery.ServerStatus](),
	}
}

func (sm *StatusManager) SetStatus(serverID string, status discovery.ServerStatus) error {
	sm.statuses.Store(serverID, status)

	sm.notifyWatchers(serverID, status)

	return nil
}

func (sm *StatusManager) GetStatus(serverID string) (discovery.ServerStatus, error) {
	status, exists := sm.statuses.Load(serverID)
	if !exists {
		return "", fmt.Errorf("server not found: %s", serverID)
	}

	return status, nil
}

func (sm *StatusManager) WatchStatus(serverID string) (<-chan discovery.ServerStatus, error) {
	watchChan := make(chan discovery.ServerStatus, 10)

	// 加载现有的watchers
	watchers, exists := sm.watchers.Load(serverID)
	if !exists {
		watchers = []chan discovery.ServerStatus{}
	}

	// 添加新的watcher
	watchers = append(watchers, watchChan)
	sm.watchers.Store(serverID, watchers)

	return watchChan, nil
}

func (sm *StatusManager) notifyWatchers(serverID string, status discovery.ServerStatus) {
	watchers, exists := sm.watchers.Load(serverID)
	if !exists {
		return
	}

	for _, watcher := range watchers {
		select {
		case watcher <- status:
		default:
		}
	}
}

func (sm *StatusManager) Stop() {
	// 关闭所有watcher channels
	sm.watchers.Range(func(serverID string, watchers []chan discovery.ServerStatus) bool {
		for _, watcher := range watchers {
			close(watcher)
		}
		return true
	})

	// 清空maps
	sm.statuses.Clear()
	sm.watchers.Clear()
}
