// serverregistry 服务器注册表
// 服务器注册表负责管理服务器列表的动态状态，包括心跳检测、缓存更新等。
package serverregistry

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pzqf/zCommon/db/models"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

const (
	// DefaultHeartbeatTimeout 默认心跳超时时间（分钟）
	DefaultHeartbeatTimeout = 5 * time.Minute
)

// ServerRuntimeStatus 服务器运行时状态
type ServerRuntimeStatus struct {
	ServerID      int32     `json:"serverId"`
	Port          int32     `json:"port"`
	Status        int32     `json:"status"` // 1=在线, 0=维护/离线
	OnlineCount   int32     `json:"onlineCount"`
	Version       string    `json:"version"`
	Address       string    `json:"address"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	UpdateTime    time.Time `json:"updateTime"`
}

// ServerFullInfo 完整服务器信息（静态+动态）
type ServerFullInfo struct {
	// 静态数据（来自MySQL）
	GroupID        int32  `json:"groupId"`
	ServerID       int32  `json:"serverId"`
	MaxOnlineCount int32  `json:"maxOnlineCount"`
	OnlineCount    int32  `json:"onlineCount"` // 动态数据（来自服务发现）为了内存对齐而放上来了
	ServerName     string `json:"serverName"`
	ServerType     string `json:"serverType"`
	Region         string `json:"region"`

	// 动态数据（来自服务发现）
	Port          int32     `json:"port"`
	Status        int32     `json:"status"`
	Address       string    `json:"address"`
	Version       string    `json:"version"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
}

// ServerListManager 服务器列表管理器
type ServerListManager struct {
	serviceDiscovery *discovery.ServiceDiscovery
	staticServers    *zMap.TypedMap[int32, *models.GameServer] // 静态配置缓存
	heartbeatTimeout time.Duration
	cleanupTicker    *time.Ticker
	stopCleanup      chan struct{}
	serviceCache     *zMap.TypedMap[string, *discovery.ServerInfo] // 服务发现缓存

	// 服务器列表缓存
	serverListCache     []*ServerFullInfo
	serverListCacheTime time.Time
	serverListCacheTTL  time.Duration // 缓存过期时间
}

var (
	serverListManager     *ServerListManager
	serverListManagerOnce sync.Once
)

// GetServerListManager 获取服务器列表管理器单例实例
func GetServerListManager() *ServerListManager {
	return serverListManager
}

// InitServerListManager 初始化服务器列表管理器
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
			serverListCacheTTL: 30 * time.Second, // 缓存30秒
		}

		// 启动清理协程
		serverListManager.startCleanupRoutine()

		// 启动服务发现监听
		serverListManager.startServiceDiscoveryWatch()

		zLog.Info("ServerListManager initialized with service discovery")
	})
	return err
}

// Close 关闭管理器
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

// LoadStaticServers 从MySQL加载静态服务器配置
func (m *ServerListManager) LoadStaticServers(servers []*models.GameServer) {
	m.staticServers.Clear()
	for _, server := range servers {
		m.staticServers.Store(server.ServerID, server)
	}

	zLog.Info("Static servers loaded",
		zap.Int("count", len(servers)),
	)
}

// ReloadStaticServers 重新加载静态服务器配置
func (m *ServerListManager) ReloadStaticServers(servers []*models.GameServer) {
	m.LoadStaticServers(servers)
	zLog.Info("Static servers reloaded")
}

// UpdateServerStatus 更新服务器状态（已废弃，使用服务发现）
func (m *ServerListManager) UpdateServerStatus(status *ServerRuntimeStatus) error {
	zLog.Warn("UpdateServerStatus is deprecated, using service discovery instead")
	return nil
}

// GetServerRuntimeStatus 获取单个服务器运行时状态
func (m *ServerListManager) GetServerRuntimeStatus(serverID int32) (*ServerRuntimeStatus, error) {
	// 从服务发现缓存中获取
	var found *ServerRuntimeStatus
	serverIDStr := strconv.FormatInt(int64(serverID), 10)
	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		// 服务ID现在是纯数字格式（如 "100101"）
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

// GetAllServerRuntimeStatuses 获取所有服务器运行时状态
func (m *ServerListManager) GetAllServerRuntimeStatuses() ([]*ServerRuntimeStatus, error) {
	// 从服务发现缓存中获取所有服务
	var statuses []*ServerRuntimeStatus
	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		status := m.convertServiceInfoToServerRuntimeStatus(service)
		statuses = append(statuses, status)
		return true
	})

	return statuses, nil
}

// GetServerFullInfo 获取完整服务器信息（静态+动态）
func (m *ServerListManager) GetServerFullInfo(serverID int32) *ServerFullInfo {
	// 获取静态数据
	static := m.getStaticServer(serverID)
	if static == nil {
		// 对于Gateway服务器，可能没有静态配置
		// 尝试从服务发现缓存中获取动态信息
		status, err := m.GetServerRuntimeStatus(serverID)
		if err != nil || status == nil {
			return nil
		}
		// 检查心跳是否超时
		if time.Since(status.LastHeartbeat) > m.heartbeatTimeout {
			// 心跳超时，标记为离线
			status.Status = 0
		}
		// 为Gateway服务器创建基本信息
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

	// 获取动态数据
	status, err := m.GetServerRuntimeStatus(serverID)
	if err != nil {
		zLog.Warn("Failed to get server runtime status", zap.Int32("serverId", serverID), zap.Error(err))
		// 没有动态信息，使用静态信息，状态为离线
		return &ServerFullInfo{
			ServerID:       static.ServerID,
			ServerName:     static.ServerName,
			ServerType:     static.ServerType,
			GroupID:        static.GroupID,
			MaxOnlineCount: static.MaxOnlineCount,
			Region:         static.Region,
			Address:        "",
			Port:           0,
			Status:         0, // 维护中
			OnlineCount:    0,
			Version:        "",
			LastHeartbeat:  time.Time{},
		}
	}

	// 检查心跳是否超时
	if status != nil && time.Since(status.LastHeartbeat) > m.heartbeatTimeout {
		// 心跳超时，标记为离线
		status.Status = 0
	}

	return m.mergeServerFullInfo(static, status)
}

// GetAllServerFullInfos 获取所有完整服务器信息
func (m *ServerListManager) GetAllServerFullInfos() []*ServerFullInfo {
	// 检查缓存是否有效
	now := time.Now()
	if len(m.serverListCache) > 0 && now.Sub(m.serverListCacheTime) < m.serverListCacheTTL {
		zLog.Debug("Using cached server list", zap.Int("count", len(m.serverListCache)))
		return m.serverListCache
	}

	// 缓存过期，重新生成
	var infos []*ServerFullInfo
	// 先添加所有静态服务器
	m.staticServers.Range(func(serverID int32, static *models.GameServer) bool {
		info := m.GetServerFullInfo(serverID)
		if info != nil {
			infos = append(infos, info)
		}
		return true
	})

	// 再添加所有Gateway服务器（从服务发现缓存中）
	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		// 只处理gateway类型的服务
		if service.ServiceType == "gateway" && service.Status == zServer.StateHealthy {
			// 服务ID现在是纯数字格式（如 "100101"）
			serverID, err := strconv.ParseInt(service.ID, 10, 32)
			if err == nil {
				// 检查该Gateway服务器是否已经添加过
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

	// 更新缓存
	m.serverListCache = infos
	m.serverListCacheTime = now
	zLog.Debug("Updated server list cache", zap.Int("count", len(infos)))

	return infos
}

// GetServerFullInfosByGroup 按分组获取服务器完整信息
func (m *ServerListManager) GetServerFullInfosByGroup(groupID int32) []*ServerFullInfo {
	// 从服务发现缓存中获取分组内的服务器
	var infos []*ServerFullInfo
	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		// 检查服务的GroupID是否匹配
		if service.GroupID == fmt.Sprintf("%d", groupID) {
			// 尝试从服务ID中解析server_id
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

// GetOnlineServers 获取在线服务器列表（按在线人数排序）
func (m *ServerListManager) GetOnlineServers() []*ServerFullInfo {
	// 使用缓存的服务器列表
	allServers := m.GetAllServerFullInfos()

	// 过滤出在线的gateway服务器
	var onlineServers []*ServerFullInfo
	for _, server := range allServers {
		if server.ServerType == "gateway" && server.Status == 1 {
			onlineServers = append(onlineServers, server)
		}
	}

	// 按在线人数排序
	// 这里可以实现一个简单的排序算法
	// 或者使用更复杂的排序方法

	return onlineServers
}

// RemoveServerStatus 删除服务器状态（服务器下线）
func (m *ServerListManager) RemoveServerStatus(serverID int32) error {
	// 从服务发现缓存中移除
	var foundKey string
	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		// 尝试从服务ID中解析server_id
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

// 内部方法

func (m *ServerListManager) getStaticServer(serverID int32) *models.GameServer {
	server, _ := m.staticServers.Load(serverID)
	return server
}

func (m *ServerListManager) mergeServerFullInfo(static *models.GameServer, status *ServerRuntimeStatus) *ServerFullInfo {
	info := &ServerFullInfo{
		// 静态数据
		ServerID:       static.ServerID,
		ServerName:     static.ServerName,
		ServerType:     static.ServerType,
		GroupID:        static.GroupID,
		MaxOnlineCount: static.MaxOnlineCount,
		Region:         static.Region,
	}

	// 动态数据
	if status != nil {
		info.Address = status.Address
		info.Port = status.Port
		info.Status = status.Status
		info.OnlineCount = status.OnlineCount
		info.Version = status.Version
		info.LastHeartbeat = status.LastHeartbeat
	}

	return info
}

// convertServiceInfoToServerRuntimeStatus 将ServerInfo转换为ServerRuntimeStatus
func (m *ServerListManager) convertServiceInfoToServerRuntimeStatus(service *discovery.ServerInfo) *ServerRuntimeStatus {
	// 服务ID现在是纯数字格式（如 "100101"）
	serverID := int32(0)
	if id, err := strconv.ParseInt(service.ID, 10, 32); err == nil {
		serverID = int32(id)
	}

	// 转换状态
	status := int32(0)
	if service.Status == zServer.StateHealthy {
		status = 1
	}

	// 转换心跳时间
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
		Version:       "1.0.0",       // 暂时使用固定版本
		LastHeartbeat: lastHeartbeat, // 使用服务中的心跳时间
		UpdateTime:    time.Now(),
	}
}

// startServiceDiscoveryWatch 启动服务发现监听
func (m *ServerListManager) startServiceDiscoveryWatch() {
	// 监听gateway服务
	go func() {
		eventChan, err := m.serviceDiscovery.Watch("gateway", "*")
		if err != nil {
			zLog.Error("Failed to watch gateway services", zap.Error(err))
			return
		}
		m.handleServiceEvents(eventChan, "gateway")
	}()

	/*这是game和map服务的监听代码, 没有必要，因为客户端只需要gateway服务地址即可
	// 监听game服务
	go func() {
		eventChan, err := m.serviceDiscovery.Watch("game", "*")
		if err != nil {
			zLog.Error("Failed to watch game services", zap.Error(err))
			return
		}
		m.handleServiceEvents(eventChan, "game")
	}()

	// 监听map服务
	go func() {
		eventChan, err := m.serviceDiscovery.Watch("map", "*")
		if err != nil {
			zLog.Error("Failed to watch map services", zap.Error(err))
			return
		}
		m.handleServiceEvents(eventChan, "map")
	}()
	*/

	zLog.Info("Service discovery watch started")
}

// handleServiceEvents 处理服务事件
func (m *ServerListManager) handleServiceEvents(eventChan <-chan *discovery.ServerEvent, serviceType string) {
	for event := range eventChan {
		// 生成缓存键
		key := fmt.Sprintf("%s:%s:%s", serviceType, event.GroupID, event.ServerID)
		switch event.EventType {
		case "PUT":
			// 更新或添加服务
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
			// 删除服务
			m.serviceCache.Delete(key)
			zLog.Info("Service deleted",
				zap.String("service_type", serviceType),
				zap.String("server_id", event.ServerID))
		}
	}
}

func (m *ServerListManager) startCleanupRoutine() {
	m.cleanupTicker = time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-m.cleanupTicker.C:
				m.cleanupExpiredServers()
			case <-m.stopCleanup:
				return
			}
		}
	}()
}

func (m *ServerListManager) cleanupExpiredServers() {
	// 清理过期的服务器状态
	now := time.Now().Unix()
	var expiredKeys []string
	m.serviceCache.Range(func(key string, service *discovery.ServerInfo) bool {
		// 检查心跳是否超时
		if service.LastHeartbeat > 0 && now-service.LastHeartbeat > int64(m.heartbeatTimeout.Seconds()) {
			expiredKeys = append(expiredKeys, key)
		}
		return true
	})

	// 删除过期的服务
	for _, key := range expiredKeys {
		m.serviceCache.Delete(key)
		zLog.Info("Expired server status removed from cache", zap.String("key", key))
	}
}

// RefreshServiceCache 刷新服务缓存
func (m *ServerListManager) RefreshServiceCache() error {
	if m.serviceDiscovery == nil {
		return fmt.Errorf("service discovery not initialized")
	}

	// 重新发现所有服务
	serviceTypes := []string{"gateway" /*, "game", "map"*/}
	for _, serviceType := range serviceTypes {
		services, err := m.serviceDiscovery.Discover(serviceType, "*")
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
