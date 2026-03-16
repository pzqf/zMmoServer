package handler

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/dungeon"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
)

type DungeonHandler struct {
	sessionManager *session.SessionManager
	dungeonManager *dungeon.DungeonManager
}

func NewDungeonHandler(sessionManager *session.SessionManager, dungeonManager *dungeon.DungeonManager) *DungeonHandler {
	return &DungeonHandler{
		sessionManager: sessionManager,
		dungeonManager: dungeonManager,
	}
}

// HandleDungeonList 获取副本列表
func (dh *DungeonHandler) HandleDungeonList(sessionID string) (*protocol.CommonResponse, error) {
	zLog.Info("Handling dungeon list request", zap.String("session_id", sessionID))

	_, exists := dh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	dungeons := dh.dungeonManager.GetAllDungeons()

	// 构建副本列表响应
	dungeonList := make([]*protocol.DungeonInfo, 0, len(dungeons))
	for _, d := range dungeons {
		dungeonList = append(dungeonList, &protocol.DungeonInfo{
			DungeonId:        int32(d.DungeonID),
			DungeonName:      d.Name,
			DungeonDesc:      d.Description,
			LevelRequirement: int32(d.MinLevel),
			RecommendedLevel: int32(d.MaxLevel),
			MaxPlayers:       int32(d.MaxPlayers),
			Difficulty:       int32(d.Difficulty),
			EntryCost:        0, // 暂时设为0
			GoldReward:       d.RewardGold,
			ExpReward:        d.RewardExp,
			CooldownHours:    0, // 暂时设为0
			IsUnlocked:       d.IsOpen,
		})
	}

	// 这里应该直接返回DungeonListResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}

// HandleDungeonEnter 进入副本
func (dh *DungeonHandler) HandleDungeonEnter(sessionID string, dungeonID id.DungeonIdType) (*protocol.CommonResponse, error) {
	zLog.Info("Handling dungeon enter request", zap.String("session_id", sessionID), zap.Uint64("dungeon_id", uint64(dungeonID)))

	session, exists := dh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 获取玩家等级（假设从playerService获取）
	playerLevel := 10 // 临时值

	// 检查是否可以进入
	if !dh.dungeonManager.CanEnterDungeon(session.PlayerID, dungeonID, playerLevel) {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Cannot enter dungeon",
		}, nil
	}

	// 创建副本实例
	instance := dh.dungeonManager.CreateInstance(dungeonID, session.PlayerID)
	if instance == nil {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Failed to create dungeon instance",
		}, nil
	}

	// 开始副本
	dh.dungeonManager.StartInstance(instance.InstanceID)

	// 这里应该直接返回DungeonEnterResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}

// HandleDungeonComplete 完成副本
func (dh *DungeonHandler) HandleDungeonComplete(sessionID string, instanceID id.InstanceIdType) (*protocol.CommonResponse, error) {
	zLog.Info("Handling dungeon complete request", zap.String("session_id", sessionID), zap.Uint64("instance_id", uint64(instanceID)))

	_, exists := dh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 完成副本
	_ = dh.dungeonManager.CompleteInstance(instanceID)

	// 这里应该直接返回DungeonCompleteResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}

// HandleDungeonFail 副本失败
func (dh *DungeonHandler) HandleDungeonFail(sessionID string, instanceID id.InstanceIdType) (*protocol.CommonResponse, error) {
	zLog.Info("Handling dungeon fail request", zap.String("session_id", sessionID), zap.Uint64("instance_id", uint64(instanceID)))

	_, exists := dh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 标记副本失败
	_ = dh.dungeonManager.FailInstance(instanceID)

	// 这里应该直接返回DungeonFailResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}
