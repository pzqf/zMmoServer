package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewNetworkMetrics(t *testing.T) {
	nm := NewNetworkMetrics()
	assert.NotNil(t, nm)
}

func TestNetworkMetrics_IncActiveConnections(t *testing.T) {
	nm := NewNetworkMetrics()

	// 增加活跃连接数
	nm.IncActiveConnections()
	stats := nm.GetStats()
	assert.Equal(t, 1, stats["active_connections"])
	assert.Equal(t, int64(1), stats["total_connections"])

	// 再次增加
	nm.IncActiveConnections()
	stats = nm.GetStats()
	assert.Equal(t, 2, stats["active_connections"])
	assert.Equal(t, int64(2), stats["total_connections"])
}

func TestNetworkMetrics_DecActiveConnections(t *testing.T) {
	nm := NewNetworkMetrics()

	// 增加活跃连接数
	nm.IncActiveConnections()
	nm.IncActiveConnections()

	// 减少活跃连接数
	nm.DecActiveConnections()
	stats := nm.GetStats()
	assert.Equal(t, 1, stats["active_connections"])

	// 再次减少
	nm.DecActiveConnections()
	stats = nm.GetStats()
	assert.Equal(t, 0, stats["active_connections"])

	// 测试边界情况：减少到负数
	nm.DecActiveConnections()
	stats = nm.GetStats()
	assert.Equal(t, 0, stats["active_connections"])
}

func TestNetworkMetrics_RecordLatency(t *testing.T) {
	nm := NewNetworkMetrics()

	// 记录延迟
	nm.RecordLatency(10 * time.Millisecond)
	nm.RecordLatency(20 * time.Millisecond)
	nm.RecordLatency(30 * time.Millisecond)

	stats := nm.GetStats()
	avgLatency := stats["avg_latency_ms"].(float64)
	maxLatency := stats["max_latency_ms"].(float64)
	minLatency := stats["min_latency_ms"].(float64)

	assert.InDelta(t, 20.0, avgLatency, 0.1)
	assert.InDelta(t, 30.0, maxLatency, 0.1)
	assert.InDelta(t, 10.0, minLatency, 0.1)
}

func TestNetworkMetrics_RecordBytes(t *testing.T) {
	nm := NewNetworkMetrics()

	// 记录发送字节数
	nm.RecordBytesSent(100)
	nm.RecordBytesSent(200)

	// 记录接收字节数
	nm.RecordBytesReceived(50)
	nm.RecordBytesReceived(150)

	stats := nm.GetStats()
	assert.Equal(t, int64(300), stats["total_bytes_sent"])
	assert.Equal(t, int64(200), stats["total_bytes_received"])
	assert.Equal(t, int64(2), stats["total_packets_sent"])
	assert.Equal(t, int64(2), stats["total_packets_received"])
}

func TestNetworkMetrics_IncErrors(t *testing.T) {
	nm := NewNetworkMetrics()

	// 增加各种错误计数
	nm.IncEncodingErrors()
	nm.IncDecodingErrors()
	nm.IncCompressionErrors()
	nm.IncDroppedPackets()

	stats := nm.GetStats()
	assert.Equal(t, int64(1), stats["encoding_errors"])
	assert.Equal(t, int64(1), stats["decoding_errors"])
	assert.Equal(t, int64(1), stats["compression_errors"])
	assert.Equal(t, int64(1), stats["dropped_packets"])
}

func TestNetworkMetrics_Reset(t *testing.T) {
	nm := NewNetworkMetrics()

	// 增加一些指标
	nm.IncActiveConnections()
	nm.RecordLatency(10 * time.Millisecond)
	nm.RecordBytesSent(100)
	nm.RecordBytesReceived(50)
	nm.IncEncodingErrors()

	// 重置
	nm.Reset()

	stats := nm.GetStats()
	// 活跃连接数应该保持不变
	assert.Equal(t, 1, stats["active_connections"])
	// 其他指标应该被重置
	assert.Equal(t, int64(0), stats["total_connections"])
	assert.InDelta(t, 0.0, stats["avg_latency_ms"], 0.1)
	assert.InDelta(t, 0.0, stats["max_latency_ms"], 0.1)
	assert.InDelta(t, 3600000.0, stats["min_latency_ms"], 0.1) // 初始化为1小时
	assert.Equal(t, int64(0), stats["total_bytes_sent"])
	assert.Equal(t, int64(0), stats["total_bytes_received"])
	assert.Equal(t, int64(0), stats["encoding_errors"])
}
