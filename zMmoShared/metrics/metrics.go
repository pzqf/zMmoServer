package metrics

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zMetrics"
)

// 为了向后兼容，保留这些常量
const (
	MetricTypeCounter   = zMetrics.MetricTypeCounter
	MetricTypeGauge     = zMetrics.MetricTypeGauge
	MetricTypeHistogram = zMetrics.MetricTypeHistogram
)

const (
	CategoryNetwork  = zMetrics.CategoryNetwork
	CategoryBusiness = "business"
	CategorySystem   = zMetrics.CategorySystem
	CategoryDatabase = zMetrics.CategoryDatabase
	CategoryCustom   = zMetrics.CategoryCustom
)

// MetricConfig 指标配置（别名）
type MetricConfig = zMetrics.MetricConfig

// MetricsManager 指标管理器（别名）
type MetricsManager = zMetrics.MetricsManager

// NetworkMetrics 网络指标（别名）
type NetworkMetrics = zMetrics.NetworkMetrics

// NewMetricsManager 创建指标管理器实例
func NewMetricsManager() *MetricsManager {
	return zMetrics.NewMetricsManager()
}

// NewNetworkMetrics 创建网络指标监控实例
func NewNetworkMetrics() *NetworkMetrics {
	return zMetrics.NewNetworkMetrics()
}

// BusinessMetrics 业务指标监控（游戏特定）
type BusinessMetrics struct {
	mu sync.RWMutex

	// 计数器
	counters map[string]int64

	// 计时器
	timers map[string]time.Duration

	// 采样时间
	lastSampleTime time.Time
}

// NewBusinessMetrics 创建业务指标监控实例
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

// AddCounter 添加计数器值
func (m *BusinessMetrics) AddCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[name] += value
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

	// 返回副本
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

	// 返回副本
	timers := make(map[string]time.Duration, len(m.timers))
	for k, v := range m.timers {
		timers[k] = v
	}

	return timers
}

// Reset 重置所有指标
func (m *BusinessMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters = make(map[string]int64)
	m.timers = make(map[string]time.Duration)
	m.lastSampleTime = time.Now()
}

// GetLastSampleTime 获取上次采样时间
func (m *BusinessMetrics) GetLastSampleTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.lastSampleTime
}

// GlobalMetrics 全局指标实例
var GlobalMetrics = NewMetricsManager()

// GetGlobalMetrics 获取全局指标实例
func GetGlobalMetrics() *MetricsManager {
	return GlobalMetrics
}

// GetBusinessMetrics 获取业务指标实例（独立函数，不在类型上定义方法）
func GetBusinessMetrics(name string) *BusinessMetrics {
	// 这里使用一个简单实现，实际可以根据需要扩展
	return NewBusinessMetrics()
}
