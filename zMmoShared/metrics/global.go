package metrics

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// globalMetricsManager 全局指标管理器实例
	globalMetricsManager *MetricsManager
	// once 确保全局指标管理器只初始化一次
	once sync.Once
)

// GetMetricsManager 获取全局指标管理器实例
func GetMetricsManager() *MetricsManager {
	once.Do(func() {
		globalMetricsManager = NewMetricsManager()
	})

	return globalMetricsManager
}

// GetNetworkMetrics 获取全局网络指标实例
func GetNetworkMetrics() *NetworkMetrics {
	return GetMetricsManager().GetNetworkMetrics()
}

// GetBusinessMetrics 获取或创建全局业务指标实例
func GetBusinessMetrics(name string) *BusinessMetrics {
	return GetMetricsManager().GetBusinessMetrics(name)
}

// ResetAllMetrics 重置所有指标
func ResetAllMetrics() {
	GetMetricsManager().ResetAll()
}

// RegisterCounter 注册一个全局的counter类型指标
func RegisterCounter(name, help string, labels map[string]string) prometheus.Counter {
	return GetMetricsManager().RegisterCounter(name, help, labels)
}

// RegisterCounterWithCategory 注册一个带分类的全局counter类型指标
func RegisterCounterWithCategory(name, help, category string, labels map[string]string) prometheus.Counter {
	return GetMetricsManager().RegisterCounterWithCategory(name, help, category, labels)
}

// RegisterHistogram 注册一个全局的histogram类型指标
func RegisterHistogram(name, help string, buckets []float64, labels map[string]string) prometheus.Histogram {
	return GetMetricsManager().RegisterHistogram(name, help, buckets, labels)
}

// RegisterHistogramWithCategory 注册一个带分类的全局histogram类型指标
func RegisterHistogramWithCategory(name, help, category string, buckets []float64, labels map[string]string) prometheus.Histogram {
	return GetMetricsManager().RegisterHistogramWithCategory(name, help, category, buckets, labels)
}

// RegisterGauge 注册一个全局的gauge类型指标
func RegisterGauge(name, help string, labels map[string]string) prometheus.Gauge {
	return GetMetricsManager().RegisterGauge(name, help, labels)
}

// RegisterGaugeWithCategory 注册一个带分类的全局gauge类型指标
func RegisterGaugeWithCategory(name, help, category string, labels map[string]string) prometheus.Gauge {
	return GetMetricsManager().RegisterGaugeWithCategory(name, help, category, labels)
}

// BatchRegisterMetrics 批量注册全局指标
func BatchRegisterMetrics(configs []MetricConfig) map[string]interface{} {
	return GetMetricsManager().BatchRegisterMetrics(configs)
}

// GetCounter 获取一个全局的counter类型指标
func GetCounter(name string) prometheus.Counter {
	return GetMetricsManager().GetCounter(name)
}

// GetHistogram 获取一个全局的histogram类型指标
func GetHistogram(name string) prometheus.Histogram {
	return GetMetricsManager().GetHistogram(name)
}

// GetGauge 获取一个全局的gauge类型指标
func GetGauge(name string) prometheus.Gauge {
	return GetMetricsManager().GetGauge(name)
}

// GetMetricsByCategory 获取指定分类的所有全局指标
func GetMetricsByCategory(category string) []string {
	return GetMetricsManager().GetMetricsByCategory(category)
}

// GetAllMetrics 获取所有全局指标
func GetAllMetrics() []string {
	return GetMetricsManager().GetAllMetrics()
}

// GetMetricConfig 获取全局指标配置
func GetMetricConfig(name string) (MetricConfig, bool) {
	return GetMetricsManager().GetMetricConfig(name)
}

// GetRegistry 获取全局的prometheus registry
func GetRegistry() *prometheus.Registry {
	return GetMetricsManager().GetRegistry()
}

// RegisterBasicMetrics 注册基本的Prometheus指标
func RegisterBasicMetrics() {
	// 注册服务器启动时间指标
	startTime := time.Now()
	RegisterGaugeWithCategory("server_start_time", "Server start time in Unix timestamp", CategorySystem, nil)
	if gauge := GetGauge("server_start_time"); gauge != nil {
		gauge.Set(float64(startTime.Unix()))
	}

	// 使用批量注册功能注册网络相关指标
	networkMetrics := []MetricConfig{
		{
			Name:     "active_connections",
			Help:     "Number of active connections",
			Type:     MetricTypeGauge,
			Category: CategoryNetwork,
			Labels:   nil,
		},
		{
			Name:     "total_connections",
			Type:     MetricTypeCounter,
			Help:     "Total number of connections",
			Category: CategoryNetwork,
			Labels:   nil,
		},
		{
			Name:     "dropped_connections",
			Type:     MetricTypeCounter,
			Help:     "Number of dropped connections",
			Category: CategoryNetwork,
			Labels:   nil,
		},
		{
			Name:     "total_bytes_sent",
			Type:     MetricTypeCounter,
			Help:     "Total bytes sent",
			Category: CategoryNetwork,
			Labels:   nil,
		},
		{
			Name:     "total_bytes_received",
			Type:     MetricTypeCounter,
			Help:     "Total bytes received",
			Category: CategoryNetwork,
			Labels:   nil,
		},
		{
			Name:     "encoding_errors",
			Type:     MetricTypeCounter,
			Help:     "Number of encoding errors",
			Category: CategoryNetwork,
			Labels:   nil,
		},
		{
			Name:     "decoding_errors",
			Type:     MetricTypeCounter,
			Help:     "Number of decoding errors",
			Category: CategoryNetwork,
			Labels:   nil,
		},
		{
			Name:     "compression_errors",
			Type:     MetricTypeCounter,
			Help:     "Number of compression errors",
			Category: CategoryNetwork,
			Labels:   nil,
		},
		{
			Name:     "dropped_packets",
			Type:     MetricTypeCounter,
			Help:     "Number of dropped packets",
			Category: CategoryNetwork,
			Labels:   nil,
		},
	}

	// 批量注册指标
	BatchRegisterMetrics(networkMetrics)

	// 注册业务相关指标
	businessMetrics := []MetricConfig{
		// 玩家相关指标
		{
			Name:     "player_login_count",
			Type:     MetricTypeCounter,
			Help:     "Number of player logins",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "player_logout_count",
			Type:     MetricTypeCounter,
			Help:     "Number of player logouts",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "player_register_count",
			Type:     MetricTypeCounter,
			Help:     "Number of player registrations",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "online_player_count",
			Type:     MetricTypeGauge,
			Help:     "Number of online players",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "player_login_latency",
			Type:     MetricTypeHistogram,
			Help:     "Player login latency in seconds",
			Category: CategoryBusiness,
			Labels:   nil,
			Buckets:  []float64{0.1, 0.2, 0.5, 1, 2, 5, 10},
		},
		{
			Name:     "player_session_duration",
			Type:     MetricTypeHistogram,
			Help:     "Player session duration in seconds",
			Category: CategoryBusiness,
			Labels:   nil,
			Buckets:  []float64{60, 300, 600, 1800, 3600, 7200, 14400},
		},

		// 游戏经济相关指标
		{
			Name:     "gold_transaction_count",
			Type:     MetricTypeCounter,
			Help:     "Number of gold transactions",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "gold_transaction_amount",
			Type:     MetricTypeCounter,
			Help:     "Total amount of gold transactions",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "item_transaction_count",
			Type:     MetricTypeCounter,
			Help:     "Number of item transactions",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "auction_transaction_count",
			Type:     MetricTypeCounter,
			Help:     "Number of auction transactions",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "auction_transaction_amount",
			Type:     MetricTypeCounter,
			Help:     "Total amount of auction transactions",
			Category: CategoryBusiness,
			Labels:   nil,
		},

		// 游戏系统相关指标
		{
			Name:     "battle_count",
			Type:     MetricTypeCounter,
			Help:     "Number of battles",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "battle_win_count",
			Type:     MetricTypeCounter,
			Help:     "Number of battle wins",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "skill_usage_count",
			Type:     MetricTypeCounter,
			Help:     "Number of skill usages",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "quest_complete_count",
			Type:     MetricTypeCounter,
			Help:     "Number of quests completed",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "guild_count",
			Type:     MetricTypeGauge,
			Help:     "Number of guilds",
			Category: CategoryBusiness,
			Labels:   nil,
		},
		{
			Name:     "guild_member_count",
			Type:     MetricTypeGauge,
			Help:     "Number of guild members",
			Category: CategoryBusiness,
			Labels:   nil,
		},

		// 服务器性能相关指标
		{
			Name:     "memory_usage_mb",
			Type:     MetricTypeGauge,
			Help:     "Memory usage in MB",
			Category: CategorySystem,
			Labels:   nil,
		},
		{
			Name:     "cpu_usage_percent",
			Type:     MetricTypeGauge,
			Help:     "CPU usage percentage",
			Category: CategorySystem,
			Labels:   nil,
		},
		{
			Name:     "gc_count",
			Type:     MetricTypeCounter,
			Help:     "Number of garbage collections",
			Category: CategorySystem,
			Labels:   nil,
		},
		{
			Name:     "gc_duration_seconds",
			Type:     MetricTypeCounter,
			Help:     "Total duration of garbage collections in seconds",
			Category: CategorySystem,
			Labels:   nil,
		},
		{
			Name:     "goroutine_count",
			Type:     MetricTypeGauge,
			Help:     "Number of goroutines",
			Category: CategorySystem,
			Labels:   nil,
		},

		// 网络相关指标扩展
		{
			Name:     "packet_processing_time",
			Type:     MetricTypeHistogram,
			Help:     "Packet processing time in seconds",
			Category: CategoryNetwork,
			Labels:   nil,
			Buckets:  []float64{0.001, 0.01, 0.1, 0.5, 1, 2, 5},
		},
		{
			Name:     "request_queue_length",
			Type:     MetricTypeGauge,
			Help:     "Length of request queue",
			Category: CategoryNetwork,
			Labels:   nil,
		},
		{
			Name:     "response_time",
			Type:     MetricTypeHistogram,
			Help:     "Response time in seconds",
			Category: CategoryNetwork,
			Labels:   nil,
			Buckets:  []float64{0.001, 0.01, 0.1, 0.5, 1, 2, 5},
		},
	}

	// 批量注册业务指标
	BatchRegisterMetrics(businessMetrics)

	// 使用标准库log记录日志，避免循环依赖
	log.Println("Basic Prometheus metrics registered successfully")
}

// ValidateMetricName 验证指标名称是否合法
func ValidateMetricName(name string) error {
	if name == "" {
		return fmt.Errorf("metric name cannot be empty")
	}

	// 简单的指标名称验证，符合Prometheus规范
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == ':') {
			return fmt.Errorf("metric name contains invalid character: %c", char)
		}
	}

	return nil
}

// GenerateMetricName 生成规范化的指标名称
func GenerateMetricName(category, name string) string {
	return fmt.Sprintf("%s_%s", category, name)
}
