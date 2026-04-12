package pool

import (
	"sync"

	"github.com/pzqf/zEngine/zNet"
)

var netPacketPool = sync.Pool{
	New: func() interface{} {
		return &zNet.NetPacket{
			Data: make([]byte, 0, 256),
		}
	},
}

func AcquireNetPacket() *zNet.NetPacket {
	return netPacketPool.Get().(*zNet.NetPacket)
}

func ReleaseNetPacket(pkt *zNet.NetPacket) {
	pkt.ProtoId = 0
	pkt.Version = 0
	pkt.DataSize = 0
	pkt.IsCompressed = 0
	pkt.Sequence = 0
	pkt.Timestamp = 0
	pkt.KeyID = 0
	pkt.Data = pkt.Data[:0]
	netPacketPool.Put(pkt)
}

var byteSlicePool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 1024)
		return &b
	},
}

func AcquireByteSlice() *[]byte {
	return byteSlicePool.Get().(*[]byte)
}

func ReleaseByteSlice(b *[]byte) {
	*b = (*b)[:0]
	byteSlicePool.Put(b)
}

type TypedPool[T any] struct {
	pool sync.Pool
}

func NewTypedPool[T any](newFn func() *T) *TypedPool[T] {
	return &TypedPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFn()
			},
		},
	}
}

func (p *TypedPool[T]) Acquire() *T {
	return p.pool.Get().(*T)
}

func (p *TypedPool[T]) Release(obj *T) {
	p.pool.Put(obj)
}
