package message

import (
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoServer/GameServer/game/maps"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type MapHandler struct {
	mapService    *maps.MapService
	playerManager *player.PlayerManager
	serverID      int32
	// crossInFlight 正在进行跨服进图的玩家集合（playerID → struct{}）。
	// 跨服进图是异步的（见 handleCrossEnter），没有这道闸门的话客户端连点就能刷出一堆
	// goroutine，还可能把同一玩家并发绑到不同落点。
	crossInFlight sync.Map
}

func NewMapHandler(mapService *maps.MapService, playerManager *player.PlayerManager, serverID int32) *MapHandler {
	return &MapHandler{
		mapService:    mapService,
		playerManager: playerManager,
		serverID:      serverID,
	}
}

func (h *MapHandler) sendToClient(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, protoId int32, data []byte) error {
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
	wrappedData := crossserver.Wrap(meta, crossMsgData)

	return gwSession.Send(zNet.ProtoIdType(protoId), wrappedData)
}

func (h *MapHandler) Handle(session zNet.Session, protoId int32, data []byte) error {
	switch protoId {
	case int32(protocol.MapMsgId_MSG_MAP_ENTER):
		return h.handleMapEnter(session, data)
	case int32(protocol.MapMsgId_MSG_MAP_LEAVE):
		return h.handleMapLeave(session, data)
	case int32(protocol.MapMsgId_MSG_MAP_MOVE):
		return h.handleMapMove(session, data)
	case int32(protocol.MapMsgId_MSG_MAP_ATTACK):
		return h.handleMapAttack(session, data)
	case int32(protocol.MapMsgId_MSG_MAP_CROSS_ENTER):
		return h.handleCrossEnter(session, data)
	default:
		zLog.Warn("Unknown map message", zap.Int32("proto_id", protoId))
		return nil
	}
}

func (h *MapHandler) handleMapEnter(session zNet.Session, data []byte) error {
	var req protocol.ClientMapEnterRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal map enter request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	mapID := id.MapIdType(req.MapId)
	clientSessionID := getClientSessionID(session)
	zLog.Debug("Map enter request", zap.Int64("player_id", int64(playerID)), zap.Int32("map_id", req.MapId))

	if mapID <= 0 && h.mapService != nil {
		mapID = h.mapService.GetDefaultMapID()
	}

	pos := common.Vector3{X: 250, Y: 250, Z: 0}

	msg, callback := player.NewPlayerMessageWithCallback(
		playerID, player.SourceGateway, player.MsgNetMapEnter,
		&player.NetMapEnterRequest{
			PlayerID: playerID,
			MapID:    mapID,
			PosX:     pos.X,
			PosY:     pos.Y,
			PosZ:     pos.Z,
		},
	)

	if err := h.playerManager.RouteMessage(playerID, msg); err != nil {
		zLog.Error("Failed to route map enter message", zap.Error(err))
		h.sendMapEnterResponse(session, clientSessionID, playerID, 1, err.Error(), 0, nil)
		return nil
	}

	select {
	case resp := <-callback:
		if netResp, ok := resp.(*player.NetResponse); ok {
			return h.sendToClient(session, clientSessionID, playerID, int32(netResp.ProtoId), netResp.Data)
		}
		if errResp, ok := resp.(*player.BaseResponse); ok && !errResp.Success {
			h.sendMapEnterResponse(session, clientSessionID, playerID, 1, errResp.Error, 0, nil)
			return nil
		}
		h.sendMapEnterResponse(session, clientSessionID, playerID, 0, "", int32(mapID), &protocol.Position{X: pos.X, Y: pos.Y, Z: pos.Z})
	case <-time.After(5 * time.Second):
		zLog.Warn("Map enter timeout", zap.Int64("player_id", int64(playerID)))
		h.sendMapEnterResponse(session, clientSessionID, playerID, 1, "timeout", 0, nil)
	}

	return nil
}

// handleCrossEnter 进跨服活动实例（realm §5.1）。
//
// **必须异步**：解析落点要打 GlobalServer 的 HTTP、连外域 MapServer、等它建实例，可达数秒。
// 而 zNet 的会话是**每连接一个 goroutine 且收发共用**（TcpServerSession.process 同时消费
// receiveChan 与 sendChan）——Gateway↔GameServer 又是**一条连接承载全服玩家**。在这里同步等，
// 等于把该 GameServer 上所有玩家的收发一起卡住数秒。故本函数立刻返回，慢活丢给独立 goroutine，
// 完成后再把响应推给客户端（客户端本就是等 MSG_MAP_CROSS_ENTER_RESPONSE，天然容得下异步）。
func (h *MapHandler) handleCrossEnter(session zNet.Session, data []byte) error {
	var req protocol.ClientCrossEnterRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal cross enter request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	clientSessionID := getClientSessionID(session)
	zLog.Info("Cross enter request",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("activity_id", req.ActivityId),
		zap.Int32("map_config_id", req.MapConfigId))

	// 同一玩家同时只允许一次进行中的跨服进图：既防客户端连点刷出一堆 goroutine，
	// 也防同一玩家并发绑定到不同落点。
	if _, busy := h.crossInFlight.LoadOrStore(int64(playerID), struct{}{}); busy {
		h.sendCrossEnterResponse(session, clientSessionID, playerID, 1, "cross enter already in progress", req.ActivityId)
		return nil
	}

	go func() {
		defer h.crossInFlight.Delete(int64(playerID))
		h.runCrossEnter(session, clientSessionID, playerID, req.ActivityId, req.MapConfigId)
	}()

	return nil
}

// runCrossEnter 在独立 goroutine 上跑完整条跨服进图：慢的落点解析在 actor 之外，
// 快的绑定+进图在玩家 actor 上（铁律1：改玩家状态只在 actor 内）。
func (h *MapHandler) runCrossEnter(session zNet.Session, clientSessionID zNet.SessionIdType,
	playerID id.PlayerIdType, activityID int64, mapConfigID int32) {

	if h.mapService == nil {
		h.sendCrossEnterResponse(session, clientSessionID, playerID, 1, "map service not available", activityID)
		return
	}

	// 慢：问 GlobalServer 分配 → 连外域 MapServer → 请它建/取实例。不碰玩家状态。
	target, err := h.mapService.ResolveCrossActivity(activityID, mapConfigID)
	if err != nil {
		zLog.Error("Failed to resolve cross activity",
			zap.Int64("player_id", int64(playerID)),
			zap.Int64("activity_id", activityID), zap.Error(err))
		h.sendCrossEnterResponse(session, clientSessionID, playerID, 1, err.Error(), activityID)
		return
	}

	// 快：投递到玩家 actor 完成绑定 + 发进图请求。
	pos := common.Vector3{X: 250, Y: 250, Z: 0}
	msg, callback := player.NewPlayerMessageWithCallback(
		playerID, player.SourceGateway, player.MsgNetMapCrossEnter,
		&player.NetCrossEnterRequest{
			PlayerID:    playerID,
			ActivityID:  activityID,
			MapServerID: target.MapServerID,
			MapID:       target.MapID,
			PosX:        pos.X,
			PosY:        pos.Y,
			PosZ:        pos.Z,
		},
	)

	if err := h.playerManager.RouteMessage(playerID, msg); err != nil {
		zLog.Error("Failed to route cross enter message", zap.Error(err))
		h.sendCrossEnterResponse(session, clientSessionID, playerID, 1, err.Error(), activityID)
		return
	}

	select {
	case resp := <-callback:
		if netResp, ok := resp.(*player.NetResponse); ok {
			if err := h.sendToClient(session, clientSessionID, playerID, int32(netResp.ProtoId), netResp.Data); err != nil {
				zLog.Error("Failed to send cross enter response", zap.Error(err))
			}
			return
		}
		if errResp, ok := resp.(*player.BaseResponse); ok && !errResp.Success {
			h.sendCrossEnterResponse(session, clientSessionID, playerID, 1, errResp.Error, activityID)
			return
		}
		h.sendCrossEnterResponse(session, clientSessionID, playerID, 1, "unexpected response", activityID)
	case <-time.After(5 * time.Second):
		zLog.Warn("Cross enter attach timeout", zap.Int64("player_id", int64(playerID)))
		h.sendCrossEnterResponse(session, clientSessionID, playerID, 1, "timeout", activityID)
	}
}

func (h *MapHandler) sendCrossEnterResponse(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, result int32, errMsg string, activityID int64) {
	resp := &protocol.ClientCrossEnterResponse{
		Result:     result,
		ErrorMsg:   errMsg,
		ActivityId: activityID,
	}
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal cross enter response", zap.Error(err))
		return
	}
	if err := h.sendToClient(gwSession, clientSessionID, playerID, int32(protocol.MapMsgId_MSG_MAP_CROSS_ENTER_RESPONSE), respData); err != nil {
		zLog.Error("Failed to send cross enter response", zap.Error(err))
	}
}

func (h *MapHandler) handleMapLeave(session zNet.Session, data []byte) error {
	var req protocol.ClientMapLeaveRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal map leave request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	mapID := id.MapIdType(req.MapId)
	clientSessionID := getClientSessionID(session)
	zLog.Info("Map leave request", zap.Int64("player_id", int64(playerID)))

	msg, callback := player.NewPlayerMessageWithCallback(
		playerID, player.SourceGateway, player.MsgNetMapLeave,
		&player.NetMapLeaveRequest{PlayerID: playerID, MapID: mapID},
	)

	if err := h.playerManager.RouteMessage(playerID, msg); err != nil {
		zLog.Error("Failed to route map leave message", zap.Error(err))
		h.sendMapLeaveResponse(session, clientSessionID, playerID, 1)
		return nil
	}

	select {
	case resp := <-callback:
		if netResp, ok := resp.(*player.NetResponse); ok {
			return h.sendToClient(session, clientSessionID, playerID, int32(netResp.ProtoId), netResp.Data)
		}
		h.sendMapLeaveResponse(session, clientSessionID, playerID, 0)
	case <-time.After(5 * time.Second):
		zLog.Warn("Map leave timeout", zap.Int64("player_id", int64(playerID)))
		h.sendMapLeaveResponse(session, clientSessionID, playerID, 0)
	}

	return nil
}

func (h *MapHandler) handleMapMove(session zNet.Session, data []byte) error {
	var req protocol.ClientMapMoveRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal map move request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	mapID := id.MapIdType(req.MapId)
	clientSessionID := getClientSessionID(session)
	zLog.Debug("Map move request", zap.Int64("player_id", int64(playerID)), zap.Int32("map_id", req.MapId))

	msg, callback := player.NewPlayerMessageWithCallback(
		playerID, player.SourceGateway, player.MsgNetMapMove,
		&player.NetMapMoveRequest{
			PlayerID: playerID,
			MapID:    mapID,
			PosX:     req.Pos.X,
			PosY:     req.Pos.Y,
			PosZ:     req.Pos.Z,
		},
	)

	if err := h.playerManager.RouteMessage(playerID, msg); err != nil {
		zLog.Error("Failed to route map move message", zap.Error(err))
		h.sendMapMoveResponse(session, clientSessionID, playerID, 1, nil)
		return nil
	}

	select {
	case resp := <-callback:
		if netResp, ok := resp.(*player.NetResponse); ok {
			return h.sendToClient(session, clientSessionID, playerID, int32(netResp.ProtoId), netResp.Data)
		}
		h.sendMapMoveResponse(session, clientSessionID, playerID, 0, &protocol.Position{X: req.Pos.X, Y: req.Pos.Y, Z: req.Pos.Z})
	case <-time.After(3 * time.Second):
		zLog.Warn("Map move timeout", zap.Int64("player_id", int64(playerID)))
		h.sendMapMoveResponse(session, clientSessionID, playerID, 1, nil)
	}

	return nil
}

func (h *MapHandler) handleMapAttack(session zNet.Session, data []byte) error {
	var req protocol.ClientMapAttackRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal map attack request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	mapID := id.MapIdType(req.MapId)
	targetID := id.ObjectIdType(req.TargetId)
	clientSessionID := getClientSessionID(session)
	zLog.Debug("Map attack request", zap.Int64("player_id", int64(playerID)), zap.Int64("target_id", int64(targetID)))

	msg, callback := player.NewPlayerMessageWithCallback(
		playerID, player.SourceGateway, player.MsgNetMapAttack,
		&player.NetMapAttackRequest{
			PlayerID: playerID,
			MapID:    mapID,
			TargetID: targetID,
		},
	)

	if err := h.playerManager.RouteMessage(playerID, msg); err != nil {
		zLog.Error("Failed to route map attack message", zap.Error(err))
		h.sendMapAttackResponse(session, clientSessionID, playerID, 1, req.TargetId, 0, 0)
		return nil
	}

	select {
	case resp := <-callback:
		if netResp, ok := resp.(*player.NetResponse); ok {
			return h.sendToClient(session, clientSessionID, playerID, int32(netResp.ProtoId), netResp.Data)
		}
		h.sendMapAttackResponse(session, clientSessionID, playerID, 0, req.TargetId, 0, 0)
	case <-time.After(5 * time.Second):
		zLog.Warn("Map attack timeout", zap.Int64("player_id", int64(playerID)))
		h.sendMapAttackResponse(session, clientSessionID, playerID, 1, req.TargetId, 0, 0)
	}

	return nil
}

func (h *MapHandler) sendMapEnterResponse(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, result int32, errMsg string, mapID int32, pos *protocol.Position) {
	resp := &protocol.ClientMapEnterResponse{
		Result:   result,
		ErrorMsg: errMsg,
		MapId:    mapID,
		Pos:      pos,
	}
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal map enter response", zap.Error(err))
		return
	}
	if err := h.sendToClient(gwSession, clientSessionID, playerID, int32(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE), respData); err != nil {
		zLog.Error("Failed to send map enter response", zap.Error(err))
	}
}

func (h *MapHandler) sendMapLeaveResponse(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, result int32) {
	resp := &protocol.ClientMapLeaveResponse{Result: result}
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal map leave response", zap.Error(err))
		return
	}
	if err := h.sendToClient(gwSession, clientSessionID, playerID, int32(protocol.MapMsgId_MSG_MAP_LEAVE_RESPONSE), respData); err != nil {
		zLog.Error("Failed to send map leave response", zap.Error(err))
	}
}

func (h *MapHandler) sendMapMoveResponse(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, result int32, pos *protocol.Position) {
	resp := &protocol.ClientMapMoveResponse{Result: result, Pos: pos}
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal map move response", zap.Error(err))
		return
	}
	if err := h.sendToClient(gwSession, clientSessionID, playerID, int32(protocol.MapMsgId_MSG_MAP_MOVE_RESPONSE), respData); err != nil {
		zLog.Error("Failed to send map move response", zap.Error(err))
	}
}

func (h *MapHandler) sendMapAttackResponse(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, result int32, targetID int64, damage int64, targetHP int64) {
	resp := &protocol.ClientMapAttackResponse{
		Result:   result,
		TargetId: targetID,
		Damage:   damage,
		TargetHp: targetHP,
	}
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal map attack response", zap.Error(err))
		return
	}
	if err := h.sendToClient(gwSession, clientSessionID, playerID, int32(protocol.MapMsgId_MSG_MAP_ATTACK_RESPONSE), respData); err != nil {
		zLog.Error("Failed to send map attack response", zap.Error(err))
	}
}
