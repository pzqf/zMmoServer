package health

import (
	zhealth "github.com/pzqf/zCommon/health"
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
