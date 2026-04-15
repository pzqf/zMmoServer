package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	sharedMetrics "github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/config"
)

type Metrics struct {
	*sharedMetrics.ServerMetrics

	config *config.Config

	mapEnterTotal  prometheus.Counter
	mapLeaveTotal  prometheus.Counter
	mapMoveTotal   prometheus.Counter
	mapAttackTotal prometheus.Counter
	mapsCurrent    prometheus.Gauge
	connections    prometheus.Gauge
	players        prometheus.Gauge
	requestTotal   prometheus.Counter
	requestDuration prometheus.Histogram
	serviceRegister      prometheus.Counter
	serviceDiscovery     prometheus.Counter
	serviceHeartbeat     prometheus.Counter
}

func NewMetrics(cfg *config.Config) *Metrics {
	commonCfg := &sharedMetrics.MetricsConfig{
		Enabled:       cfg.Metrics.Enabled,
		ListenAddress: cfg.Metrics.ListenAddress,
	}

	m := &Metrics{
		ServerMetrics: sharedMetrics.NewServerMetrics(commonCfg),
		config:        cfg,
	}

	m.registerMapServerMetrics()
	return m
}

func (m *Metrics) registerMapServerMetrics() {
	mgr := m.GetMetricsManager()

	m.requestTotal = mgr.RegisterCounter("mapserver_requests_total", "Total number of requests", nil)
	m.connections = mgr.RegisterGauge("mapserver_connections", "Current number of connections", nil)
	m.players = mgr.RegisterGauge("mapserver_players", "Current number of players", nil)
	m.mapsCurrent = mgr.RegisterGauge("mapserver_maps", "Current number of maps", nil)
	m.requestDuration = mgr.RegisterHistogram("mapserver_request_duration_seconds", "Request duration in seconds", []float64{0.001, 0.01, 0.1, 1, 5, 10}, nil)
	m.serviceRegister = mgr.RegisterCounter("mapserver_service_discovery_register_total", "Total number of service discovery register attempts", nil)
	m.serviceDiscovery = mgr.RegisterCounter("mapserver_service_discovery_discover_total", "Total number of service discovery discover attempts", nil)
	m.mapEnterTotal = mgr.RegisterCounter("mapserver_map_enter_total", "Total number of map enter requests", nil)
	m.mapLeaveTotal = mgr.RegisterCounter("mapserver_map_leave_total", "Total number of map leave requests", nil)
	m.mapMoveTotal = mgr.RegisterCounter("mapserver_map_move_total", "Total number of map move requests", nil)
	m.mapAttackTotal = mgr.RegisterCounter("mapserver_map_attack_total", "Total number of map attack requests", nil)
	m.serviceHeartbeat = mgr.RegisterCounter("mapserver_service_heartbeat_total", "Total number of service heartbeats", nil)

	zLog.Info("Map metrics registered successfully")
}

func (m *Metrics) Start() error {
	return m.ServerMetrics.Start()
}

func (m *Metrics) IncrementRequests() {
	if m.requestTotal != nil {
		m.requestTotal.Inc()
	}
}

func (m *Metrics) UpdateConnections(count int) {
	if m.connections != nil {
		m.connections.Set(float64(count))
	}
}

func (m *Metrics) UpdatePlayers(count int) {
	if m.players != nil {
		m.players.Set(float64(count))
	}
}

func (m *Metrics) UpdateMaps(count int) {
	if m.mapsCurrent != nil {
		m.mapsCurrent.Set(float64(count))
	}
}

func (m *Metrics) RecordRequestDuration(duration float64) {
	if m.requestDuration != nil {
		m.requestDuration.Observe(duration)
	}
}

func (m *Metrics) IncrementMapEnter() {
	if m.mapEnterTotal != nil {
		m.mapEnterTotal.Inc()
	}
}

func (m *Metrics) IncrementMapLeave() {
	if m.mapLeaveTotal != nil {
		m.mapLeaveTotal.Inc()
	}
}

func (m *Metrics) IncrementMapMove() {
	if m.mapMoveTotal != nil {
		m.mapMoveTotal.Inc()
	}
}

func (m *Metrics) IncrementMapAttack() {
	if m.mapAttackTotal != nil {
		m.mapAttackTotal.Inc()
	}
}

func (m *Metrics) IncrementServiceRegister() {
	if m.serviceRegister != nil {
		m.serviceRegister.Inc()
	}
}

func (m *Metrics) IncrementServiceDiscovery() {
	if m.serviceDiscovery != nil {
		m.serviceDiscovery.Inc()
	}
}

func (m *Metrics) IncrementServiceHeartbeat() {
	if m.serviceHeartbeat != nil {
		m.serviceHeartbeat.Inc()
	}
}
