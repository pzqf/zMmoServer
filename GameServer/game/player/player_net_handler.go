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

// handleNetCrossEnter 进跨服活动实例（realm §5.1）。
//
// 与普通进图的关键差别：目标地图**不是客户端给的 mapID**——先由 GlobalServer 分配承载服务器、
// 再由该 MapServer 按 activityID 建/取实例，实例 mapID 是结果而非输入。整条链在玩家自己的
// actor 上同步等待（与攻击等响应同一模式），故 MapService 侧对每一跳都设了短超时。
func (p *Player) handleNetCrossEnter(msg *PlayerMessage) {
	req, ok := msg.Data.(*NetCrossEnterRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}

	respond := func(resp *protocol.ClientCrossEnterResponse) {
		respData, err := proto.Marshal(resp)
		if err != nil {
			p.sendErrorResponse(msg, fmt.Sprintf("marshal error: %v", err))
			return
		}
		if msg.Callback != nil {
			msg.Callback <- &NetResponse{
				ProtoId: int32(protocol.MapMsgId_MSG_MAP_CROSS_ENTER_RESPONSE),
				Data:    respData,
			}
		}
	}

	if p.mapOp == nil {
		respond(&protocol.ClientCrossEnterResponse{
			Result: 1, ErrorMsg: "map operator not available", ActivityId: req.ActivityID,
		})
		return
	}

	pos := common.Vector3{X: req.PosX, Y: req.PosY, Z: req.PosZ}
	mapServerID, mapID, err := p.mapOp.EnterCrossMap(req.PlayerID, req.ActivityID, req.MapConfigID, pos)
	if err != nil {
		zLog.Error("Failed to enter cross-server map",
			zap.Int64("player_id", int64(req.PlayerID)),
			zap.Int64("activity_id", req.ActivityID), zap.Error(err))
		respond(&protocol.ClientCrossEnterResponse{
			Result: 1, ErrorMsg: err.Error(), ActivityId: req.ActivityID,
		})
		return
	}

	p.SetCurrentMapID(mapID)

	respond(&protocol.ClientCrossEnterResponse{
		Result:      0,
		ActivityId:  req.ActivityID,
		MapServerId: int32(mapServerID),
		MapId:       int32(mapID),
		Pos:         &protocol.Position{X: pos.X, Y: pos.Y, Z: pos.Z},
	})
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

	// GS-3: 攻击是"转发到 MapServer 权威 + 回填响应",不改玩家本地状态。此前直接在 Player
	// actor goroutine 上同步等 MapServer 响应(最多 1.5s),期间该玩家的 move/AOI 等消息在
	// 邮箱(容量 100)里堆积、超限即被静默丢弃(尤其人多时 AOI 事件密集)。改为把跨服往返放到
	// 独立 goroutine,actor 立即返回继续服务本玩家其它消息;响应就绪后经 msg.Callback(缓冲 1,
	// 不会因 handler 已超时而阻塞)回填,handler 侧仍带 3s 超时等待,客户端语义不变。
	mapOp := p.mapOp
	callback := msg.Callback
	go func() {
		var damage, targetHP int64
		result := int32(0)
		if mapOp != nil {
			d, hp, err := mapOp.Attack(req.PlayerID, req.MapID, req.TargetID)
			if err != nil {
				zLog.Error("Failed to attack in map", zap.Error(err))
				result = 1
			} else {
				damage, targetHP = d, hp
			}
		}

		if callback == nil {
			return
		}
		resp := &protocol.ClientMapAttackResponse{
			Result:   result,
			TargetId: int64(req.TargetID),
			Damage:   damage,
			TargetHp: targetHP,
		}
		respData, err := proto.Marshal(resp)
		if err != nil {
			zLog.Error("Failed to marshal attack response", zap.Error(err))
			return
		}
		callback <- &NetResponse{
			ProtoId: int32(protocol.MapMsgId_MSG_MAP_ATTACK_RESPONSE),
			Data:    respData,
		}
	}()
}
