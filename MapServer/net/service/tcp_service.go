package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoServer/MapServer/maps"
	"go.uber.org/zap"
)

type TCPService struct {
	config                  *config.Config
	connManager             *connection.ConnectionManager
	mapService              *maps.MapManager
	playerGameServerManager *maps.PlayerGameServerManager
	tcpServer               *zNet.TcpServer
	isRunning               bool
	wg                      sync.WaitGroup
}

func NewTCPService(cfg *config.Config, connManager *connection.ConnectionManager, mapService *maps.MapManager) *TCPService {
	return &TCPService{
		config:      cfg,
		connManager: connManager,
		mapService:  mapService,
		isRunning:   false,
	}
}

func (ts *TCPService) SetPlayerGameServerManager(manager *maps.PlayerGameServerManager) {
	ts.playerGameServerManager = manager
}

func (ts *TCPService) Name() string {
	return "TCPService"
}

func (ts *TCPService) Start(ctx context.Context) error {
	if ts.isRunning {
		return nil
	}

	zLog.Info("Starting TCP service...", zap.String("addr", ts.config.Server.ListenAddr))

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

	// 创建zNet.TcpServer
	ts.tcpServer = zNet.NewTcpServer(tcpConfig)

	// 注册消息处理器
	ts.tcpServer.RegisterDispatcher(ts.handleConnectionMessage)

	// 启动服务
	err := ts.tcpServer.Start()
	if err != nil {
		return fmt.Errorf("failed to start TCP service: %v", err)
	}

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

	ts.isRunning = false
	ts.wg.Wait()

	zLog.Info("TCP service stopped")

	return nil
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

// handleGameServerMessage 处理来自GameServer的消息
func (ts *TCPService) handleGameServerMessage(session zNet.Session, protoId int32, data []byte) {
	meta, payload, wrapped, unwrapErr := crossserver.Unwrap(data)
	if unwrapErr != nil {
		zLog.Error("Invalid cross-server envelope from GameServer", zap.Error(unwrapErr))
		return
	}
	if wrapped {
		zLog.Debug("Received cross-server envelope from GameServer",
			zap.Uint64("trace_id", meta.TraceID),
			zap.Uint64("request_id", meta.RequestID),
			zap.Int32("proto_id", protoId))
	}

	var crossMsg crossserver.CrossServerMessage
	if err := json.Unmarshal(payload, &crossMsg); err != nil {
		zLog.Error("Failed to unmarshal cross server message", zap.Error(err))
		return
	}

	baseMsg := crossMsg.Message

	switch protoId {
	case 400:
		ts.handleMapEnterRequest(session, &baseMsg, &meta)
	case 402:
		ts.handleMapLeaveRequest(session, &baseMsg, &meta)
	case 404:
		ts.handleMapMoveRequest(session, &baseMsg, &meta)
	case 406:
		ts.handleMapAttackRequest(session, &baseMsg, &meta)
	default:
		zLog.Info("Received unknown message from GameServer", zap.Int32("proto_id", protoId))
	}
}

// handleMapEnterRequest 处理玩家进入地图请求
func (ts *TCPService) handleMapEnterRequest(session zNet.Session, baseMsg *crossserver.BaseMessage, meta *crossserver.Meta) {
	var req protocol.MapEnterRequest
	if err := proto.Unmarshal(baseMsg.Data, &req); err != nil {
		zLog.Error("Failed to unmarshal MapEnterRequest", zap.Error(err))
		return
	}

	zLog.Info("Map enter request",
		zap.Int64("player_id", req.PlayerId),
		zap.Int64("map_id", req.MapId),
		zap.Uint32("game_server_id", baseMsg.ServerID),
		zap.Float32("x", req.X),
		zap.Float32("y", req.Y),
		zap.Float32("z", req.Z))

	if ts.playerGameServerManager != nil {
		ts.playerGameServerManager.SetPlayerGameServer(
			id.PlayerIdType(req.PlayerId),
			baseMsg.ServerID,
			"",
			id.MapIdType(req.MapId),
		)
	}

	if ts.mapService != nil {
		err := ts.mapService.HandlePlayerEnterMap(req.PlayerId, req.MapId, req.X, req.Y, req.Z)
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

func (ts *TCPService) handleMapLeaveRequest(session zNet.Session, baseMsg *crossserver.BaseMessage, meta *crossserver.Meta) {
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
func (ts *TCPService) handleMapMoveRequest(session zNet.Session, baseMsg *crossserver.BaseMessage, meta *crossserver.Meta) {
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
func (ts *TCPService) handleMapAttackRequest(session zNet.Session, baseMsg *crossserver.BaseMessage, meta *crossserver.Meta) {
	var req protocol.MapAttackRequest
	if err := proto.Unmarshal(baseMsg.Data, &req); err != nil {
		zLog.Error("Failed to unmarshal MapAttackRequest", zap.Error(err))
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

// sendResponse 发送响应消息
func (ts *TCPService) sendResponse(session zNet.Session, msgID int, msg proto.Message, baseMsg *crossserver.BaseMessage, meta *crossserver.Meta) {
	data, err := proto.Marshal(msg)
	if err != nil {
		zLog.Error("Failed to marshal response", zap.Error(err))
		return
	}

	respBaseMsg := crossserver.BaseMessage{
		MsgID:       uint32(msgID),
		SessionID:   baseMsg.SessionID,
		PlayerID:    baseMsg.PlayerID,
		ServerID:    uint32(ts.config.Server.ServerID),
		Timestamp:   uint64(time.Now().Unix()),
		Data:        data,
		MapID:       baseMsg.MapID,
		MapServerID: uint32(ts.config.Server.ServerID),
	}

	respCrossMsg := crossserver.CrossServerMessage{
		TraceID:      meta.TraceID,
		FromService:  crossserver.ServiceTypeMap,
		ToService:    crossserver.ServiceTypeGame,
		FromServerID: uint32(ts.config.Server.ServerID),
		ToServerID:   baseMsg.ServerID,
		Message:      respBaseMsg,
	}

	crossMsgData, err := json.Marshal(respCrossMsg)
	if err != nil {
		zLog.Error("Failed to marshal cross server message", zap.Error(err))
		return
	}

	// 使用zNet发送消息
	err = session.Send(zNet.ProtoIdType(msgID), crossMsgData)
	if err != nil {
		zLog.Error("Failed to send response", zap.Error(err))
	}
}
