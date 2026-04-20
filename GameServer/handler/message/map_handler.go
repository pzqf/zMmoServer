package message

import (
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
	zLog.Info("Map enter request", zap.Int64("player_id", int64(playerID)), zap.Int32("map_id", req.MapId))

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
	zLog.Info("Map attack request", zap.Int64("player_id", int64(playerID)), zap.Int64("target_id", int64(targetID)))

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
