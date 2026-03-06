package team

import (
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// TeamManager 队伍管理器
type TeamManager struct {
	teams      map[id.TeamIdType]*Team
	playerTeam map[id.PlayerIdType]id.TeamIdType
	mutex      sync.RWMutex
}

// NewTeamManager 创建队伍管理器
func NewTeamManager() *TeamManager {
	return &TeamManager{
		teams:      make(map[id.TeamIdType]*Team),
		playerTeam: make(map[id.PlayerIdType]id.TeamIdType),
	}
}

// CreateTeam 创建队伍
func (tm *TeamManager) CreateTeam(leaderID id.PlayerIdType, leaderName string, leaderLevel, leaderClass int) *Team {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// 检查玩家是否已有队伍
	if _, exists := tm.playerTeam[leaderID]; exists {
		return nil
	}

	team := NewTeam(leaderID, leaderName, leaderLevel, leaderClass)
	tm.teams[team.TeamID] = team
	tm.playerTeam[leaderID] = team.TeamID

	zLog.Info("Team created",
		zap.Uint64("team_id", uint64(team.TeamID)),
		zap.Uint64("leader_id", uint64(leaderID)),
		zap.String("leader_name", leaderName))

	return team
}

// GetTeam 获取队伍
func (tm *TeamManager) GetTeam(teamID id.TeamIdType) *Team {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	return tm.teams[teamID]
}

// GetPlayerTeam 获取玩家所在队伍
func (tm *TeamManager) GetPlayerTeam(playerID id.PlayerIdType) *Team {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	if teamID, exists := tm.playerTeam[playerID]; exists {
		return tm.teams[teamID]
	}
	return nil
}

// AddMember 添加成员到队伍
func (tm *TeamManager) AddMember(teamID id.TeamIdType, playerID id.PlayerIdType, playerName string, level, class int) bool {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// 检查玩家是否已有队伍
	if _, exists := tm.playerTeam[playerID]; exists {
		return false
	}

	team := tm.teams[teamID]
	if team == nil {
		return false
	}

	if team.AddMember(playerID, playerName, level, class) {
		tm.playerTeam[playerID] = teamID
		zLog.Info("Player joined team",
			zap.Uint64("team_id", uint64(teamID)),
			zap.Uint64("player_id", uint64(playerID)),
			zap.String("player_name", playerName))
		return true
	}

	return false
}

// RemoveMember 从队伍移除成员
func (tm *TeamManager) RemoveMember(playerID id.PlayerIdType) bool {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	teamID, exists := tm.playerTeam[playerID]
	if !exists {
		return false
	}

	team := tm.teams[teamID]
	if team == nil {
		return false
	}

	if team.RemoveMember(playerID) {
		delete(tm.playerTeam, playerID)
		
		// 如果队伍为空，删除队伍
		if team.GetMemberCount() == 0 {
			delete(tm.teams, teamID)
			zLog.Info("Team disbanded", zap.Uint64("team_id", uint64(teamID)))
		} else {
			zLog.Info("Player left team",
				zap.Uint64("team_id", uint64(teamID)),
				zap.Uint64("player_id", uint64(playerID)))
		}
		return true
	}

	return false
}

// ChangeLeader 更换队长
func (tm *TeamManager) ChangeLeader(teamID id.TeamIdType, newLeaderID id.PlayerIdType) bool {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	team := tm.teams[teamID]
	if team == nil {
		return false
	}

	if team.ChangeLeader(newLeaderID) {
		zLog.Info("Team leader changed",
			zap.Uint64("team_id", uint64(teamID)),
			zap.Uint64("new_leader_id", uint64(newLeaderID)))
		return true
	}

	return false
}

// DisbandTeam 解散队伍
func (tm *TeamManager) DisbandTeam(teamID id.TeamIdType) bool {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	team := tm.teams[teamID]
	if team == nil {
		return false
	}

	// 移除所有成员的队伍关联
	for playerID := range team.Members {
		delete(tm.playerTeam, playerID)
	}

	delete(tm.teams, teamID)
	zLog.Info("Team disbanded", zap.Uint64("team_id", uint64(teamID)))
	return true
}

// SetOnlineStatus 设置玩家在线状态
func (tm *TeamManager) SetOnlineStatus(playerID id.PlayerIdType, isOnline bool) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	teamID, exists := tm.playerTeam[playerID]
	if !exists {
		return
	}

	team := tm.teams[teamID]
	if team != nil {
		team.SetOnlineStatus(playerID, isOnline)
	}
}

// GetTeamCount 获取队伍数量
func (tm *TeamManager) GetTeamCount() int {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	return len(tm.teams)
}

// GetPlayerCount 获取队伍总玩家数量
func (tm *TeamManager) GetPlayerCount() int {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	return len(tm.playerTeam)
}
