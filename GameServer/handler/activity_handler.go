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
func (ah *ActivityHandler) HandleActivityList(sessionID string) (*protocol.ActivityListResponse, error) {
	zLog.Info("Handling activity list request", zap.String("session_id", sessionID))

	_, exists := ah.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.ActivityListResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 获取所有活动
	activities := ah.activityManager.GetAllActivities()

	response := &protocol.ActivityListResponse{
		Result:     0,
		Activities: make([]*protocol.ActivityDetail, 0, len(activities)),
	}

	for _, activity := range activities {
		activityDetail := &protocol.ActivityDetail{
			ActivityId:   int32(activity.ActivityID),
			ActivityName: activity.Name,
			ActivityDesc: activity.Description,
			StartTime:    activity.StartTime.Unix(),
			EndTime:      activity.EndTime.Unix(),
			Status:       int32(activity.Status),
			Config:       make(map[string]string),
		}
		response.Activities = append(response.Activities, activityDetail)
	}

	return response, nil
}

// HandleActivityJoin 参与活动
func (ah *ActivityHandler) HandleActivityJoin(sessionID string, activityID id.ActivityIdType) (*protocol.CommonResponse, error) {
	zLog.Info("Handling activity join request", zap.String("session_id", sessionID), zap.Uint64("activity_id", uint64(activityID)))

	session, exists := ah.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 获取玩家信息（假设从playerService获取）
	playerLevel := 10 // 临时值

	// 参与活动
	_ = ah.activityManager.JoinActivity(session.PlayerID, activityID, playerLevel)

	// 这里应该直接返回ActivityJoinResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}

// HandleActivityProgress 更新活动进度
func (ah *ActivityHandler) HandleActivityProgress(sessionID string, activityID id.ActivityIdType, progress int) (*protocol.CommonResponse, error) {
	zLog.Info("Handling activity progress update", zap.String("session_id", sessionID), zap.Uint64("activity_id", uint64(activityID)), zap.Int("progress", progress))

	session, exists := ah.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 更新进度
	_ = ah.activityManager.UpdateActivityProgress(session.PlayerID, activityID, progress)

	// 这里应该直接返回ActivityProgressResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}

// HandleActivityClaim 领取活动奖励
func (ah *ActivityHandler) HandleActivityClaim(sessionID string, activityID id.ActivityIdType) (*protocol.CommonResponse, error) {
	zLog.Info("Handling activity reward claim", zap.String("session_id", sessionID), zap.Uint64("activity_id", uint64(activityID)))

	session, exists := ah.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 领取奖励
	_ = ah.activityManager.ClaimActivityReward(session.PlayerID, activityID)

	// 这里应该直接返回ActivityClaimResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}
