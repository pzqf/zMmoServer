package pool

import (
	"sync"
	"time"

	"github.com/pzqf/zCommon/gameevent"
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

var byteSlicePool4K = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 4096)
		return &b
	},
}

func AcquireByteSlice4K() *[]byte {
	return byteSlicePool4K.Get().(*[]byte)
}

func ReleaseByteSlice4K(b *[]byte) {
	*b = (*b)[:0]
	byteSlicePool4K.Put(b)
}

var byteSlicePool16K = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 16384)
		return &b
	},
}

func AcquireByteSlice16K() *[]byte {
	return byteSlicePool16K.Get().(*[]byte)
}

func ReleaseByteSlice16K(b *[]byte) {
	*b = (*b)[:0]
	byteSlicePool16K.Put(b)
}

var eventPool = sync.Pool{
	New: func() interface{} {
		return &gameevent.Event{}
	},
}

func AcquireEvent() *gameevent.Event {
	return eventPool.Get().(*gameevent.Event)
}

func ReleaseEvent(e *gameevent.Event) {
	e.ID = 0
	e.Name = ""
	e.Source = nil
	e.Data = nil
	e.Canceled = false
	eventPool.Put(e)
}

func AcquireEventWith(id gameevent.EventID, name string, source interface{}, data interface{}) *gameevent.Event {
	e := AcquireEvent()
	e.ID = id
	e.Name = name
	e.Timestamp = time.Now()
	e.Source = source
	e.Data = data
	return e
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

type PoolStats struct {
	Name string
	Get  uint64
	Put  uint64
}

type SizedBytePool struct {
	pools map[int]*sync.Pool
}

func NewSizedBytePool() *SizedBytePool {
	sizes := []int{128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768}
	pools := make(map[int]*sync.Pool, len(sizes))
	for _, size := range sizes {
		s := size
		pools[s] = &sync.Pool{
			New: func() interface{} {
				b := make([]byte, 0, s)
				return &b
			},
		}
	}
	return &SizedBytePool{pools: pools}
}

func (sp *SizedBytePool) Acquire(need int) *[]byte {
	for size, pool := range sp.pools {
		if size >= need {
			return pool.Get().(*[]byte)
		}
	}
	b := make([]byte, 0, need)
	return &b
}

func (sp *SizedBytePool) Release(b *[]byte) {
	c := cap(*b)
	for size, pool := range sp.pools {
		if c == size {
			*b = (*b)[:0]
			pool.Put(b)
			return
		}
	}
}

var DefaultSizedBytePool = NewSizedBytePool()
