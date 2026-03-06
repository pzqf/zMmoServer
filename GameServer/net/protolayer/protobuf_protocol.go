package protolayer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

var (
	ErrInvalidPacketLength = errors.New("invalid packet length")
	ErrInvalidMessageType  = errors.New("invalid message type")
)

type Protocol interface {
	Encode(msgID uint32, data []byte) ([]byte, error)
	Decode(data []byte) (uint32, []byte, error)
	GetName() string
}

type ProtobufProtocol struct{}

func NewProtobufProtocol() *ProtobufProtocol {
	return &ProtobufProtocol{}
}

func (p *ProtobufProtocol) Encode(msgID uint32, data []byte) ([]byte, error) {
	totalLength := 4 + 2 + len(data)
	buffer := bytes.NewBuffer(make([]byte, 0, totalLength))

	if err := binary.Write(buffer, binary.BigEndian, uint32(totalLength)); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.BigEndian, uint16(msgID)); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(data); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (p *ProtobufProtocol) Decode(data []byte) (uint32, []byte, error) {
	if len(data) < 6 {
		return 0, nil, ErrInvalidPacketLength
	}

	reader := bytes.NewReader(data)

	var length uint32
	if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
		return 0, nil, err
	}

	var msgID uint32
	if err := binary.Read(reader, binary.BigEndian, &msgID); err != nil {
		return 0, nil, err
	}

	payload := make([]byte, len(data)-6)
	if _, err := reader.Read(payload); err != nil && err != io.EOF {
		return 0, nil, err
	}

	return msgID, payload, nil
}

func (p *ProtobufProtocol) GetName() string {
	return "protobuf"
}

func NewProtocolByName(name string) (Protocol, error) {
	switch name {
	case "protobuf":
		return NewProtobufProtocol(), nil
	default:
		return NewProtobufProtocol(), nil
	}
}
