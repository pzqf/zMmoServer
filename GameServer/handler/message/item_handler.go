package message

import (
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// ItemHandler 网关侧物品/仓库消息处理器（业务层建设 2026-07-25）。
// 把客户端 protobuf 请求转成 PlayerMessage 投递到 Player actor，等回调后把 protobuf 响应下发客户端。
// 结构与 MapHandler 一致（同一转发/回调模式）。
type ItemHandler struct {
	playerManager *player.PlayerManager
	serverID      int32
}

func NewItemHandler(playerManager *player.PlayerManager, serverID int32) *ItemHandler {
	return &ItemHandler{playerManager: playerManager, serverID: serverID}
}

func (h *ItemHandler) sendToClient(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, protoId int32, data []byte) error {
	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: uint64(clientSessionID),
		PlayerId:  uint64(playerID),
		ServerId:  uint32(h.serverID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}
	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(h.serverID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeGateway),
		Message:      baseMsg,
	}
	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		zLog.Error("Failed to marshal cross server message", zap.Error(err))
		return err
	}
	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, h.serverID)
	return gwSession.Send(zNet.ProtoIdType(protoId), crossserver.Wrap(meta, crossMsgData))
}

// dispatch 把 req 作为 PlayerMessage 投递到 Player actor，等回调后下发响应。
func (h *ItemHandler) dispatch(session zNet.Session, playerID id.PlayerIdType, msgType player.MessageType, req interface{}) error {
	clientSessionID := getClientSessionID(session)
	msg, callback := player.NewPlayerMessageWithCallback(playerID, player.SourceGateway, msgType, req)
	if err := h.playerManager.RouteMessage(playerID, msg); err != nil {
		zLog.Error("Failed to route item message", zap.Error(err), zap.Int64("player_id", int64(playerID)))
		return nil
	}
	select {
	case resp := <-callback:
		if netResp, ok := resp.(*player.NetResponse); ok {
			return h.sendToClient(session, clientSessionID, playerID, int32(netResp.ProtoId), netResp.Data)
		}
	case <-time.After(3 * time.Second):
		zLog.Warn("Item message timeout", zap.Uint64("player_id", uint64(playerID)))
	}
	return nil
}

func (h *ItemHandler) Handle(session zNet.Session, protoId int32, data []byte) error {
	switch protoId {
	case int32(protocol.ItemMsgId_MSG_ITEM_LIST):
		var req protocol.ClientItemListRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		return h.dispatch(session, id.PlayerIdType(req.PlayerId), player.MsgNetItemList, &req)
	case int32(protocol.ItemMsgId_MSG_ITEM_USE):
		var req protocol.ClientItemUseRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		return h.dispatch(session, id.PlayerIdType(req.PlayerId), player.MsgNetItemUse, &req)
	case int32(protocol.ItemMsgId_MSG_ITEM_MOVE):
		var req protocol.ClientItemMoveRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		return h.dispatch(session, id.PlayerIdType(req.PlayerId), player.MsgNetItemMove, &req)
	case int32(protocol.ItemMsgId_MSG_ITEM_PICKUP):
		var req protocol.ClientItemPickupRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		// 拾取是 fire：转发到 MapServer；物品经 ItemGrant(506) 异步回推后由 Player actor 主动推客户端 1307，不在此等回调。
		return h.playerManager.RouteMessage(id.PlayerIdType(req.PlayerId),
			player.NewPlayerMessage(id.PlayerIdType(req.PlayerId), player.SourceGateway, player.MsgNetItemPickup, &req))
	case int32(protocol.ItemMsgId_MSG_WAREHOUSE_LIST):
		var req protocol.ClientWarehouseListRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		return h.dispatch(session, id.PlayerIdType(req.PlayerId), player.MsgNetWarehouseList, &req)
	case int32(protocol.ItemMsgId_MSG_WAREHOUSE_STORE):
		var req protocol.ClientWarehouseStoreRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		return h.dispatch(session, id.PlayerIdType(req.PlayerId), player.MsgNetWarehouseStore, &req)
	case int32(protocol.ItemMsgId_MSG_WAREHOUSE_RETRIEVE):
		var req protocol.ClientWarehouseRetrieveRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		return h.dispatch(session, id.PlayerIdType(req.PlayerId), player.MsgNetWarehouseRetrieve, &req)
	default:
		zLog.Warn("Unknown item message", zap.Int32("proto_id", protoId))
		return nil
	}
}
