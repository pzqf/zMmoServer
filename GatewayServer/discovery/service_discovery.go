package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"go.uber.org/zap"
)

// ServiceDiscoveryImpl 服务发现管理器实现
type ServiceDiscoveryImpl struct {
	config      *config.Config
	discovery   *discovery.ServiceDiscovery
	serverID    string
	groupID     string
	serviceInfo *discovery.ServerInfo
}

// NewServiceDiscovery 创建服务发现管理器
func NewServiceDiscovery(cfg *config.Config) (ServiceDiscovery, error) {
	// 解析ServerID
	serverID, err := id.ParseServerIDInt(int32(cfg.Server.ServerID))
	if err != nil {
		return nil, fmt.Errorf("invalid gateway ServerID %d: %w", cfg.Server.ServerID, err)
	}

	serverIDStr := id.ServerIDString(serverID)
	groupID := id.GroupIDStringFromServerID(serverID)

	// 创建服务发现客户端
	etcdEndpoints := []string{cfg.Etcd.Endpoints}
	etcdConfig := &discovery.EtcdConfig{
		Endpoints:      cfg.Etcd.Endpoints,
		Username:       cfg.Etcd.Username,
		Password:       cfg.Etcd.Password,
		CACertPath:     cfg.Etcd.CACertPath,
		ClientCertPath: cfg.Etcd.ClientCertPath,
		ClientKeyPath:  cfg.Etcd.ClientKeyPath,
	}

	serviceDiscovery, err := discovery.NewServiceDiscoveryWithConfig(etcdEndpoints, etcdConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create service discovery: %w", err)
	}

	return &ServiceDiscoveryImpl{
		config:    cfg,
		discovery: serviceDiscovery,
		serverID:  serverIDStr,
		groupID:   groupID,
	}, nil
}

// Register 注册服务
func (sd *ServiceDiscoveryImpl) Register() error {
	serviceInfo := &discovery.ServerInfo{
		ID:            sd.serverID,
		ServiceType:   "gateway",
		GroupID:       sd.groupID,
		Status:        "initializing",
		Address:       sd.config.Server.ListenAddr,
		Port:          0,
		Load:          0,
		Players:       0,
		ReadyTime:     time.Now().Unix(),
		LastHeartbeat: time.Now().Unix(),
	}

	// 注册服务，添加重试机制
	maxRetries := 3
	var registerErr error
	for i := 0; i < maxRetries; i++ {
		if err := sd.discovery.Register(serviceInfo); err != nil {
			zLog.Warn("Failed to register service", zap.Error(err), zap.Int("retry", i+1))
			registerErr = err
			time.Sleep(time.Duration(i+1) * time.Second)
		} else {
			zLog.Info("Service registered successfully", zap.String("service_id", serviceInfo.ID))
			sd.serviceInfo = serviceInfo
			return nil
		}
	}

	return fmt.Errorf("failed to register service after %d retries: %w", maxRetries, registerErr)
}

// Unregister 注销服务
func (sd *ServiceDiscoveryImpl) Unregister() error {
	if err := sd.discovery.Unregister("gateway", sd.groupID, sd.serverID); err != nil {
		zLog.Warn("Failed to unregister service", zap.Error(err))
		return err
	}
	zLog.Info("Service unregistered successfully", zap.String("service_id", sd.serverID))
	return nil
}

// DiscoverGameServer 发现GameServer
func (sd *ServiceDiscoveryImpl) DiscoverGameServer() ([]*discovery.ServerInfo, error) {
	return sd.discovery.Discover("game", sd.groupID)
}

// WatchGameServer 监控GameServer状态变化
func (sd *ServiceDiscoveryImpl) WatchGameServer(ctx context.Context, callback func(*discovery.ServerEvent)) error {
	eventChan, err := sd.discovery.Watch("game", sd.groupID)
	if err != nil {
		return fmt.Errorf("failed to start watching GameServer status: %w", err)
	}

	zLog.Info("Started watching GameServer status changes")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventChan:
				if !ok {
					zLog.Warn("GameServer status watch channel closed")
					return
				}
				callback(event)
			}
		}
	}()

	return nil
}

// UpdateHeartbeat 更新心跳
func (sd *ServiceDiscoveryImpl) UpdateHeartbeat(status string, players int) error {
	if sd.serviceInfo == nil {
		return fmt.Errorf("service not registered")
	}

	updatedInfo := &discovery.ServerInfo{
		ID:            sd.serverID,
		ServiceType:   "gateway",
		GroupID:       sd.groupID,
		Status:        zServer.ServerState(status),
		Address:       sd.serviceInfo.Address,
		Port:          sd.serviceInfo.Port,
		Load:          0.0,
		Players:       players,
		ReadyTime:     sd.serviceInfo.ReadyTime,
		LastHeartbeat: time.Now().Unix(),
	}

	if err := sd.discovery.Register(updatedInfo); err != nil {
		zLog.Warn("Failed to send heartbeat", zap.Error(err))
		return err
	}

	sd.serviceInfo = updatedInfo
	return nil
}

// Close 关闭服务发现
func (sd *ServiceDiscoveryImpl) Close() error {
	return sd.discovery.Close()
}

// GetServerID 获取服务器ID
func (sd *ServiceDiscoveryImpl) GetServerID() string {
	return sd.serverID
}

// GetGroupID 获取组ID
func (sd *ServiceDiscoveryImpl) GetGroupID() string {
	return sd.groupID
}
