package discovery

import (
	"context"
	"time"
)

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name     string
	ID       string
	Address  string
	Port     int
	Metadata map[string]string
}

// ServiceDiscovery 服务发现接口
type ServiceDiscovery interface {
	// Register 注册服务
	Register(ctx context.Context, service *ServiceInfo) error

	// Unregister 注销服务
	Unregister(ctx context.Context, serviceID string) error

	// Discover 发现服务
	Discover(ctx context.Context, serviceName string) ([]*ServiceInfo, error)

	// Watch 监听服务变化
	Watch(ctx context.Context, serviceName string, callback func([]*ServiceInfo)) error

	// Close 关闭服务发现
	Close() error
}

// NewServiceDiscovery 创建服务发现实例
func NewServiceDiscovery(endpoints []string, username, password string) (ServiceDiscovery, error) {
	return NewEtcdServiceDiscovery(endpoints, username, password)
}
