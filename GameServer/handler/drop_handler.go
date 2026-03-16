package handler

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/drop"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
)

type DropHandler struct {
	sessionManager    *session.SessionManager
	dropManager       *drop.DropManager
	dropEntityManager *drop.DropEntityManager
}

func NewDropHandler(sessionManager *session.SessionManager, dropManager *drop.DropManager, dropEntityManager *drop.DropEntityManager) *DropHandler {
	return &DropHandler{
		sessionManager:    sessionManager,
		dropManager:       dropManager,
		dropEntityManager: dropEntityManager,
	}
}

// HandleDropPickup 拾取掉落
func (dh *DropHandler) HandleDropPickup(sessionID string, entityID id.ObjectIdType) (*protocol.CommonResponse, error) {
	zLog.Info("Handling drop pickup request", zap.String("session_id", sessionID), zap.Uint64("entity_id", uint64(entityID)))

	session, exists := dh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 获取掉落实体
	dropEntity := dh.dropEntityManager.GetEntity(entityID)
	if dropEntity == nil {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Drop entity not found",
		}, nil
	}

	// 检查是否可以拾取
	if !dropEntity.CanPickup(session.PlayerID) {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Cannot pickup this drop",
		}, nil
	}

	// 拾取所有物品
	items := make([]*protocol.DropItemInfo, 0, len(dropEntity.Items))
	for _, item := range dropEntity.Items {
		items = append(items, &protocol.DropItemInfo{
			ItemId:   item.ItemID,
			Count:    int32(item.Count),
			IsRare:   item.IsRare,
			BindType: int32(item.BindType),
		})
	}

	_ = dropEntity.PickupGold()
	_ = dropEntity.PickupExp()

	// 检查是否为空，为空则移除
	if dropEntity.IsEmpty() {
		dh.dropEntityManager.RemoveEntity(entityID)
	}

	// 这里应该直接返回DropPickupResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}

// HandleDropList 获取掉落列表
func (dh *DropHandler) HandleDropList(sessionID string, x, y, z, radius float64) (*protocol.CommonResponse, error) {
	zLog.Info("Handling drop list request", zap.String("session_id", sessionID))

	_, exists := dh.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.CommonResponse{
			Result:   1,
			ErrorMsg: "Session not found",
		}, nil
	}

	// 获取范围内的掉落实体
	dropEntities := dh.dropEntityManager.GetEntitiesInRange(x, y, z, radius)

	// 构建掉落列表响应
	dropList := make([]*protocol.DropEntityInfo, 0, len(dropEntities))
	for _, entity := range dropEntities {
		items := make([]*protocol.DropItemInfo, 0, len(entity.Items))
		for _, item := range entity.Items {
			items = append(items, &protocol.DropItemInfo{
				ItemId:   item.ItemID,
				Count:    int32(item.Count),
				IsRare:   item.IsRare,
				BindType: int32(item.BindType),
			})
		}

		dropList = append(dropList, &protocol.DropEntityInfo{
			EntityId:    int64(entity.EntityID),
			Items:       items,
			Gold:        entity.Gold,
			Exp:         entity.Exp,
			OwnerId:     int64(entity.OwnerID),
			IsProtected: entity.IsProtected,
			ProtectTime: int32(entity.ProtectTime),
			PositionX:   float32(entity.PositionX),
			PositionY:   float32(entity.PositionY),
			PositionZ:   float32(entity.PositionZ),
		})
	}

	// 这里应该直接返回DropListResponse，但由于函数签名限制，暂时返回CommonResponse
	return &protocol.CommonResponse{
		Result: 0,
	}, nil
}
