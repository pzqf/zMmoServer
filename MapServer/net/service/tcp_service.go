package service

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
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
	listener                net.Listener
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

	listener, err := net.Listen("tcp", ts.config.Server.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", ts.config.Server.ListenAddr, err)
	}

	ts.listener = listener
	ts.isRunning = true

	ts.wg.Add(1)
	go ts.acceptConnections(ctx)

	zLog.Info("TCP service started successfully", zap.String("addr", ts.config.Server.ListenAddr))

	return nil
}

func (ts *TCPService) Stop(ctx context.Context) error {
	if !ts.isRunning {
		return nil
	}

	zLog.Info("Stopping TCP service...")

	if ts.listener != nil {
		ts.listener.Close()
	}

	ts.isRunning = false
	ts.wg.Wait()

	zLog.Info("TCP service stopped")

	return nil
}

func (ts *TCPService) acceptConnections(ctx context.Context) {
	defer ts.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := ts.listener.Accept()
			if err != nil {
				if ts.isRunning {
					zLog.Error("Failed to accept connection", zap.Error(err))
				}
				continue
			}

			ts.wg.Add(1)
			go ts.handleConnection(ctx, conn)
		}
	}
}

func (ts *TCPService) handleConnection(ctx context.Context, conn net.Conn) {
	defer ts.wg.Done()
	defer conn.Close()

	gameServerAddr := conn.RemoteAddr().String()
	connID := fmt.Sprintf("%d", time.Now().UnixNano())

	zLog.Info("New GameServer connection", zap.String("conn_id", connID), zap.String("game_server_addr", gameServerAddr))

	// 处理GameServer连接
	ts.handleGameServerConnection(ctx, conn, connID)
}

// handleGameServerConnection 处理GameServer连接
func (ts *TCPService) handleGameServerConnection(ctx context.Context, conn net.Conn, connID string) {
	buffer := make([]byte, 4096)
	var pendingData []byte

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := conn.Read(buffer)
			if err != nil {
				if ts.isRunning {
					zLog.Info("GameServer connection closed", zap.String("conn_id", connID), zap.Error(err))
				}
				return
			}

			if n > 0 {
				conn.SetReadDeadline(time.Now().Add(60 * time.Second))

				// 将新读取的数据添加到待处理数据中
				pendingData = append(pendingData, buffer[:n]...)
				// 处理待处理数据
				ts.processGameServerData(conn, &pendingData)
			}
		}
	}
}

// processGameServerData 处理GameServer数据
func (ts *TCPService) processGameServerData(conn net.Conn, pendingData *[]byte) {
	for {
		// 检查是否有足够的数据来解析zNet消息头（16字节）
		if len(*pendingData) < 16 {
			// 数据不足，等待更多数据
			break
		}

		// 解析zNet格式的消息头
		// zNet消息格式：4字节ProtoId + 4字节Version + 4字节DataSize + 4字节IsCompressed
		protoId := int(binary.BigEndian.Uint32((*pendingData)[:4]))
		version := int(binary.BigEndian.Uint32((*pendingData)[4:8]))
		dataLen := int(binary.BigEndian.Uint32((*pendingData)[8:12]))
		isCompressed := int(binary.BigEndian.Uint32((*pendingData)[12:16]))

		zLog.Debug("zNet message header parsed",
			zap.Int("proto_id", protoId),
			zap.Int("version", version),
			zap.Int("data_len", dataLen),
			zap.Int("is_compressed", isCompressed))

		if dataLen > 1024*1024 {
			zLog.Error("Message too long from GameServer", zap.Int("length", dataLen))
			// 丢弃此消息，继续处理下一个消息
			*pendingData = (*pendingData)[16:]
			continue
		}

		// 计算总消息长度：16字节头部 + 数据长度
		totalLen := 16 + dataLen
		if len(*pendingData) < totalLen {
			// 数据不足，等待更多数据
			zLog.Debug("Insufficient data",
				zap.Int("available", len(*pendingData)),
				zap.Int("required", totalLen))
			break
		}

		// 提取完整的消息
		message := (*pendingData)[:totalLen]
		// 从待处理数据中移除已处理的消息
		*pendingData = (*pendingData)[totalLen:]

		// 处理消息
		ts.handleGameServerMessage(conn, message)
	}
}

// handleGameServerMessage 处理来自GameServer的消息
func (ts *TCPService) handleGameServerMessage(conn net.Conn, data []byte) {
	if len(data) < 16 {
		zLog.Error("Invalid zNet message format from GameServer", zap.Int("size", len(data)))
		return
	}

	protoId := int(binary.BigEndian.Uint32(data[:4]))
	_ = int(binary.BigEndian.Uint32(data[4:8]))
	dataLen := int(binary.BigEndian.Uint32(data[8:12]))
	_ = int(binary.BigEndian.Uint32(data[12:16]))

	if dataLen > 1024*1024 {
		zLog.Error("Message too long from GameServer", zap.Int("length", dataLen))
		return
	}

	totalLen := 16 + dataLen
	if len(data) < totalLen {
		zLog.Error("Insufficient data from GameServer", zap.Int("actual", len(data)), zap.Int("expected", totalLen))
		return
	}

	rawPayload := data[16:totalLen]
	meta, payload, wrapped, unwrapErr := crossserver.Unwrap(rawPayload)
	if unwrapErr != nil {
		zLog.Error("Invalid cross-server envelope from GameServer", zap.Error(unwrapErr))
		return
	}
	if wrapped {
		zLog.Debug("Received cross-server envelope from GameServer",
			zap.Uint64("trace_id", meta.TraceID),
			zap.Uint64("request_id", meta.RequestID),
			zap.Int("proto_id", protoId))
	}

	var crossMsg crossserver.CrossServerMessage
	if err := json.Unmarshal(payload, &crossMsg); err != nil {
		zLog.Error("Failed to unmarshal cross server message", zap.Error(err))
		return
	}

	baseMsg := crossMsg.Message

	switch protoId {
	case 400:
		ts.handleMapEnterRequest(conn, &baseMsg, &meta)
	case 402:
		ts.handleMapLeaveRequest(conn, &baseMsg, &meta)
	case 404:
		ts.handleMapMoveRequest(conn, &baseMsg, &meta)
	case 406:
		ts.handleMapAttackRequest(conn, &baseMsg, &meta)
	default:
		zLog.Info("Received unknown message from GameServer", zap.Int("proto_id", protoId))
	}
}

// handleMapEnterRequest 处理玩家进入地图请求
func (ts *TCPService) handleMapEnterRequest(conn net.Conn, baseMsg *crossserver.BaseMessage, meta *crossserver.Meta) {
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
			ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_RESPONSE), resp, baseMsg, meta)
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
	ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_RESPONSE), resp, baseMsg, meta)
}

func (ts *TCPService) handleMapLeaveRequest(conn net.Conn, baseMsg *crossserver.BaseMessage, meta *crossserver.Meta) {
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
			ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_LEAVE_RESPONSE), resp, baseMsg, meta)
			return
		}
	}

	resp := &protocol.MapLeaveResponse{
		Success:  true,
		PlayerId: req.PlayerId,
	}
	ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_LEAVE_RESPONSE), resp, baseMsg, meta)
}

// handleMapMoveRequest 处理玩家移动请求
func (ts *TCPService) handleMapMoveRequest(conn net.Conn, baseMsg *crossserver.BaseMessage, meta *crossserver.Meta) {
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
	ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_MOVE_SYNC), resp, baseMsg, meta)
}

// handleMapAttackRequest 处理玩家攻击请求
func (ts *TCPService) handleMapAttackRequest(conn net.Conn, baseMsg *crossserver.BaseMessage, meta *crossserver.Meta) {
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
		ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_INTERNAL_COMBAT_ACTION), resp, baseMsg, meta)
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
		ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_INTERNAL_COMBAT_ACTION), resp, baseMsg, meta)
		return
	}

	resp := &protocol.MapAttackResponse{
		Success:  true,
		PlayerId: req.PlayerId,
		TargetId: req.TargetId,
		Damage:   damage,
		TargetHp: targetHp,
	}
	ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_INTERNAL_COMBAT_ACTION), resp, baseMsg, meta)
}

// sendResponse 发送响应消息
func (ts *TCPService) sendResponse(conn net.Conn, msgID int, msg proto.Message, baseMsg *crossserver.BaseMessage, meta *crossserver.Meta) {
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

	header := make([]byte, 16)
	binary.BigEndian.PutUint32(header[:4], uint32(msgID))
	binary.BigEndian.PutUint32(header[4:8], uint32(1))
	binary.BigEndian.PutUint32(header[8:12], uint32(len(crossMsgData)))
	binary.BigEndian.PutUint32(header[12:16], uint32(0))

	responseData := append(header, crossMsgData...)
	_, err = conn.Write(responseData)
	if err != nil {
		zLog.Error("Failed to send response", zap.Error(err))
	}
}
