package protolayer

import (
	"errors"
	"strings"

	"github.com/pzqf/zEngine/zNet"
)

// Protocol 定义协议接口，实现协议的编解码与zNet层解耦
type Protocol interface {
	// Encode 将应用层消息编码为二进制数据
	Encode(protoId int32, version int32, data interface{}) (*zNet.NetPacket, error)
	// Decode 将二进制数据解码为应用层消息
	Decode(packet *zNet.NetPacket) (interface{}, error)
}

// ProtocolType 协议类型枚举
type ProtocolType int

const (
	ProtocolTypeProtobuf ProtocolType = iota
	ProtocolTypeJSON
	ProtocolTypeXML
)

// ProtocolFactory 协议工厂函数类型
type ProtocolFactory func() Protocol

// 协议注册映射
var protocolFactories = make(map[string]ProtocolFactory)

// RegisterProtocol 注册协议工厂
func RegisterProtocol(name string, factory ProtocolFactory) {
	protocolFactories[strings.ToLower(name)] = factory
}

// GetProtocolFactory 获取协议工厂
func GetProtocolFactory(name string) (ProtocolFactory, error) {
	if factory, ok := protocolFactories[strings.ToLower(name)]; ok {
		return factory, nil
	}
	return nil, errors.New("protocol not found: " + name)
}

// NewProtocol 根据类型创建协议实例
func NewProtocol(protocolType ProtocolType) Protocol {
	switch protocolType {
	case ProtocolTypeProtobuf:
		return NewProtobufProtocol()
	case ProtocolTypeJSON:
		return NewJSONProtocol()
	case ProtocolTypeXML:
		return NewXMLProtocol()
	default:
		return NewProtobufProtocol() // 默认使用protobuf
	}
}

// NewProtocolByName 根据名称创建协议实例
func NewProtocolByName(name string) (Protocol, error) {
	factory, err := GetProtocolFactory(name)
	if err != nil {
		return nil, err
	}
	return factory(), nil
}

// 初始化函数，注册默认协议
func init() {
	RegisterProtocol("protobuf", func() Protocol {
		return NewProtobufProtocol()
	})
	RegisterProtocol("json", func() Protocol {
		return NewJSONProtocol()
	})
	RegisterProtocol("xml", func() Protocol {
		return NewXMLProtocol()
	})
}
