package metrics

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// MetricsConfig 监控指标配置
type MetricsConfig struct {
	Enabled       bool   `ini:"enabled"`        // 是否启用监控
	ListenAddress string `ini:"listen_address"` // 监控服务监听地址
}

// ServerMetrics 通用服务器监控指标
type ServerMetrics struct {
	config     *MetricsConfig
	metricsMgr *MetricsManager
	startTime  time.Time

	// HTTP 请求指标
	httpRequestsTotal   prometheus.Counter
	httpRequestDuration prometheus.Histogram

	// 系统指标
	uptimeSeconds prometheus.Gauge
}

// NewServerMetrics 创建通用服务器监控指标
func NewServerMetrics(cfg *MetricsConfig) *ServerMetrics {
	m := &ServerMetrics{
		config:     cfg,
		metricsMgr: NewMetricsManager(),
		startTime:  time.Now(),
	}

	m.registerCommonMetrics()
	return m
}

// Start 启动监控服务
func (m *ServerMetrics) Start() error {
	if !m.config.Enabled {
		zLog.Info("Metrics disabled")
		return nil
	}

	// 尝试监听端口，检查是否可用
	ln, err := net.Listen("tcp", m.config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen metrics on %s: %w", m.config.ListenAddress, err)
	}
	ln.Close()

	// 注册健康检查路由
	http.HandleFunc("/health", m.healthCheck)
	http.Handle("/metrics", promhttp.HandlerFor(m.metricsMgr.GetRegistry(), promhttp.HandlerOpts{}))

	// 启动HTTP服务
	go func() {
		zLog.Info("Metrics server started", zap.String("addr", m.config.ListenAddress))
		err := http.ListenAndServe(m.config.ListenAddress, nil)
		if err != nil {
			zLog.Error("Metrics server error", zap.Error(err))
		}
	}()

	// 启动系统指标收集
	go m.collectSystemMetrics()

	zLog.Info("Metrics service started successfully")
	return nil
}

// healthCheck 健康检查
func (m *ServerMetrics) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// registerCommonMetrics 注册通用指标
func (m *ServerMetrics) registerCommonMetrics() {
	// 注册HTTP请求指标
	m.httpRequestsTotal = m.metricsMgr.RegisterCounter(
		"http_requests_total",
		"Total number of HTTP requests",
		nil,
	)
	m.httpRequestDuration = m.metricsMgr.RegisterHistogram(
		"http_request_duration_seconds",
		"HTTP request duration in seconds",
		[]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		nil,
	)

	// 注册系统指标
	m.uptimeSeconds = m.metricsMgr.RegisterGauge(
		"uptime_seconds",
		"Server uptime in seconds",
		nil,
	)

	zLog.Info("Common server metrics registered successfully")
}

// collectSystemMetrics 收集系统指标
func (m *ServerMetrics) collectSystemMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 更新运行时间
		if m.uptimeSeconds != nil {
			m.uptimeSeconds.Set(time.Since(m.startTime).Seconds())
		}
	}
}

// RecordHTTPRequest 记录HTTP请求
func (m *ServerMetrics) RecordHTTPRequest(duration time.Duration) {
	if m.httpRequestsTotal != nil {
		m.httpRequestsTotal.Inc()
	}
	if m.httpRequestDuration != nil {
		m.httpRequestDuration.Observe(duration.Seconds())
	}
}

// RegisterCounter 注册计数器
func (m *ServerMetrics) RegisterCounter(name, help string, labels map[string]string) prometheus.Counter {
	return m.metricsMgr.RegisterCounter(name, help, labels)
}

// RegisterHistogram 注册直方图
func (m *ServerMetrics) RegisterHistogram(name, help string, buckets []float64, labels map[string]string) prometheus.Histogram {
	return m.metricsMgr.RegisterHistogram(name, help, buckets, labels)
}

// RegisterGauge 注册仪表
func (m *ServerMetrics) RegisterGauge(name, help string, labels map[string]string) prometheus.Gauge {
	return m.metricsMgr.RegisterGauge(name, help, labels)
}

// GetMetricsManager 获取指标管理器
func (m *ServerMetrics) GetMetricsManager() *MetricsManager {
	return m.metricsMgr
}
