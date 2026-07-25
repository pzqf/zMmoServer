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

// aoiSendJob 一条待发送的 AOI 视野事件（全部为值，无共享对象引用 → 可安全跨 goroutine 传递）。
type aoiSendJob struct {
	watcherPlayerID int64
	mapID           int32
	eventType       uint32
	targetID        int64
	pos             mapcommon.Vector3
}

// NotifyAOI 实现 maps.AOINotifier：把一条 AOI 视野事件回传给拥有 watcher 的 GameServer。
//
// MAP-7：本方法由地图的单写者 goroutine（map actor）在 handleAOIEvent 中同步调用。若在此直接做
// 网络 marshal+send，一旦某 GameServer 连接的发送缓冲被打满，就会阻塞整张地图的 actor（帧更新/
// 攻击/移动全部卡住）。故这里只把「值化」的事件投递到发送队列，由独立的 aoiSendLoop goroutine
// 实际发送；队列满则丢弃该 AOI 事件（AOI 是尽力而为，丢一帧视野同步远好于卡死整图）。
func (ts *TCPService) NotifyAOI(watcherPlayerID int64, mapID int32, eventType uint32, targetID int64, pos mapcommon.Vector3) {
	job := aoiSendJob{
		watcherPlayerID: watcherPlayerID,
		mapID:           mapID,
		eventType:       eventType,
		targetID:        targetID,
		pos:             pos,
	}
	if ts.aoiSendCh == nil {
		// 兜底：发送循环未启动（理论上 Start 后必非 nil），就地发送以不丢事件。
		ts.sendAOINotify(job)
		return
	}
	select {
	case ts.aoiSendCh <- job:
	default:
		zLog.Warn("AOI send queue full, dropping notify",
			zap.Int64("watcher", watcherPlayerID), zap.Int64("target", targetID), zap.Uint32("evt", eventType))
	}
}

// aoiSendLoop 独立 goroutine：串行发送 AOI 事件，把网络 I/O 从地图 actor goroutine 上剥离（MAP-7）。
func (ts *TCPService) aoiSendLoop() {
	defer ts.wg.Done()
	for {
		select {
		case <-ts.aoiStop:
			return
		case job := <-ts.aoiSendCh:
			ts.sendAOINotify(job)
		}
	}
}

// sendAOINotify 实际执行一条 AOI 事件的查会话 + marshal + pack + send。
// watcher 非玩家（如怪）或其 GameServer 未知时静默跳过。承载消息复用客户端 notify，
// 事件类型经外层/内层 protoId（500/501/502）区分。见 docs/成熟化改造-执行计划.md 2.3。
func (ts *TCPService) sendAOINotify(job aoiSendJob) {
	if ts.playerGameServerManager == nil {
		return
	}
	serverID, ok := ts.playerGameServerManager.GetGameServerID(id.PlayerIdType(job.watcherPlayerID))
	if !ok {
		return // watcher 非本服玩家（如怪）或映射未建立
	}
	session, ok := ts.gameServerSessions.Load(serverID)
	if !ok {
		return
	}
	zLog.Debug("NotifyAOI sending", zap.Int64("watcher", job.watcherPlayerID), zap.Int64("target", job.targetID), zap.Uint32("evt", job.eventType), zap.Uint32("server_id", serverID))

	var innerData []byte
	var err error
	switch job.eventType {
	case crossserver.MsgInternalAOIEnter:
		innerData, err = proto.Marshal(&protocol.EntityEnterViewNotify{
			EntityId: job.targetID,
			Pos:      &protocol.Position{X: job.pos.X, Y: job.pos.Y, Z: job.pos.Z},
		})
	case crossserver.MsgInternalAOILeave:
		innerData, err = proto.Marshal(&protocol.EntityLeaveViewNotify{EntityId: job.targetID})
	case crossserver.MsgInternalAOIMove:
		innerData, err = proto.Marshal(&protocol.EntityMoveNotify{
			EntityId: job.targetID,
			NewPos:   &protocol.Position{X: job.pos.X, Y: job.pos.Y, Z: job.pos.Z},
		})
	default:
		return
	}
	if err != nil {
		zLog.Error("Failed to marshal AOI notify", zap.Error(err))
		return
	}

	base := crossserver.BuildBaseMessage(job.eventType, uint64(job.watcherPlayerID),
		uint32(ts.config.Server.ServerID), uint32(job.mapID), innerData)
	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeMap, int32(ts.config.Server.ServerID))
	enveloped, err := crossserver.PackMessage(meta, crossserver.ServiceTypeMap, crossserver.ServiceTypeGame,
		uint32(ts.config.Server.ServerID), serverID, base)
	if err != nil {
		zLog.Error("Failed to pack AOI notify", zap.Error(err))
		return
	}
	if err := session.Send(zNet.ProtoIdType(job.eventType), enveloped); err != nil {
		zLog.Warn("Failed to send AOI notify to GameServer", zap.Error(err))
	}
}
