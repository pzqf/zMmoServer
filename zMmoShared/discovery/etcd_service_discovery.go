package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdServiceDiscovery etcd服务发现实现
type EtcdServiceDiscovery struct {
	client *clientv3.Client
	lease  clientv3.LeaseID
	ctx    context.Context
	cancel context.CancelFunc
}

// NewEtcdServiceDiscovery 创建etcd服务发现实例
func NewEtcdServiceDiscovery(endpoints []string, username, password string) (*EtcdServiceDiscovery, error) {
	ctx, cancel := context.WithCancel(context.Background())

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		Username:    username,
		Password:    password,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &EtcdServiceDiscovery{
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Register 注册服务
func (s *EtcdServiceDiscovery) Register(ctx context.Context, service *ServiceInfo) error {
	// 创建租约
	leaseResp, err := s.client.Grant(ctx, 60)
	if err != nil {
		return fmt.Errorf("failed to grant lease: %w", err)
	}
	s.lease = leaseResp.ID

	// 服务路径
	servicePath := filepath.Join("services", service.Name, service.GroupID, service.ID)

	// 服务信息序列化
	serviceData, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %w", err)
	}

	// 注册服务
	_, err = s.client.Put(ctx, servicePath, string(serviceData), clientv3.WithLease(s.lease))
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// 保持租约
	go func() {
		ch, err := s.client.KeepAlive(s.ctx, s.lease)
		if err != nil {
			return
		}

		for range ch {
			// 保持租约
		}
	}()

	return nil
}

// Unregister 注销服务
func (s *EtcdServiceDiscovery) Unregister(ctx context.Context, serviceName, groupID, serviceID string) error {
	// 取消租约
	if s.lease != 0 {
		_, err := s.client.Revoke(ctx, s.lease)
		if err != nil {
			return fmt.Errorf("failed to revoke lease: %w", err)
		}
	}

	// 删除服务注册
	servicePath := filepath.Join("services", serviceName, groupID, serviceID)
	_, err := s.client.Delete(ctx, servicePath)
	if err != nil {
		return fmt.Errorf("failed to delete service registration: %w", err)
	}

	return nil
}

// Discover 发现服务
func (s *EtcdServiceDiscovery) Discover(ctx context.Context, serviceName, groupID string) ([]*ServiceInfo, error) {
	// 服务前缀
	var servicePrefix string
	if groupID != "" {
		servicePrefix = filepath.Join("services", serviceName, groupID)
	} else {
		servicePrefix = filepath.Join("services", serviceName)
	}

	// 列出所有服务
	resp, err := s.client.Get(ctx, servicePrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	// 解析服务信息
	services := make([]*ServiceInfo, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var service ServiceInfo
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			continue
		}
		services = append(services, &service)
	}

	return services, nil
}

// Watch 监听服务变化
func (s *EtcdServiceDiscovery) Watch(ctx context.Context, serviceName, groupID string, callback func([]*ServiceInfo)) error {
	// 服务前缀
	var servicePrefix string
	if groupID != "" {
		servicePrefix = filepath.Join("services", serviceName, groupID)
	} else {
		servicePrefix = filepath.Join("services", serviceName)
	}

	// 创建监听器
	watcher := s.client.Watch(ctx, servicePrefix, clientv3.WithPrefix())

	// 处理变更
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.ctx.Done():
				return
			case _, ok := <-watcher:
				if !ok {
					return
				}

				// 重新发现服务
				services, err := s.Discover(ctx, serviceName, groupID)
				if err == nil {
					callback(services)
				}
			}
		}
	}()

	return nil
}

// Close 关闭服务发现
func (s *EtcdServiceDiscovery) Close() error {
	s.cancel()
	return s.client.Close()
}
