package protolayer

import (
	"encoding/xml"

	"github.com/pzqf/zEngine/zNet"
)

// XMLProtocol XML协议实现
type XMLProtocol struct{}

// NewXMLProtocol 创建XML协议实例
func NewXMLProtocol() *XMLProtocol {
	return &XMLProtocol{}
}

// Encode 将应用层消息编码为XML格式
func (xp *XMLProtocol) Encode(protoId int32, version int32, data interface{}) (*zNet.NetPacket, error) {
	// 序列化XML数据
	body, err := xml.Marshal(data)
	if err != nil {
		return nil, err
	}

	// 创建NetPacket
	packet := &zNet.NetPacket{
		ProtoId:  protoId,
		Version:  version,
		DataSize: int32(len(body)),
		Data:     body,
	}

	return packet, nil
}

// Decode 将XML格式数据解码为应用层消息
func (xp *XMLProtocol) Decode(packet *zNet.NetPacket) (interface{}, error) {
	// 这里只返回原始数据，实际使用时需要根据protoId解析为具体的消息类型
	return packet.Data, nil
}
