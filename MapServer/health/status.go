package health

import (
	zhealth "github.com/pzqf/zCommon/health"
)

type HealthStatus = zhealth.HealthStatus
type CheckResult = zhealth.CheckResult
type HealthReport = zhealth.HealthReport
type HealthCheck = zhealth.HealthCheck
type ComponentStatus = zhealth.ComponentStatus

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

type Checker struct {
	*zhealth.Checker
}

func NewChecker() *Checker {
	return &Checker{
		Checker: zhealth.NewChecker(),
	}
}
