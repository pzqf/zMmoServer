package handler

import (
	"google.golang.org/protobuf/proto"
)

// marshalResponse 序列化响应消息
func marshalResponse(msg proto.Message) []byte {
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil
	}
	return data
}

// unmarshalRequest 反序列化请求消息
func unmarshalRequest(data []byte, msg proto.Message) error {
	return proto.Unmarshal(data, msg)
}
