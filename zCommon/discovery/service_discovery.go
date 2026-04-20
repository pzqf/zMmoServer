package discovery

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zMap"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

const (
	CacheTTL          = 30
	DiscoveryInterval = 5
	MaxRetries        = 3
	RetryIntervalSec  = 2
)

type cacheEntry struct {
	services  []*ServerInfo
	timestamp time.Time
}

type ServiceDiscovery struct {
	etcdClient    *clientv3.Client
	servicePrefix string
	leaseID       uint64 // 使用atomic存储leaseID
	keepAliveCh   <-chan *clientv3.LeaseKeepAliveResponse
	cache         *zMap.TypedMap[string, *cacheEntry]
	cacheTTL      time.Duration
	leaseMu       sync.Mutex
	etcdAvailable bool
	lastEtcdError time.Time
	failureCount  int
}

type EtcdConfig struct {
	Endpoints      string `ini:"Endpoints"`
	Username       string `ini:"Username"`
	Password       string `ini:"Password"`
	CACertPath     string `ini:"CACertPath"`
	ClientCertPath string `ini:"ClientCertPath"`
	ClientKeyPath  string `ini:"ClientKeyPath"`
}

func NewServiceDiscovery(endpoints []string) (*ServiceDiscovery, error) {
	return NewServiceDiscoveryWithConfig(endpoints, &EtcdConfig{
		CACertPath:     "../resources/etcd/ca.crt",
		ClientCertPath: "../resources/etcd/server.crt",
		ClientKeyPath:  "../resources/etcd/server.key",
	})
}

func NewServiceDiscoveryWithConfig(endpoints []string, cfg *EtcdConfig) (*ServiceDiscovery, error) {
	zLog.Info("Creating ServiceDiscovery", zap.Strings("endpoints", endpoints))

	tlsConfig, err := CreateTLSConfig(cfg)
	if err != nil {
		return nil, err
	}

	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		TLS:         tlsConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("create etcd client failed: %w", err)
	}

	return &ServiceDiscovery{
		etcdClient:    etcdClient,
		servicePrefix: "/services/",
		leaseID:       0,
		cache:         zMap.NewTypedMap[string, *cacheEntry](),
		cacheTTL:      CacheTTL * time.Second,
		etcdAvailable: true,
		lastEtcdError: time.Time{},
		failureCount:  0,
	}, nil
}

func (sd *ServiceDiscovery) Register(info *ServerInfo) error {
	// 更新最后心跳时间
	info.LastHeartbeat = time.Now().Unix()

	if info.ServiceType == "" {
		return fmt.Errorf("service type is required")
	}

	serviceKey := fmt.Sprintf("%s%s/%s/%s", sd.servicePrefix, info.ServiceType, info.GroupID, info.ID)

	serviceData, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal service info failed: %w", err)
	}

	// 使用带重试机制的操作
	err = withRetry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := sd.ensureLeaseAlive(ctx, 30*time.Second); err != nil {
			return err
		}

		_, err = sd.etcdClient.Put(ctx, serviceKey, string(serviceData), clientv3.WithLease(clientv3.LeaseID(atomic.LoadUint64(&sd.leaseID))))
		if err != nil {
			return fmt.Errorf("put service info failed: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	zLog.Debug("Service registered to etcd",
		zap.String("key", serviceKey),
		zap.String("id", info.ID),
		zap.String("service_type", info.ServiceType),
		zap.String("group_id", info.GroupID),
		zap.String("status", string(info.Status)),
		zap.Float64("load", info.Load),
		zap.Int("players", info.Players))

	return nil
}

func CreateTLSConfig(cfg *EtcdConfig) (*tls.Config, error) {
	if cfg == nil {
		return nil, nil
	}

	if strings.HasPrefix(cfg.Endpoints, "http://") {
		return nil, nil
	}

	if cfg.CACertPath == "" && cfg.ClientCertPath == "" && cfg.ClientKeyPath == "" {
		return nil, nil
	}

	caCert, err := ioutil.ReadFile(cfg.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("read CA certificate failed: %w", err)
	}

	clientCert, err := ioutil.ReadFile(cfg.ClientCertPath)
	if err != nil {
		return nil, fmt.Errorf("read client certificate failed: %w", err)
	}

	clientKey, err := ioutil.ReadFile(cfg.ClientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read client key failed: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("add CA certificate to pool failed")
	}

	cert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, fmt.Errorf("create client certificate failed: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}, nil
}

func (sd *ServiceDiscovery) Unregister(serviceType string, groupID string, serverID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if serviceType == "" {
		return fmt.Errorf("service type is required")
	}

	serviceKey := fmt.Sprintf("%s%s/%s/%s", sd.servicePrefix, serviceType, groupID, serverID)

	_, err := sd.etcdClient.Delete(ctx, serviceKey)
	if err != nil {
		return fmt.Errorf("delete service key failed: %w", err)
	}

	if atomic.LoadUint64(&sd.leaseID) != 0 {
		_, err := sd.etcdClient.Revoke(ctx, clientv3.LeaseID(atomic.LoadUint64(&sd.leaseID)))
		if err != nil {
			return fmt.Errorf("revoke lease failed: %w", err)
		}
		atomic.StoreUint64(&sd.leaseID, 0)
		sd.leaseMu.Lock()
		sd.keepAliveCh = nil
		sd.leaseMu.Unlock()
	}

	zLog.Info("Service unregistered from etcd",
		zap.String("key", serviceKey),
		zap.String("service_type", serviceType),
		zap.String("group_id", groupID),
		zap.String("server_id", serverID))
	return nil
}

func (sd *ServiceDiscovery) GetService(serviceType string, groupID string, serverID string) (*ServerInfo, error) {
	if serviceType == "" {
		return nil, fmt.Errorf("service type is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serviceKey := fmt.Sprintf("%s%s/%s/%s", sd.servicePrefix, serviceType, groupID, serverID)

	resp, err := sd.etcdClient.Get(ctx, serviceKey)
	if err != nil {
		return nil, fmt.Errorf("get service failed: %w", err)
	}

	if resp.Count == 0 {
		return nil, fmt.Errorf("service not found: %s", serverID)
	}

	var info ServerInfo
	if err := json.Unmarshal(resp.Kvs[0].Value, &info); err != nil {
		return nil, fmt.Errorf("unmarshal service info failed: %w", err)
	}

	return &info, nil
}

func (sd *ServiceDiscovery) GetAllServices() ([]*ServerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := sd.etcdClient.Get(ctx, sd.servicePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("get all services failed: %w", err)
	}

	var services []*ServerInfo
	for _, kv := range resp.Kvs {
		var info ServerInfo
		if err := json.Unmarshal(kv.Value, &info); err != nil {
			continue
		}
		services = append(services, &info)
	}

	return services, nil
}

func (sd *ServiceDiscovery) Discover(serviceType string, groupID string) ([]*ServerInfo, error) {
	cacheKey := fmt.Sprintf("%s:%s", serviceType, groupID)

	entry, exists := sd.cache.Load(cacheKey)

	if exists {
		if !sd.etcdAvailable {
			zLog.Warn("etcd is unavailable, using cached services",
				zap.String("service_type", serviceType),
				zap.String("group_id", groupID))
			return entry.services, nil
		}

		if time.Since(entry.timestamp) < sd.cacheTTL {
			return entry.services, nil
		}
	}

	var resp *clientv3.GetResponse
	var err error

	err = withRetry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		prefix := fmt.Sprintf("%s%s/", sd.servicePrefix, serviceType)
		var innerErr error
		resp, innerErr = sd.etcdClient.Get(ctx, prefix, clientv3.WithPrefix())
		if innerErr != nil {
			return fmt.Errorf("discover services failed: %w", innerErr)
		}
		return nil
	})

	if err != nil {
		sd.etcdAvailable = false
		sd.lastEtcdError = time.Now()
		sd.failureCount++
		zLog.Error("etcd discovery failed, marking etcd as unavailable",
			zap.Error(err),
			zap.Int("failure_count", sd.failureCount))

		if exists {
			zLog.Warn("etcd discovery failed, using cached services", zap.Error(err))
			return entry.services, nil
		}
		return nil, err
	}

	if !sd.etcdAvailable {
		sd.etcdAvailable = true
		sd.failureCount = 0
		zLog.Info("etcd recovered and is now available")
	}

	var filteredServices []*ServerInfo
	currentTime := time.Now().Unix()

	for _, kv := range resp.Kvs {
		var info ServerInfo
		if err := json.Unmarshal(kv.Value, &info); err != nil {
			continue
		}

		if groupID != "" && info.GroupID != groupID {
			continue
		}

		if info.LastHeartbeat > 0 && currentTime-info.LastHeartbeat > 60 {
			zLog.Warn("Skipping inactive service",
				zap.String("service_id", info.ID),
				zap.Int64("last_heartbeat", info.LastHeartbeat),
				zap.Int64("current_time", currentTime))
			continue
		}

		filteredServices = append(filteredServices, &info)
	}

	sd.cache.Store(cacheKey, &cacheEntry{
		services:  filteredServices,
		timestamp: time.Now(),
	})

	return filteredServices, nil
}

// IsEtcdAvailable 检查etcd是否可用
func (sd *ServiceDiscovery) IsEtcdAvailable() bool {
	return sd.etcdAvailable
}

// GetEtcdStatus 获取etcd状态
func (sd *ServiceDiscovery) GetEtcdStatus() (bool, time.Time, int) {
	return sd.etcdAvailable, sd.lastEtcdError, sd.failureCount
}

func (sd *ServiceDiscovery) Watch(serviceType string, groupID string) (<-chan *ServerEvent, error) {
	watchChan := make(chan *ServerEvent, 100)

	go func() {
		prefix := fmt.Sprintf("%s%s/", sd.servicePrefix, serviceType)
		watcher := sd.etcdClient.Watch(context.Background(), prefix, clientv3.WithPrefix())
		for watchResp := range watcher {
			if watchResp.Err() != nil {
				zLog.Error("Watch error", zap.Error(watchResp.Err()))
				continue
			}

			for _, event := range watchResp.Events {
				var info ServerInfo
				if err := json.Unmarshal(event.Kv.Value, &info); err != nil {
					continue
				}

				// 根据groupID过滤事件
				if groupID != "" && info.GroupID != groupID {
					continue
				}

				serviceEvent := &ServerEvent{
					ServerID:  info.ID,
					EventType: string(event.Type),
					Status:    info.Status,
					Timestamp: time.Now().Unix(),
					Data:      &info,
				}

				select {
				case watchChan <- serviceEvent:
				default:
					zLog.Warn("Watch channel full, dropping event", zap.String("server_id", info.ID))
				}
			}
		}
	}()

	return watchChan, nil
}

// withRetry 带重试机制的函数执行
func withRetry(fn func() error) error {
	var lastErr error
	for i := 0; i < MaxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err
			zLog.Warn("Operation failed, retrying...", zap.Error(err), zap.Int("attempt", i+1))
			time.Sleep(time.Duration(RetryIntervalSec) * time.Second)
			continue
		}
		return nil
	}
	return lastErr
}

func (sd *ServiceDiscovery) Close() error {
	if atomic.LoadUint64(&sd.leaseID) != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = sd.etcdClient.Revoke(ctx, clientv3.LeaseID(atomic.LoadUint64(&sd.leaseID)))
		cancel()
		atomic.StoreUint64(&sd.leaseID, 0)
		sd.leaseMu.Lock()
		sd.keepAliveCh = nil
		sd.leaseMu.Unlock()
	}

	if sd.etcdClient != nil {
		if err := sd.etcdClient.Close(); err != nil {
			return fmt.Errorf("close etcd client failed: %w", err)
		}
	}

	return nil
}

func (sd *ServiceDiscovery) ensureLeaseAlive(ctx context.Context, ttl time.Duration) error {
	sd.leaseMu.Lock()
	defer sd.leaseMu.Unlock()

	if atomic.LoadUint64(&sd.leaseID) != 0 && sd.keepAliveCh != nil {
		return nil
	}

	leaseResp, err := sd.etcdClient.Grant(ctx, int64(ttl.Seconds()))
	if err != nil {
		return fmt.Errorf("grant lease failed: %w", err)
	}
	atomic.StoreUint64(&sd.leaseID, uint64(leaseResp.ID))

	sd.keepAliveCh, err = sd.etcdClient.KeepAlive(ctx, clientv3.LeaseID(atomic.LoadUint64(&sd.leaseID)))
	if err != nil {
		return fmt.Errorf("keepalive failed: %w", err)
	}

	go func(ch <-chan *clientv3.LeaseKeepAliveResponse) {
		for range ch {
		}
		sd.leaseMu.Lock()
		if sd.keepAliveCh == ch {
			sd.keepAliveCh = nil
			atomic.StoreUint64(&sd.leaseID, 0)
		}
		sd.leaseMu.Unlock()
	}(sd.keepAliveCh)

	return nil
}
