// Package message 客户端-服务器消息编解码
// 职责：定义客户端与网关/游戏服务器之间的二进制消息格式（Length+MsgID+Data）
// 边界：仅处理客户端协议层的消息编码/解码，不涉及服务器间通信
// 服务器间通信请使用 crossserver 包
package message

import (
	"errors"
	"sync"

	"github.com/pzqf/zEngine/zNet"
)

// MessageHeader 消息头
type MessageHeader struct {
	Length uint32 // 消息总长度
	MsgID  uint32 // 消息ID
}

// Message 消息结构
type Message struct {
	Header MessageHeader
	Data   []byte // 消息数据
}

// 消息对象池
var messagePool = sync.Pool{
	New: func() interface{} {
		return &Message{}
	},
}

// GetMessage 从对象池获取消息对象
func GetMessage() *Message {
	return messagePool.Get().(*Message)
}

// PutMessage 将消息对象归还到对象池
func PutMessage(msg *Message) {
	// 重置消息对象
	msg.Header.Length = 0
	msg.Header.MsgID = 0
	msg.Data = nil
	messagePool.Put(msg)
}

// Encode 编码消息
func Encode(msgID uint32, data []byte) ([]byte, error) {
	totalLen := 8 + len(data)
	buffer := make([]byte, totalLen)

	// 编码长度
	order := zNet.GetByteOrder()
	order.PutUint32(buffer[0:4], uint32(totalLen))
	// 编码消息ID
	order.PutUint32(buffer[4:8], msgID)
	// 编码数据
	copy(buffer[8:], data)

	return buffer, nil
}

// Decode 解码消息
func Decode(data []byte) (*Message, error) {
	if len(data) < 8 {
		return nil, errors.New("invalid message format: insufficient data for header")
	}

	// 从对象池获取消息对象
	message := GetMessage()
	order := zNet.GetByteOrder()
	message.Header.Length = order.Uint32(data[0:4])
	message.Header.MsgID = order.Uint32(data[4:8])

	if len(data) < int(message.Header.Length) {
		PutMessage(message)
		return nil, errors.New("invalid message format: insufficient data for message")
	}

	message.Data = data[8:message.Header.Length]

	return message, nil
}
