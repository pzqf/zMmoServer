package message

import (
	"google.golang.org/protobuf/proto"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoServer/GameServer/game/maps"
	"go.uber.org/zap"
)

// MapHandler 地图消息处理器
type MapHandler struct {
	mapService *maps.MapService
}

// NewMapHandler 创建地图消息处理器
func NewMapHandler(mapService *maps.MapService) *MapHandler {
	return &MapHandler{
		mapService: mapService,
	}
}

// Handle 处理地图消息
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
		zLog.Info("Received unknown map message", zap.Int32("proto_id", protoId))
		return nil
	}
}

// handleMapEnter 处理玩家进入地图
func (h *MapHandler) handleMapEnter(session zNet.Session, data []byte) error {
	var req protocol.ClientMapEnterRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal map enter request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	mapID := id.MapIdType(req.MapId)
	zLog.Info("Map enter request", zap.Uint64("player_id", uint64(playerID)), zap.Int32("map_id", req.MapId))

	// 如果地图ID无效，使用默认地图
	if mapID <= 0 && h.mapService != nil {
		mapID = h.mapService.GetDefaultMapID()
	}

	pos := common.Vector3{X: 250, Y: 250, Z: 0}

	// 处理玩家进入地图
	if h.mapService != nil {
		err := h.mapService.HandlePlayerEnterMap(playerID, mapID, pos)
		if err != nil {
			zLog.Error("Failed to handle player enter map", zap.Error(err))
			// 发送进入地图失败的响应
			resp := &protocol.ClientMapEnterResponse{
				Result:   1,
				ErrorMsg: err.Error(),
			}
			respData, err := proto.Marshal(resp)
			if err != nil {
				zLog.Error("Failed to marshal map enter response", zap.Error(err))
				return err
			}
			err = session.Send(zNet.ProtoIdType(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE), respData)
			if err != nil {
				zLog.Error("Failed to send map enter response", zap.Error(err))
				return err
			}
			return nil
		}
	}

	// 发送进入地图成功的响应
	resp := &protocol.ClientMapEnterResponse{
		Result: 0,
		MapId:  int32(mapID),
		Pos: &protocol.Position{
			X: pos.X,
			Y: pos.Y,
			Z: pos.Z,
		},
	}

	// 序列化响应
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal map enter response", zap.Error(err))
		return err
	}

	// 发送响应
	err = session.Send(zNet.ProtoIdType(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE), respData)
	if err != nil {
		zLog.Error("Failed to send map enter response", zap.Error(err))
		return err
	}

	zLog.Info("Map enter response sent", zap.Uint64("player_id", uint64(playerID)), zap.Int32("map_id", int32(mapID)))
	return nil
}

// handleMapLeave 处理玩家离开地图
func (h *MapHandler) handleMapLeave(session zNet.Session, data []byte) error {
	var req protocol.ClientMapLeaveRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal map leave request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	zLog.Info("Map leave request", zap.Uint64("player_id", uint64(playerID)))

	// 处理玩家离开地图
	if h.mapService != nil {
		err := h.mapService.HandlePlayerLeaveMap(playerID, 0) // 暂时传递0作为地图ID
		if err != nil {
			zLog.Error("Failed to handle player leave map", zap.Error(err))
			// 发送离开地图失败的响应
			resp := &protocol.ClientMapLeaveResponse{
				Result: 1,
			}
			respData, err := proto.Marshal(resp)
			if err != nil {
				zLog.Error("Failed to marshal map leave response", zap.Error(err))
				return err
			}
			err = session.Send(zNet.ProtoIdType(protocol.MapMsgId_MSG_MAP_LEAVE_RESPONSE), respData)
			if err != nil {
				zLog.Error("Failed to send map leave response", zap.Error(err))
				return err
			}
			return nil
		}
	}

	// 发送离开地图成功的响应
	resp := &protocol.ClientMapLeaveResponse{
		Result: 0,
	}

	// 序列化响应
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal map leave response", zap.Error(err))
		return err
	}

	// 发送响应
	err = session.Send(zNet.ProtoIdType(protocol.MapMsgId_MSG_MAP_LEAVE_RESPONSE), respData)
	if err != nil {
		zLog.Error("Failed to send map leave response", zap.Error(err))
		return err
	}

	zLog.Info("Map leave response sent", zap.Uint64("player_id", uint64(playerID)))
	return nil
}

// handleMapMove 处理玩家移动
func (h *MapHandler) handleMapMove(session zNet.Session, data []byte) error {
	var req protocol.ClientMapMoveRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal map move request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	mapID := id.MapIdType(req.MapId)
	pos := common.Vector3{X: req.Pos.X, Y: req.Pos.Y, Z: req.Pos.Z}

	zLog.Info("Map move request",
		zap.Uint64("player_id", uint64(playerID)),
		zap.Int32("map_id", req.MapId),
		zap.Float32("x", req.Pos.X),
		zap.Float32("y", req.Pos.Y),
		zap.Float32("z", req.Pos.Z))

	// 处理玩家移动
	if h.mapService != nil {
		err := h.mapService.HandlePlayerMove(playerID, mapID, pos)
		if err != nil {
			zLog.Error("Failed to handle player move", zap.Error(err))
			// 发送移动失败的响应
			resp := &protocol.ClientMapMoveResponse{
				Result: 1,
			}
			respData, err := proto.Marshal(resp)
			if err != nil {
				zLog.Error("Failed to marshal map move response", zap.Error(err))
				return err
			}
			err = session.Send(zNet.ProtoIdType(protocol.MapMsgId_MSG_MAP_MOVE_RESPONSE), respData)
			if err != nil {
				zLog.Error("Failed to send map move response", zap.Error(err))
				return err
			}
			return nil
		}
	}

	// 发送移动成功的响应
	resp := &protocol.ClientMapMoveResponse{
		Result: 0,
		Pos: &protocol.Position{
			X: req.Pos.X,
			Y: req.Pos.Y,
			Z: req.Pos.Z,
		},
	}

	// 序列化响应
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal map move response", zap.Error(err))
		return err
	}

	// 发送响应
	err = session.Send(zNet.ProtoIdType(protocol.MapMsgId_MSG_MAP_MOVE_RESPONSE), respData)
	if err != nil {
		zLog.Error("Failed to send map move response", zap.Error(err))
		return err
	}

	zLog.Info("Map move response sent", zap.Uint64("player_id", uint64(playerID)))
	return nil
}

// handleMapAttack 处理玩家攻击
func (h *MapHandler) handleMapAttack(session zNet.Session, data []byte) error {
	var req protocol.ClientMapAttackRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal map attack request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	mapID := id.MapIdType(req.MapId)
	targetID := id.ObjectIdType(req.TargetId)

	zLog.Info("Map attack request",
		zap.Uint64("player_id", uint64(playerID)),
		zap.Int32("map_id", req.MapId),
		zap.Int64("target_id", req.TargetId))

	// 处理玩家攻击
	damage := int64(0)
	targetHP := int64(0)
	if h.mapService != nil {
		damage, targetHP, err := h.mapService.HandlePlayerAttack(playerID, mapID, targetID)
		if err != nil {
			zLog.Error("Failed to handle player attack", zap.Error(err))
			// 发送攻击失败的响应
			resp := &protocol.ClientMapAttackResponse{
				Result:   1,
				TargetId: req.TargetId,
			}
			respData, marshalErr := proto.Marshal(resp)
			if marshalErr != nil {
				zLog.Error("Failed to marshal map attack error response", zap.Error(marshalErr))
				return marshalErr
			}
			if sendErr := session.Send(zNet.ProtoIdType(protocol.MapMsgId_MSG_MAP_ATTACK_RESPONSE), respData); sendErr != nil {
				zLog.Error("Failed to send map attack error response", zap.Error(sendErr))
				return sendErr
			}
			return err
		}
		zLog.Info("Player attack handled", zap.Int64("damage", damage), zap.Int64("target_hp", targetHP))
	}

	// 发送攻击成功的响应
	resp := &protocol.ClientMapAttackResponse{
		Result:   0,
		TargetId: req.TargetId,
		Damage:   damage,
		TargetHp: targetHP,
	}

	// 序列化响应
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal map attack response", zap.Error(err))
		return err
	}

	// 发送响应
	err = session.Send(zNet.ProtoIdType(protocol.MapMsgId_MSG_MAP_ATTACK_RESPONSE), respData)
	if err != nil {
		zLog.Error("Failed to send map attack response", zap.Error(err))
		return err
	}

	zLog.Info("Map attack response sent", zap.Uint64("player_id", uint64(playerID)))
	return nil
}
