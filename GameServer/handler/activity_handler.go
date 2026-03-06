package handler

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/activity"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
)

type ActivityHandler struct {
	sessionManager  *session.SessionManager
	activityManager *activity.ActivityManager
}

func NewActivityHandler(sessionManager *session.SessionManager, activityManager *activity.ActivityManager) *ActivityHandler {
	return &ActivityHandler{
		sessionManager:  sessionManager,
		activityManager: activityManager,
	}
}

// HandleActivityList 获取活动列表
func (ah *ActivityHandler) HandleActivityList(sessionID string) (*protocol.Response, error) {
	zLog.Info("Handling activity list request", zap.String("session_id", sessionID))

	_, exists := ah.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 获取所有活动
	activities := ah.activityManager.GetAllActivities()

	// 构建活动列表响应
	activityList := make([]*protocol.ActivityInfo, 0, len(activities))
	for _, act := range activities {
		activityList = append(activityList, &protocol.ActivityInfo{
			ActivityId:     int32(act.ActivityID),
			ActivityName:   act.Name,
			ActivityDesc:   act.Description,
			ActivityType:   int32(act.Type),
			Status:         int32(act.Status),
			StartTime:      act.StartTime.Unix(),
			EndTime:        act.EndTime.Unix(),
			MinLevel:       int32(act.MinLevel),
		})
	}

	response := &protocol.ActivityListResponse{
		Success:  true,
		Activities: activityList,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleActivityJoin 参与活动
func (ah *ActivityHandler) HandleActivityJoin(sessionID string, activityID id.ActivityIdType) (*protocol.Response, error) {
	zLog.Info("Handling activity join request", zap.String("session_id", sessionID), zap.Uint64("activity_id", uint64(activityID)))

	session, exists := ah.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 获取玩家信息（假设从playerService获取）
	playerLevel := 10 // 临时值

	// 参与活动
	success := ah.activityManager.JoinActivity(session.PlayerID, activityID, playerLevel)

	response := &protocol.ActivityJoinResponse{
		Success: success,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleActivityProgress 更新活动进度
func (ah *ActivityHandler) HandleActivityProgress(sessionID string, activityID id.ActivityIdType, progress int) (*protocol.Response, error) {
	zLog.Info("Handling activity progress update", zap.String("session_id", sessionID), zap.Uint64("activity_id", uint64(activityID)), zap.Int("progress", progress))

	session, exists := ah.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 更新进度
	success := ah.activityManager.UpdateActivityProgress(session.PlayerID, activityID, progress)

	response := &protocol.ActivityProgressResponse{
		Success: success,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}

// HandleActivityClaim 领取活动奖励
func (ah *ActivityHandler) HandleActivityClaim(sessionID string, activityID id.ActivityIdType) (*protocol.Response, error) {
	zLog.Info("Handling activity reward claim", zap.String("session_id", sessionID), zap.Uint64("activity_id", uint64(activityID)))

	session, exists := ah.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 领取奖励
	success := ah.activityManager.ClaimActivityReward(session.PlayerID, activityID)

	response := &protocol.ActivityClaimResponse{
		Success: success,
	}

	return &protocol.Response{
		Result: 0,
		Data:   marshalResponse(response),
	}, nil
}
