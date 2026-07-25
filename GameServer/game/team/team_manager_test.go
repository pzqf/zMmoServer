package team

import (
	"sync"
	"testing"

	"github.com/pzqf/zCommon/common/id"
)

func TestTeamManager_CreateJoinLeave(t *testing.T) {
	tm := NewTeamManager()
	leader := id.PlayerIdType(1)

	tt, err := tm.Create(leader)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(tt.Members) != 1 || tt.Leader != leader {
		t.Fatalf("建队快照错误: members=%d leader=%d", len(tt.Members), tt.Leader)
	}
	if _, err := tm.Create(leader); err == nil {
		t.Fatalf("重复建队应失败")
	}

	m2 := id.PlayerIdType(2)
	j, err := tm.Join(tt.ID, m2)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}
	if len(j.Members) != 2 {
		t.Fatalf("加入后应 2 人, got %d", len(j.Members))
	}
	if _, err := tm.Join(id.TeamIdType(999999), m2); err == nil {
		t.Fatalf("加入不存在的队应失败")
	}

	// 队长离开 → m2 接任,不解散
	left, disbanded, err := tm.Leave(leader)
	if err != nil {
		t.Fatalf("Leave leader: %v", err)
	}
	if disbanded {
		t.Fatalf("尚有 1 人不应解散")
	}
	if left.Leader != m2 {
		t.Fatalf("队长应顺位给 m2, got %d", left.Leader)
	}

	// 最后一人离开 → 解散
	_, disbanded2, err := tm.Leave(m2)
	if err != nil {
		t.Fatalf("Leave m2: %v", err)
	}
	if !disbanded2 {
		t.Fatalf("最后一人离开应解散")
	}
}

// TestTeamManager_ConcurrentNoRace 多 goroutine 并发建/加/离/查,-race 下应零竞争。
func TestTeamManager_ConcurrentNoRace(t *testing.T) {
	tm := NewTeamManager()
	base, err := tm.Create(id.PlayerIdType(1000))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		pid := id.PlayerIdType(2000 + i)
		go func() {
			defer wg.Done()
			_, _ = tm.Join(base.ID, pid) // 超 maxTeamSize 的会失败,无妨
			_, _ = tm.GetTeamIDOf(pid)
			_, _, _ = tm.Leave(pid)
		}()
	}
	wg.Wait()
}
