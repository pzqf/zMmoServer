package metrics

import (
	"fmt"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/connection"
	"github.com/pzqf/zMmoShared/metrics"
	"go.uber.org/zap"
)

// Metrics 监控指标
type Metrics struct {
	config      *config.Config
	connManager *connection.ConnectionManager
	metricsMgr  *metrics.MetricsManager
}

// NewMetrics 创建监控指标
func NewMetrics(cfg *config.Config, connManager *connection.ConnectionManager) *Metrics {
	m := &Metrics{
		config:      cfg,
		connManager: connManager,
		metricsMgr:  metrics.NewMetricsManager(),
	}

	m.registerMetrics()
	return m
}

// Start 启动监控服务
func (m *Metrics) Start() error {
	if !m.config.Metrics.Enabled {
		zLog.Info("Metrics disabled")
		return nil
	}

	// 尝试监听端口，检查是否可用
	ln, err := net.Listen("tcp", m.config.Metrics.MetricsAddr)
	if err != nil {
		return fmt.Errorf("failed to listen metrics on %s: %w", m.config.Metrics.MetricsAddr, err)
	}
	ln.Close()

	// 注册健康检查路由
	http.HandleFunc("/health", m.healthCheck)
	http.Handle("/metrics", promhttp.HandlerFor(m.metricsMgr.GetRegistry(), promhttp.HandlerOpts{}))

	// 启动HTTP服务
	go func() {
		zLog.Info("Metrics server started", zap.String("addr", m.config.Metrics.MetricsAddr))
		err := http.ListenAndServe(m.config.Metrics.MetricsAddr, nil)
		if err != nil {
			zLog.Error("Metrics server error", zap.Error(err))
		}
	}()

	zLog.Info("Metrics service started successfully")
	return nil
}

// healthCheck 健康检查
func (m *Metrics) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// registerMetrics 注册监控指标
func (m *Metrics) registerMetrics() {
	// 注册连接指标
	m.metricsMgr.RegisterCounter("gateway_connections_total", "Total number of connections", nil)
	m.metricsMgr.RegisterGauge("gateway_connections_current", "Current number of connections", nil)

	// 注册消息指标
	m.metricsMgr.RegisterCounter("gateway_messages_received_total", "Total number of messages received", nil)
	m.metricsMgr.RegisterCounter("gateway_messages_sent_total", "Total number of messages sent", nil)

	// 注册Token指标
	m.metricsMgr.RegisterCounter("gateway_token_verifications_total", "Total number of token verifications", nil)
	m.metricsMgr.RegisterCounter("gateway_token_failures_total", "Total number of token verification failures", nil)

	// 注册系统指标
	m.metricsMgr.RegisterGauge("gateway_uptime_seconds", "Gateway server uptime in seconds", nil)

	zLog.Info("Gateway metrics registered successfully")
}

// IncrementConnections 增加连接计数
func (m *Metrics) IncrementConnections() {
	if counter, ok := m.metricsMgr.GetCounter("gateway_connections_total"); ok && counter != nil {
		counter.Inc()
	}
	if gauge, ok := m.metricsMgr.GetGauge("gateway_connections_current"); ok && gauge != nil {
		gauge.Set(float64(m.connManager.GetConnectionCount()))
	}
}

// IncrementMessagesReceived 增加消息接收计数
func (m *Metrics) IncrementMessagesReceived() {
	if counter, ok := m.metricsMgr.GetCounter("gateway_messages_received_total"); ok && counter != nil {
		counter.Inc()
	}
}

// IncrementMessagesSent 增加消息发送计数
func (m *Metrics) IncrementMessagesSent() {
	if counter, ok := m.metricsMgr.GetCounter("gateway_messages_sent_total"); ok && counter != nil {
		counter.Inc()
	}
}

// IncrementTokenVerifications 增加Token验证计数
func (m *Metrics) IncrementTokenVerifications() {
	if counter, ok := m.metricsMgr.GetCounter("gateway_token_verifications_total"); ok && counter != nil {
		counter.Inc()
	}
}

// IncrementTokenFailures 增加Token验证失败计数
func (m *Metrics) IncrementTokenFailures() {
	if counter, ok := m.metricsMgr.GetCounter("gateway_token_failures_total"); ok && counter != nil {
		counter.Inc()
	}
}

// UpdateCurrentConnections 更新当前连接数
func (m *Metrics) UpdateCurrentConnections() {
	if gauge, ok := m.metricsMgr.GetGauge("gateway_connections_current"); ok && gauge != nil {
		gauge.Set(float64(m.connManager.GetConnectionCount()))
	}
}
