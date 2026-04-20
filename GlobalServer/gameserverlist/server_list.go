package gameserverlist

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/db/models"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

const (
	DefaultHeartbeatTimeout = 5 * time.Minute
)

type ServerRuntimeStatus struct {
	ServerID      int32     `json:"serverId"`
	Port          int32     `json:"port"`
	Status        int32     `json:"status"`
	OnlineCount   int32     `json:"onlineCount"`
	Version       string    `json:"version"`
	Address       string    `json:"address"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	UpdateTime    time.Time `json:"updateTime"`
}

type ServerFullInfo struct {
	GroupID        int32  `json:"groupId"`
	ServerID       int32  `json:"serverId"`
	MaxOnlineCount int32  `json:"maxOnlineCount"`
	OnlineCount    int32  `json:"onlineCount"`
	ServerName     string `json:"serverName"`
	ServerType     string `json:"serverType"`
	Region         string `json:"region"`

	Port          int32     `json:"port"`
	Status        int32     `json:"status"`
	Address       string    `json:"address"`
	Version       string    `json:"version"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
}

type ServerListManager struct {
	serviceDiscovery *discovery.ServiceDiscovery
	staticServers    *zMap.TypedMap[int32, *models.GameServer]
	heartbeatTimeout time.Duration
	cleanupTicker    *time.Ticker
	stopCleanup      chan struct{}
	serviceCache     *zMap.TypedMap[string, *discovery.ServerInfo]

	serverListCache     []*ServerFullInfo
	serverListCacheTime time.Time
	serverListCacheTTL  time.Duration
}

var (
	serverListManager     *ServerListManager
	serverListManagerOnce sync.Once
)

func GetServerListManager() *ServerListManager {
	return serverListManager
}

func InitServerListManager(serviceDiscovery *discovery.ServiceDiscovery) error {
	var err error
	serverListManagerOnce.Do(func() {
		serverListManager = &ServerListManager{
			serviceDiscovery:   serviceDiscovery,
			staticServers:      zMap.NewTypedMap[int32, *models.GameServer](),
			heartbeatTimeout:   DefaultHeartbeatTimeout,
			stopCleanup:        make(chan struct{}),
			serviceCache:       zMap.NewTypedMap[string, *discovery.ServerInfo](),
			serverListCache:    []*ServerFullInfo{},
			serverListCacheTTL: 30 * time.Second,
		}

		serverListManager.startCleanupRoutine()
		serverListManager.startServiceDiscoveryWatch()

		zLog.Info("ServerListManager initialized with service discovery")
	})
	return err
}

func (m *ServerListManager) Close() {
	if m.stopCleanup != nil {
		close(m.stopCleanup)
	}
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}
	if m.serviceDiscovery != nil {
		m.serviceDiscovery.Close()
	}
}

func (m *ServerListManager) LoadStaticServers(servers []*models.GameServer) {
	m.staticServers.Clear()
	for _, server := range servers {
		m.staticServers.Store(server.ServerID, server)
	}

	zLog.Info("Static servers loaded",
		zap.Int("count", len(servers)),
	)
}

func (m *ServerListManager) ReloadStaticServers(servers []*models.GameServer) {
	m.LoadStaticServers(servers)
	zLog.Info("Static servers reloaded")
}

func (m *ServerListManager) UpdateServerStatus(status *ServerRuntimeStatus) error {
	zLog.Warn("UpdateServerStatus is deprecated, using service discovery instead")
	return nil
}

func (m *ServerListManager) GetServerRuntimeStatus(serverID int32) (*ServerRuntimeStatus, error) {
	var found *ServerRuntimeStatus
	serverIDStr := id.ServerIDString(id.MustParseServerIDInt(serverID))
	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		if service.ID == serverIDStr {
			found = m.convertServiceInfoToServerRuntimeStatus(service)
			return false
		}
		return true
	})

	if found == nil {
		return nil, nil
	}

	return found, nil
}

func (m *ServerListManager) GetAllServerRuntimeStatuses() ([]*ServerRuntimeStatus, error) {
	var statuses []*ServerRuntimeStatus
	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		status := m.convertServiceInfoToServerRuntimeStatus(service)
		statuses = append(statuses, status)
		return true
	})

	return statuses, nil
}

func (m *ServerListManager) GetServerFullInfo(serverID int32) *ServerFullInfo {
	static := m.getStaticServer(serverID)
	if static == nil {
		status, err := m.GetServerRuntimeStatus(serverID)
		if err != nil || status == nil {
			return nil
		}
		if time.Since(status.LastHeartbeat) > m.heartbeatTimeout {
			status.Status = 0
		}
		return &ServerFullInfo{
			ServerID:       status.ServerID,
			ServerName:     fmt.Sprintf("Gateway-%d", status.ServerID),
			ServerType:     "gateway",
			GroupID:        0,
			MaxOnlineCount: 10000,
			Region:         "default",
			Address:        status.Address,
			Port:           status.Port,
			Status:         status.Status,
			OnlineCount:    status.OnlineCount,
			Version:        status.Version,
			LastHeartbeat:  status.LastHeartbeat,
		}
	}

	status, err := m.GetServerRuntimeStatus(serverID)
	if err != nil {
		zLog.Warn("Failed to get server runtime status", zap.Int32("serverId", serverID), zap.Error(err))
		return &ServerFullInfo{
			ServerID:       static.ServerID,
			ServerName:     static.ServerName,
			ServerType:     static.ServerType,
			GroupID:        static.GroupID,
			MaxOnlineCount: static.MaxOnlineCount,
			Region:         static.Region,
			Address:        "",
			Port:           0,
			Status:         0,
			OnlineCount:    0,
			Version:        "",
			LastHeartbeat:  time.Time{},
		}
	}

	if status != nil && time.Since(status.LastHeartbeat) > m.heartbeatTimeout {
		status.Status = 0
	}

	return m.mergeServerFullInfo(static, status)
}

func (m *ServerListManager) GetAllServerFullInfos() []*ServerFullInfo {
	now := time.Now()
	if len(m.serverListCache) > 0 && now.Sub(m.serverListCacheTime) < m.serverListCacheTTL {
		zLog.Debug("Using cached server list", zap.Int("count", len(m.serverListCache)))
		return m.serverListCache
	}

	var infos []*ServerFullInfo
	m.staticServers.Range(func(serverID int32, static *models.GameServer) bool {
		info := m.GetServerFullInfo(serverID)
		if info != nil {
			infos = append(infos, info)
		}
		return true
	})

	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		if service.ServiceType == "gateway" && service.Status == zServer.StateHealthy {
			serverID, err := strconv.ParseInt(service.ID, 10, 32)
			if err != nil {
				serverIDInt := id.ParseServerIDString(service.ID)
				if serverIDInt > 0 {
					serverID = int64(serverIDInt)
					err = nil
				}
			}
			if err == nil {
				alreadyAdded := false
				for _, info := range infos {
					if info.ServerID == int32(serverID) {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					info := m.GetServerFullInfo(int32(serverID))
					if info != nil {
						infos = append(infos, info)
					}
				}
			}
		}
		return true
	})

	m.serverListCache = infos
	m.serverListCacheTime = now
	zLog.Debug("Updated server list cache", zap.Int("count", len(infos)))

	return infos
}

func (m *ServerListManager) GetServerFullInfosByGroup(groupID int32) []*ServerFullInfo {
	var infos []*ServerFullInfo
	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		if service.GroupID == fmt.Sprintf("%d", groupID) {
			parts := strings.Split(service.ID, "-")
			if len(parts) == 2 {
				serverID, err := strconv.ParseInt(parts[1], 10, 32)
				if err == nil {
					info := m.GetServerFullInfo(int32(serverID))
					if info != nil {
						infos = append(infos, info)
					}
				}
			}
		}
		return true
	})

	return infos
}

func (m *ServerListManager) GetOnlineServers() []*ServerFullInfo {
	allServers := m.GetAllServerFullInfos()

	var onlineServers []*ServerFullInfo
	for _, server := range allServers {
		if server.ServerType == "gateway" && server.Status == 1 {
			onlineServers = append(onlineServers, server)
		}
	}

	return onlineServers
}

func (m *ServerListManager) RemoveServerStatus(serverID int32) error {
	var foundKey string
	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		parts := strings.Split(service.ID, "-")
		if len(parts) == 2 {
			if parsedID, err := strconv.ParseInt(parts[1], 10, 32); err == nil {
				if int32(parsedID) == serverID {
					foundKey = key
					return false
				}
			}
		}
		return true
	})

	if foundKey != "" {
		m.serviceCache.Delete(foundKey)
		zLog.Info("Server status removed from cache", zap.Int32("serverId", serverID))
	}

	return nil
}

func (m *ServerListManager) getStaticServer(serverID int32) *models.GameServer {
	server, _ := m.staticServers.Load(serverID)
	return server
}

func (m *ServerListManager) mergeServerFullInfo(static *models.GameServer, status *ServerRuntimeStatus) *ServerFullInfo {
	info := &ServerFullInfo{
		ServerID:       static.ServerID,
		ServerName:     static.ServerName,
		ServerType:     static.ServerType,
		GroupID:        static.GroupID,
		MaxOnlineCount: static.MaxOnlineCount,
		Region:         static.Region,
		Address:        static.Address,
		Port:           static.Port,
	}

	if status != nil {
		if status.Address != "" && status.Address != "0.0.0.0" {
			info.Address = status.Address
		}
		if status.Port > 0 {
			info.Port = status.Port
		}
		info.Status = status.Status
		info.OnlineCount = status.OnlineCount
		info.Version = status.Version
		info.LastHeartbeat = status.LastHeartbeat
	}

	return info
}

func (m *ServerListManager) convertServiceInfoToServerRuntimeStatus(service *discovery.ServerInfo) *ServerRuntimeStatus {
	serverID := int32(0)
	if id, err := strconv.ParseInt(service.ID, 10, 32); err == nil {
		serverID = int32(id)
	}

	status := int32(0)
	if service.Status == zServer.StateHealthy {
		status = 1
	}

	lastHeartbeat := time.Now()
	if service.LastHeartbeat > 0 {
		lastHeartbeat = time.Unix(service.LastHeartbeat, 0)
	}

	return &ServerRuntimeStatus{
		ServerID:      serverID,
		Address:       service.Address,
		Port:          int32(service.Port),
		Status:        status,
		OnlineCount:   int32(service.Players),
		Version:       "1.0.0",
		LastHeartbeat: lastHeartbeat,
		UpdateTime:    time.Now(),
	}
}

func (m *ServerListManager) startServiceDiscoveryWatch() {
	if err := m.RefreshServiceCache(); err != nil {
		zLog.Warn("Failed to refresh service cache on startup", zap.Error(err))
	}

	go func() {
		eventChan, err := m.serviceDiscovery.Watch("gateway", "")
		if err != nil {
			zLog.Error("Failed to watch gateway services", zap.Error(err))
			return
		}
		m.handleServiceEvents(eventChan, "gateway")
	}()

	zLog.Info("Service discovery watch started")
}

func (m *ServerListManager) handleServiceEvents(eventChan <-chan *discovery.ServerEvent, serviceType string) {
	for event := range eventChan {
		key := fmt.Sprintf("%s:%s:%s", serviceType, event.GroupID, event.ServerID)
		switch event.EventType {
		case "PUT":
			if event.Data != nil {
				if serverInfo, ok := event.Data.(*discovery.ServerInfo); ok {
					m.serviceCache.Store(key, serverInfo)
					zLog.Info("Service updated",
						zap.String("service_type", serviceType),
						zap.String("server_id", event.ServerID),
						zap.String("status", string(event.Status)))
				}
			}
		case "DELETE":
			m.serviceCache.Delete(key)
			zLog.Info("Service deleted",
				zap.String("service_type", serviceType),
				zap.String("server_id", event.ServerID))
		}
	}
}

func (m *ServerListManager) startCleanupRoutine() {
	m.cleanupTicker = time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-m.cleanupTicker.C:
				if err := m.RefreshServiceCache(); err != nil {
					zLog.Warn("Failed to refresh service cache", zap.Error(err))
				}
				m.cleanupExpiredServers()
			case <-m.stopCleanup:
				return
			}
		}
	}()
}

func (m *ServerListManager) cleanupExpiredServers() {
	now := time.Now().Unix()
	var expiredKeys []string
	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		if service.LastHeartbeat > 0 && now-service.LastHeartbeat > int64(m.heartbeatTimeout.Seconds()) {
			expiredKeys = append(expiredKeys, key)
		}
		return true
	})

	for _, key := range expiredKeys {
		m.serviceCache.Delete(key)
		zLog.Info("Expired server status removed from cache", zap.String("key", key))
	}
}

func (m *ServerListManager) RefreshServiceCache() error {
	if m.serviceDiscovery == nil {
		return fmt.Errorf("service discovery not initialized")
	}

	serviceTypes := []string{"gateway"}
	for _, serviceType := range serviceTypes {
		services, err := m.serviceDiscovery.Discover(serviceType, "")
		if err != nil {
			zLog.Warn("Failed to discover services", zap.String("service_type", serviceType), zap.Error(err))
			continue
		}

		for _, service := range services {
			key := fmt.Sprintf("%s:%s:%s", serviceType, service.GroupID, service.ID)
			m.serviceCache.Store(key, service)
		}

		zLog.Info("Service cache refreshed", zap.String("service_type", serviceType), zap.Int("count", len(services)))
	}

	return nil
}
