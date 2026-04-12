package discovery

import (
	"context"

	zdisc "github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zMmoServer/GameServer/config"
)

type ServiceDiscovery interface {
	Register() error
	Unregister() error
	DiscoverGateway() ([]*zdisc.ServerInfo, error)
	Discover(serviceType string, groupID string) ([]*zdisc.ServerInfo, error)
	WatchGateway(ctx context.Context, callback func(*zdisc.ServerEvent)) error
	UpdateHeartbeat(status string, players int) error
	Close() error
	GetServerID() string
	GetGroupID() string
}

type ServiceDiscoveryImpl struct {
	*zdisc.ServerServiceDiscovery
}

func NewServiceDiscovery(cfg *config.Config) (ServiceDiscovery, error) {
	sd, err := zdisc.NewServerServiceDiscovery(&zdisc.ServerServiceDiscoveryConfig{
		ServiceType: "game",
		ServerID:    int32(cfg.Server.ServerID),
		ListenAddr:  cfg.Server.ListenAddr,
		Etcd:        &cfg.Etcd,
	})
	if err != nil {
		return nil, err
	}

	return &ServiceDiscoveryImpl{
		ServerServiceDiscovery: sd,
	}, nil
}

func (sd *ServiceDiscoveryImpl) DiscoverGateway() ([]*zdisc.ServerInfo, error) {
	return sd.ServerServiceDiscovery.Discover("gateway")
}

func (sd *ServiceDiscoveryImpl) Discover(serviceType string, groupID string) ([]*zdisc.ServerInfo, error) {
	return sd.ServerServiceDiscovery.DiscoverInGroup(serviceType, groupID)
}

func (sd *ServiceDiscoveryImpl) WatchGateway(ctx context.Context, callback func(*zdisc.ServerEvent)) error {
	return sd.ServerServiceDiscovery.Watch(ctx, "gateway", callback)
}
