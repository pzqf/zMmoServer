package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	sharedMetrics "github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/client/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
)

type Metrics struct {
	*sharedMetrics.ServerMetrics

	config  *config.Config
	connMgr *connection.ClientConnMgr

	connectionsTotal   prometheus.Counter
	connectionsCurrent prometheus.Gauge
	messagesReceived   prometheus.Counter
	messagesSent       prometheus.Counter
	tokenVerifications prometheus.Counter
	tokenFailures      prometheus.Counter
}

func NewMetrics(cfg *config.Config, connMgr *connection.ClientConnMgr) *Metrics {
	commonCfg := &sharedMetrics.MetricsConfig{
		Enabled:       cfg.Metrics.Enabled,
		ListenAddress: cfg.Metrics.ListenAddress,
	}

	m := &Metrics{
		ServerMetrics: sharedMetrics.NewServerMetrics(commonCfg),
		config:        cfg,
		connMgr:       connMgr,
	}

	m.registerGatewayMetrics()
	return m
}

func (m *Metrics) registerGatewayMetrics() {
	mgr := m.GetMetricsManager()

	m.connectionsTotal = mgr.RegisterCounter("gateway_connections_total", "Total number of connections", nil)
	m.connectionsCurrent = mgr.RegisterGauge("gateway_connections_current", "Current number of connections", nil)
	m.messagesReceived = mgr.RegisterCounter("gateway_messages_received_total", "Total number of messages received", nil)
	m.messagesSent = mgr.RegisterCounter("gateway_messages_sent_total", "Total number of messages sent", nil)
	m.tokenVerifications = mgr.RegisterCounter("gateway_token_verifications_total", "Total number of token verifications", nil)
	m.tokenFailures = mgr.RegisterCounter("gateway_token_failures_total", "Total number of token verification failures", nil)

	zLog.Info("Gateway metrics registered successfully")
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
	if m.connectionsCurrent != nil && m.connMgr != nil {
		m.connectionsCurrent.Set(float64(m.connMgr.GetConnectionCount()))
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

func (m *Metrics) IncrementTokenVerifications() {
	if m.tokenVerifications != nil {
		m.tokenVerifications.Inc()
	}
}

func (m *Metrics) IncrementTokenFailures() {
	if m.tokenFailures != nil {
		m.tokenFailures.Inc()
	}
}
