package discovery

import (
	zdisc "github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zMmoServer/MapServer/config"
)

type ServiceDiscovery = zdisc.ServerServiceDiscovery

func NewServiceDiscovery(cfg *config.Config) (*ServiceDiscovery, error) {
	return zdisc.NewServerServiceDiscovery(&zdisc.ServerServiceDiscoveryConfig{
		ServiceType: "map",
		ServerID:    int32(cfg.Server.ServerID),
		ListenAddr:  cfg.Server.ListenAddr,
		Etcd:        &cfg.Etcd,
	})
}
