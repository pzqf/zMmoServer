package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"

	"google.golang.org/protobuf/proto"
)

const (
	MsgHeaderLen = 4
	MaxMsgLen    = 1024 * 1024
)

type Message struct {
	MsgType uint32
	Data    []byte
}

type ProtocolHandler struct{}

func NewProtocolHandler() *ProtocolHandler {
	return &ProtocolHandler{}
}

func (ph *ProtocolHandler) Encode(msgType uint32, data interface{}) ([]byte, error) {
	var serializedData []byte
	var err error

	if pbMsg, ok := data.(proto.Message); ok {
		serializedData, err = proto.Marshal(pbMsg)
		if err != nil {
			return nil, err
		}
	} else if bytesData, ok := data.([]byte); ok {
		serializedData = bytesData
	} else {
		return nil, errors.New("unsupported data type")
	}

	totalLen := MsgHeaderLen + 4 + len(serializedData)
	if totalLen > MaxMsgLen {
		return nil, errors.New("message too long")
	}

	buffer := new(bytes.Buffer)

	err = binary.Write(buffer, binary.BigEndian, uint32(totalLen))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, uint32(msgType))
	if err != nil {
		return nil, err
	}

	_, err = buffer.Write(serializedData)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (ph *ProtocolHandler) Decode(data []byte) (*Message, error) {
	if len(data) < MsgHeaderLen {
		return nil, errors.New("insufficient data")
	}

	length := binary.BigEndian.Uint32(data[:MsgHeaderLen])
	if length > MaxMsgLen {
		return nil, errors.New("message too long")
	}

	if len(data) < int(length) {
		return nil, errors.New("insufficient data")
	}

	msgType := binary.BigEndian.Uint32(data[MsgHeaderLen : MsgHeaderLen+4])
	msgData := data[MsgHeaderLen+4 : length]

	return &Message{
		MsgType: msgType,
		Data:    msgData,
	}, nil
}
