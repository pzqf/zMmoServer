package parallel

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type UpdateFunc func(deltaTime time.Duration)

type Partition struct {
	ID       int
	updateFn UpdateFunc
}

type PartitionScheduler struct {
	partitions *zMap.TypedMap[int, *Partition]
	workers    int
	running    atomic.Bool
	wg         sync.WaitGroup
	stats      SchedulerStats
}

type SchedulerStats struct {
	TotalUpdates  atomic.Uint64
	TotalDuration atomic.Int64
	MaxDuration   atomic.Int64
}

func NewPartitionScheduler(workers int) *PartitionScheduler {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &PartitionScheduler{
		partitions: zMap.NewTypedMap[int, *Partition](),
		workers:    workers,
	}
}

func (ps *PartitionScheduler) RegisterPartition(id int, updateFn UpdateFunc) {
	ps.partitions.Store(id, &Partition{
		ID:       id,
		updateFn: updateFn,
	})
	zLog.Debug("Partition registered",
		zap.Int("partition_id", id))
}

func (ps *PartitionScheduler) UnregisterPartition(id int) {
	ps.partitions.Delete(id)
}

func (ps *PartitionScheduler) Update(deltaTime time.Duration) {
	if !ps.running.Load() {
		return
	}

	start := time.Now()

	partitions := make([]*Partition, 0)
	ps.partitions.Range(func(id int, p *Partition) bool {
		partitions = append(partitions, p)
		return true
	})

	if len(partitions) == 0 {
		return
	}

	if len(partitions) <= ps.workers {
		ps.updateParallel(partitions, deltaTime)
	} else {
		ps.updateBatched(partitions, deltaTime)
	}

	elapsed := time.Since(start)
	ps.stats.TotalUpdates.Add(1)
	ps.stats.TotalDuration.Add(int64(elapsed))
	for {
		current := ps.stats.MaxDuration.Load()
		if int64(elapsed) <= current || ps.stats.MaxDuration.CompareAndSwap(current, int64(elapsed)) {
			break
		}
	}
}

func (ps *PartitionScheduler) updateParallel(partitions []*Partition, deltaTime time.Duration) {
	var wg sync.WaitGroup
	wg.Add(len(partitions))

	for _, p := range partitions {
		p := p
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					zLog.Error("Partition update panic",
						zap.Int("partition_id", p.ID),
						zap.Any("recover", r))
				}
			}()
			p.updateFn(deltaTime)
		}()
	}

	wg.Wait()
}

func (ps *PartitionScheduler) updateBatched(partitions []*Partition, deltaTime time.Duration) {
	batchSize := (len(partitions) + ps.workers - 1) / ps.workers
	var wg sync.WaitGroup

	for i := 0; i < len(partitions); i += batchSize {
		end := i + batchSize
		if end > len(partitions) {
			end = len(partitions)
		}

		batch := partitions[i:end]
		wg.Add(1)

		go func(b []*Partition) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					zLog.Error("Partition batch update panic",
						zap.Any("recover", r))
				}
			}()
			for _, p := range b {
				p.updateFn(deltaTime)
			}
		}(batch)
	}

	wg.Wait()
}

func (ps *PartitionScheduler) Start(tickInterval time.Duration) {
	if !ps.running.CompareAndSwap(false, true) {
		return
	}

	ps.wg.Add(1)
	go func() {
		defer ps.wg.Done()

		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()

		lastTick := time.Now()
		for ps.running.Load() {
			select {
			case <-ticker.C:
				now := time.Now()
				deltaTime := now.Sub(lastTick)
				lastTick = now
				ps.Update(deltaTime)
			}
		}
	}()

	zLog.Info("PartitionScheduler started",
		zap.Int("workers", ps.workers),
		zap.Duration("tick_interval", tickInterval))
}

func (ps *PartitionScheduler) Stop() {
	if !ps.running.CompareAndSwap(true, false) {
		return
	}
	ps.wg.Wait()
	zLog.Info("PartitionScheduler stopped")
}

func (ps *PartitionScheduler) PartitionCount() int {
	return int(ps.partitions.Len())
}

func (ps *PartitionScheduler) AvgUpdateDuration() time.Duration {
	total := ps.stats.TotalUpdates.Load()
	if total == 0 {
		return 0
	}
	return time.Duration(ps.stats.TotalDuration.Load() / int64(total))
}

func (ps *PartitionScheduler) MaxUpdateDuration() time.Duration {
	return time.Duration(ps.stats.MaxDuration.Load())
}

type MapUpdateScheduler struct {
	scheduler *PartitionScheduler
}

func NewMapUpdateScheduler(workers int) *MapUpdateScheduler {
	return &MapUpdateScheduler{
		scheduler: NewPartitionScheduler(workers),
	}
}

func (mus *MapUpdateScheduler) RegisterMap(mapID int32, updateFn UpdateFunc) {
	mus.scheduler.RegisterPartition(int(mapID), updateFn)
}

func (mus *MapUpdateScheduler) UnregisterMap(mapID int32) {
	mus.scheduler.UnregisterPartition(int(mapID))
}

func (mus *MapUpdateScheduler) Start(tickInterval time.Duration) {
	mus.scheduler.Start(tickInterval)
}

func (mus *MapUpdateScheduler) Stop() {
	mus.scheduler.Stop()
}

func (mus *MapUpdateScheduler) Stats() (avgDuration, maxDuration time.Duration, updateCount uint64) {
	return mus.scheduler.AvgUpdateDuration(), mus.scheduler.MaxUpdateDuration(), mus.scheduler.stats.TotalUpdates.Load()
}
