package message

import (
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// SkillHandler 网关侧技能消息处理器（业务层建设 2026-07-25）。同 ItemHandler 的转发/回调模式。
type SkillHandler struct {
	playerManager *player.PlayerManager
	serverID      int32
}

func NewSkillHandler(playerManager *player.PlayerManager, serverID int32) *SkillHandler {
	return &SkillHandler{playerManager: playerManager, serverID: serverID}
}

func (h *SkillHandler) sendToClient(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, protoId int32, data []byte) error {
	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: uint64(clientSessionID),
		PlayerId:  uint64(playerID),
		ServerId:  uint32(h.serverID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}
	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(h.serverID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeGateway),
		Message:      baseMsg,
	}
	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		zLog.Error("Failed to marshal cross server message", zap.Error(err))
		return err
	}
	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, h.serverID)
	return gwSession.Send(zNet.ProtoIdType(protoId), crossserver.Wrap(meta, crossMsgData))
}

func (h *SkillHandler) dispatch(session zNet.Session, playerID id.PlayerIdType, msgType player.MessageType, req interface{}) error {
	clientSessionID := getClientSessionID(session)
	msg, callback := player.NewPlayerMessageWithCallback(playerID, player.SourceGateway, msgType, req)
	if err := h.playerManager.RouteMessage(playerID, msg); err != nil {
		zLog.Error("Failed to route skill message", zap.Error(err), zap.Int64("player_id", int64(playerID)))
		return nil
	}
	select {
	case resp := <-callback:
		if netResp, ok := resp.(*player.NetResponse); ok {
			return h.sendToClient(session, clientSessionID, playerID, int32(netResp.ProtoId), netResp.Data)
		}
	case <-time.After(3 * time.Second):
		zLog.Warn("Skill message timeout", zap.Uint64("player_id", uint64(playerID)))
	}
	return nil
}

func (h *SkillHandler) Handle(session zNet.Session, protoId int32, data []byte) error {
	switch protoId {
	case int32(protocol.SkillMsgId_MSG_SKILL_LIST):
		var req protocol.ClientSkillListRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		return h.dispatch(session, id.PlayerIdType(req.PlayerId), player.MsgNetSkillList, &req)
	case int32(protocol.SkillMsgId_MSG_SKILL_LEARN):
		var req protocol.ClientSkillLearnRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		return h.dispatch(session, id.PlayerIdType(req.PlayerId), player.MsgNetSkillLearn, &req)
	case int32(protocol.SkillMsgId_MSG_SKILL_UPGRADE):
		var req protocol.ClientSkillUpgradeRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		return h.dispatch(session, id.PlayerIdType(req.PlayerId), player.MsgNetSkillUpgrade, &req)
	case int32(protocol.SkillMsgId_MSG_SKILL_CAST):
		var req protocol.ClientSkillCastRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		return h.dispatch(session, id.PlayerIdType(req.PlayerId), player.MsgNetSkillCast, &req)
	default:
		zLog.Warn("Unknown skill message", zap.Int32("proto_id", protoId))
		return nil
	}
}
