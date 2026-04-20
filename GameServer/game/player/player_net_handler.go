package player

import (
	"fmt"

	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func (p *Player) handleNetEnterGame(msg *PlayerMessage) {
	req, ok := msg.Data.(*NetEnterGameRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	zLog.Info("Player entering game via Actor",
		zap.Int64("player_id", int64(req.PlayerID)))

	resp := &protocol.PlayerLoginResponse{
		Result: 0,
		PlayerInfo: &protocol.PlayerBasicInfo{
			PlayerId: int64(p.GetPlayerID()),
			Name:     p.GetName(),
			Level:    int32(p.attrs.GetLevel()),
			Gold:     p.GetGold(),
		},
	}

	respData, err := proto.Marshal(resp)
	if err != nil {
		p.sendErrorResponse(msg, fmt.Sprintf("marshal error: %v", err))
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &NetResponse{
			ProtoId: int32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE),
			Data:    respData,
		}
	}
}

func (p *Player) handleNetLeaveGame(msg *PlayerMessage) {
	req, ok := msg.Data.(*NetLeaveGameRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	zLog.Info("Player leaving game via Actor",
		zap.Int64("player_id", int64(req.PlayerID)))

	resp := &protocol.CommonResponse{Result: 0}
	respData, err := proto.Marshal(resp)
	if err != nil {
		p.sendErrorResponse(msg, fmt.Sprintf("marshal error: %v", err))
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &NetResponse{
			ProtoId: int32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME),
			Data:    respData,
		}
	}
}

func (p *Player) handleNetMapEnter(msg *PlayerMessage) {
	req, ok := msg.Data.(*NetMapEnterRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	pos := common.Vector3{X: req.PosX, Y: req.PosY, Z: req.PosZ}

	if p.mapOp != nil {
		if err := p.mapOp.EnterMap(req.PlayerID, req.MapID, pos); err != nil {
			zLog.Error("Failed to enter map", zap.Error(err))
			resp := &protocol.ClientMapEnterResponse{
				Result:   1,
				ErrorMsg: err.Error(),
			}
			respData, _ := proto.Marshal(resp)
			if msg.Callback != nil {
				msg.Callback <- &NetResponse{
					ProtoId: int32(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE),
					Data:    respData,
				}
			}
			return
		}
	}

	p.SetCurrentMapID(req.MapID)

	resp := &protocol.ClientMapEnterResponse{
		Result: 0,
		MapId:  int32(req.MapID),
		Pos: &protocol.Position{
			X: pos.X,
			Y: pos.Y,
			Z: pos.Z,
		},
	}

	respData, err := proto.Marshal(resp)
	if err != nil {
		p.sendErrorResponse(msg, fmt.Sprintf("marshal error: %v", err))
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &NetResponse{
			ProtoId: int32(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE),
			Data:    respData,
		}
	}
}

func (p *Player) handleNetMapLeave(msg *PlayerMessage) {
	req, ok := msg.Data.(*NetMapLeaveRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	if p.mapOp != nil {
		if err := p.mapOp.LeaveMap(req.PlayerID, req.MapID); err != nil {
			zLog.Error("Failed to leave map", zap.Error(err))
		}
	}

	p.SetCurrentMapID(0)

	resp := &protocol.ClientMapLeaveResponse{Result: 0}
	respData, err := proto.Marshal(resp)
	if err != nil {
		p.sendErrorResponse(msg, fmt.Sprintf("marshal error: %v", err))
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &NetResponse{
			ProtoId: int32(protocol.MapMsgId_MSG_MAP_LEAVE_RESPONSE),
			Data:    respData,
		}
	}
}

func (p *Player) handleNetMapMove(msg *PlayerMessage) {
	req, ok := msg.Data.(*NetMapMoveRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	pos := common.Vector3{X: req.PosX, Y: req.PosY, Z: req.PosZ}

	if p.mapOp != nil {
		if err := p.mapOp.Move(req.PlayerID, req.MapID, pos); err != nil {
			zLog.Error("Failed to move in map", zap.Error(err))
			resp := &protocol.ClientMapMoveResponse{Result: 1}
			respData, _ := proto.Marshal(resp)
			if msg.Callback != nil {
				msg.Callback <- &NetResponse{
					ProtoId: int32(protocol.MapMsgId_MSG_MAP_MOVE_RESPONSE),
					Data:    respData,
				}
			}
			return
		}
	}

	resp := &protocol.ClientMapMoveResponse{
		Result: 0,
		Pos: &protocol.Position{
			X: pos.X,
			Y: pos.Y,
			Z: pos.Z,
		},
	}

	respData, err := proto.Marshal(resp)
	if err != nil {
		p.sendErrorResponse(msg, fmt.Sprintf("marshal error: %v", err))
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &NetResponse{
			ProtoId: int32(protocol.MapMsgId_MSG_MAP_MOVE_RESPONSE),
			Data:    respData,
		}
	}
}

func (p *Player) handleNetMapAttack(msg *PlayerMessage) {
	req, ok := msg.Data.(*NetMapAttackRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	var damage, targetHP int64
	if p.mapOp != nil {
		var err error
		damage, targetHP, err = p.mapOp.Attack(req.PlayerID, req.MapID, req.TargetID)
		if err != nil {
			zLog.Error("Failed to attack in map", zap.Error(err))
			resp := &protocol.ClientMapAttackResponse{
				Result:   1,
				TargetId: int64(req.TargetID),
			}
			respData, _ := proto.Marshal(resp)
			if msg.Callback != nil {
				msg.Callback <- &NetResponse{
					ProtoId: int32(protocol.MapMsgId_MSG_MAP_ATTACK_RESPONSE),
					Data:    respData,
				}
			}
			return
		}
	}

	resp := &protocol.ClientMapAttackResponse{
		Result:   0,
		TargetId: int64(req.TargetID),
		Damage:   damage,
		TargetHp: targetHP,
	}

	respData, err := proto.Marshal(resp)
	if err != nil {
		p.sendErrorResponse(msg, fmt.Sprintf("marshal error: %v", err))
		return
	}

	if msg.Callback != nil {
		msg.Callback <- &NetResponse{
			ProtoId: int32(protocol.MapMsgId_MSG_MAP_ATTACK_RESPONSE),
			Data:    respData,
		}
	}
}
