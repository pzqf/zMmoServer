package player

import (
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func (p *Player) handleAOIEnterView(msg *PlayerMessage) {
	req, ok := msg.Data.(*AOIViewRequest)
	if !ok {
		return
	}

	zLog.Debug("AOI enter view notification",
		zap.Int64("watcher_id", int64(req.WatcherID)),
		zap.Int64("target_id", req.TargetID),
		zap.Int32("map_id", int32(req.MapID)))

	notify := &protocol.EntityEnterViewNotify{
		EntityId: req.TargetID,
		Pos: &protocol.Position{
			X: req.PosX,
			Y: req.PosY,
			Z: req.PosZ,
		},
	}

	data, err := proto.Marshal(notify)
	if err != nil {
		zLog.Error("Failed to marshal enter view notify", zap.Error(err))
		return
	}

	p.pushToClient(int32(protocol.MapMsgId_MSG_MAP_ENTER_VIEW), data)
}

func (p *Player) handleAOILeaveView(msg *PlayerMessage) {
	req, ok := msg.Data.(*AOIViewRequest)
	if !ok {
		return
	}

	zLog.Debug("AOI leave view notification",
		zap.Int64("watcher_id", int64(req.WatcherID)),
		zap.Int64("target_id", req.TargetID))

	notify := &protocol.EntityLeaveViewNotify{
		EntityId: req.TargetID,
	}

	data, err := proto.Marshal(notify)
	if err != nil {
		zLog.Error("Failed to marshal leave view notify", zap.Error(err))
		return
	}

	p.pushToClient(int32(protocol.MapMsgId_MSG_MAP_LEAVE_VIEW), data)
}

func (p *Player) handleAOIMove(msg *PlayerMessage) {
	req, ok := msg.Data.(*AOIViewRequest)
	if !ok {
		return
	}

	notify := &protocol.EntityMoveNotify{
		EntityId: req.TargetID,
		OldPos: &protocol.Position{
			X: req.OldPosX,
			Y: req.OldPosY,
			Z: req.OldPosZ,
		},
		NewPos: &protocol.Position{
			X: req.PosX,
			Y: req.PosY,
			Z: req.PosZ,
		},
	}

	data, err := proto.Marshal(notify)
	if err != nil {
		zLog.Error("Failed to marshal move notify", zap.Error(err))
		return
	}

	p.pushToClient(int32(protocol.MapMsgId_MSG_MAP_ENTITY_MOVE), data)
}

// pushToClient 推送消息到客户端
func (p *Player) pushToClient(protoId int32, data []byte) {
	if p.clientSender == nil || p.sessionID == nil {
		zLog.Debug("Cannot push to client: no session",
			zap.Int64("player_id", int64(p.GetPlayerID())))
		return
	}

	if err := p.clientSender.SendToClient(p.sessionID, protoId, data); err != nil {
		zLog.Warn("Failed to push message to client",
			zap.Int64("player_id", int64(p.GetPlayerID())),
			zap.Int32("proto_id", protoId),
			zap.Error(err))
	}
}
