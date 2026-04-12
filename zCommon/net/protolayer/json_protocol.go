package protolayer

import (
	"encoding/json"

	"github.com/pzqf/zEngine/zNet"
)

// JSONProtocol JSON协议实现
type JSONProtocol struct{}

// NewJSONProtocol 创建JSON协议实例
func NewJSONProtocol() *JSONProtocol {
	return &JSONProtocol{}
}

// Encode 将应用层消息编码为JSON格式
func (jp *JSONProtocol) Encode(protoId int32, version int32, data interface{}) (*zNet.NetPacket, error) {
	// 序列化JSON数据
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// 创建NetPacket
	packet := &zNet.NetPacket{
		ProtoId:  zNet.ProtoIdType(protoId),
		Version:  version,
		DataSize: int32(len(body)),
		Data:     body,
	}

	return packet, nil
}

// Decode 将JSON格式数据解码为应用层消息
func (jp *JSONProtocol) Decode(packet *zNet.NetPacket) (interface{}, error) {
	// 这里只返回原始数据，实际使用时需要根据protoId解析为具体的消息类型
	return packet.Data, nil
}
