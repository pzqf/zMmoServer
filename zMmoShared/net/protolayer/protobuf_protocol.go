package protolayer

import (
	"time"

	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoShared/metrics"
	"google.golang.org/protobuf/proto"
)

// 全局网络指标监控实例
var globalNetworkMetrics *metrics.NetworkMetrics

// SetNetworkMetrics 设置网络指标监控实例
func SetNetworkMetrics(metrics *metrics.NetworkMetrics) {
	globalNetworkMetrics = metrics
}

// ProtobufProtocol Protobuf协议实现
type ProtobufProtocol struct {
	compressionConfig *CompressionConfig
}

// NewProtobufProtocol 创建Protobuf协议实例
func NewProtobufProtocol() *ProtobufProtocol {
	return &ProtobufProtocol{
		compressionConfig: NewCompressionConfig(),
	}
}

// SetCompressionConfig 设置压缩配置
func (pp *ProtobufProtocol) SetCompressionConfig(config *CompressionConfig) {
	pp.compressionConfig = config
}

// GetCompressionConfig 获取压缩配置
func (pp *ProtobufProtocol) GetCompressionConfig() *CompressionConfig {
	return pp.compressionConfig
}

// UpdateNetworkQuality 更新网络质量评估
func (pp *ProtobufProtocol) UpdateNetworkQuality(quality int) {
	if quality < 0 {
		quality = 0
	}
	if quality > 100 {
		quality = 100
	}
	pp.compressionConfig.NetworkQuality = quality
}

// Encode 将应用层消息编码为二进制数据
func (pp *ProtobufProtocol) Encode(protoId int32, version int32, data interface{}) (*zNet.NetPacket, error) {
	// 记录开始编码时间
	startTime := time.Now()

	// 将data转换为proto.Message
	msg, ok := data.(proto.Message)
	if !ok {
		if globalNetworkMetrics != nil {
			globalNetworkMetrics.IncEncodingErrors()
		}
		return nil, ErrInvalidProtobufMessage
	}

	// 序列化Protobuf数据
	body, err := proto.Marshal(msg)
	if err != nil {
		if globalNetworkMetrics != nil {
			globalNetworkMetrics.IncEncodingErrors()
		}
		return nil, err
	}

	// 压缩数据
	compressedData, isCompressed, err := Compress(body, pp.compressionConfig)
	if err != nil {
		// 压缩失败，使用原始数据
		isCompressed = false
		compressedData = body
		if globalNetworkMetrics != nil {
			globalNetworkMetrics.IncCompressionErrors()
		}
	}

	// 创建NetPacket
	var isCompressedInt int32
	if isCompressed {
		isCompressedInt = 1
	}
	packet := &zNet.NetPacket{
		ProtoId:      protoId,
		Version:      version,
		DataSize:     int32(len(compressedData)),
		Data:         compressedData,
		IsCompressed: isCompressedInt,
	}

	// 记录编码延迟和发送的数据包大小
	if globalNetworkMetrics != nil {
		latency := time.Since(startTime)
		globalNetworkMetrics.RecordLatency(latency)
		globalNetworkMetrics.RecordBytesSent(len(compressedData) + zNet.NetPacketHeadSize)
	}

	return packet, nil
}

// MessageCreator 消息创建函数类型
type MessageCreator func() proto.Message

// MessageRegistry 消息注册表
var MessageRegistry = make(map[int32]MessageCreator)

// RegisterMessage 注册消息类型
func RegisterMessage(protoId int32, creator MessageCreator) {
	MessageRegistry[protoId] = creator
}

// Decode 将二进制数据解码为应用层消息
func (pp *ProtobufProtocol) Decode(packet *zNet.NetPacket) (interface{}, error) {
	// 记录开始解码时间
	startTime := time.Now()

	data := packet.Data

	// 如果数据被压缩，先解压
	if packet.IsCompressed != 0 {
		decompressed, err := Decompress(data)
		if err == nil {
			data = decompressed
		} else {
			if globalNetworkMetrics != nil {
				globalNetworkMetrics.IncCompressionErrors()
			}
		}
	}

	// 根据protoId创建对应的消息实例
	creator, ok := MessageRegistry[packet.ProtoId]
	if !ok {
		if globalNetworkMetrics != nil {
			globalNetworkMetrics.IncDecodingErrors()
		}
		return nil, ErrUnknownMessageType
	}

	msg := creator()
	if err := proto.Unmarshal(data, msg); err != nil {
		if globalNetworkMetrics != nil {
			globalNetworkMetrics.IncDecodingErrors()
		}
		return nil, err
	}

	// 记录解码延迟
	if globalNetworkMetrics != nil {
		latency := time.Since(startTime)
		globalNetworkMetrics.RecordLatency(latency)
	}

	return msg, nil
}
