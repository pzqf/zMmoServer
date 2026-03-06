package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// 指标类型常量
const (
	MetricTypeCounter   = "counter"
	MetricTypeGauge     = "gauge"
	MetricTypeHistogram = "histogram"
)

// 指标分类常量
const (
	CategoryNetwork  = "network"
	CategoryBusiness = "business"
	CategorySystem   = "system"
	CategoryDatabase = "database"
	CategoryCustom   = "custom"
)

// MetricConfig 指标配置
type MetricConfig struct {
	Name     string
	Help     string
	Type     string
	Category string
	Labels   map[string]string
	Buckets  []float64 // 仅用于Histogram类型
}

// MetricsManager 指标管理管理器
type MetricsManager struct {
	mu              sync.RWMutex
	networkMetrics  *NetworkMetrics
	businessMetrics map[string]*BusinessMetrics

	// Prometheus 相关
	registry   *prometheus.Registry
	counters   map[string]prometheus.Counter
	histograms map[string]prometheus.Histogram
	gauges     map[string]prometheus.Gauge

	// 指标分类管理
	metricsByCategory map[string]map[string]bool

	// 指标配置管理
	metricConfigs map[string]MetricConfig
}

// BusinessMetrics 业务指标监控
type BusinessMetrics struct {
	mu sync.RWMutex

	// 计数器
	counters map[string]int64

	// 计时器
	timers map[string]time.Duration

	// 采样时间
	lastSampleTime time.Time
}

// NewMetricsManager 创建指标管理器实例
func NewMetricsManager() *MetricsManager {
	return &MetricsManager{
		networkMetrics:    NewNetworkMetrics(),
		businessMetrics:   make(map[string]*BusinessMetrics),
		registry:          prometheus.NewRegistry(),
		counters:          make(map[string]prometheus.Counter),
		histograms:        make(map[string]prometheus.Histogram),
		gauges:            make(map[string]prometheus.Gauge),
		metricsByCategory: make(map[string]map[string]bool),
		metricConfigs:     make(map[string]MetricConfig),
	}
}

// GetNetworkMetrics 获取网络指标实例
func (m *MetricsManager) GetNetworkMetrics() *NetworkMetrics {
	return m.networkMetrics
}

// GetBusinessMetrics 获取或创建业务指标实例
func (m *MetricsManager) GetBusinessMetrics(name string) *BusinessMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()

	if metrics, exists := m.businessMetrics[name]; exists {
		return metrics
	}

	metrics := NewBusinessMetrics()
	m.businessMetrics[name] = metrics
	return metrics
}

// GetAllBusinessMetrics 获取所有业务指标实例
func (m *MetricsManager) GetAllBusinessMetrics() map[string]*BusinessMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本，避免并发修改
	metrics := make(map[string]*BusinessMetrics, len(m.businessMetrics))
	for k, v := range m.businessMetrics {
		metrics[k] = v
	}

	return metrics
}

// ResetAll 重置所有指标
func (m *MetricsManager) ResetAll() {
	m.networkMetrics.Reset()

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, metrics := range m.businessMetrics {
		metrics.Reset()
	}
}

// RegisterCounter 注册一个counter类型的指标
func (m *MetricsManager) RegisterCounter(name, help string, labels map[string]string) prometheus.Counter {
	return m.RegisterCounterWithCategory(name, help, CategoryCustom, labels)
}

// RegisterCounterWithCategory 注册一个带分类的counter类型指标
func (m *MetricsManager) RegisterCounterWithCategory(name, help, category string, labels map[string]string) prometheus.Counter {
	m.mu.Lock()
	defer m.mu.Unlock()

	if counter, exists := m.counters[name]; exists {
		return counter
	}

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name:        name,
		Help:        help,
		ConstLabels: labels,
	})

	m.registry.MustRegister(counter)
	m.counters[name] = counter

	// 记录指标分类
	m.addMetricToCategory(name, category)

	// 记录指标配置
	m.metricConfigs[name] = MetricConfig{
		Name:     name,
		Help:     help,
		Type:     MetricTypeCounter,
		Category: category,
		Labels:   labels,
	}

	return counter
}

// RegisterHistogram 注册一个histogram类型的指标
func (m *MetricsManager) RegisterHistogram(name, help string, buckets []float64, labels map[string]string) prometheus.Histogram {
	return m.RegisterHistogramWithCategory(name, help, CategoryCustom, buckets, labels)
}

// RegisterHistogramWithCategory 注册一个带分类的histogram类型指标
func (m *MetricsManager) RegisterHistogramWithCategory(name, help, category string, buckets []float64, labels map[string]string) prometheus.Histogram {
	m.mu.Lock()
	defer m.mu.Unlock()

	if histogram, exists := m.histograms[name]; exists {
		return histogram
	}

	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:        name,
		Help:        help,
		Buckets:     buckets,
		ConstLabels: labels,
	})

	m.registry.MustRegister(histogram)
	m.histograms[name] = histogram

	// 记录指标分类
	m.addMetricToCategory(name, category)

	// 记录指标配置
	m.metricConfigs[name] = MetricConfig{
		Name:     name,
		Help:     help,
		Type:     MetricTypeHistogram,
		Category: category,
		Labels:   labels,
		Buckets:  buckets,
	}

	return histogram
}

// RegisterGauge 注册一个gauge类型的指标
func (m *MetricsManager) RegisterGauge(name, help string, labels map[string]string) prometheus.Gauge {
	return m.RegisterGaugeWithCategory(name, help, CategoryCustom, labels)
}

// RegisterGaugeWithCategory 注册一个带分类的gauge类型指标
func (m *MetricsManager) RegisterGaugeWithCategory(name, help, category string, labels map[string]string) prometheus.Gauge {
	m.mu.Lock()
	defer m.mu.Unlock()

	if gauge, exists := m.gauges[name]; exists {
		return gauge
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        name,
		Help:        help,
		ConstLabels: labels,
	})

	m.registry.MustRegister(gauge)
	m.gauges[name] = gauge

	// 记录指标分类
	m.addMetricToCategory(name, category)

	// 记录指标配置
	m.metricConfigs[name] = MetricConfig{
		Name:     name,
		Help:     help,
		Type:     MetricTypeGauge,
		Category: category,
		Labels:   labels,
	}

	return gauge
}

// BatchRegisterMetrics 批量注册指标
func (m *MetricsManager) BatchRegisterMetrics(configs []MetricConfig) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(map[string]interface{})

	for _, config := range configs {
		var metric interface{}

		switch config.Type {
		case MetricTypeCounter:
			counter := prometheus.NewCounter(prometheus.CounterOpts{
				Name:        config.Name,
				Help:        config.Help,
				ConstLabels: config.Labels,
			})
			m.registry.MustRegister(counter)
			m.counters[config.Name] = counter
			metric = counter

		case MetricTypeGauge:
			gauge := prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        config.Name,
				Help:        config.Help,
				ConstLabels: config.Labels,
			})
			m.registry.MustRegister(gauge)
			m.gauges[config.Name] = gauge
			metric = gauge

		case MetricTypeHistogram:
			histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
				Name:        config.Name,
				Help:        config.Help,
				Buckets:     config.Buckets,
				ConstLabels: config.Labels,
			})
			m.registry.MustRegister(histogram)
			m.histograms[config.Name] = histogram
			metric = histogram

		default:
			continue
		}

		// 记录指标分类
		m.addMetricToCategory(config.Name, config.Category)

		// 记录指标配置
		m.metricConfigs[config.Name] = config

		result[config.Name] = metric
	}

	return result
}

// GetCounter 获取一个counter类型的指标
func (m *MetricsManager) GetCounter(name string) prometheus.Counter {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.counters[name]
}

// GetHistogram 获取一个histogram类型的指标
func (m *MetricsManager) GetHistogram(name string) prometheus.Histogram {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.histograms[name]
}

// GetGauge 获取一个gauge类型的指标
func (m *MetricsManager) GetGauge(name string) prometheus.Gauge {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.gauges[name]
}

// GetMetricsByCategory 获取指定分类的所有指标
func (m *MetricsManager) GetMetricsByCategory(category string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make([]string, 0)
	if metricMap, exists := m.metricsByCategory[category]; exists {
		for metric := range metricMap {
			metrics = append(metrics, metric)
		}
	}

	return metrics
}

// GetAllMetrics 获取所有指标
func (m *MetricsManager) GetAllMetrics() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make([]string, 0)

	// 收集所有Counter类型指标
	for metric := range m.counters {
		metrics = append(metrics, metric)
	}

	// 收集所有Gauge类型指标
	for metric := range m.gauges {
		metrics = append(metrics, metric)
	}

	// 收集所有Histogram类型指标
	for metric := range m.histograms {
		metrics = append(metrics, metric)
	}

	return metrics
}

// GetMetricConfig 获取指标配置
func (m *MetricsManager) GetMetricConfig(name string) (MetricConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.metricConfigs[name]
	return config, exists
}

// GetRegistry 获取prometheus的registry
func (m *MetricsManager) GetRegistry() *prometheus.Registry {
	return m.registry
}

// addMetricToCategory 将指标添加到分类
func (m *MetricsManager) addMetricToCategory(name, category string) {
	if _, exists := m.metricsByCategory[category]; !exists {
		m.metricsByCategory[category] = make(map[string]bool)
	}
	m.metricsByCategory[category][name] = true
}

// NewBusinessMetrics 创建业务指标实例
func NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{
		counters:       make(map[string]int64),
		timers:         make(map[string]time.Duration),
		lastSampleTime: time.Now(),
	}
}

// IncCounter 增加计数器
func (m *BusinessMetrics) IncCounter(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[name]++
}

// AddCounter 增加计数器指定值
func (m *BusinessMetrics) AddCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[name] += value
}

// SetCounter 设置计数器值
func (m *BusinessMetrics) SetCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[name] = value
}

// GetCounter 获取计数器值
func (m *BusinessMetrics) GetCounter(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.counters[name]
}

// RecordTimer 记录计时器
func (m *BusinessMetrics) RecordTimer(name string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.timers[name] = duration
}

// GetTimer 获取计时器值
func (m *BusinessMetrics) GetTimer(name string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.timers[name]
}

// GetAllCounters 获取所有计数器
func (m *BusinessMetrics) GetAllCounters() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本，避免并发修改
	counters := make(map[string]int64, len(m.counters))
	for k, v := range m.counters {
		counters[k] = v
	}

	return counters
}

// GetAllTimers 获取所有计时器
func (m *BusinessMetrics) GetAllTimers() map[string]time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本，避免并发修改
	timers := make(map[string]time.Duration, len(m.timers))
	for k, v := range m.timers {
		timers[k] = v
	}

	return timers
}

// Reset 重置业务指标
func (m *BusinessMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters = make(map[string]int64)
	m.timers = make(map[string]time.Duration)
	m.lastSampleTime = time.Now()
}
