package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsManager(t *testing.T) {
	mm := NewMetricsManager()
	assert.NotNil(t, mm)
	assert.NotNil(t, mm.GetNetworkMetrics())
}

func TestRegisterCounter(t *testing.T) {
	mm := NewMetricsManager()
	counter := mm.RegisterCounter("test_counter", "Test counter", nil)
	assert.NotNil(t, counter)

	// 测试计数器增加
	counter.Inc()
	// 由于我们无法直接获取计数器值，这里主要测试注册过程
}

func TestRegisterGauge(t *testing.T) {
	mm := NewMetricsManager()
	gauge := mm.RegisterGauge("test_gauge", "Test gauge", nil)
	assert.NotNil(t, gauge)

	// 测试 gauge 设置
	gauge.Set(100)
}

func TestRegisterHistogram(t *testing.T) {
	mm := NewMetricsManager()
	histogram := mm.RegisterHistogram("test_histogram", "Test histogram", []float64{0.1, 0.5, 1.0}, nil)
	assert.NotNil(t, histogram)

	// 测试直方图记录
	histogram.Observe(0.5)
}

func TestGetBusinessMetrics(t *testing.T) {
	mm := NewMetricsManager()
	businessMetrics := mm.GetBusinessMetrics("test_business")
	assert.NotNil(t, businessMetrics)

	// 测试业务指标计数器
	businessMetrics.IncCounter("test_count")
	assert.Equal(t, int64(1), businessMetrics.GetCounter("test_count"))

	// 测试业务指标计时器
	businessMetrics.RecordTimer("test_timer", 100)
	assert.Equal(t, int64(100), businessMetrics.GetTimer("test_timer"))
}

func TestResetAll(t *testing.T) {
	mm := NewMetricsManager()
	businessMetrics := mm.GetBusinessMetrics("test_business")
	businessMetrics.IncCounter("test_count")

	// 重置所有指标
	mm.ResetAll()

	// 检查业务指标是否被重置
	assert.Equal(t, int64(0), businessMetrics.GetCounter("test_count"))
}
