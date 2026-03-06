package handler

import (
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/buff"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
)

type BuffHandler struct {
	sessionManager *session.SessionManager
	buffManager    *buff.BuffManager
}

func NewBuffHandler(sessionManager *session.SessionManager, buffManager *buff.BuffManager) *BuffHandler {
	return &BuffHandler{
		sessionManager: sessionManager,
		buffManager:    buffManager,
	}
}

// HandleBuffList 获取Buff列表
func (bh *BuffHandler) HandleBuffList(sessionID string) (*protocol.Response, error) {
	zLog.Info("Handling buff list request", zap.String("session_id", sessionID))

	session, exists := bh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 获取玩家的Buff列表
	buffs := bh.buffManager.GetTargetBuffs(id.ObjectIdType(session.PlayerID))

	// 构建Buff列表响应
	buffList := make([]*protocol.BuffInfo, 0, len(buffs))
	for _, b := range buffs {
		buffDef := bh.buffManager.GetBuff(b.BuffID)
		if buffDef != nil {
			buffList = append(buffList, &protocol.BuffInfo{
				BuffId:        int32(b.BuffID),
				BuffName:      buffDef.Name,
				BuffDesc:      buffDef.Description,
				BuffType:      int32(buffDef.Type),
				Duration:      int32(buffDef.Duration),
				RemainingTime: int32(time.Until(b.EndTime).Seconds()),
				Stacks:        int32(b.StackCount),
				IsBeneficial:  buffDef.Type == 1, // 假设1是有益的
			})
		}
	}

	response := &protocol.BuffListResponse{
		Success: true,
		Buffs:   buffList,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleBuffApply 施加Buff
func (bh *BuffHandler) HandleBuffApply(sessionID string, buffID id.BuffIdType, targetID id.ObjectIdType) (*protocol.Response, error) {
	zLog.Info("Handling buff apply request", zap.String("session_id", sessionID), zap.Uint64("buff_id", uint64(buffID)), zap.Uint64("target_id", uint64(targetID)))

	session, exists := bh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 施加Buff
	instance := bh.buffManager.ApplyBuff(buffID, id.ObjectIdType(session.PlayerID), targetID)

	var buffInfo *protocol.BuffInfo
	if instance != nil {
		buffDef := bh.buffManager.GetBuff(buffID)
		if buffDef != nil {
			buffInfo = &protocol.BuffInfo{
				BuffId:        int32(buffID),
				BuffName:      buffDef.Name,
				BuffDesc:      buffDef.Description,
				BuffType:      int32(buffDef.Type),
				Duration:      int32(buffDef.Duration),
				RemainingTime: int32(buffDef.Duration),
				Stacks:        1,
				IsBeneficial:  buffDef.Type == 1,
			}
		}
	}

	response := &protocol.BuffApplyResponse{
		Success: instance != nil,
		Buff:    buffInfo,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleBuffRemove 移除Buff
func (bh *BuffHandler) HandleBuffRemove(sessionID string, instanceID id.BuffInstanceIdType) (*protocol.Response, error) {
	zLog.Info("Handling buff remove request", zap.String("session_id", sessionID), zap.Uint64("instance_id", uint64(instanceID)))

	_, exists := bh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 移除Buff
	success := bh.buffManager.RemoveBuff(instanceID)

	response := &protocol.BuffRemoveResponse{
		Success: success,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleBuffDispel 驱散Buff
func (bh *BuffHandler) HandleBuffDispel(sessionID string, targetID id.ObjectIdType, count int, dispelDebuff bool) (*protocol.Response, error) {
	zLog.Info("Handling buff dispel request", zap.String("session_id", sessionID), zap.Uint64("target_id", uint64(targetID)), zap.Int("count", count), zap.Bool("dispel_debuff", dispelDebuff))

	_, exists := bh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 驱散Buff
	dispelled := bh.buffManager.DispelBuffs(targetID, count, dispelDebuff)

	response := &protocol.BuffDispelResponse{
		Success:   true,
		Dispelled: int32(dispelled),
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}
