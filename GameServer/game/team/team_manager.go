// Package team 提供 GameServer 级的队伍(组队)共享状态管理。
//
// 这是本业务引入的一类新「跨玩家共享权威状态」——多个玩家 Actor / 网关 handler goroutine 并发
// 访问同一份队伍花名册,故内部用 RWMutex 保护;对外返回的均为花名册**快照**(脱锁后可安全读、
// 可安全跨 goroutine 广播)。队伍状态仅存于本 GameServer(同服成员);跨服组队需 Global 层,未做。
package team

import (
	"fmt"
	"sync"

	"github.com/pzqf/zCommon/common/id"
)

const maxTeamSize = 5

// Team 一支队伍(快照或内部态)。
type Team struct {
	ID      id.TeamIdType
	Leader  id.PlayerIdType
	Members []id.PlayerIdType
}

// TeamManager 队伍权威:建/加/离 + 花名册查询。一人一队。
type TeamManager struct {
	mu         sync.RWMutex
	teams      map[id.TeamIdType]*Team
	playerTeam map[id.PlayerIdType]id.TeamIdType
	nextID     int32 // 队伍 ID 由本 manager 内部自增分配(队伍为 GameServer 本地态,无需全局唯一)
}

func NewTeamManager() *TeamManager {
	return &TeamManager{
		teams:      make(map[id.TeamIdType]*Team),
		playerTeam: make(map[id.PlayerIdType]id.TeamIdType),
	}
}

// Create 建队,leader 入队。返回花名册快照。
func (tm *TeamManager) Create(leader id.PlayerIdType) (*Team, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if _, ok := tm.playerTeam[leader]; ok {
		return nil, fmt.Errorf("player %d already in a team", leader)
	}
	tm.nextID++
	tid := id.TeamIdType(tm.nextID)
	t := &Team{ID: tid, Leader: leader, Members: []id.PlayerIdType{leader}}
	tm.teams[tid] = t
	tm.playerTeam[leader] = tid
	return snapshot(t), nil
}

// Join 加入队伍。返回更新后的花名册快照。
func (tm *TeamManager) Join(teamID id.TeamIdType, playerID id.PlayerIdType) (*Team, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if _, ok := tm.playerTeam[playerID]; ok {
		return nil, fmt.Errorf("player %d already in a team", playerID)
	}
	t, ok := tm.teams[teamID]
	if !ok {
		return nil, fmt.Errorf("team %d not found", teamID)
	}
	if len(t.Members) >= maxTeamSize {
		return nil, fmt.Errorf("team %d full", teamID)
	}
	t.Members = append(t.Members, playerID)
	tm.playerTeam[playerID] = teamID
	return snapshot(t), nil
}

// Leave 离队。返回(剩余队伍快照或 nil, 是否解散, err)。最后一人离开→解散;队长离开→顺位接任。
func (tm *TeamManager) Leave(playerID id.PlayerIdType) (*Team, bool, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tid, ok := tm.playerTeam[playerID]
	if !ok {
		return nil, false, fmt.Errorf("player %d not in a team", playerID)
	}
	delete(tm.playerTeam, playerID)
	t := tm.teams[tid]
	for i, m := range t.Members {
		if m == playerID {
			t.Members = append(t.Members[:i], t.Members[i+1:]...)
			break
		}
	}
	if len(t.Members) == 0 {
		delete(tm.teams, tid)
		return &Team{ID: tid}, true, nil
	}
	if t.Leader == playerID {
		t.Leader = t.Members[0]
	}
	return snapshot(t), false, nil
}

// GetTeamIDOf 返回玩家所在队伍 ID。
func (tm *TeamManager) GetTeamIDOf(playerID id.PlayerIdType) (id.TeamIdType, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tid, ok := tm.playerTeam[playerID]
	return tid, ok
}

// snapshot 深拷贝一支队伍,脱锁后可安全读/广播。调用方须持锁。
func snapshot(t *Team) *Team {
	members := make([]id.PlayerIdType, len(t.Members))
	copy(members, t.Members)
	return &Team{ID: t.ID, Leader: t.Leader, Members: members}
}
