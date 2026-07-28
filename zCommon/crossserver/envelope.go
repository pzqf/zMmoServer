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
	// MessageTypeError 是响应的失败变体：handler 返回 error 时由 CrossTransport 回投，
	// payload = 错误文本。有它，等待方才能立刻拿到失败原因，而不是干等到超时。
	MessageTypeError uint8 = 3
)

const (
	ServiceTypeUnknown uint8 = 0
	ServiceTypeGame    uint8 = 1
	ServiceTypeMap     uint8 = 2
	ServiceTypeGateway uint8 = 3
	ServiceTypeGlobal  uint8 = 4
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

// ComposeRequestID 把**发送方服务器 ID** 混进请求 ID 的高位，使 requestID 全集群唯一。
//
// 为什么必须这么做（2026-07-28 跨 realm E2E 实测踩到的真缺陷）：requestID 同时是**幂等去重键**，
// 而收侧（MapServer 的进图/攻击去重、GameServer 的响应去重、迁移 inbox）都是按**裸 requestID**
// 去重的。各进程的序号各自从 1 开始递增，于是两个 realm 的 GameServer 会发出同号请求——目标
// MapServer 把第二个 realm 玩家的进图请求当成重放直接丢弃，那个玩家永远进不去跨服实例。
//
// 高 24 位放 serverID（6 位十进制 ID 最大 999999，占 20 位），低 40 位放进程内序号（够用一万亿次）。
// serverID=0（未指定）时退化为纯进程内序号，与旧行为一致。
func ComposeRequestID(serverID int32, seq uint64) uint64 {
	return uint64(uint32(serverID)&0xFFFFFF)<<40 | (seq & 0xFFFFFFFFFF)
}

func NewRequestMeta(sourceService uint8, sourceServerID int32) Meta {
	id := ComposeRequestID(sourceServerID, atomic.AddUint64(&requestSeq, 1))
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
