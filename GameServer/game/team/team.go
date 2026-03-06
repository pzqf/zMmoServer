package team

import (
	"github.com/pzqf/zMmoShared/common/id"
	"time"
)

// Team 队伍结构
type Team struct {
	TeamID        id.TeamIdType   // 队伍ID
	LeaderID      id.PlayerIdType  // 队长ID
	Members       map[id.PlayerIdType]*TeamMember // 成员列表
	MaxMembers    int              // 最大成员数
	IsAutoAccept  bool             // 是否自动接受邀请
	IsLocked      bool             // 是否锁定队伍
	CreateTime    time.Time        // 创建时间
	LastActivity  time.Time        // 最后活动时间
}

// TeamMember 队伍成员
type TeamMember struct {
	PlayerID    id.PlayerIdType // 玩家ID
	PlayerName  string          // 玩家名称
	Level       int             // 等级
	Class       int             // 职业
	IsOnline    bool            // 是否在线
	JoinTime    time.Time       // 加入时间
}

// NewTeam 创建新队伍
func NewTeam(leaderID id.PlayerIdType, leaderName string, leaderLevel, leaderClass int) *Team {
	team := &Team{
		TeamID:        id.TeamIdType(id.GenerateId()),
		LeaderID:      leaderID,
		Members:       make(map[id.PlayerIdType]*TeamMember),
		MaxMembers:    5, // 默认5人队伍
		IsAutoAccept:  false,
		IsLocked:      false,
		CreateTime:    time.Now(),
		LastActivity:  time.Now(),
	}

	// 添加队长到队伍
	team.AddMember(leaderID, leaderName, leaderLevel, leaderClass)

	return team
}

// AddMember 添加成员
func (t *Team) AddMember(playerID id.PlayerIdType, playerName string, level, class int) bool {
	if len(t.Members) >= t.MaxMembers {
		return false
	}

	if _, exists := t.Members[playerID]; exists {
		return false
	}

	t.Members[playerID] = &TeamMember{
		PlayerID:    playerID,
		PlayerName:  playerName,
		Level:       level,
		Class:       class,
		IsOnline:    true,
		JoinTime:    time.Now(),
	}

	t.LastActivity = time.Now()
	return true
}

// RemoveMember 移除成员
func (t *Team) RemoveMember(playerID id.PlayerIdType) bool {
	if _, exists := t.Members[playerID]; !exists {
		return false
	}

	delete(t.Members, playerID)
	t.LastActivity = time.Now()

	// 如果队长离开，重新选举队长
	if playerID == t.LeaderID && len(t.Members) > 0 {
		for memberID := range t.Members {
			t.LeaderID = memberID
			break
		}
	}

	return true
}

// ChangeLeader 更换队长
func (t *Team) ChangeLeader(newLeaderID id.PlayerIdType) bool {
	if _, exists := t.Members[newLeaderID]; !exists {
		return false
	}

	t.LeaderID = newLeaderID
	t.LastActivity = time.Now()
	return true
}

// IsMember 检查是否是队伍成员
func (t *Team) IsMember(playerID id.PlayerIdType) bool {
	_, exists := t.Members[playerID]
	return exists
}

// GetMemberCount 获取成员数量
func (t *Team) GetMemberCount() int {
	return len(t.Members)
}

// GetMember 获取成员信息
func (t *Team) GetMember(playerID id.PlayerIdType) *TeamMember {
	return t.Members[playerID]
}

// SetOnlineStatus 设置成员在线状态
func (t *Team) SetOnlineStatus(playerID id.PlayerIdType, isOnline bool) {
	if member, exists := t.Members[playerID]; exists {
		member.IsOnline = isOnline
		t.LastActivity = time.Now()
	}
}

// SetAutoAccept 设置自动接受邀请
func (t *Team) SetAutoAccept(autoAccept bool) {
	t.IsAutoAccept = autoAccept
	t.LastActivity = time.Now()
}

// SetLocked 设置队伍锁定状态
func (t *Team) SetLocked(locked bool) {
	t.IsLocked = locked
	t.LastActivity = time.Now()
}
