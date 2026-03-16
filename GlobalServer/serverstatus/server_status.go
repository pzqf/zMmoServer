package serverstatus

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	redisv8 "github.com/go-redis/redis/v8"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/db/models"
	"github.com/pzqf/zMmoShared/redis"
	"go.uber.org/zap"
)

const (
	// ServerStatusKeyPrefix 服务器状态Key前缀
	ServerStatusKeyPrefix = "game_server:"
	// OnlineServersKey 在线服务器集合Key
	OnlineServersKey = "online_servers"
	// ServerGroupKeyPrefix 服务器分组Key前缀
	ServerGroupKeyPrefix = "server_group:"
	// DefaultHeartbeatTimeout 默认心跳超时时间（5分钟）
	DefaultHeartbeatTimeout = 5 * time.Minute
	// DefaultRedisExpire Redis数据默认过期时间
	DefaultRedisExpire = 10 * time.Minute
)

// ServerStatus 服务器动态状态
type ServerStatus struct {
	ServerID      int32     `json:"serverId"`
	Address       string    `json:"address"`
	Port          int32     `json:"port"`
	Status        int32     `json:"status"` // 1=在线, 0=维护/离线
	OnlineCount   int32     `json:"onlineCount"`
	Version       string    `json:"version"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	UpdateTime    time.Time `json:"updateTime"`
}

// ServerInfo 完整服务器信息（静态+动态）
type ServerInfo struct {
	// 静态数据（来自MySQL）
	ServerID       int32  `json:"serverId"`
	ServerName     string `json:"serverName"`
	ServerType     string `json:"serverType"`
	GroupID        int32  `json:"groupId"`
	MaxOnlineCount int32  `json:"maxOnlineCount"`
	Region         string `json:"region"`

	// 动态数据（来自Redis）
	Address       string    `json:"address"`
	Port          int32     `json:"port"`
	Status        int32     `json:"status"`
	OnlineCount   int32     `json:"onlineCount"`
	Version       string    `json:"version"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
}

// Manager 服务器状态管理器
type Manager struct {
	redisClient      *redis.Client
	staticServers    map[int32]*models.GameServer // 静态配置缓存
	staticServersMux sync.RWMutex
	heartbeatTimeout time.Duration
	cleanupTicker    *time.Ticker
	stopCleanup      chan struct{}
}

var (
	manager     *Manager
	managerOnce sync.Once
)

// GetManager 获取单例实例
func GetManager() *Manager {
	return manager
}

// InitManager 初始化服务器状态管理器
func InitManager(redisCfg redis.RedisConfig) error {
	var err error
	managerOnce.Do(func() {
		client, err := redis.NewClient(redisCfg)
		if err != nil {
			return
		}

		manager = &Manager{
			redisClient:      client,
			staticServers:    make(map[int32]*models.GameServer),
			heartbeatTimeout: DefaultHeartbeatTimeout,
			stopCleanup:      make(chan struct{}),
		}

		// 启动清理协程
		manager.startCleanupRoutine()

		zLog.Info("ServerStatusManager initialized",
			zap.String("redis", fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port)),
		)
	})
	return err
}

// Close 关闭管理器
func (m *Manager) Close() {
	if m.stopCleanup != nil {
		close(m.stopCleanup)
	}
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}
	if m.redisClient != nil {
		m.redisClient.Close()
	}
}

// LoadStaticServers 从MySQL加载静态服务器配置
func (m *Manager) LoadStaticServers(servers []*models.GameServer) {
	m.staticServersMux.Lock()
	defer m.staticServersMux.Unlock()

	m.staticServers = make(map[int32]*models.GameServer)
	for _, server := range servers {
		m.staticServers[server.ServerID] = server
	}

	zLog.Info("Static servers loaded",
		zap.Int("count", len(servers)),
	)
}

// ReloadStaticServers 重新加载静态服务器配置
func (m *Manager) ReloadStaticServers(servers []*models.GameServer) {
	m.LoadStaticServers(servers)
	zLog.Info("Static servers reloaded")
}

// UpdateServerStatus 更新服务器状态（心跳上报）
func (m *Manager) UpdateServerStatus(status *ServerStatus) error {
	key := m.getServerKey(status.ServerID)

	// 使用Hash存储服务器状态
	data := map[string]interface{}{
		"address":        status.Address,
		"port":           status.Port,
		"status":         status.Status,
		"online_count":   status.OnlineCount,
		"version":        status.Version,
		"last_heartbeat": status.LastHeartbeat.Unix(),
		"update_time":    time.Now().Unix(),
	}

	// 保存到Redis
	if err := m.redisClient.HSet(key, data); err != nil {
		return fmt.Errorf("failed to update server status: %w", err)
	}

	// 设置过期时间
	if err := m.redisClient.Expire(key, DefaultRedisExpire); err != nil {
		zLog.Warn("Failed to set expire", zap.Error(err))
	}

	// 添加到在线服务器集合（使用Sorted Set，按在线人数排序）
	member := &redisv8.Z{
		Score:  float64(status.OnlineCount),
		Member: status.ServerID,
	}
	if err := m.redisClient.ZAdd(OnlineServersKey, member); err != nil {
		zLog.Warn("Failed to add to online servers", zap.Error(err))
	}

	// 添加到分组集合
	staticServer := m.getStaticServer(status.ServerID)
	if staticServer != nil {
		groupKey := m.getGroupKey(staticServer.GroupID)
		if err := m.redisClient.SAdd(groupKey, status.ServerID); err != nil {
			zLog.Warn("Failed to add to group", zap.Error(err))
		}
	}

	return nil
}

// GetServerStatus 获取单个服务器状态
func (m *Manager) GetServerStatus(serverID int32) (*ServerStatus, error) {
	key := m.getServerKey(serverID)
	data, err := m.redisClient.HGetAll(key)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	return m.parseServerStatus(serverID, data), nil
}

// GetAllServerStatuses 获取所有服务器状态
func (m *Manager) GetAllServerStatuses() ([]*ServerStatus, error) {
	// 获取所有在线服务器ID
	serverIDs, err := m.redisClient.ZRange(OnlineServersKey, 0, -1)
	if err != nil {
		return nil, err
	}

	var statuses []*ServerStatus
	for _, idStr := range serverIDs {
		serverID, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			continue
		}

		status, err := m.GetServerStatus(int32(serverID))
		if err != nil {
			continue
		}

		if status != nil {
			statuses = append(statuses, status)
		}
	}

	return statuses, nil
}

// GetServerInfo 获取完整服务器信息（静态+动态）
func (m *Manager) GetServerInfo(serverID int32) *ServerInfo {
	// 获取静态数据
	static := m.getStaticServer(serverID)
	if static == nil {
		return nil
	}

	// 获取动态数据
	status, err := m.GetServerStatus(serverID)
	if err != nil {
		zLog.Warn("Failed to get server status", zap.Int32("serverId", serverID), zap.Error(err))
		return nil
	}

	// 检查心跳是否超时
	if status != nil && time.Since(status.LastHeartbeat) > m.heartbeatTimeout {
		// 心跳超时，标记为离线
		status.Status = 0
	}

	return m.mergeServerInfo(static, status)
}

// GetAllServerInfos 获取所有完整服务器信息
func (m *Manager) GetAllServerInfos() []*ServerInfo {
	m.staticServersMux.RLock()
	defer m.staticServersMux.RUnlock()

	var infos []*ServerInfo
	for serverID := range m.staticServers {
		info := m.GetServerInfo(serverID)
		if info != nil {
			infos = append(infos, info)
		}
	}

	return infos
}

// GetServerInfosByGroup 按分组获取服务器信息
func (m *Manager) GetServerInfosByGroup(groupID int32) []*ServerInfo {
	// 从Redis获取分组内的服务器ID
	groupKey := m.getGroupKey(groupID)
	serverIDs, err := m.redisClient.SMembers(groupKey)
	if err != nil {
		zLog.Warn("Failed to get group members", zap.Int32("groupId", groupID), zap.Error(err))
		return nil
	}

	var infos []*ServerInfo
	for _, idStr := range serverIDs {
		serverID, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			continue
		}

		info := m.GetServerInfo(int32(serverID))
		if info != nil {
			infos = append(infos, info)
		}
	}

	return infos
}

// GetOnlineServers 获取在线服务器列表（按在线人数排序）
func (m *Manager) GetOnlineServers() []*ServerInfo {
	// 从Sorted Set获取排序后的服务器ID
	serverIDs, err := m.redisClient.ZRange(OnlineServersKey, 0, -1)
	if err != nil {
		zLog.Warn("Failed to get online servers", zap.Error(err))
		return nil
	}

	var infos []*ServerInfo
	for _, idStr := range serverIDs {
		serverID, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			continue
		}

		info := m.GetServerInfo(int32(serverID))
		if info != nil && info.Status == 1 {
			infos = append(infos, info)
		}
	}

	return infos
}

// RemoveServerStatus 删除服务器状态（服务器下线）
func (m *Manager) RemoveServerStatus(serverID int32) error {
	key := m.getServerKey(serverID)

	// 删除服务器状态
	if err := m.redisClient.Del(key); err != nil {
		return err
	}

	// 从在线集合中移除
	if err := m.redisClient.ZRem(OnlineServersKey, serverID); err != nil {
		zLog.Warn("Failed to remove from online servers", zap.Error(err))
	}

	// 从分组中移除
	static := m.getStaticServer(serverID)
	if static != nil {
		groupKey := m.getGroupKey(static.GroupID)
		if err := m.redisClient.SRem(groupKey, serverID); err != nil {
			zLog.Warn("Failed to remove from group", zap.Error(err))
		}
	}

	zLog.Info("Server status removed", zap.Int32("serverId", serverID))
	return nil
}

// 内部方法

func (m *Manager) getServerKey(serverID int32) string {
	return fmt.Sprintf("%s%d", ServerStatusKeyPrefix, serverID)
}

func (m *Manager) getGroupKey(groupID int32) string {
	return fmt.Sprintf("%s%d", ServerGroupKeyPrefix, groupID)
}

func (m *Manager) getStaticServer(serverID int32) *models.GameServer {
	m.staticServersMux.RLock()
	defer m.staticServersMux.RUnlock()
	return m.staticServers[serverID]
}

func (m *Manager) parseServerStatus(serverID int32, data map[string]string) *ServerStatus {
	status := &ServerStatus{ServerID: serverID}

	if v, ok := data["address"]; ok {
		status.Address = v
	}
	if v, ok := data["port"]; ok {
		port, _ := strconv.ParseInt(v, 10, 32)
		status.Port = int32(port)
	}
	if v, ok := data["status"]; ok {
		s, _ := strconv.ParseInt(v, 10, 32)
		status.Status = int32(s)
	}
	if v, ok := data["online_count"]; ok {
		count, _ := strconv.ParseInt(v, 10, 32)
		status.OnlineCount = int32(count)
	}
	if v, ok := data["version"]; ok {
		status.Version = v
	}
	if v, ok := data["last_heartbeat"]; ok {
		timestamp, _ := strconv.ParseInt(v, 10, 64)
		status.LastHeartbeat = time.Unix(timestamp, 0)
	}
	if v, ok := data["update_time"]; ok {
		timestamp, _ := strconv.ParseInt(v, 10, 64)
		status.UpdateTime = time.Unix(timestamp, 0)
	}

	return status
}

func (m *Manager) mergeServerInfo(static *models.GameServer, status *ServerStatus) *ServerInfo {
	info := &ServerInfo{
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

func (m *Manager) startCleanupRoutine() {
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

func (m *Manager) cleanupExpiredServers() {
	statuses, err := m.GetAllServerStatuses()
	if err != nil {
		zLog.Warn("Failed to get server statuses for cleanup", zap.Error(err))
		return
	}

	now := time.Now()
	for _, status := range statuses {
		if now.Sub(status.LastHeartbeat) > m.heartbeatTimeout {
			// 心跳超时，移除
			if err := m.RemoveServerStatus(status.ServerID); err != nil {
				zLog.Warn("Failed to remove expired server",
					zap.Int32("serverId", status.ServerID),
					zap.Error(err),
				)
			}
		}
	}
}
