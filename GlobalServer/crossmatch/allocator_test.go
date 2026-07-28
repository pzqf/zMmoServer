package crossmatch

import (
	"errors"
	"testing"
	"time"

	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zEngine/zServer"
)

// fakeDiscoverer 可控的服务发现替身：记录被问到的 groupID，便于断言"确实是跨 realm 发现"。
type fakeDiscoverer struct {
	servers      []*discovery.ServerInfo
	err          error
	lastGroupID  string
	discoverCall int
}

func (f *fakeDiscoverer) Discover(serviceType string, groupID string) ([]*discovery.ServerInfo, error) {
	f.discoverCall++
	f.lastGroupID = groupID
	if f.err != nil {
		return nil, f.err
	}
	return f.servers, nil
}

func mapServer(idStr, groupID, host string, port, players int, status string) *discovery.ServerInfo {
	return &discovery.ServerInfo{
		ID:          idStr,
		ServiceType: "map",
		GroupID:     groupID,
		Status:      zServer.ServerState(status),
		Address:     host,
		Port:        port,
		Players:     players,
	}
}

// TestAllocator_DiscoversAcrossRealms 分配必须**跨 realm** 找候选：groupID 传空串。
// 传本 group 就退化成"只在本 realm 里选"，跨服活动永远凑不齐不同 realm 的玩家。
func TestAllocator_DiscoversAcrossRealms(t *testing.T) {
	d := &fakeDiscoverer{servers: []*discovery.ServerInfo{
		mapServer("000101", "1", "10.0.0.1", 9001, 0, "healthy"),
	}}
	a := NewAllocator(d, 0)

	if _, err := a.Allocate(7001, 5001); err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if d.lastGroupID != "" {
		t.Fatalf("必须以 groupID=\"\" 跨 realm 发现, got %q", d.lastGroupID)
	}
}

// TestAllocator_StickyPerActivity 同一活动反复请求必须落到同一台服务器——
// 这是"不同 realm 的玩家进同一张实例"的前提。即使后来出现了人更少的服务器也不许改选。
func TestAllocator_StickyPerActivity(t *testing.T) {
	d := &fakeDiscoverer{servers: []*discovery.ServerInfo{
		mapServer("000101", "1", "10.0.0.1", 9001, 10, "healthy"),
		mapServer("000201", "2", "10.0.0.2", 9002, 20, "healthy"),
	}}
	a := NewAllocator(d, 0)

	first, err := a.Allocate(7002, 5001)
	if err != nil {
		t.Fatalf("first Allocate: %v", err)
	}
	if first.ServerID != 101 {
		t.Fatalf("应选人最少的 101, got %d", first.ServerID)
	}

	// 出现一台更空的服务器，粘性仍应保持原选择。
	d.servers = append(d.servers, mapServer("000301", "3", "10.0.0.3", 9003, 0, "healthy"))
	second, err := a.Allocate(7002, 5001)
	if err != nil {
		t.Fatalf("second Allocate: %v", err)
	}
	if second.ServerID != first.ServerID {
		t.Fatalf("同一活动必须粘住同一台: first=%d second=%d", first.ServerID, second.ServerID)
	}

	// 另一场活动则可以选新的空服务器。
	other, err := a.Allocate(7003, 5001)
	if err != nil {
		t.Fatalf("other Allocate: %v", err)
	}
	if other.ServerID != 301 {
		t.Fatalf("新活动应选最空的 301, got %d", other.ServerID)
	}
}

// TestAllocator_FailoverWhenHostGone 粘住的那台从服务发现消失后必须改选，否则活动永远进不去。
func TestAllocator_FailoverWhenHostGone(t *testing.T) {
	d := &fakeDiscoverer{servers: []*discovery.ServerInfo{
		mapServer("000101", "1", "10.0.0.1", 9001, 0, "healthy"),
		mapServer("000201", "2", "10.0.0.2", 9002, 5, "healthy"),
	}}
	a := NewAllocator(d, 0)

	first, err := a.Allocate(7004, 5001)
	if err != nil {
		t.Fatalf("first Allocate: %v", err)
	}
	if first.ServerID != 101 {
		t.Fatalf("expected 101, got %d", first.ServerID)
	}

	// 101 下线。
	d.servers = []*discovery.ServerInfo{mapServer("000201", "2", "10.0.0.2", 9002, 5, "healthy")}
	second, err := a.Allocate(7004, 5001)
	if err != nil {
		t.Fatalf("failover Allocate: %v", err)
	}
	if second.ServerID != 201 {
		t.Fatalf("承载服务器消失后应改选存活的 201, got %d", second.ServerID)
	}
}

// TestAllocator_DeterministicTieBreak 人数相同时按 serverID 升序选，保证可复现。
func TestAllocator_DeterministicTieBreak(t *testing.T) {
	d := &fakeDiscoverer{servers: []*discovery.ServerInfo{
		mapServer("000301", "3", "10.0.0.3", 9003, 7, "healthy"),
		mapServer("000101", "1", "10.0.0.1", 9001, 7, "healthy"),
		mapServer("000201", "2", "10.0.0.2", 9002, 7, "healthy"),
	}}

	for i := 0; i < 5; i++ {
		a := NewAllocator(d, 0)
		alloc, err := a.Allocate(int64(8000+i), 5001)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		if alloc.ServerID != 101 {
			t.Fatalf("平票应选最小 serverID 101, got %d", alloc.ServerID)
		}
	}
}

// TestAllocator_SkipsUnhealthyAndPortless 不健康/没端口/ID 不可解析的候选一律跳过，
// 否则会把玩家分配到一台连不上的服务器。
func TestAllocator_SkipsUnhealthyAndPortless(t *testing.T) {
	d := &fakeDiscoverer{servers: []*discovery.ServerInfo{
		mapServer("000101", "1", "10.0.0.1", 9001, 0, "draining"),
		mapServer("000201", "2", "10.0.0.2", 0, 0, "healthy"),
		mapServer("bad-id", "3", "10.0.0.3", 9003, 0, "healthy"),
		mapServer("000401", "4", "0.0.0.0", 9004, 3, "ready"),
	}}
	a := NewAllocator(d, 0)

	alloc, err := a.Allocate(7005, 5001)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if alloc.ServerID != 401 {
		t.Fatalf("应只剩 401 可选, got %d", alloc.ServerID)
	}
	if alloc.Address != "127.0.0.1:9004" {
		t.Fatalf("0.0.0.0 应转成可拨号回环地址, got %q", alloc.Address)
	}
}

// TestAllocator_NoCandidates 无可用 MapServer 时明确报错（而不是返回一个空落点让上层去连空地址）。
func TestAllocator_NoCandidates(t *testing.T) {
	a := NewAllocator(&fakeDiscoverer{}, 0)
	if _, err := a.Allocate(7006, 5001); err == nil {
		t.Fatalf("无候选时应报错")
	}

	a2 := NewAllocator(&fakeDiscoverer{err: errors.New("etcd down")}, 0)
	if _, err := a2.Allocate(7007, 5001); err == nil {
		t.Fatalf("发现失败时应报错")
	}
}

// TestAllocator_CleanupExpired 空闲超时的分配被释放，之后可重新选服（活动已结束）。
func TestAllocator_CleanupExpired(t *testing.T) {
	d := &fakeDiscoverer{servers: []*discovery.ServerInfo{
		mapServer("000101", "1", "10.0.0.1", 9001, 0, "healthy"),
	}}
	a := NewAllocator(d, time.Minute)

	now := time.Now()
	a.now = func() time.Time { return now }

	if _, err := a.Allocate(7008, 5001); err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if a.Count() != 1 {
		t.Fatalf("expected 1 allocation, got %d", a.Count())
	}

	// 还没过期。
	a.now = func() time.Time { return now.Add(30 * time.Second) }
	if n := a.Cleanup(); n != 0 {
		t.Fatalf("未到期不应释放, got %d", n)
	}

	a.now = func() time.Time { return now.Add(2 * time.Minute) }
	if n := a.Cleanup(); n != 1 {
		t.Fatalf("到期应释放 1 条, got %d", n)
	}
	if a.Count() != 0 {
		t.Fatalf("释放后应为空, got %d", a.Count())
	}
}

// TestAllocator_ReleaseAndList 显式释放与列表输出。
func TestAllocator_ReleaseAndList(t *testing.T) {
	d := &fakeDiscoverer{servers: []*discovery.ServerInfo{
		mapServer("000101", "1", "10.0.0.1", 9001, 0, "healthy"),
	}}
	a := NewAllocator(d, 0)

	for _, activityID := range []int64{9002, 9001} {
		if _, err := a.Allocate(activityID, 5001); err != nil {
			t.Fatalf("Allocate: %v", err)
		}
	}
	list := a.List()
	if len(list) != 2 || list[0].ActivityID != 9001 {
		t.Fatalf("List 应按 activityID 升序: %+v", list)
	}

	a.Release(9001)
	if a.Count() != 1 {
		t.Fatalf("Release 后应剩 1 条, got %d", a.Count())
	}
}
