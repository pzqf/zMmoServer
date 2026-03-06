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
func (dh *DungeonHandler) HandleDungeonList(sessionID string) (*protocol.Response, error) {
	zLog.Info("Handling dungeon list request", zap.String("session_id", sessionID))

	_, exists := dh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	dungeons := dh.dungeonManager.GetAllDungeons()

	// 构建副本列表响应
	dungeonList := make([]*protocol.DungeonInfo, 0, len(dungeons))
	for _, d := range dungeons {
		dungeonList = append(dungeonList, &protocol.DungeonInfo{
			DungeonId:     int32(d.DungeonID),
			DungeonName:   d.Name,
			DungeonDesc:   d.Description,
			DungeonType:   int32(d.Type),
			MinLevel:      int32(d.MinLevel),
			MaxLevel:      int32(d.MaxLevel),
			MinPlayers:    int32(d.MinPlayers),
			MaxPlayers:    int32(d.MaxPlayers),
			TimeLimit:     int32(d.TimeLimit),
			DailyLimit:    int32(d.DailyLimit),
			Status:        1, // 假设1是可用状态
			Progress:      0,
			RemainingTime: int32(d.TimeLimit),
		})
	}

	response := &protocol.DungeonListResponse{
		Success:  true,
		Dungeons: dungeonList,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleDungeonEnter 进入副本
func (dh *DungeonHandler) HandleDungeonEnter(sessionID string, dungeonID id.DungeonIdType) (*protocol.Response, error) {
	zLog.Info("Handling dungeon enter request", zap.String("session_id", sessionID), zap.Uint64("dungeon_id", uint64(dungeonID)))

	session, exists := dh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 获取玩家等级（假设从playerService获取）
	playerLevel := 10 // 临时值

	// 检查是否可以进入
	if !dh.dungeonManager.CanEnterDungeon(session.PlayerID, dungeonID, playerLevel) {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Cannot enter dungeon",
		}, nil
	}

	// 创建副本实例
	instance := dh.dungeonManager.CreateInstance(dungeonID, session.PlayerID)
	if instance == nil {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Failed to create dungeon instance",
		}, nil
	}

	// 开始副本
	dh.dungeonManager.StartInstance(instance.InstanceID)

	response := &protocol.DungeonEnterResponse{
		Success:    true,
		InstanceId: int32(instance.InstanceID),
		MapId:      0, // 暂时设为0，需要从副本配置中获取
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleDungeonComplete 完成副本
func (dh *DungeonHandler) HandleDungeonComplete(sessionID string, instanceID id.InstanceIdType) (*protocol.Response, error) {
	zLog.Info("Handling dungeon complete request", zap.String("session_id", sessionID), zap.Uint64("instance_id", uint64(instanceID)))

	_, exists := dh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 完成副本
	success := dh.dungeonManager.CompleteInstance(instanceID)

	response := &protocol.DungeonCompleteResponse{
		Success: success,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleDungeonFail 副本失败
func (dh *DungeonHandler) HandleDungeonFail(sessionID string, instanceID id.InstanceIdType) (*protocol.Response, error) {
	zLog.Info("Handling dungeon fail request", zap.String("session_id", sessionID), zap.Uint64("instance_id", uint64(instanceID)))

	_, exists := dh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 标记副本失败
	success := dh.dungeonManager.FailInstance(instanceID)

	response := &protocol.DungeonFailResponse{
		Success: success,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}
