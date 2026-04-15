package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	sharedMetrics "github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	"github.com/pzqf/zMmoServer/GameServer/game/maps"
	"github.com/pzqf/zMmoServer/GameServer/session"
)

type Metrics struct {
	*sharedMetrics.ServerMetrics

	config         *config.Config
	connManager    *connection.ConnectionManager
	sessionManager *session.SessionManager
	mapService     *maps.MapService

	connectionsTotal     prometheus.Counter
	connectionsCurrent   prometheus.Gauge
	sessionsCurrent      prometheus.Gauge
	messagesReceived     prometheus.Counter
	messagesSent         prometheus.Counter
	messagesFailed       prometheus.Counter
	messageDuration      prometheus.Histogram
	playersOnline        prometheus.Gauge
	playersLoginTotal    prometheus.Counter
	playersLogoutTotal   prometheus.Counter
	playersCreateTotal   prometheus.Counter
	mapsLoaded           prometheus.Gauge
	mapEnterTotal        prometheus.Counter
	mapLeaveTotal        prometheus.Counter
	mapLoadDuration      prometheus.Histogram
	serviceRegister      prometheus.Counter
	serviceUnregister    prometheus.Counter
	serviceDiscoveryOK   prometheus.Counter
	serviceDiscoveryFail prometheus.Counter
	serviceHeartbeat     prometheus.Counter
	healthCheckOK        prometheus.Counter
	healthCheckFail      prometheus.Counter
	outboxPending        prometheus.Gauge
	outboxDead           prometheus.Gauge
	gatewayDedupeHits    prometheus.Gauge
}

func NewMetrics(cfg *config.Config, connManager *connection.ConnectionManager, sessionManager *session.SessionManager, mapService *maps.MapService) *Metrics {
	commonCfg := &sharedMetrics.MetricsConfig{
		Enabled:       cfg.Metrics.Enabled,
		ListenAddress: cfg.Metrics.ListenAddress,
	}

	m := &Metrics{
		ServerMetrics:  sharedMetrics.NewServerMetrics(commonCfg),
		config:         cfg,
		connManager:    connManager,
		sessionManager: sessionManager,
		mapService:     mapService,
	}

	m.registerGameServerMetrics()
	return m
}

func (m *Metrics) registerGameServerMetrics() {
	mgr := m.GetMetricsManager()

	m.connectionsTotal = mgr.RegisterCounter("game_connections_total", "Total number of connections", nil)
	m.connectionsCurrent = mgr.RegisterGauge("game_connections_current", "Current number of connections", nil)
	m.sessionsCurrent = mgr.RegisterGauge("game_sessions_current", "Current number of sessions", nil)
	m.messagesReceived = mgr.RegisterCounter("game_messages_received_total", "Total number of messages received", nil)
	m.messagesSent = mgr.RegisterCounter("game_messages_sent_total", "Total number of messages sent", nil)
	m.messagesFailed = mgr.RegisterCounter("game_messages_failed_total", "Total number of failed messages", nil)
	m.messageDuration = mgr.RegisterHistogram("game_message_processing_duration_seconds", "Message processing duration in seconds", []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0}, nil)
	m.playersOnline = mgr.RegisterGauge("game_players_online", "Current number of online players", nil)
	m.playersLoginTotal = mgr.RegisterCounter("game_players_login_total", "Total number of player logins", nil)
	m.playersLogoutTotal = mgr.RegisterCounter("game_players_logout_total", "Total number of player logouts", nil)
	m.playersCreateTotal = mgr.RegisterCounter("game_players_create_total", "Total number of player character creations", nil)
	m.mapsLoaded = mgr.RegisterGauge("game_maps_loaded", "Current number of loaded maps", nil)
	m.mapEnterTotal = mgr.RegisterCounter("game_map_enter_total", "Total number of map enter events", nil)
	m.mapLeaveTotal = mgr.RegisterCounter("game_map_leave_total", "Total number of map leave events", nil)
	m.mapLoadDuration = mgr.RegisterHistogram("game_map_load_duration_seconds", "Map load duration in seconds", []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0}, nil)
	m.serviceRegister = mgr.RegisterCounter("game_service_register_total", "Total number of service registrations", nil)
	m.serviceUnregister = mgr.RegisterCounter("game_service_unregister_total", "Total number of service unregistrations", nil)
	m.serviceDiscoveryOK = mgr.RegisterCounter("game_service_discovery_success_total", "Total number of successful service discoveries", nil)
	m.serviceDiscoveryFail = mgr.RegisterCounter("game_service_discovery_failure_total", "Total number of failed service discoveries", nil)
	m.serviceHeartbeat = mgr.RegisterCounter("game_service_heartbeat_total", "Total number of service heartbeats", nil)
	m.healthCheckOK = mgr.RegisterCounter("game_health_check_success_total", "Total number of successful health checks", nil)
	m.healthCheckFail = mgr.RegisterCounter("game_health_check_failure_total", "Total number of failed health checks", nil)
	m.outboxPending = mgr.RegisterGauge("game_outbox_pending", "Current outbox pending message count", nil)
	m.outboxDead = mgr.RegisterGauge("game_outbox_dead", "Current outbox dead-letter count", nil)
	m.gatewayDedupeHits = mgr.RegisterGauge("game_gateway_dedupe_hits_total", "Total gateway dedupe hit count", nil)

	zLog.Info("Game metrics registered successfully")
}

func (m *Metrics) Start() error {
	return m.ServerMetrics.Start()
}

func (m *Metrics) IncrementConnections() {
	if m.connectionsTotal != nil {
		m.connectionsTotal.Inc()
	}
	m.UpdateCurrentConnections()
}

func (m *Metrics) UpdateCurrentConnections() {
	if m.connectionsCurrent != nil && m.connManager != nil {
		m.connectionsCurrent.Set(float64(m.connManager.GetConnectionCount()))
	}
	if m.sessionsCurrent != nil && m.sessionManager != nil {
		m.sessionsCurrent.Set(float64(m.sessionManager.GetSessionCount()))
	}
}

func (m *Metrics) IncrementMessagesReceived() {
	if m.messagesReceived != nil {
		m.messagesReceived.Inc()
	}
}

func (m *Metrics) IncrementMessagesSent() {
	if m.messagesSent != nil {
		m.messagesSent.Inc()
	}
}

func (m *Metrics) IncrementMessagesFailed() {
	if m.messagesFailed != nil {
		m.messagesFailed.Inc()
	}
}

func (m *Metrics) RecordMessageProcessingDuration(duration float64) {
	if m.messageDuration != nil {
		m.messageDuration.Observe(duration)
	}
}

func (m *Metrics) UpdateOnlinePlayers(count int) {
	if m.playersOnline != nil {
		m.playersOnline.Set(float64(count))
	}
}

func (m *Metrics) IncrementPlayerLogin() {
	if m.playersLoginTotal != nil {
		m.playersLoginTotal.Inc()
	}
}

func (m *Metrics) IncrementPlayerLogout() {
	if m.playersLogoutTotal != nil {
		m.playersLogoutTotal.Inc()
	}
}

func (m *Metrics) IncrementPlayerCreate() {
	if m.playersCreateTotal != nil {
		m.playersCreateTotal.Inc()
	}
}

func (m *Metrics) UpdateMapsLoaded(count int) {
	if m.mapsLoaded != nil {
		m.mapsLoaded.Set(float64(count))
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

func (m *Metrics) RecordMapLoadDuration(duration float64) {
	if m.mapLoadDuration != nil {
		m.mapLoadDuration.Observe(duration)
	}
}

func (m *Metrics) IncrementServiceRegister() {
	if m.serviceRegister != nil {
		m.serviceRegister.Inc()
	}
}

func (m *Metrics) IncrementServiceUnregister() {
	if m.serviceUnregister != nil {
		m.serviceUnregister.Inc()
	}
}

func (m *Metrics) IncrementServiceDiscoverySuccess() {
	if m.serviceDiscoveryOK != nil {
		m.serviceDiscoveryOK.Inc()
	}
}

func (m *Metrics) IncrementServiceDiscoveryFailure() {
	if m.serviceDiscoveryFail != nil {
		m.serviceDiscoveryFail.Inc()
	}
}

func (m *Metrics) IncrementServiceHeartbeat() {
	if m.serviceHeartbeat != nil {
		m.serviceHeartbeat.Inc()
	}
}

func (m *Metrics) IncrementHealthCheckSuccess() {
	if m.healthCheckOK != nil {
		m.healthCheckOK.Inc()
	}
}

func (m *Metrics) IncrementHealthCheckFailure() {
	if m.healthCheckFail != nil {
		m.healthCheckFail.Inc()
	}
}

func (m *Metrics) UpdateConsistencyOutbox(pending, dead int) {
	if m.outboxPending != nil {
		m.outboxPending.Set(float64(pending))
	}
	if m.outboxDead != nil {
		m.outboxDead.Set(float64(dead))
	}
}

func (m *Metrics) UpdateGatewayDedupeHits(total uint64) {
	if m.gatewayDedupeHits != nil {
		m.gatewayDedupeHits.Set(float64(total))
	}
}

func (m *Metrics) RecordHTTPRequest(duration time.Duration) {
	m.ServerMetrics.RecordHTTPRequest(duration)
}
