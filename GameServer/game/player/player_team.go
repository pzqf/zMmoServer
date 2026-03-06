package player

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/team"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// PlayerTeam 玩家队伍组件
type PlayerTeam struct {
	playerID   id.PlayerIdType
	teamID     id.TeamIdType
	isLeader   bool
	teamData   *team.Team
}

// NewPlayerTeam 创建玩家队伍组件
func NewPlayerTeam(playerID id.PlayerIdType) *PlayerTeam {
	return &PlayerTeam{
		playerID: playerID,
		teamID:   0,
		isLeader: false,
		teamData: nil,
	}
}

// GetTeamID 获取队伍ID
func (pt *PlayerTeam) GetTeamID() id.TeamIdType {
	return pt.teamID
}

// SetTeamID 设置队伍ID
func (pt *PlayerTeam) SetTeamID(teamID id.TeamIdType) {
	pt.teamID = teamID
}

// IsInTeam 是否在队伍中
func (pt *PlayerTeam) IsInTeam() bool {
	return pt.teamID > 0
}

// IsTeamLeader 是否是队长
func (pt *PlayerTeam) IsTeamLeader() bool {
	if pt.teamData == nil {
		return false
	}
	return pt.teamData.LeaderID == pt.playerID
}

// GetTeamData 获取队伍数据
func (pt *PlayerTeam) GetTeamData() *team.Team {
	return pt.teamData
}

// SetTeamData 设置队伍数据
func (pt *PlayerTeam) SetTeamData(teamData *team.Team) {
	pt.teamData = teamData
	if teamData != nil {
		pt.teamID = teamData.TeamID
		pt.isLeader = teamData.LeaderID == pt.playerID
		zLog.Info("Player joined team",
			zap.Uint64("player_id", uint64(pt.playerID)),
			zap.Uint64("team_id", uint64(pt.teamID)),
			zap.Bool("is_leader", pt.isLeader))
	} else {
		pt.teamID = 0
		pt.isLeader = false
	}
}

// LeaveTeam 离开队伍
func (pt *PlayerTeam) LeaveTeam() {
	if pt.teamID > 0 {
		zLog.Info("Player left team",
			zap.Uint64("player_id", uint64(pt.playerID)),
			zap.Uint64("team_id", uint64(pt.teamID)))
	}
	pt.teamID = 0
	pt.isLeader = false
	pt.teamData = nil
}
