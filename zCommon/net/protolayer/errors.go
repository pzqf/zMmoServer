package protolayer

import "errors"

// 协议层错误定义
var (
	ErrInvalidProtocolType    = errors.New("invalid protocol type")
	ErrInvalidProtobufMessage = errors.New("invalid protobuf message")
	ErrProtocolEncodeFailed   = errors.New("protocol encode failed")
	ErrProtocolDecodeFailed   = errors.New("protocol decode failed")
	ErrUnknownMessageType     = errors.New("unknown message type")
)
