package metrics

import (
	"fmt"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	"github.com/pzqf/zMmoServer/GameServer/game/maps"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"go.uber.org/zap"
)

// Metrics 监控指标
type Metrics struct {
	config         *config.Config
	connManager    *connection.ConnectionManager
	sessionManager *session.SessionManager
	mapService     *maps.MapService
	metricsMgr     *metrics.MetricsManager
}

// NewMetrics 创建监控指标
func NewMetrics(cfg *config.Config, connManager *connection.ConnectionManager, sessionManager *session.SessionManager, mapService *maps.MapService) *Metrics {
	m := &Metrics{
		config:         cfg,
		connManager:    connManager,
		sessionManager: sessionManager,
		mapService:     mapService,
		metricsMgr:     metrics.NewMetricsManager(),
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
	m.metricsMgr.RegisterCounter("game_connections_total", "Total number of connections", nil)
	m.metricsMgr.RegisterGauge("game_connections_current", "Current number of connections", nil)

	// 注册会话指标
	m.metricsMgr.RegisterGauge("game_sessions_current", "Current number of sessions", nil)

	// 注册消息指标
	m.metricsMgr.RegisterCounter("game_messages_received_total", "Total number of messages received", nil)
	m.metricsMgr.RegisterCounter("game_messages_sent_total", "Total number of messages sent", nil)

	// 注册玩家指标
	m.metricsMgr.RegisterGauge("game_players_online", "Current number of online players", nil)

	// 注册系统指标
	m.metricsMgr.RegisterGauge("game_uptime_seconds", "Game server uptime in seconds", nil)
	m.metricsMgr.RegisterGauge("game_outbox_pending", "Current outbox pending message count", nil)
	m.metricsMgr.RegisterGauge("game_outbox_dead", "Current outbox dead-letter count", nil)
	m.metricsMgr.RegisterGauge("game_gateway_dedupe_hits_total", "Total gateway dedupe hit count", nil)

	zLog.Info("Game metrics registered successfully")
}

// IncrementConnections 增加连接计数
func (m *Metrics) IncrementConnections() {
	if counter, ok := m.metricsMgr.GetCounter("game_connections_total"); ok && counter != nil {
		counter.Inc()
	}
	m.UpdateCurrentConnections()
}

// UpdateCurrentConnections 更新当前连接数
func (m *Metrics) UpdateCurrentConnections() {
	if gauge, ok := m.metricsMgr.GetGauge("game_connections_current"); ok && gauge != nil {
		gauge.Set(float64(m.connManager.GetConnectionCount()))
	}
	if gauge, ok := m.metricsMgr.GetGauge("game_sessions_current"); ok && gauge != nil {
		gauge.Set(float64(m.sessionManager.GetSessionCount()))
	}
}

// IncrementMessagesReceived 增加消息接收计数
func (m *Metrics) IncrementMessagesReceived() {
	if counter, ok := m.metricsMgr.GetCounter("game_messages_received_total"); ok && counter != nil {
		counter.Inc()
	}
}

// IncrementMessagesSent 增加消息发送计数
func (m *Metrics) IncrementMessagesSent() {
	if counter, ok := m.metricsMgr.GetCounter("game_messages_sent_total"); ok && counter != nil {
		counter.Inc()
	}
}

// UpdateOnlinePlayers 更新在线玩家数
func (m *Metrics) UpdateOnlinePlayers(count int) {
	if gauge, ok := m.metricsMgr.GetGauge("game_players_online"); ok && gauge != nil {
		gauge.Set(float64(count))
	}
}

// UpdateConsistencyOutbox 更新跨服一致性Outbox指标
func (m *Metrics) UpdateConsistencyOutbox(pending, dead int) {
	if gauge, ok := m.metricsMgr.GetGauge("game_outbox_pending"); ok && gauge != nil {
		gauge.Set(float64(pending))
	}
	if gauge, ok := m.metricsMgr.GetGauge("game_outbox_dead"); ok && gauge != nil {
		gauge.Set(float64(dead))
	}
}

// UpdateGatewayDedupeHits 更新Gateway重复请求去重累计命中次数
func (m *Metrics) UpdateGatewayDedupeHits(total uint64) {
	if gauge, ok := m.metricsMgr.GetGauge("game_gateway_dedupe_hits_total"); ok && gauge != nil {
		gauge.Set(float64(total))
	}
}
