package net

import (
	"github.com/golang/snappy"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
)

// CompressionLevel 压缩级别
const (
	CompressionLevelNone = 0 // 不压缩
	CompressionLevelLow  = 1 // 低压缩率，高速度
	CompressionLevelHigh = 2 // 高压缩率，低速度
)

// CompressionConfig 压缩配置
type CompressionConfig struct {
	Enabled         bool // 是否启用压缩
	Threshold       int  // 压缩阈值
	Level           int  // 压缩级别
	NetworkQuality  int  // 网络质量评估 (0-100)
	MaxCompressSize int  // 最大压缩大小
}

// NewCompressionConfig 创建默认压缩配置
func NewCompressionConfig(cfg *config.NetCompressionConfig) *CompressionConfig {
	return &CompressionConfig{
		Enabled:         cfg.Enabled,
		Threshold:       cfg.Threshold,
		Level:           cfg.Level,
		NetworkQuality:  80,          // 默认网络质量良好
		MaxCompressSize: 1024 * 1024, // 最大压缩1MB
	}
}

// ShouldCompress 判断是否应该压缩
func (c *CompressionConfig) ShouldCompress(dataSize int) bool {
	if !c.Enabled {
		return false
	}

	if dataSize < c.Threshold {
		return false
	}

	if dataSize > c.MaxCompressSize {
		return false
	}

	// 根据网络质量动态调整
	// 网络质量差时，更积极地压缩
	return dataSize > c.Threshold*(100-c.NetworkQuality)/50
}

// Compress 压缩数据
func Compress(data []byte, config *CompressionConfig) ([]byte, bool, error) {
	if !config.ShouldCompress(len(data)) {
		return data, false, nil
	}

	compressed := snappy.Encode(nil, data)

	// 检查压缩是否有效
	if len(compressed) >= len(data) {
		return data, false, nil
	}

	return compressed, true, nil
}

// Decompress 解压数据
func Decompress(data []byte) ([]byte, error) {
	decompressed, err := snappy.Decode(nil, data)
	if err != nil {
		return data, err
	}

	return decompressed, nil
}
