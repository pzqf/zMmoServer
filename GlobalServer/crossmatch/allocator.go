// Package crossmatch 是**跨服活动的分配服务**：为一场跨服活动挑一台承载它的 MapServer，
// 并让所有 realm 的 GameServer 都拿到**同一个答案**。
//
// 为什么必须有它（realm §5.1 ③）：realm 架构下每个 realm 自成世界，GameServer 只发现本 realm 的
// MapServer。跨服玩法要把不同 realm 的玩家凑到同一张实例地图上，就需要一个**全区唯一的决策方**
// 来回答"这场活动放在哪台 MapServer 上"。GlobalServer 是全区全服进程（账号/服务器列表），
// 天然是这个决策方。
//
// 三条设计约束：
//   - **粘性**：同一 activityID 反复请求必须返回同一台服务器，否则各 realm 各进各的、玩家碰不到面。
//   - **确定性**：候选相同时选择必须可复现（先按人数、再按 serverID），便于排查。
//   - **故障转移**：粘住的那台掉线了要能改选，但只在它真的从服务发现里消失时才改。
//
// 本包只做"选服务器"，**不建实例**：实例由目标 MapServer 按 activityID 幂等创建
//（MSG_INTERNAL_CROSS_INSTANCE_ENSURE），所以这里不需要 GlobalServer→MapServer 的连接。
package crossmatch

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

const (
	// DefaultAllocationTTL 分配记录的空闲过期时间：超过这么久没人再请求这场活动就释放，
	// 下次请求会重新选服（活动已结束/无人参与）。
	DefaultAllocationTTL = 30 * time.Minute
	// serviceTypeMap 服务发现里 MapServer 的服务类型。
	serviceTypeMap = "map"
)

// Discoverer 服务发现的最小依赖面（便于单测注入假实现）。
// groupID 传空串 = **跨所有 realm** 发现（discovery.Discover 的既有能力）。
type Discoverer interface {
	Discover(serviceType string, groupID string) ([]*discovery.ServerInfo, error)
}

// Candidate 一台可承载跨服活动的 MapServer。
type Candidate struct {
	ServerID uint32
	GroupID  string
	Address  string // host:port，GameServer 可直接拨号
	Players  int
}

// Allocation 一场跨服活动的落点。
type Allocation struct {
	ActivityID  int64
	MapConfigID int32
	ServerID    uint32
	GroupID     string
	Address     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Allocator 跨服活动分配器。
type Allocator struct {
	mu          sync.Mutex
	discoverer  Discoverer
	allocations map[int64]*Allocation
	ttl         time.Duration
	now         func() time.Time // 可注入，便于测试过期逻辑
}

// NewAllocator 创建分配器。ttl<=0 取 DefaultAllocationTTL。
func NewAllocator(d Discoverer, ttl time.Duration) *Allocator {
	if ttl <= 0 {
		ttl = DefaultAllocationTTL
	}
	return &Allocator{
		discoverer:  d,
		allocations: make(map[int64]*Allocation),
		ttl:         ttl,
		now:         time.Now,
	}
}

// Allocate 为一场跨服活动选定承载的 MapServer（粘性 + 故障转移）。
func (a *Allocator) Allocate(activityID int64, mapConfigID int32) (*Allocation, error) {
	if activityID <= 0 {
		return nil, fmt.Errorf("invalid activity id %d", activityID)
	}
	if a.discoverer == nil {
		return nil, fmt.Errorf("cross allocator has no service discovery")
	}

	candidates, err := a.candidates()
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no healthy map server available for cross-server activity")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now()

	// 粘性：已分配且那台还在 → 原样返回（这是"所有 realm 落到同一张实例"的前提）。
	if existing, ok := a.allocations[activityID]; ok {
		for _, c := range candidates {
			if c.ServerID == existing.ServerID {
				existing.UpdatedAt = now
				// 地址可能因重启而变，跟随最新发现结果。
				existing.Address = c.Address
				existing.GroupID = c.GroupID
				return cloneAllocation(existing), nil
			}
		}
		zLog.Warn("Cross activity host map server disappeared, re-allocating",
			zap.Int64("activity_id", activityID),
			zap.Uint32("old_server_id", existing.ServerID))
		delete(a.allocations, activityID)
	}

	best := pickCandidate(candidates)
	alloc := &Allocation{
		ActivityID:  activityID,
		MapConfigID: mapConfigID,
		ServerID:    best.ServerID,
		GroupID:     best.GroupID,
		Address:     best.Address,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	a.allocations[activityID] = alloc

	zLog.Info("Cross activity allocated to map server",
		zap.Int64("activity_id", activityID),
		zap.Int32("map_config_id", mapConfigID),
		zap.Uint32("server_id", best.ServerID),
		zap.String("group_id", best.GroupID),
		zap.String("address", best.Address))

	return cloneAllocation(alloc), nil
}

// pickCandidate 确定性选择：人少优先，人数相同选 serverID 小的（可复现，便于排查）。
func pickCandidate(candidates []Candidate) Candidate {
	sorted := make([]Candidate, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Players != sorted[j].Players {
			return sorted[i].Players < sorted[j].Players
		}
		return sorted[i].ServerID < sorted[j].ServerID
	})
	return sorted[0]
}

// candidates 跨所有 realm 发现健康的 MapServer。
func (a *Allocator) candidates() ([]Candidate, error) {
	services, err := a.discoverer.Discover(serviceTypeMap, "") // "" = 不限 group，跨 realm
	if err != nil {
		return nil, fmt.Errorf("discover map servers: %w", err)
	}

	out := make([]Candidate, 0, len(services))
	for _, s := range services {
		if s == nil {
			continue
		}
		if status := string(s.Status); status != "healthy" && status != "ready" {
			continue
		}
		serverID := id.ParseServerIDString(s.ID)
		if serverID <= 0 {
			zLog.Warn("Skipping map server with unparseable id", zap.String("id", s.ID))
			continue
		}
		host := s.Address
		if host == "" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		if s.Port <= 0 {
			zLog.Warn("Skipping map server without port", zap.String("id", s.ID))
			continue
		}
		out = append(out, Candidate{
			ServerID: uint32(serverID),
			GroupID:  s.GroupID,
			Address:  fmt.Sprintf("%s:%d", host, s.Port),
			Players:  s.Players,
		})
	}
	return out, nil
}

// Release 显式释放一场活动的分配（活动结束）。
func (a *Allocator) Release(activityID int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.allocations, activityID)
}

// Cleanup 释放空闲超过 ttl 的分配，返回释放条数。
func (a *Allocator) Cleanup() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now()
	var dead []int64
	for activityID, alloc := range a.allocations {
		if now.Sub(alloc.UpdatedAt) > a.ttl {
			dead = append(dead, activityID)
		}
	}
	for _, activityID := range dead {
		delete(a.allocations, activityID)
	}
	if len(dead) > 0 {
		zLog.Info("Cross activity allocations expired", zap.Int("count", len(dead)))
	}
	return len(dead)
}

// List 当前所有分配（运维/排查用），按 activityID 升序。
func (a *Allocator) List() []*Allocation {
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make([]*Allocation, 0, len(a.allocations))
	for _, alloc := range a.allocations {
		out = append(out, cloneAllocation(alloc))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ActivityID < out[j].ActivityID })
	return out
}

// Count 当前分配数。
func (a *Allocator) Count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.allocations)
}

func cloneAllocation(a *Allocation) *Allocation {
	c := *a
	return &c
}

// —— 进程内单例（与 gameserverlist 同风格，供 HTTP handler 取用）——

var (
	allocator     *Allocator
	allocatorOnce sync.Once
)

// InitAllocator 初始化全局分配器（GlobalServer 启动时调用一次）。
func InitAllocator(d Discoverer) {
	allocatorOnce.Do(func() {
		allocator = NewAllocator(d, DefaultAllocationTTL)
		zLog.Info("Cross-server match allocator initialized")
	})
}

// GetAllocator 取全局分配器（未初始化返回 nil）。
func GetAllocator() *Allocator { return allocator }
