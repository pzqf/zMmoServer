package security

import (
	"context"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"go.uber.org/zap"
)

const (
	// EtcdBanIPPrefix etcd中封禁IP的前缀
	EtcdBanIPPrefix = "/gateway/ban/ip/"
)

// IPManager IP管理器
type IPManager struct {
	config      *config.SecurityConfig
	etcdClient  *clientv3.Client
	bannedIPs   map[string]struct{}
	ipMutex     sync.RWMutex
	connections map[string]int
	connMutex   sync.RWMutex
}

// NewIPManager 创建IP管理器
func NewIPManager(cfg *config.SecurityConfig, etcdClient *clientv3.Client) *IPManager {
	return &IPManager{
		config:      cfg,
		etcdClient:  etcdClient,
		bannedIPs:   make(map[string]struct{}),
		connections: make(map[string]int),
	}
}

// CheckIPAllowed 检查IP是否被允许
func (sm *IPManager) CheckIPAllowed(ip string) bool {
	sm.ipMutex.RLock()
	_, exists := sm.bannedIPs[ip]
	sm.ipMutex.RUnlock()

	// 检查IP是否被封禁
	if exists {
		zLog.Warn("IP is banned", zap.String("ip", ip))
		return false
	}

	// 检查连接数限制
	sm.connMutex.RLock()
	connCount, exists := sm.connections[ip]
	sm.connMutex.RUnlock()

	if exists && connCount >= 5000 { // 暂时硬编码，后续从配置读取
		zLog.Warn("IP connection limit reached", zap.String("ip", ip), zap.Int("count", connCount))
		return false
	}

	return true
}

// BanIP 封禁IP
func (sm *IPManager) BanIP(ip string, duration int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 存储到etcd，使用过期时间
	key := EtcdBanIPPrefix + ip
	_, err := sm.etcdClient.Put(ctx, key, "", clientv3.WithLease(clientv3.LeaseID(uint64(duration))))
	if err != nil {
		zLog.Error("Failed to ban IP in etcd", zap.String("ip", ip), zap.Error(err))
		return err
	}

	zLog.Info("IP banned", zap.String("ip", ip), zap.Duration("duration", time.Duration(duration)*time.Second))
	return nil
}

// UnbanIP 解除IP封禁
func (sm *IPManager) UnbanIP(ip string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 从etcd中删除
	key := EtcdBanIPPrefix + ip
	_, err := sm.etcdClient.Delete(ctx, key)
	if err != nil {
		zLog.Error("Failed to unban IP in etcd", zap.String("ip", ip), zap.Error(err))
		return err
	}

	zLog.Info("IP unbanned", zap.String("ip", ip))
	return nil
}

// AddConnection 添加连接
func (sm *IPManager) AddConnection(ip string) {
	sm.connMutex.Lock()
	defer sm.connMutex.Unlock()

	sm.connections[ip]++
}

// RemoveConnection 移除连接
func (sm *IPManager) RemoveConnection(ip string) {
	sm.connMutex.Lock()
	defer sm.connMutex.Unlock()

	if count, exists := sm.connections[ip]; exists {
		if count > 1 {
			sm.connections[ip]--
		} else {
			delete(sm.connections, ip)
		}
	}
}

// LoadBannedIPs 从etcd加载初始封禁IP列表
func (sm *IPManager) LoadBannedIPs() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 列出所有封禁的IP
	resp, err := sm.etcdClient.Get(ctx, EtcdBanIPPrefix, clientv3.WithPrefix())
	if err != nil {
		zLog.Error("Failed to load banned IPs from etcd", zap.Error(err))
		return err
	}

	sm.ipMutex.Lock()
	for _, kv := range resp.Kvs {
		ip := string(kv.Key[len(EtcdBanIPPrefix):])
		sm.bannedIPs[ip] = struct{}{}
	}
	sm.ipMutex.Unlock()

	zLog.Info("Loaded banned IPs from etcd", zap.Int("count", len(resp.Kvs)))
	return nil
}

// StartWatchTask 启动etcd watch任务
func (sm *IPManager) StartWatchTask() {
	go func() {
		ctx := context.Background()
		watcher := sm.etcdClient.Watch(ctx, EtcdBanIPPrefix, clientv3.WithPrefix())

		for resp := range watcher {
			for _, event := range resp.Events {
				ip := string(event.Kv.Key[len(EtcdBanIPPrefix):])
				sm.ipMutex.Lock()
				switch event.Type {
				case mvccpb.PUT:
					sm.bannedIPs[ip] = struct{}{}
					zLog.Info("IP added to ban list", zap.String("ip", ip))
				case mvccpb.DELETE:
					delete(sm.bannedIPs, ip)
					zLog.Info("IP removed from ban list", zap.String("ip", ip))
				}
				sm.ipMutex.Unlock()
			}
		}
	}()

	zLog.Info("Started etcd watch task for banned IPs")
}

// StartCleanupTask 启动清理任务（保留此函数以保持兼容性）
func (sm *IPManager) StartCleanupTask() {
	// 加载初始封禁IP
	if err := sm.LoadBannedIPs(); err != nil {
		zLog.Warn("Failed to load banned IPs", zap.Error(err))
	}

	// 启动watch任务
	sm.StartWatchTask()
}
