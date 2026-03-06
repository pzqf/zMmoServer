package metrics

import (
	"sync"
	"time"
)

// NetworkMetrics 网络指标监控
type NetworkMetrics struct {
	mu sync.RWMutex

	// 连接统计
	activeConnections  int
	totalConnections   int64
	droppedConnections int64

	// 延迟统计
	avgLatency     time.Duration
	maxLatency     time.Duration
	minLatency     time.Duration
	totalLatency   time.Duration
	latencySamples int64

	// 吞吐量统计
	totalBytesSent       int64
	totalBytesReceived   int64
	totalPacketsSent     int64
	totalPacketsReceived int64

	// 错误统计
	encodingErrors    int64
	decodingErrors    int64
	compressionErrors int64
	droppedPackets    int64

	// 采样时间
	lastSampleTime time.Time
}

// NewNetworkMetrics 创建网络指标监控实例
func NewNetworkMetrics() *NetworkMetrics {
	return &NetworkMetrics{
		lastSampleTime: time.Now(),
		minLatency:     time.Hour, // 初始化为较大值
	}
}

// IncActiveConnections 增加活跃连接数
func (m *NetworkMetrics) IncActiveConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.activeConnections++
	m.totalConnections++
}

// DecActiveConnections 减少活跃连接数
func (m *NetworkMetrics) DecActiveConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeConnections > 0 {
		m.activeConnections--
	}
}

// IncDroppedConnections 增加丢弃连接数
func (m *NetworkMetrics) IncDroppedConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.droppedConnections++
}

// RecordLatency 记录延迟
func (m *NetworkMetrics) RecordLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalLatency += latency
	m.latencySamples++

	if latency > m.maxLatency {
		m.maxLatency = latency
	}

	if latency < m.minLatency {
		m.minLatency = latency
	}

	// 计算平均延迟
	if m.latencySamples > 0 {
		m.avgLatency = m.totalLatency / time.Duration(m.latencySamples)
	}
}

// RecordBytesSent 记录发送字节数
func (m *NetworkMetrics) RecordBytesSent(bytes int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalBytesSent += int64(bytes)
	m.totalPacketsSent++
}

// RecordBytesReceived 记录接收字节数
func (m *NetworkMetrics) RecordBytesReceived(bytes int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalBytesReceived += int64(bytes)
	m.totalPacketsReceived++
}

// IncEncodingErrors 增加编码错误数
func (m *NetworkMetrics) IncEncodingErrors() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.encodingErrors++
}

// IncDecodingErrors 增加解码错误数
func (m *NetworkMetrics) IncDecodingErrors() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.decodingErrors++
}

// IncCompressionErrors 增加压缩错误数
func (m *NetworkMetrics) IncCompressionErrors() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.compressionErrors++
}

// IncDroppedPackets 增加丢弃数据包数
func (m *NetworkMetrics) IncDroppedPackets() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.droppedPackets++
}

// GetStats 获取统计信息
func (m *NetworkMetrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	elapsed := time.Since(m.lastSampleTime)
	throughputSent := float64(m.totalBytesSent) / elapsed.Seconds()
	throughputReceived := float64(m.totalBytesReceived) / elapsed.Seconds()

	return map[string]interface{}{
		"active_connections":      m.activeConnections,
		"total_connections":       m.totalConnections,
		"dropped_connections":     m.droppedConnections,
		"avg_latency_ms":          float64(m.avgLatency.Milliseconds()),
		"max_latency_ms":          float64(m.maxLatency.Milliseconds()),
		"min_latency_ms":          float64(m.minLatency.Milliseconds()),
		"throughput_sent_bps":     throughputSent,
		"throughput_received_bps": throughputReceived,
		"total_bytes_sent":        m.totalBytesSent,
		"total_bytes_received":    m.totalBytesReceived,
		"total_packets_sent":      m.totalPacketsSent,
		"total_packets_received":  m.totalPacketsReceived,
		"encoding_errors":         m.encodingErrors,
		"decoding_errors":         m.decodingErrors,
		"compression_errors":      m.compressionErrors,
		"dropped_packets":         m.droppedPackets,
		"sample_time":             m.lastSampleTime,
		"elapsed_seconds":         elapsed.Seconds(),
	}
}

// Reset 重置统计信息
func (m *NetworkMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 保留活跃连接数
	activeConnections := m.activeConnections

	// 手动重置所有字段，而不是完全替换对象
	m.activeConnections = activeConnections
	m.totalConnections = 0
	m.droppedConnections = 0
	m.avgLatency = 0
	m.maxLatency = 0
	m.minLatency = time.Hour
	m.totalLatency = 0
	m.latencySamples = 0
	m.totalBytesSent = 0
	m.totalBytesReceived = 0
	m.totalPacketsSent = 0
	m.totalPacketsReceived = 0
	m.encodingErrors = 0
	m.decodingErrors = 0
	m.compressionErrors = 0
	m.droppedPackets = 0
	m.lastSampleTime = time.Now()
}

// PrintStats 打印统计信息
func (m *NetworkMetrics) PrintStats() {
	_ = m.GetStats()

	// 这里可以根据需要输出到日志或监控系统
	// 例如使用zap日志框架
}
