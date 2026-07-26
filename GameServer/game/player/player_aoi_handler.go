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

	// OldPos 不再承载：MapServer 分层 AOI 的 move 事件只回传当前(新)坐标，客户端只用 NewPos；
	// 原 OldPos 全链路无写入点、恒 (0,0,0)，故移除承载避免"看着有其实恒 0"的隐形字段。
	notify := &protocol.EntityMoveNotify{
		EntityId: req.TargetID,
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

// handleAOIAttr 视野内实体血量变更 → 推客户端 EntityAttrNotify。
// PosX=当前血量, PosY=最大血量（见 connection_manager.handleAOINotify 的 AOIAttr 透传）。
func (p *Player) handleAOIAttr(msg *PlayerMessage) {
	req, ok := msg.Data.(*AOIViewRequest)
	if !ok {
		return
	}
	notify := &protocol.EntityAttrNotify{
		EntityId: req.TargetID,
		CurHp:    int64(req.PosX),
		MaxHp:    int64(req.PosY),
	}
	data, err := proto.Marshal(notify)
	if err != nil {
		zLog.Error("Failed to marshal entity attr notify", zap.Error(err))
		return
	}
	p.pushToClient(int32(protocol.MapMsgId_MSG_MAP_ENTITY_ATTR), data)
}

// handleAOIDeath 视野内实体死亡 → 推客户端 EntityDeathNotify。PosX=killer 对象 ID。
func (p *Player) handleAOIDeath(msg *PlayerMessage) {
	req, ok := msg.Data.(*AOIViewRequest)
	if !ok {
		return
	}
	notify := &protocol.EntityDeathNotify{
		EntityId: req.TargetID,
		KillerId: int64(req.PosX),
	}
	data, err := proto.Marshal(notify)
	if err != nil {
		zLog.Error("Failed to marshal entity death notify", zap.Error(err))
		return
	}
	p.pushToClient(int32(protocol.MapMsgId_MSG_MAP_ENTITY_DEATH), data)
}

// handleAOIBuff 视野内实体 buff 增删 → 推客户端 EntityBuffNotify。
// PosX=buffID, PosY=added(1/0), PosZ=remaining_ms。
func (p *Player) handleAOIBuff(msg *PlayerMessage) {
	req, ok := msg.Data.(*AOIViewRequest)
	if !ok {
		return
	}
	notify := &protocol.EntityBuffNotify{
		EntityId:    req.TargetID,
		BuffId:      int32(req.PosX),
		Added:       req.PosY > 0.5,
		RemainingMs: int64(req.PosZ),
	}
	data, err := proto.Marshal(notify)
	if err != nil {
		zLog.Error("Failed to marshal entity buff notify", zap.Error(err))
		return
	}
	p.pushToClient(int32(protocol.MapMsgId_MSG_MAP_ENTITY_BUFF), data)
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
