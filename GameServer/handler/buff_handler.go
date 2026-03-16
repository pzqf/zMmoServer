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
func (bh *BuffHandler) HandleBuffList(sessionID string) (*protocol.BuffListResponse, error) {
	zLog.Info("Handling buff list request", zap.String("session_id", sessionID))

	session, exists := bh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.BuffListResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 获取玩家的Buff列表
	buffs := bh.buffManager.GetTargetBuffs(id.ObjectIdType(session.PlayerID))

	response := &protocol.BuffListResponse{
		Result: 0,
		Buffs:  make([]*protocol.BuffDetail, 0, len(buffs)),
	}

	for _, buffInstance := range buffs {
		// 获取Buff定义
		buffDef := bh.buffManager.GetBuff(buffInstance.BuffID)
		if buffDef == nil {
			continue
		}

		// 计算剩余时间
		remainingTime := int(buffInstance.EndTime.Sub(time.Now()).Seconds())
		if remainingTime < 0 {
			remainingTime = 0
		}

		buffDetail := &protocol.BuffDetail{
			BuffId:        int32(buffInstance.BuffID),
			BuffName:      buffDef.Name,
			BuffDesc:      buffDef.Description,
			BuffType:      int32(buffDef.Type),
			Duration:      int32(buffDef.Duration),
			RemainingTime: int32(remainingTime),
			Stacks:        int32(buffInstance.StackCount),
			IsBeneficial:  buffDef.Type == buff.BuffTypeBuff,
		}
		response.Buffs = append(response.Buffs, buffDetail)
	}

	return response, nil
}

// HandleBuffApply 施加Buff
func (bh *BuffHandler) HandleBuffApply(sessionID string, buffID id.BuffIdType, targetID id.ObjectIdType) (*protocol.CommonResponse, error) {
	zLog.Info("Handling buff apply request", zap.String("session_id", sessionID), zap.Uint64("buff_id", uint64(buffID)), zap.Uint64("target_id", uint64(targetID)))

	session, exists := bh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 施加Buff
	instance := bh.buffManager.ApplyBuff(buffID, id.ObjectIdType(session.PlayerID), targetID)

	if instance != nil {
		buffDef := bh.buffManager.GetBuff(buffID)
		if buffDef != nil {
			_ = &protocol.BuffDetail{
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

	// 这里应该直接返回BuffApplyResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}

// HandleBuffRemove 移除Buff
func (bh *BuffHandler) HandleBuffRemove(sessionID string, instanceID id.BuffInstanceIdType) (*protocol.CommonResponse, error) {
	zLog.Info("Handling buff remove request", zap.String("session_id", sessionID), zap.Uint64("instance_id", uint64(instanceID)))

	_, exists := bh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 移除Buff
	_ = bh.buffManager.RemoveBuff(instanceID)

	// 这里应该直接返回BuffRemoveResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}

// HandleBuffDispel 驱散Buff
func (bh *BuffHandler) HandleBuffDispel(sessionID string, targetID id.ObjectIdType, count int, dispelDebuff bool) (*protocol.CommonResponse, error) {
	zLog.Info("Handling buff dispel request", zap.String("session_id", sessionID), zap.Uint64("target_id", uint64(targetID)), zap.Int("count", count), zap.Bool("dispel_debuff", dispelDebuff))

	_, exists := bh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 驱散Buff
	_ = bh.buffManager.DispelBuffs(targetID, count, dispelDebuff)

	// 这里应该直接返回BuffDispelResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}
