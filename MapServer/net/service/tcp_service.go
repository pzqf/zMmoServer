package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"github.com/pzqf/zCommon/common/id"
	consistency "github.com/pzqf/zEngine/zConsistency"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/netutil"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/maps"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type TCPService struct {
	config                  *config.Config
	mapService              *maps.MapManager
	playerGameServerManager *maps.PlayerGameServerManager
	tcpServer               *zNet.TcpServer
	isRunning               bool
	wg                      sync.WaitGroup
	// gameServerSessions: GameServer 服务器 ID → 其连接会话，用于服务端主动推送
	// （如 AOI 视野事件回程）。在 handleGameServerMessage 里按每条请求更新。
	gameServerSessions *zMap.TypedMap[uint32, zNet.Session]
	// metricsRecorder: 可选网络指标上报器（Phase 3.5），注入 zNet 上报连接/流量/错误。
	metricsRecorder zNet.NetworkMetricsRecorder
	// aoiSendCh/aoiStop: AOI 回程发送队列 + 停止信号（MAP-7）。把网络发送从地图 actor
	// goroutine 上剥离，避免慢连接阻塞整张地图。见 aoi_push.go。
	aoiSendCh chan aoiSendJob
	aoiStop   chan struct{}
	// inbox: 请求侧幂等去重（GS-4）。GameServer 的 outboxRetryLoop 会重发失败的 enter/attack
	// 请求，MapServer 无去重则会重放（双倍伤害 / 重复进图）。按 requestID 去重非幂等操作。
	// 周期 Cleanup 限制其增长（INF-11）。
	inbox consistency.InboxStore
}

func NewTCPService(cfg *config.Config, mapService *maps.MapManager) *TCPService {
	return &TCPService{
		config:             cfg,
		mapService:         mapService,
		isRunning:          false,
		gameServerSessions: zMap.NewTypedMap[uint32, zNet.Session](),
		inbox:              consistency.NewMemoryInbox(),
	}
}

func (ts *TCPService) SetPlayerGameServerManager(manager *maps.PlayerGameServerManager) {
	ts.playerGameServerManager = manager
}

// SetMetricsRecorder 注入网络指标上报器（Phase 3.5），须在 Start 前调用。
func (ts *TCPService) SetMetricsRecorder(r zNet.NetworkMetricsRecorder) {
	ts.metricsRecorder = r
}

func (ts *TCPService) Name() string {
	return "TCPService"
}

func (ts *TCPService) Start(ctx context.Context) error {
	if ts.isRunning {
		return nil
	}

	zLog.Info("Starting TCP service...", zap.String("addr", ts.config.Server.ListenAddr))

	// 端口占用探测（主动拨号，规避 Windows SO_REUSEADDR 静默双绑；占用则记占用方 PID/进程名 + fail-fast）。
	if err := netutil.EnsurePortFree(ts.config.Server.ListenAddr); err != nil {
		zLog.Error("TCP 监听端口被占用，拒绝启动", zap.String("addr", ts.config.Server.ListenAddr), zap.Error(err))
		return err
	}

	// 创建zNet.TcpServer配置
	tcpConfig := &zNet.TcpConfig{
		ListenAddress:     ts.config.Server.ListenAddr,
		MaxClientCount:    ts.config.Server.MaxConnections,
		HeartbeatDuration: ts.config.Server.HeartbeatInterval,
		// 使用默认值
		ChanSize:            1024,
		MaxPacketDataSize:   1024 * 1024,
		UseWorkerPool:       false,
		WorkerPoolSize:      10,
		WorkerQueueSize:     100,
		DisableEncryption:   true,
		EnableKeyRotation:   false,
		KeyRotationInterval: 300 * time.Second,
		MaxHistoryKeys:      5,
		EnableSequenceCheck: false,
		SequenceWindowSize:  100,
		TimestampTolerance:  5000,
	}

	// 创建zNet.TcpServer（可选注入网络指标上报器）
	var opts []zNet.Options
	if ts.metricsRecorder != nil {
		opts = append(opts, zNet.WithServerMetrics(ts.metricsRecorder))
	}
	ts.tcpServer = zNet.NewTcpServer(tcpConfig, opts...)

	// 注册消息处理器
	ts.tcpServer.RegisterDispatcher(ts.handleConnectionMessage)

	// 启动服务
	err := ts.tcpServer.Start()
	if err != nil {
		return fmt.Errorf("failed to start TCP service: %v", err)
	}

	// 启动 AOI 回程发送 goroutine（MAP-7）：把网络发送从地图 actor 上剥离。
	ts.aoiSendCh = make(chan aoiSendJob, 4096)
	ts.aoiStop = make(chan struct{})
	ts.wg.Add(1)
	go ts.aoiSendLoop()

	// 启动请求去重表清理 goroutine（GS-4 + INF-11）：定期驱逐旧 requestID，限制 inbox 增长。
	ts.wg.Add(1)
	go ts.inboxCleanupLoop()

	ts.isRunning = true

	zLog.Info("TCP service started successfully", zap.String("addr", ts.config.Server.ListenAddr))

	return nil
}

func (ts *TCPService) Stop(ctx context.Context) error {
	if !ts.isRunning {
		return nil
	}

	zLog.Info("Stopping TCP service...")

	if ts.tcpServer != nil {
		ts.tcpServer.Close()
	}

	// 停止 AOI 发送 goroutine（MAP-7）。aoiSendCh 不关闭：NotifyAOI 用非阻塞 select 投递，
	// 循环退出后残留投递会填满缓冲后被丢弃，不会 send-on-closed panic。
	if ts.aoiStop != nil {
		close(ts.aoiStop)
	}

	ts.isRunning = false
	ts.wg.Wait()

	zLog.Info("TCP service stopped")

	return nil
}

// inboxCleanupLoop 周期驱逐旧的请求去重记录，限制 inbox 增长（GS-4 + INF-11）。
// 5 分钟窗口远大于 outbox 重试上限（maxRetry×1s），足以覆盖任何重放又不无界累积。
func (ts *TCPService) inboxCleanupLoop() {
	defer ts.wg.Done()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ts.aoiStop:
			return
		case <-ticker.C:
			ts.inbox.Cleanup(5 * time.Minute)
		}
	}
}

// handleConnectionMessage 处理来自客户端的消息
func (ts *TCPService) handleConnectionMessage(session zNet.Session, packet *zNet.NetPacket) error {
	// 处理消息
	sessionID := session.GetSid()
	gameServerAddr := session.GetClientIP()

	zLog.Info("New GameServer connection", zap.Uint64("session_id", uint64(sessionID)), zap.String("game_server_addr", gameServerAddr))

	// 处理GameServer消息
	ts.handleGameServerMessage(session, int32(packet.ProtoId), packet.Data)

	return nil
}

// handleGameServerMessage 处理来自GameServer的消息。
// 统一经 crossserver 共用 codec 解包（Envelope + protobuf CrossServerMessage），
// 按外层 protoId（== InternalMsgId 请求值）分派。见 docs/协议契约.md。
func (ts *TCPService) handleGameServerMessage(session zNet.Session, protoId int32, data []byte) {
	meta, _, baseMsg, err := crossserver.UnpackMessage(data)
	if err != nil {
		zLog.Error("Invalid cross-server message from GameServer",
			zap.Int32("proto_id", protoId), zap.Error(err))
		return
	}
	if baseMsg == nil {
		zLog.Error("Cross-server message missing inner base message", zap.Int32("proto_id", protoId))
		return
	}

	// 记录该 GameServer 的会话，供服务端主动推送（AOI 回程）按 serverID 找会话
	if baseMsg.ServerId != 0 {
		ts.gameServerSessions.Store(baseMsg.ServerId, session)
	}

	zLog.Debug("Received cross-server message from GameServer",
		zap.Uint64("trace_id", meta.TraceID),
		zap.Uint64("request_id", meta.RequestID),
		zap.Int32("proto_id", protoId))

	switch protoId {
	case int32(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_REQUEST):
		ts.handleMapEnterRequest(session, baseMsg, &meta)
	case int32(protocol.InternalMsgId_MSG_INTERNAL_MAP_LEAVE_REQUEST):
		ts.handleMapLeaveRequest(session, baseMsg, &meta)
	case int32(protocol.InternalMsgId_MSG_INTERNAL_MAP_MOVE_REQUEST):
		ts.handleMapMoveRequest(session, baseMsg, &meta)
	case int32(protocol.InternalMsgId_MSG_INTERNAL_MAP_ATTACK_REQUEST):
		ts.handleMapAttackRequest(session, baseMsg, &meta)
	case int32(protocol.InternalMsgId_MSG_INTERNAL_MAP_PICKUP_REQUEST):
		ts.handleMapPickupRequest(session, baseMsg, &meta)
	case int32(protocol.InternalMsgId_MSG_INTERNAL_CROSS_INSTANCE_ENSURE):
		ts.handleCrossInstanceEnsure(session, baseMsg, &meta)
	default:
		zLog.Info("Received unknown message from GameServer", zap.Int32("proto_id", protoId))
	}
}

// handleMapEnterRequest 处理玩家进入地图请求
func (ts *TCPService) handleMapEnterRequest(session zNet.Session, baseMsg *protocol.BaseMessage, meta *crossserver.Meta) {
	var req protocol.MapEnterRequest
	if err := proto.Unmarshal(baseMsg.Data, &req); err != nil {
		zLog.Error("Failed to unmarshal MapEnterRequest", zap.Error(err))
		return
	}

	// GS-4: 请求侧幂等去重——GameServer 的 outboxRetryLoop 可能重发本请求，重复进图会重跑
	// AddObject/AOI 进入。按 requestID 去重，重放直接丢弃。
	if meta != nil && meta.RequestID != 0 && !ts.inbox.TryAccept(meta.RequestID) {
		zLog.Warn("Duplicate map enter request ignored",
			zap.Uint64("request_id", meta.RequestID), zap.Int64("player_id", req.PlayerId))
		return
	}

	zLog.Info("Map enter request",
		zap.Int64("player_id", req.PlayerId),
		zap.Int64("map_id", req.MapId),
		zap.Uint32("game_server_id", baseMsg.ServerId),
		zap.Float32("x", req.X),
		zap.Float32("y", req.Y),
		zap.Float32("z", req.Z))

	if ts.playerGameServerManager != nil {
		ts.playerGameServerManager.SetPlayerGameServer(
			id.PlayerIdType(req.PlayerId),
			baseMsg.ServerId,
			"",
			id.MapIdType(req.MapId),
		)
	}

	if ts.mapService != nil {
		err := ts.mapService.HandlePlayerEnterMap(req.PlayerId, req.MapId, req.X, req.Y, req.Z, req.Level, req.Hp)
		if err != nil {
			zLog.Error("Failed to handle player enter map", zap.Error(err))
			resp := &protocol.MapEnterResponse{
				Success:  false,
				ErrorMsg: err.Error(),
			}
			ts.sendResponse(session, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_RESPONSE), resp, baseMsg, meta)
			return
		}
	}

	resp := &protocol.MapEnterResponse{
		Success:  true,
		ObjectId: req.PlayerId,
		MapId:    req.MapId,
		X:        req.X,
		Y:        req.Y,
		Z:        req.Z,
	}
	ts.sendResponse(session, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_RESPONSE), resp, baseMsg, meta)
}

func (ts *TCPService) handleMapLeaveRequest(session zNet.Session, baseMsg *protocol.BaseMessage, meta *crossserver.Meta) {
	var req protocol.MapLeaveRequest
	if err := proto.Unmarshal(baseMsg.Data, &req); err != nil {
		zLog.Error("Failed to unmarshal MapLeaveRequest", zap.Error(err))
		return
	}

	zLog.Info("Map leave request",
		zap.Int64("player_id", req.PlayerId),
		zap.Int64("map_id", req.MapId),
		zap.Int32("reason", req.Reason))

	if ts.playerGameServerManager != nil {
		ts.playerGameServerManager.RemovePlayerGameServer(id.PlayerIdType(req.PlayerId))
	}

	if ts.mapService != nil {
		err := ts.mapService.HandlePlayerLeaveMap(req.PlayerId, req.MapId)
		if err != nil {
			zLog.Error("Failed to handle player leave map", zap.Error(err))
			resp := &protocol.MapLeaveResponse{
				Success:  false,
				ErrorMsg: err.Error(),
			}
			ts.sendResponse(session, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_LEAVE_RESPONSE), resp, baseMsg, meta)
			return
		}
	}

	resp := &protocol.MapLeaveResponse{
		Success:  true,
		PlayerId: req.PlayerId,
	}
	ts.sendResponse(session, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_LEAVE_RESPONSE), resp, baseMsg, meta)
}

// handleMapMoveRequest 处理玩家移动请求
func (ts *TCPService) handleMapMoveRequest(session zNet.Session, baseMsg *protocol.BaseMessage, meta *crossserver.Meta) {
	var req protocol.MapMoveRequest
	if err := proto.Unmarshal(baseMsg.Data, &req); err != nil {
		zLog.Error("Failed to unmarshal MapMoveRequest", zap.Error(err))
		return
	}

	zLog.Info("Map move request",
		zap.Int64("player_id", req.PlayerId),
		zap.Int64("object_id", req.ObjectId),
		zap.Int64("map_id", req.MapId),
		zap.Float32("x", req.X),
		zap.Float32("y", req.Y),
		zap.Float32("z", req.Z))

	if ts.mapService != nil {
		err := ts.mapService.HandlePlayerMove(req.PlayerId, req.ObjectId, req.MapId, req.X, req.Y, req.Z)
		if err != nil {
			zLog.Error("Failed to handle player move", zap.Error(err))
		}
	}

	resp := &protocol.MapMoveResponse{
		Success:  true,
		PlayerId: req.PlayerId,
		X:        req.X,
		Y:        req.Y,
		Z:        req.Z,
	}
	ts.sendResponse(session, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_MOVE_SYNC), resp, baseMsg, meta)
}

// handleMapAttackRequest 处理玩家攻击请求
func (ts *TCPService) handleMapAttackRequest(session zNet.Session, baseMsg *protocol.BaseMessage, meta *crossserver.Meta) {
	var req protocol.MapAttackRequest
	if err := proto.Unmarshal(baseMsg.Data, &req); err != nil {
		zLog.Error("Failed to unmarshal MapAttackRequest", zap.Error(err))
		return
	}

	// GS-4: 请求侧幂等去重——重发的攻击会重复扣血（双倍伤害）。按 requestID 去重，重放直接丢弃：
	// 原始请求已应答，此处不再扣血也不再回包（避免重复伤害的危害大于丢一个重复响应）。
	if meta != nil && meta.RequestID != 0 && !ts.inbox.TryAccept(meta.RequestID) {
		zLog.Warn("Duplicate map attack request ignored",
			zap.Uint64("request_id", meta.RequestID),
			zap.Int64("player_id", req.PlayerId), zap.Int64("target_id", req.TargetId))
		return
	}

	zLog.Info("Map attack request",
		zap.Int64("player_id", req.PlayerId),
		zap.Int64("object_id", req.ObjectId),
		zap.Int64("map_id", req.MapId),
		zap.Int64("target_id", req.TargetId))

	if ts.mapService == nil {
		resp := &protocol.MapAttackResponse{
			Success:  false,
			ErrorMsg: "map service not initialized",
			PlayerId: req.PlayerId,
			TargetId: req.TargetId,
		}
		ts.sendResponse(session, int(protocol.InternalMsgId_MSG_INTERNAL_COMBAT_ACTION), resp, baseMsg, meta)
		return
	}

	damage, targetHp, err := ts.mapService.HandlePlayerAttack(req.PlayerId, req.ObjectId, req.MapId, req.TargetId)
	if err != nil {
		zLog.Error("Failed to handle player attack", zap.Error(err))
		resp := &protocol.MapAttackResponse{
			Success:  false,
			ErrorMsg: err.Error(),
			PlayerId: req.PlayerId,
			TargetId: req.TargetId,
		}
		ts.sendResponse(session, int(protocol.InternalMsgId_MSG_INTERNAL_COMBAT_ACTION), resp, baseMsg, meta)
		return
	}

	resp := &protocol.MapAttackResponse{
		Success:  true,
		PlayerId: req.PlayerId,
		TargetId: req.TargetId,
		Damage:   damage,
		TargetHp: targetHp,
	}
	ts.sendResponse(session, int(protocol.InternalMsgId_MSG_INTERNAL_COMBAT_ACTION), resp, baseMsg, meta)
}

// handleMapPickupRequest 处理玩家就近拾取请求（GameServer → MapServer）。
// 地图侧权威判定并移除掉落物；拾到则把 grant 推回该玩家所在 GameServer 落背包（无阻塞等待）。
func (ts *TCPService) handleMapPickupRequest(session zNet.Session, baseMsg *protocol.BaseMessage, meta *crossserver.Meta) {
	var req protocol.ClientItemPickupRequest
	if err := proto.Unmarshal(baseMsg.Data, &req); err != nil {
		zLog.Error("Failed to unmarshal ClientItemPickupRequest", zap.Error(err))
		return
	}
	// 幂等去重：拾取会移除地图物 + 发放背包，重放会致重复发放，按 requestID 丢弃重放。
	if meta != nil && meta.RequestID != 0 && !ts.inbox.TryAccept(meta.RequestID) {
		zLog.Warn("Duplicate map pickup request ignored",
			zap.Uint64("request_id", meta.RequestID), zap.Int64("player_id", req.PlayerId))
		return
	}
	if ts.mapService == nil {
		return
	}
	itemID, count, ok := ts.mapService.HandlePlayerPickup(req.PlayerId, int64(req.MapId))
	if !ok {
		zLog.Debug("Pickup: nothing in range", zap.Int64("player_id", req.PlayerId))
		return
	}
	zLog.Info("Player picked up item",
		zap.Int64("player_id", req.PlayerId), zap.Int32("item_id", itemID), zap.Int32("count", count))
	ts.notifyItemGrant(req.PlayerId, itemID, count)
}

// handleCrossInstanceEnsure 处理"建/取跨服活动实例"（GameServer → MapServer，跨 realm 物理路由 ④）。
//
// 幂等由 activityID 承担（MapManager.EnsureCrossServerInstance）：多个 realm 的 GameServer 会各自
// 为同一场活动来请求，必须收敛到同一张实例地图。故此处**不**按 requestID 走 inbox 去重——
// 重投得到的是同一个 mapID，重复应答无害；而按 requestID 丢弃重投会让请求方拿不到 mapID 干等超时。
func (ts *TCPService) handleCrossInstanceEnsure(session zNet.Session, baseMsg *protocol.BaseMessage, meta *crossserver.Meta) {
	var req protocol.CrossInstanceEnsureRequest
	if err := proto.Unmarshal(baseMsg.Data, &req); err != nil {
		zLog.Error("Failed to unmarshal CrossInstanceEnsureRequest", zap.Error(err))
		return
	}

	respond := func(resp *protocol.CrossInstanceEnsureResponse) {
		ts.sendResponse(session, int(protocol.InternalMsgId_MSG_INTERNAL_CROSS_INSTANCE_ENSURE_RESPONSE),
			resp, baseMsg, meta)
	}

	if ts.mapService == nil {
		respond(&protocol.CrossInstanceEnsureResponse{
			Success: false, ErrorMsg: "map service not initialized", ActivityId: req.ActivityId,
		})
		return
	}

	_, mapID, created := ts.mapService.EnsureCrossServerInstance(
		req.ActivityId, req.MapConfigId, req.Name, req.Width, req.Height,
		int32(ts.config.Server.ServerID))
	if mapID == 0 {
		respond(&protocol.CrossInstanceEnsureResponse{
			Success: false, ErrorMsg: "invalid activity id", ActivityId: req.ActivityId,
		})
		return
	}

	zLog.Info("Cross-server instance ensured for requester",
		zap.Int64("activity_id", req.ActivityId),
		zap.Int32("map_id", int32(mapID)),
		zap.Bool("created", created),
		zap.Uint32("from_game_server", baseMsg.ServerId))

	respond(&protocol.CrossInstanceEnsureResponse{
		Success:    true,
		ActivityId: req.ActivityId,
		MapId:      int32(mapID),
	})
}

// sendResponse 发送响应消息
func (ts *TCPService) sendResponse(session zNet.Session, msgID int, msg proto.Message, baseMsg *protocol.BaseMessage, meta *crossserver.Meta) {
	data, err := proto.Marshal(msg)
	if err != nil {
		zLog.Error("Failed to marshal response", zap.Error(err))
		return
	}

	// 统一经共用 codec 打包响应（Envelope + protobuf CrossServerMessage）。
	// 响应 meta 由请求 meta 派生，保持 TraceID/RequestID 以便 GameServer 关联等待中的请求。
	respBase := crossserver.BuildBaseMessage(uint32(msgID), baseMsg.PlayerId,
		uint32(ts.config.Server.ServerID), baseMsg.MapId, data)
	respBase.SessionId = baseMsg.SessionId
	respBase.MapServerId = uint32(ts.config.Server.ServerID)

	respMeta := crossserver.NewResponseMetaFromRequest(*meta, crossserver.ServiceTypeMap, int32(ts.config.Server.ServerID))
	enveloped, err := crossserver.PackMessage(respMeta, crossserver.ServiceTypeMap, crossserver.ServiceTypeGame,
		uint32(ts.config.Server.ServerID), baseMsg.ServerId, respBase)
	if err != nil {
		zLog.Error("Failed to pack response message", zap.Error(err))
		return
	}

	if err := session.Send(zNet.ProtoIdType(msgID), enveloped); err != nil {
		zLog.Error("Failed to send response", zap.Error(err))
	}
}
