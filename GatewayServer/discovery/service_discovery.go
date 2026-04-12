package discovery

import (
	"context"

	zdisc "github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
)

type ServiceDiscovery interface {
	Register() error
	Unregister() error
	DiscoverGameServer() ([]*zdisc.ServerInfo, error)
	WatchGameServer(ctx context.Context, callback func(*zdisc.ServerEvent)) error
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
		ServiceType: "gateway",
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

func (sd *ServiceDiscoveryImpl) DiscoverGameServer() ([]*zdisc.ServerInfo, error) {
	return sd.ServerServiceDiscovery.Discover("game")
}

func (sd *ServiceDiscoveryImpl) WatchGameServer(ctx context.Context, callback func(*zdisc.ServerEvent)) error {
	return sd.ServerServiceDiscovery.Watch(ctx, "game", callback)
}
