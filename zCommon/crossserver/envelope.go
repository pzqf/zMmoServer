package crossserver

import (
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	envelopeMagic   uint32 = 0x5A4D4D4F // ZMMO
	envelopeVersion uint16 = 1
	envelopeSize           = 40
)

const (
	MessageTypeRequest  uint8 = 1
	MessageTypeResponse uint8 = 2
)

const (
	ServiceTypeUnknown uint8 = 0
	ServiceTypeGame    uint8 = 1
	ServiceTypeMap     uint8 = 2
)

type Meta struct {
	TraceID        uint64
	RequestID      uint64
	TimestampUnix  int64
	SourceServerID int32
	SourceService  uint8
	MessageType    uint8
}

var requestSeq uint64

func NewRequestMeta(sourceService uint8, sourceServerID int32) Meta {
	id := atomic.AddUint64(&requestSeq, 1)
	return Meta{
		TraceID:        id,
		RequestID:      id,
		TimestampUnix:  time.Now().UnixMilli(),
		SourceServerID: sourceServerID,
		SourceService:  sourceService,
		MessageType:    MessageTypeRequest,
	}
}

func NewResponseMetaFromRequest(request Meta, sourceService uint8, sourceServerID int32) Meta {
	return Meta{
		TraceID:        request.TraceID,
		RequestID:      request.RequestID,
		TimestampUnix:  time.Now().UnixMilli(),
		SourceServerID: sourceServerID,
		SourceService:  sourceService,
		MessageType:    MessageTypeResponse,
	}
}

func Wrap(meta Meta, payload []byte) []byte {
	buf := make([]byte, envelopeSize+len(payload))
	binary.BigEndian.PutUint32(buf[0:4], envelopeMagic)
	binary.BigEndian.PutUint16(buf[4:6], envelopeVersion)
	binary.BigEndian.PutUint16(buf[6:8], uint16(meta.MessageType))
	binary.BigEndian.PutUint64(buf[8:16], meta.TraceID)
	binary.BigEndian.PutUint64(buf[16:24], meta.RequestID)
	binary.BigEndian.PutUint64(buf[24:32], uint64(meta.TimestampUnix))
	binary.BigEndian.PutUint32(buf[32:36], uint32(meta.SourceServerID))
	buf[36] = meta.SourceService
	buf[37] = meta.MessageType
	copy(buf[envelopeSize:], payload)
	return buf
}

func Unwrap(data []byte) (Meta, []byte, bool, error) {
	if len(data) < envelopeSize {
		return Meta{}, data, false, nil
	}
	if binary.BigEndian.Uint32(data[0:4]) != envelopeMagic {
		return Meta{}, data, false, nil
	}
	if binary.BigEndian.Uint16(data[4:6]) != envelopeVersion {
		return Meta{}, nil, true, fmt.Errorf("unsupported envelope version")
	}

	meta := Meta{
		TraceID:        binary.BigEndian.Uint64(data[8:16]),
		RequestID:      binary.BigEndian.Uint64(data[16:24]),
		TimestampUnix:  int64(binary.BigEndian.Uint64(data[24:32])),
		SourceServerID: int32(binary.BigEndian.Uint32(data[32:36])),
		SourceService:  data[36],
		MessageType:    data[37],
	}
	return meta, data[envelopeSize:], true, nil
}
