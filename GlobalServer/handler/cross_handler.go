package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GlobalServer/crossmatch"
	"go.uber.org/zap"
)

// HandleCrossAllocate POST /api/v1/cross/allocate —— 跨服活动分配（GameServer 调用）。
//
// 返回"这场活动放在哪台 MapServer 上"。**同一 activityID 必须始终得到同一个答案**，
// 否则不同 realm 的玩家会被送进不同实例、永远碰不到面（粘性由 crossmatch.Allocator 保证）。
// 本接口不建实例——实例由目标 MapServer 按 activityID 幂等创建。
func HandleCrossAllocate(c echo.Context) error {
	var req protocol.CrossAllocateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, &protocol.CrossAllocateResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "invalid request body",
		})
	}
	if req.ActivityId <= 0 {
		return c.JSON(http.StatusBadRequest, &protocol.CrossAllocateResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "activity_id is required",
		})
	}

	allocator := crossmatch.GetAllocator()
	if allocator == nil {
		zLog.Error("Cross allocator not initialized")
		return c.JSON(http.StatusInternalServerError, &protocol.CrossAllocateResponse{
			Result:     int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg:   "cross allocator not initialized",
			ActivityId: req.ActivityId,
		})
	}

	alloc, err := allocator.Allocate(req.ActivityId, req.MapConfigId)
	if err != nil {
		zLog.Warn("Cross activity allocation failed",
			zap.Int64("activity_id", req.ActivityId), zap.Error(err))
		return c.JSON(http.StatusServiceUnavailable, &protocol.CrossAllocateResponse{
			Result:     int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg:   err.Error(),
			ActivityId: req.ActivityId,
		})
	}

	return c.JSON(http.StatusOK, &protocol.CrossAllocateResponse{
		Result:      int32(protocol.ErrorCode_ERR_SUCCESS),
		ActivityId:  alloc.ActivityID,
		MapServerId: int32(alloc.ServerID),
		GroupId:     alloc.GroupID,
		Address:     alloc.Address,
	})
}

// HandleCrossAllocations GET /api/v1/cross/allocations —— 当前所有跨服活动落点（运维/排查）。
func HandleCrossAllocations(c echo.Context) error {
	allocator := crossmatch.GetAllocator()
	if allocator == nil {
		return c.JSON(http.StatusOK, map[string]any{"allocations": []any{}})
	}

	list := allocator.List()
	items := make([]map[string]any, 0, len(list))
	for _, a := range list {
		items = append(items, map[string]any{
			"activity_id":   a.ActivityID,
			"map_config_id": a.MapConfigID,
			"map_server_id": a.ServerID,
			"group_id":      a.GroupID,
			"address":       a.Address,
			"created_at":    a.CreatedAt.Unix(),
			"updated_at":    a.UpdatedAt.Unix(),
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"allocations": items})
}
