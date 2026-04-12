package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zServer"
	"go.uber.org/zap"
)

type ServerServiceDiscoveryConfig struct {
	ServiceType string
	ServerID    int32
	ListenAddr  string
	Etcd        *EtcdConfig
}

type ServerServiceDiscovery struct {
	config      *ServerServiceDiscoveryConfig
	discovery   *ServiceDiscovery
	serverID    string
	groupID     string
	serviceInfo *ServerInfo
}

func NewServerServiceDiscovery(cfg *ServerServiceDiscoveryConfig) (*ServerServiceDiscovery, error) {
	serverID, err := id.ParseServerIDInt(cfg.ServerID)
	if err != nil {
		return nil, fmt.Errorf("invalid %s ServerID %d: %w", cfg.ServiceType, cfg.ServerID, err)
	}

	serverIDStr := id.ServerIDString(serverID)
	groupID := id.GroupIDStringFromServerID(serverID)

	etcdEndpoints := []string{cfg.Etcd.Endpoints}
	serviceDiscovery, err := NewServiceDiscoveryWithConfig(etcdEndpoints, cfg.Etcd)
	if err != nil {
		return nil, fmt.Errorf("failed to create service discovery: %w", err)
	}

	return &ServerServiceDiscovery{
		config:    cfg,
		discovery: serviceDiscovery,
		serverID:  serverIDStr,
		groupID:   groupID,
	}, nil
}

func (sd *ServerServiceDiscovery) Register() error {
	serviceInfo := &ServerInfo{
		ID:            sd.serverID,
		ServiceType:   sd.config.ServiceType,
		GroupID:       sd.groupID,
		Status:        zServer.ServerState("initializing"),
		Address:       sd.config.ListenAddr,
		Port:          0,
		Load:          0,
		Players:       0,
		ReadyTime:     time.Now().Unix(),
		LastHeartbeat: time.Now().Unix(),
	}

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

func (sd *ServerServiceDiscovery) Unregister() error {
	if err := sd.discovery.Unregister(sd.config.ServiceType, sd.groupID, sd.serverID); err != nil {
		zLog.Warn("Failed to unregister service", zap.Error(err))
		return err
	}
	zLog.Info("Service unregistered successfully", zap.String("service_id", sd.serverID))
	return nil
}

func (sd *ServerServiceDiscovery) Discover(serviceType string) ([]*ServerInfo, error) {
	return sd.discovery.Discover(serviceType, sd.groupID)
}

func (sd *ServerServiceDiscovery) DiscoverInGroup(serviceType, groupID string) ([]*ServerInfo, error) {
	return sd.discovery.Discover(serviceType, groupID)
}

func (sd *ServerServiceDiscovery) Watch(ctx context.Context, serviceType string, callback func(*ServerEvent)) error {
	eventChan, err := sd.discovery.Watch(serviceType, sd.groupID)
	if err != nil {
		return fmt.Errorf("failed to start watching %s status: %w", serviceType, err)
	}

	zLog.Info("Started watching service status changes", zap.String("service_type", serviceType))

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventChan:
				if !ok {
					zLog.Warn("Service status watch channel closed", zap.String("service_type", serviceType))
					return
				}
				callback(event)
			}
		}
	}()

	return nil
}

func (sd *ServerServiceDiscovery) UpdateHeartbeat(status string, players int) error {
	if sd.serviceInfo == nil {
		return fmt.Errorf("service not registered")
	}

	updatedInfo := &ServerInfo{
		ID:            sd.serverID,
		ServiceType:   sd.config.ServiceType,
		GroupID:       sd.groupID,
		Status:        zServer.ServerState(status),
		Address:       sd.serviceInfo.Address,
		Port:          sd.serviceInfo.Port,
		Load:          0.0,
		Players:       players,
		ReadyTime:     sd.serviceInfo.ReadyTime,
		LastHeartbeat: time.Now().Unix(),
		MapIDs:        sd.serviceInfo.MapIDs,
	}

	if err := sd.discovery.Register(updatedInfo); err != nil {
		zLog.Warn("Failed to send heartbeat", zap.Error(err))
		return err
	}

	sd.serviceInfo = updatedInfo
	return nil
}

func (sd *ServerServiceDiscovery) UpdateMapIDs(mapIDs []int32) {
	if sd.serviceInfo != nil {
		sd.serviceInfo.MapIDs = mapIDs
	}
}

func (sd *ServerServiceDiscovery) Close() error {
	return sd.discovery.Close()
}

func (sd *ServerServiceDiscovery) GetServerID() string {
	return sd.serverID
}

func (sd *ServerServiceDiscovery) GetGroupID() string {
	return sd.groupID
}

func (sd *ServerServiceDiscovery) GetServiceInfo() *ServerInfo {
	return sd.serviceInfo
}

func (sd *ServerServiceDiscovery) GetServiceDiscovery() *ServiceDiscovery {
	return sd.discovery
}
