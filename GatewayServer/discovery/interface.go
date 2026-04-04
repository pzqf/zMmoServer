package discovery

import (
	"context"

	"github.com/pzqf/zCommon/discovery"
)

// ServiceDiscovery 服务发现接口
type ServiceDiscovery interface {
	// Register 注册服务
	Register() error

	// Unregister 注销服务
	Unregister() error

	// DiscoverGameServer 发现GameServer
	DiscoverGameServer() ([]*discovery.ServerInfo, error)

	// WatchGameServer 监控GameServer状态变化
	WatchGameServer(ctx context.Context, callback func(*discovery.ServerEvent)) error

	// UpdateHeartbeat 更新心跳
	UpdateHeartbeat(status string, players int) error

	// Close 关闭服务发现
	Close() error

	// GetServerID 获取服务器ID
	GetServerID() string

	// GetGroupID 获取组ID
	GetGroupID() string
}
