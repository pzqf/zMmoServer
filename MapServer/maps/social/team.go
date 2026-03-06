package social

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// TeamStatus 队伍状态
type TeamStatus int32

const (
	TeamStatusNormal   TeamStatus = 1 // 正常
	TeamStatusDisband  TeamStatus = 2 // 解散
	TeamStatusInviting TeamStatus = 3 // 邀请中
)

// TeamMember 队伍成员
type TeamMember struct {
	PlayerID   id.PlayerIdType `json:"player_id"`
	Name       string          `json:"name"`
	Level      int32           `json:"level"`
	Class      int32           `json:"class"`
	IsLeader   bool            `json:"is_leader"`
	JoinTime   time.Time       `json:"join_time"`
	Online     bool            `json:"online"`
}

// Team 队伍
type Team struct {
	TeamID      id.TeamIdType  `json:"team_id"`
	LeaderID    id.PlayerIdType `json:"leader_id"`
	Status      TeamStatus      `json:"status"`
	Members     []*TeamMember   `json:"members"`
	CreatedAt   time.Time       `json:"created_at"`
	LastActive  time.Time       `json:"last_active"`
	MaxMembers  int32           `json:"max_members"`
}

// TeamManager 队伍管理器
type TeamManager struct {
	mu     sync.RWMutex
	teams  map[id.TeamIdType]*Team
	playerTeams map[id.PlayerIdType]id.TeamIdType
}

// NewTeamManager 创建队伍管理器
func NewTeamManager() *TeamManager {
	return &TeamManager{
		teams:       make(map[id.TeamIdType]*Team),
		playerTeams: make(map[id.PlayerIdType]id.TeamIdType),
	}
}

// CreateTeam 创建队伍
func (tm *TeamManager) CreateTeam(leaderID id.PlayerIdType, leaderName string, leaderLevel, leaderClass int32) (*Team, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查玩家是否已在队伍中
	if _, exists := tm.playerTeams[leaderID]; exists {
		return nil, nil
	}

	// 生成队伍ID
	teamID := id.TeamIdType(time.Now().UnixNano() % 1000000000)

	// 创建队伍
	team := &Team{
		TeamID:     teamID,
		LeaderID:   leaderID,
		Status:     TeamStatusNormal,
		Members:    make([]*TeamMember, 0),
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		MaxMembers: 5, // 默认最大5人
	}

	// 添加队长到队伍
	leader := &TeamMember{
		PlayerID: leaderID,
		Name:     leaderName,
		Level:    leaderLevel,
		Class:    leaderClass,
		IsLeader: true,
		JoinTime: time.Now(),
		Online:   true,
	}
	team.Members = append(team.Members, leader)

	// 保存队伍和玩家队伍关系
	tm.teams[teamID] = team
	tm.playerTeams[leaderID] = teamID

	zLog.Debug("Team created",
		zap.Int64("team_id", int64(teamID)),
		zap.Int64("leader_id", int64(leaderID)),
		zap.String("leader_name", leaderName))

	return team, nil
}

// InviteToTeam 邀请玩家加入队伍
func (tm *TeamManager) InviteToTeam(teamID id.TeamIdType, inviterID, targetID id.PlayerIdType) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查队伍是否存在
	team, exists := tm.teams[teamID]
	if !exists {
		return nil
	}

	// 检查邀请者是否在队伍中
	inviterInTeam := false
	for _, member := range team.Members {
		if member.PlayerID == inviterID {
			inviterInTeam = true
			break
		}
	}
	if !inviterInTeam {
		return nil
	}

	// 检查目标玩家是否已在队伍中
	if _, exists := tm.playerTeams[targetID]; exists {
		return nil
	}

	// 检查队伍是否已满
	if int32(len(team.Members)) >= team.MaxMembers {
		return nil
	}

	// 这里可以添加邀请逻辑，比如发送邀请消息给目标玩家

	zLog.Debug("Player invited to team",
		zap.Int64("team_id", int64(teamID)),
		zap.Int64("inviter_id", int64(inviterID)),
		zap.Int64("target_id", int64(targetID)))

	return nil
}

// AcceptTeamInvite 接受队伍邀请
func (tm *TeamManager) AcceptTeamInvite(teamID id.TeamIdType, playerID id.PlayerIdType, playerName string, playerLevel, playerClass int32) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查队伍是否存在
	team, exists := tm.teams[teamID]
	if !exists {
		return nil
	}

	// 检查玩家是否已在队伍中
	if _, exists := tm.playerTeams[playerID]; exists {
		return nil
	}

	// 检查队伍是否已满
	if int32(len(team.Members)) >= team.MaxMembers {
		return nil
	}

	// 添加玩家到队伍
	member := &TeamMember{
		PlayerID: playerID,
		Name:     playerName,
		Level:    playerLevel,
		Class:    playerClass,
		IsLeader: false,
		JoinTime: time.Now(),
		Online:   true,
	}
	team.Members = append(team.Members, member)

	// 保存玩家队伍关系
	tm.playerTeams[playerID] = teamID

	// 更新队伍最后活动时间
	team.LastActive = time.Now()

	zLog.Debug("Player joined team",
		zap.Int64("team_id", int64(teamID)),
		zap.Int64("player_id", int64(playerID)),
		zap.String("player_name", playerName))

	return nil
}

// LeaveTeam 离开队伍
func (tm *TeamManager) LeaveTeam(playerID id.PlayerIdType) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查玩家是否在队伍中
	teamID, exists := tm.playerTeams[playerID]
	if !exists {
		return nil
	}

	// 检查队伍是否存在
	team, exists := tm.teams[teamID]
	if !exists {
		return nil
	}

	// 移除玩家从队伍
	newMembers := make([]*TeamMember, 0)
	isLeader := false
	for _, member := range team.Members {
		if member.PlayerID == playerID {
			isLeader = member.IsLeader
			continue
		}
		newMembers = append(newMembers, member)
	}
	team.Members = newMembers

	// 移除玩家队伍关系
	delete(tm.playerTeams, playerID)

	// 如果队长离开，重新分配队长
	if isLeader && len(team.Members) > 0 {
		team.LeaderID = team.Members[0].PlayerID
		team.Members[0].IsLeader = true
	}

	// 如果队伍为空，解散队伍
	if len(team.Members) == 0 {
		team.Status = TeamStatusDisband
		delete(tm.teams, teamID)
		zLog.Debug("Team disbanded due to no members", zap.Int64("team_id", int64(teamID)))
	} else {
		// 更新队伍最后活动时间
		team.LastActive = time.Now()
		zLog.Debug("Player left team",
			zap.Int64("team_id", int64(teamID)),
			zap.Int64("player_id", int64(playerID)))
	}

	return nil
}

// KickFromTeam 踢出队伍
func (tm *TeamManager) KickFromTeam(teamID id.TeamIdType, leaderID, targetID id.PlayerIdType) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查队伍是否存在
	team, exists := tm.teams[teamID]
	if !exists {
		return nil
	}

	// 检查是否是队长
	if team.LeaderID != leaderID {
		return nil
	}

	// 移除玩家从队伍
	newMembers := make([]*TeamMember, 0)
	for _, member := range team.Members {
		if member.PlayerID == targetID {
			continue
		}
		newMembers = append(newMembers, member)
	}

	// 如果目标玩家在队伍中
	if len(newMembers) < len(team.Members) {
		team.Members = newMembers

		// 移除玩家队伍关系
		delete(tm.playerTeams, targetID)

		// 更新队伍最后活动时间
		team.LastActive = time.Now()

		zLog.Debug("Player kicked from team",
			zap.Int64("team_id", int64(teamID)),
			zap.Int64("leader_id", int64(leaderID)),
			zap.Int64("target_id", int64(targetID)))
	}

	return nil
}

// DisbandTeam 解散队伍
func (tm *TeamManager) DisbandTeam(teamID id.TeamIdType, leaderID id.PlayerIdType) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查队伍是否存在
	team, exists := tm.teams[teamID]
	if !exists {
		return nil
	}

	// 检查是否是队长
	if team.LeaderID != leaderID {
		return nil
	}

	// 移除所有玩家的队伍关系
	for _, member := range team.Members {
		delete(tm.playerTeams, member.PlayerID)
	}

	// 标记队伍为解散状态
	team.Status = TeamStatusDisband

	// 移除队伍
	delete(tm.teams, teamID)

	zLog.Debug("Team disbanded",
		zap.Int64("team_id", int64(teamID)),
		zap.Int64("leader_id", int64(leaderID)))

	return nil
}

// GetTeam 获取队伍信息
func (tm *TeamManager) GetTeam(teamID id.TeamIdType) *Team {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.teams[teamID]
}

// GetPlayerTeam 获取玩家所在队伍
func (tm *TeamManager) GetPlayerTeam(playerID id.PlayerIdType) *Team {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	teamID, exists := tm.playerTeams[playerID]
	if !exists {
		return nil
	}

	return tm.teams[teamID]
}

// GetTeamMembers 获取队伍成员列表
func (tm *TeamManager) GetTeamMembers(teamID id.TeamIdType) []*TeamMember {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	team, exists := tm.teams[teamID]
	if !exists {
		return []*TeamMember{}
	}

	// 复制成员列表
	members := make([]*TeamMember, len(team.Members))
	copy(members, team.Members)

	return members
}

// UpdatePlayerStatus 更新玩家状态
func (tm *TeamManager) UpdatePlayerStatus(playerID id.PlayerIdType, online bool) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查玩家是否在队伍中
	teamID, exists := tm.playerTeams[playerID]
	if !exists {
		return nil
	}

	// 检查队伍是否存在
	team, exists := tm.teams[teamID]
	if !exists {
		return nil
	}

	// 更新玩家状态
	for _, member := range team.Members {
		if member.PlayerID == playerID {
			member.Online = online
			break
		}
	}

	// 更新队伍最后活动时间
	team.LastActive = time.Now()

	return nil
}