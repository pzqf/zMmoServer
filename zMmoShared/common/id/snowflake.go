package id

import (
	"errors"
	"sync"
	"time"
)

const (
	workerIDBits      = 5
	datacenterIDBits  = 5
	sequenceBits      = 12
	maxWorkerID       = -1 ^ (-1 << workerIDBits)
	maxDatacenterID   = -1 ^ (-1 << datacenterIDBits)
	sequenceMask      = -1 ^ (-1 << sequenceBits)
	workerIDShift     = sequenceBits
	datacenterIDShift = sequenceBits + workerIDBits
	timestampShift    = sequenceBits + workerIDBits + datacenterIDBits
)

var (
	ErrInvalidWorkerID   = errors.New("invalid worker ID")
	ErrInvalidDatacenter = errors.New("invalid datacenter ID")
	ErrClockMovedBack    = errors.New("clock moved backwards")
)

// Snowflake 分布式ID生成�?// 基于Twitter的Snowflake算法，生�?4位有序唯一ID
// ID结构: 时间�?41�? + 数据中心ID(5�? + 工作节点ID(5�? + 序列�?12�?
type Snowflake struct {
	mu           sync.Mutex
	workerID     int64
	datacenterID int64
	sequence     int64
	lastTime     int64
	epoch        int64
}

// NewSnowflake 创建Snowflake实例
// workerID: 工作节点ID(0-31), datacenterID: 数据中心ID(0-31)
func NewSnowflake(workerID, datacenterID int64) (*Snowflake, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, ErrInvalidWorkerID
	}
	if datacenterID < 0 || datacenterID > maxDatacenterID {
		return nil, ErrInvalidDatacenter
	}

	return &Snowflake{
		workerID:     workerID,
		datacenterID: datacenterID,
		sequence:     0,
		lastTime:     0,
		epoch:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano() / 1e6,
	}, nil
}

// NextID 生成下一个唯一ID
func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentTime := time.Now().UnixNano() / 1e6

	if currentTime < s.lastTime {
		return 0, ErrClockMovedBack
	}

	if currentTime == s.lastTime {
		s.sequence = (s.sequence + 1) & sequenceMask
		if s.sequence == 0 {
			currentTime = s.waitNextMillis(s.lastTime)
		}
	} else {
		s.sequence = 0
	}

	s.lastTime = currentTime

	id := ((currentTime - s.epoch) << timestampShift) |
		(s.datacenterID << datacenterIDShift) |
		(s.workerID << workerIDShift) |
		s.sequence

	return id, nil
}

// waitNextMillis 等待下一个毫秒，避免时钟回拨
func (s *Snowflake) waitNextMillis(lastTime int64) int64 {
	currentTime := time.Now().UnixNano() / 1e6
	for currentTime <= lastTime {
		currentTime = time.Now().UnixNano() / 1e6
	}
	return currentTime
}

// ParseID 解析ID，提取时间戳、数据中心ID、工作节点ID和序列号
func (s *Snowflake) ParseID(id int64) (timestamp, datacenterID, workerID, sequence int64) {
	sequence = id & sequenceMask
	workerID = (id >> workerIDShift) & maxWorkerID
	datacenterID = (id >> datacenterIDShift) & maxDatacenterID
	timestamp = (id >> timestampShift) + s.epoch
	return
}
