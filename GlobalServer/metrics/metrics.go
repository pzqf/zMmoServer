package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	sharedMetrics "github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GlobalServer/config"
)

// Metrics GlobalServer 监控指标
type Metrics struct {
	*sharedMetrics.ServerMetrics

	// 账号相关指标
	accountRegistrationsTotal prometheus.Counter
	accountLoginsTotal        prometheus.Counter
	accountLoginFailuresTotal prometheus.Counter

	// 服务器列表指标
	serverListRequestsTotal prometheus.Counter
	gameServersCount        prometheus.Gauge

	// 数据库指标
	dbQueriesTotal  prometheus.Counter
	dbQueryDuration prometheus.Histogram

	// 系统指标
	activeAccounts prometheus.Gauge

	// 服务发现相关指标
	serviceRegisterTotal     prometheus.Counter
	serviceUnregisterTotal   prometheus.Counter
	serviceDiscoveryFailures prometheus.Counter
	serviceHeartbeatTotal    prometheus.Counter

	// HTTP 服务指标
	httpRequestsTotal      prometheus.Counter
	httpRequestDuration    prometheus.Histogram
	httpErrorRequestsTotal prometheus.Counter

	// 业务逻辑指标
	serverListResponseTime prometheus.Histogram
	accountOperationTime   prometheus.Histogram
}

// NewMetrics 创建 GlobalServer 监控指标
func NewMetrics(cfg *config.MetricsConfig) *Metrics {
	// 转换配置为通用 metrics 配置
	commonCfg := (*sharedMetrics.MetricsConfig)(cfg)

	// 创建通用服务器指标
	commonMetrics := sharedMetrics.NewServerMetrics(commonCfg)

	m := &Metrics{
		ServerMetrics: commonMetrics,
	}

	m.registerGlobalServerMetrics()
	return m
}

// registerGlobalServerMetrics 注册 GlobalServer 特定指标
func (m *Metrics) registerGlobalServerMetrics() {
	// 注册账号相关指标
	m.accountRegistrationsTotal = m.RegisterCounter(
		"global_account_registrations_total",
		"Total number of account registrations",
		nil,
	)
	m.accountLoginsTotal = m.RegisterCounter(
		"global_account_logins_total",
		"Total number of account logins",
		nil,
	)
	m.accountLoginFailuresTotal = m.RegisterCounter(
		"global_account_login_failures_total",
		"Total number of account login failures",
		nil,
	)

	// 注册服务器列表指标
	m.serverListRequestsTotal = m.RegisterCounter(
		"global_server_list_requests_total",
		"Total number of server list requests",
		nil,
	)
	m.gameServersCount = m.RegisterGauge(
		"global_game_servers_count",
		"Current number of game servers",
		nil,
	)

	// 注册数据库指标
	m.dbQueriesTotal = m.RegisterCounter(
		"global_db_queries_total",
		"Total number of database queries",
		nil,
	)
	m.dbQueryDuration = m.RegisterHistogram(
		"global_db_query_duration_seconds",
		"Database query duration in seconds",
		[]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		nil,
	)

	// 注册系统指标
	m.activeAccounts = m.RegisterGauge(
		"global_active_accounts",
		"Number of active accounts (logged in within last hour)",
		nil,
	)

	// 注册服务发现相关指标
	m.serviceRegisterTotal = m.RegisterCounter(
		"global_service_register_total",
		"Total number of service registrations",
		nil,
	)
	m.serviceUnregisterTotal = m.RegisterCounter(
		"global_service_unregister_total",
		"Total number of service unregistrations",
		nil,
	)
	m.serviceDiscoveryFailures = m.RegisterCounter(
		"global_service_discovery_failures_total",
		"Total number of service discovery failures",
		nil,
	)
	m.serviceHeartbeatTotal = m.RegisterCounter(
		"global_service_heartbeat_total",
		"Total number of service heartbeats",
		nil,
	)

	// 注册 HTTP 服务指标
	m.httpRequestsTotal = m.RegisterCounter(
		"global_http_requests_total",
		"Total number of HTTP requests",
		nil,
	)
	m.httpRequestDuration = m.RegisterHistogram(
		"global_http_request_duration_seconds",
		"HTTP request duration in seconds",
		[]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		nil,
	)
	m.httpErrorRequestsTotal = m.RegisterCounter(
		"global_http_error_requests_total",
		"Total number of HTTP error requests",
		nil,
	)

	// 注册业务逻辑指标
	m.serverListResponseTime = m.RegisterHistogram(
		"global_server_list_response_time_seconds",
		"Server list response time in seconds",
		[]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		nil,
	)
	m.accountOperationTime = m.RegisterHistogram(
		"global_account_operation_time_seconds",
		"Account operation time in seconds",
		[]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		nil,
	)

	zLog.Info("GlobalServer specific metrics registered successfully")
}

// IncrementAccountRegistrations 增加账号注册计数
func (m *Metrics) IncrementAccountRegistrations() {
	if m.accountRegistrationsTotal != nil {
		m.accountRegistrationsTotal.Inc()
	}
}

// IncrementAccountLogins 增加账号登录计数
func (m *Metrics) IncrementAccountLogins() {
	if m.accountLoginsTotal != nil {
		m.accountLoginsTotal.Inc()
	}
}

// IncrementAccountLoginFailures 增加账号登录失败计数
func (m *Metrics) IncrementAccountLoginFailures() {
	if m.accountLoginFailuresTotal != nil {
		m.accountLoginFailuresTotal.Inc()
	}
}

// IncrementServerListRequests 增加服务器列表请求计数
func (m *Metrics) IncrementServerListRequests() {
	if m.serverListRequestsTotal != nil {
		m.serverListRequestsTotal.Inc()
	}
}

// SetGameServersCount 设置游戏服务器数量
func (m *Metrics) SetGameServersCount(count int) {
	if m.gameServersCount != nil {
		m.gameServersCount.Set(float64(count))
	}
}

// RecordDBQuery 记录数据库查询
func (m *Metrics) RecordDBQuery(duration time.Duration) {
	if m.dbQueriesTotal != nil {
		m.dbQueriesTotal.Inc()
	}
	if m.dbQueryDuration != nil {
		m.dbQueryDuration.Observe(duration.Seconds())
	}
}

// SetActiveAccounts 设置活跃账号数量
func (m *Metrics) SetActiveAccounts(count int) {
	if m.activeAccounts != nil {
		m.activeAccounts.Set(float64(count))
	}
}

// 服务发现相关指标方法

// IncrementServiceRegister 增加服务注册计数
func (m *Metrics) IncrementServiceRegister() {
	if m.serviceRegisterTotal != nil {
		m.serviceRegisterTotal.Inc()
	}
}

// IncrementServiceUnregister 增加服务注销计数
func (m *Metrics) IncrementServiceUnregister() {
	if m.serviceUnregisterTotal != nil {
		m.serviceUnregisterTotal.Inc()
	}
}

// IncrementServiceDiscoveryFailures 增加服务发现失败计数
func (m *Metrics) IncrementServiceDiscoveryFailures() {
	if m.serviceDiscoveryFailures != nil {
		m.serviceDiscoveryFailures.Inc()
	}
}

// IncrementServiceHeartbeat 增加服务心跳计数
func (m *Metrics) IncrementServiceHeartbeat() {
	if m.serviceHeartbeatTotal != nil {
		m.serviceHeartbeatTotal.Inc()
	}
}

// HTTP 服务指标方法

// IncrementHTTPRequests 增加 HTTP 请求计数
func (m *Metrics) IncrementHTTPRequests() {
	if m.httpRequestsTotal != nil {
		m.httpRequestsTotal.Inc()
	}
}

// RecordHTTPRequest 记录 HTTP 请求耗时
func (m *Metrics) RecordHTTPRequest(duration time.Duration) {
	if m.httpRequestDuration != nil {
		m.httpRequestDuration.Observe(duration.Seconds())
	}
}

// IncrementHTTPErrorRequests 增加 HTTP 错误请求计数
func (m *Metrics) IncrementHTTPErrorRequests() {
	if m.httpErrorRequestsTotal != nil {
		m.httpErrorRequestsTotal.Inc()
	}
}

// 业务逻辑指标方法

// RecordServerListResponseTime 记录服务器列表响应时间
func (m *Metrics) RecordServerListResponseTime(duration time.Duration) {
	if m.serverListResponseTime != nil {
		m.serverListResponseTime.Observe(duration.Seconds())
	}
}

// RecordAccountOperationTime 记录账号操作时间
func (m *Metrics) RecordAccountOperationTime(duration time.Duration) {
	if m.accountOperationTime != nil {
		m.accountOperationTime.Observe(duration.Seconds())
	}
}
