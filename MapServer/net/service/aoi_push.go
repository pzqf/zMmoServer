package service

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	mapcommon "github.com/pzqf/zMmoServer/MapServer/common"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// NotifyAOI 实现 maps.AOINotifier：把一条 AOI 视野事件回传给拥有 watcher 的 GameServer。
// watcher 非玩家（如怪）或其 GameServer 未知时静默跳过。承载消息复用客户端 notify，
// 事件类型经外层/内层 protoId（500/501/502）区分。见 docs/成熟化改造-执行计划.md 2.3。
func (ts *TCPService) NotifyAOI(watcherPlayerID int64, mapID int32, eventType uint32, targetID int64, pos mapcommon.Vector3) {
	if ts.playerGameServerManager == nil {
		return
	}
	serverID, ok := ts.playerGameServerManager.GetGameServerID(id.PlayerIdType(watcherPlayerID))
	if !ok {
		return // watcher 非本服玩家（如怪）或映射未建立
	}
	session, ok := ts.gameServerSessions.Load(serverID)
	if !ok {
		return
	}
	zLog.Debug("NotifyAOI sending", zap.Int64("watcher", watcherPlayerID), zap.Int64("target", targetID), zap.Uint32("evt", eventType), zap.Uint32("server_id", serverID))

	var innerData []byte
	var err error
	switch eventType {
	case crossserver.MsgInternalAOIEnter:
		innerData, err = proto.Marshal(&protocol.EntityEnterViewNotify{
			EntityId: targetID,
			Pos:      &protocol.Position{X: pos.X, Y: pos.Y, Z: pos.Z},
		})
	case crossserver.MsgInternalAOILeave:
		innerData, err = proto.Marshal(&protocol.EntityLeaveViewNotify{EntityId: targetID})
	case crossserver.MsgInternalAOIMove:
		innerData, err = proto.Marshal(&protocol.EntityMoveNotify{
			EntityId: targetID,
			NewPos:   &protocol.Position{X: pos.X, Y: pos.Y, Z: pos.Z},
		})
	default:
		return
	}
	if err != nil {
		zLog.Error("Failed to marshal AOI notify", zap.Error(err))
		return
	}

	base := crossserver.BuildBaseMessage(eventType, uint64(watcherPlayerID),
		uint32(ts.config.Server.ServerID), uint32(mapID), innerData)
	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeMap, int32(ts.config.Server.ServerID))
	enveloped, err := crossserver.PackMessage(meta, crossserver.ServiceTypeMap, crossserver.ServiceTypeGame,
		uint32(ts.config.Server.ServerID), serverID, base)
	if err != nil {
		zLog.Error("Failed to pack AOI notify", zap.Error(err))
		return
	}
	if err := session.Send(zNet.ProtoIdType(eventType), enveloped); err != nil {
		zLog.Warn("Failed to send AOI notify to GameServer", zap.Error(err))
	}
}
