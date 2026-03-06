package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.etcd.io/etcd/client/v3"
)

// ServiceInfo 服务信息
type ServiceInfo struct {
	Service string `json:"service"`
	ID      string `json:"id"`
	Group   string `json:"group"`
	Address string `json:"address"`
	Port    int    `json:"port"`
	Status  string `json:"status"`
	Load    float64 `json:"load"`
	Players int    `json:"players"`
}

// EtcdClient etcd客户端
type EtcdClient struct {
	client     *clientv3.Client
	leaseID    clientv3.LeaseID
	serviceKey string
}

// NewEtcdClient 创建新的etcd客户端
func NewEtcdClient(endpoints []string) (*EtcdClient, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %v", err)
	}

	return &EtcdClient{
		client: client,
	}, nil
}

// RegisterService 注册服务
func (e *EtcdClient) RegisterService(serviceInfo ServiceInfo) error {
	// 创建租约
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	leaseResp, err := e.client.Grant(ctx, 10)
	if err != nil {
		return fmt.Errorf("failed to grant lease: %v", err)
	}

	e.leaseID = leaseResp.ID

	// 生成服务键
	serviceInfo.Status = "healthy"
	serviceData, err := json.Marshal(serviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %v", err)
	}

	e.serviceKey = fmt.Sprintf("services/%s/%s", serviceInfo.Service, serviceInfo.ID)

	// 注册服务
	_, err = e.client.Put(ctx, e.serviceKey, string(serviceData), clientv3.WithLease(e.leaseID))
	if err != nil {
		return fmt.Errorf("failed to register service: %v", err)
	}

	// 启动租约续约
	go e.keepAlive()

	log.Printf("Service %s registered successfully", serviceInfo.ID)
	return nil
}

// keepAlive 保持租约
func (e *EtcdClient) keepAlive() {
	ch, err := e.client.KeepAlive(context.Background(), e.leaseID)
	if err != nil {
		log.Printf("Failed to start keepalive: %v", err)
		return
	}

	for range ch {
		// 租约续约成功
	}
}

// DeregisterService 注销服务
func (e *EtcdClient) DeregisterService() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := e.client.Delete(ctx, e.serviceKey)
	if err != nil {
		return fmt.Errorf("failed to deregister service: %v", err)
	}

	log.Printf("Service deregistered successfully")
	return nil
}

// DiscoverServices 发现服务
func (e *EtcdClient) DiscoverServices(serviceType string) ([]ServiceInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prefix := fmt.Sprintf("services/%s/", serviceType)
	resp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %v", err)
	}

	var services []ServiceInfo
	for _, kv := range resp.Kvs {
		var serviceInfo ServiceInfo
		if err := json.Unmarshal(kv.Value, &serviceInfo); err != nil {
			log.Printf("Failed to unmarshal service info: %v", err)
			continue
		}
		services = append(services, serviceInfo)
	}

	return services, nil
}

// UpdateServiceStatus 更新服务状态
func (e *EtcdClient) UpdateServiceStatus(status string, load float64, players int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 获取当前服务信息
	resp, err := e.client.Get(ctx, e.serviceKey)
	if err != nil {
		return fmt.Errorf("failed to get service info: %v", err)
	}

	if len(resp.Kvs) == 0 {
		return fmt.Errorf("service not found")
	}

	var serviceInfo ServiceInfo
	if err := json.Unmarshal(resp.Kvs[0].Value, &serviceInfo); err != nil {
		return fmt.Errorf("failed to unmarshal service info: %v", err)
	}

	// 更新状态
	serviceInfo.Status = status
	serviceInfo.Load = load
	serviceInfo.Players = players

	serviceData, err := json.Marshal(serviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %v", err)
	}

	// 重新注册服务
	_, err = e.client.Put(ctx, e.serviceKey, string(serviceData), clientv3.WithLease(e.leaseID))
	if err != nil {
		return fmt.Errorf("failed to update service status: %v", err)
	}

	return nil
}

// Close 关闭客户端
func (e *EtcdClient) Close() error {
	return e.client.Close()
}
