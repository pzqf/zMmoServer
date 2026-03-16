package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GlobalServer/config"
	sharedMetrics "github.com/pzqf/zMmoShared/metrics"
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
