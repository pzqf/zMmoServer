package health

import (
	zhealth "github.com/pzqf/zCommon/health"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"github.com/pzqf/zMmoServer/GameServer/gateway"
)

type Checker = zhealth.Checker
type CheckResult = zhealth.CheckResult
type HealthStatus = zhealth.HealthStatus

func NewChecker() *Checker {
	return zhealth.NewChecker()
}

var StartChecker = zhealth.StartChecker

const (
	StatusHealthy   = zhealth.StatusHealthy
	StatusUnhealthy = zhealth.StatusUnhealthy
	StatusDegraded  = zhealth.StatusDegraded
	StatusStarting  = zhealth.StatusStarting
	StatusStopping  = zhealth.StatusStopping
	StatusUnknown   = zhealth.StatusUnknown
)

const (
	ComponentTCP       = zhealth.ComponentTCP
	ComponentMap       = zhealth.ComponentMap
	ComponentDiscovery = zhealth.ComponentDiscovery
	ComponentDatabase  = zhealth.ComponentDatabase
	ComponentConfig    = zhealth.ComponentConfig
	ComponentContainer = zhealth.ComponentContainer
	ComponentGateway   = zhealth.ComponentGateway
	ComponentSession   = zhealth.ComponentSession
	ComponentPlayer    = zhealth.ComponentPlayer
)

func NewGameServerChecker(
	connectionManager *connection.ConnectionManager,
	playerManager *player.PlayerManager,
	gatewayService *gateway.ConnectionService,
) *Checker {
	c := NewChecker()

	if connectionManager != nil {
		c.RegisterCheck("gateway_connection", func() zhealth.CheckResult {
			connected := connectionManager.IsGatewayConnected()
			status := zhealth.HealthStatusHealthy
			msg := "Gateway connected"
			if !connected {
				status = zhealth.HealthStatusUnhealthy
				msg = "Gateway disconnected"
			}
			return zhealth.CheckResult{
				Status:  status,
				Message: msg,
				Details: map[string]interface{}{
					"connection_count": connectionManager.GetConnectionCount(),
				},
			}
		})
	}

	if playerManager != nil {
		c.RegisterCheck("player_manager", func() zhealth.CheckResult {
			return zhealth.CheckResult{
				Status:  zhealth.HealthStatusHealthy,
				Message: "Player manager check",
				Details: map[string]interface{}{
					"player_count": playerManager.GetPlayerCount(),
				},
			}
		})
	}

	if gatewayService != nil {
		c.RegisterCheck("gateway_service", func() zhealth.CheckResult {
			return zhealth.CheckResult{
				Status:  zhealth.HealthStatusHealthy,
				Message: "Gateway service check",
			}
		})
	}

	return c
}
