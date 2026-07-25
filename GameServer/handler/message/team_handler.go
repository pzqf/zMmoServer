package message

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"github.com/pzqf/zMmoServer/GameServer/game/team"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// TeamHandler 网关侧组队处理器（业务层建设 2026-07-25）。
// 引入一类新架构路径：**跨玩家共享的权威状态（队伍花名册）+ 只广播给成员子集**
// （用 PlayerManager.BroadcastMessageToPlayers/RouteMessage，而非聊天的全服扇出）。
// 建/加/离均为 fire；结果与最新花名册经 MSG_TEAM_UPDATE_NOTIFY 回推给相关成员。
type TeamHandler struct {
	playerManager *player.PlayerManager
	teamManager   *team.TeamManager // 本 handler 单例持有 → 队伍状态跨请求持久
	serverID      int32
}

func NewTeamHandler(pm *player.PlayerManager, serverID int32) *TeamHandler {
	h := &TeamHandler{playerManager: pm, teamManager: team.NewTeamManager(), serverID: serverID}
	// 玩家下线清理钩子（M1）：掉线/登出时把该玩家移出队伍并给剩余成员推最新花名册，
	// 避免 playerTeam/teams 悬挂 + 重登被锁"already in a team"。
	pm.RegisterOfflineHook(func(pid id.PlayerIdType) {
		if t, disbanded, err := h.teamManager.Leave(pid); err == nil && !disbanded {
			h.broadcastRoster(t.ID, t.Leader, t.Members)
		}
	})
	return h
}

func (h *TeamHandler) Handle(session zNet.Session, protoId int32, data []byte) error {
	switch protoId {
	case int32(protocol.TeamMsgId_MSG_TEAM_CREATE):
		var req protocol.ClientTeamCreateRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		t, err := h.teamManager.Create(id.PlayerIdType(req.PlayerId))
		if err != nil {
			zLog.Warn("Team create failed", zap.Int64("player", req.PlayerId), zap.Error(err))
			return nil
		}
		zLog.Info("Team created", zap.Int32("team", int32(t.ID)), zap.Int64("leader", req.PlayerId))
		h.broadcastRoster(t.ID, t.Leader, t.Members)

	case int32(protocol.TeamMsgId_MSG_TEAM_JOIN):
		var req protocol.ClientTeamJoinRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		t, err := h.teamManager.Join(id.TeamIdType(req.TeamId), id.PlayerIdType(req.PlayerId))
		if err != nil {
			zLog.Warn("Team join failed", zap.Int64("player", req.PlayerId), zap.Int32("team", req.TeamId), zap.Error(err))
			return nil
		}
		zLog.Info("Team joined", zap.Int32("team", int32(t.ID)), zap.Int64("player", req.PlayerId), zap.Int("size", len(t.Members)))
		h.broadcastRoster(t.ID, t.Leader, t.Members)

	case int32(protocol.TeamMsgId_MSG_TEAM_LEAVE):
		var req protocol.ClientTeamLeaveRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		t, disbanded, err := h.teamManager.Leave(id.PlayerIdType(req.PlayerId))
		if err != nil {
			zLog.Warn("Team leave failed", zap.Int64("player", req.PlayerId), zap.Error(err))
			return nil
		}
		// 离开者收到解散/离队通知（空花名册 + disbanded=true 表示对其而言已脱离队伍）
		h.pushLeft(id.PlayerIdType(req.PlayerId), t.ID)
		// 未解散则把最新花名册推给剩余成员
		if !disbanded {
			h.broadcastRoster(t.ID, t.Leader, t.Members)
		}
	}
	return nil
}

// broadcastRoster 把最新全量花名册推给该队全体成员（成员子集广播）。
func (h *TeamHandler) broadcastRoster(teamID id.TeamIdType, leader id.PlayerIdType, members []id.PlayerIdType) {
	out := h.buildNotify(teamID, leader, members, false)
	for _, m := range members {
		h.playerManager.RouteMessage(m,
			player.NewPlayerMessage(m, player.SourceGateway, player.MsgTeamUpdate, &player.TeamUpdateData{Data: out}))
	}
}

// pushLeft 通知单个离开者「已离队/队伍解散」。
func (h *TeamHandler) pushLeft(playerID id.PlayerIdType, teamID id.TeamIdType) {
	out := h.buildNotify(teamID, 0, nil, true)
	h.playerManager.RouteMessage(playerID,
		player.NewPlayerMessage(playerID, player.SourceGateway, player.MsgTeamUpdate, &player.TeamUpdateData{Data: out}))
}

// buildNotify 把花名册组装并 marshal 成一份 ClientTeamUpdateNotify 字节（成员名字经 pm 查）。
func (h *TeamHandler) buildNotify(teamID id.TeamIdType, leader id.PlayerIdType, members []id.PlayerIdType, disbanded bool) []byte {
	infos := make([]*protocol.TeamMemberInfo, 0, len(members))
	for _, m := range members {
		name := ""
		if p, err := h.playerManager.GetPlayer(m); err == nil && p != nil {
			name = p.GetName()
		}
		infos = append(infos, &protocol.TeamMemberInfo{PlayerId: int64(m), Name: name, IsLeader: m == leader})
	}
	out, err := proto.Marshal(&protocol.ClientTeamUpdateNotify{TeamId: int32(teamID), Members: infos, Disbanded: disbanded})
	if err != nil {
		zLog.Error("Failed to marshal team update notify", zap.Error(err))
		return nil
	}
	return out
}
