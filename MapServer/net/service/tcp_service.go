package service

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoServer/MapServer/maps"
	"github.com/pzqf/zMmoServer/zMmoShared/protocol"
	"go.uber.org/zap"
)

type TCPService struct {
	config      *config.Config
	connManager *connection.ConnectionManager
	mapService  *maps.MapManager
	listener    net.Listener
	isRunning   bool
	wg          sync.WaitGroup
}

func NewTCPService(cfg *config.Config, connManager *connection.ConnectionManager, mapService *maps.MapManager) *TCPService {
	return &TCPService{
		config:      cfg,
		connManager: connManager,
		mapService:  mapService,
		isRunning:   false,
	}
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
	// 解析zNet格式的消息
	if len(data) < 16 {
		zLog.Error("Invalid zNet message format from GameServer", zap.Int("size", len(data)))
		return
	}

	// 解析zNet消息头
	protoId := int(binary.BigEndian.Uint32(data[:4]))
	_ = int(binary.BigEndian.Uint32(data[4:8])) // version
	dataLen := int(binary.BigEndian.Uint32(data[8:12]))
	_ = int(binary.BigEndian.Uint32(data[12:16])) // isCompressed

	// 检查数据长度
	if dataLen > 1024*1024 {
		zLog.Error("Message too long from GameServer", zap.Int("length", dataLen))
		return
	}

	// 检查总消息长度
	totalLen := 16 + dataLen
	if len(data) < totalLen {
		zLog.Error("Insufficient data from GameServer", zap.Int("actual", len(data)), zap.Int("expected", totalLen))
		return
	}

	// 提取数据部分
	payload := data[16:totalLen]

	// 根据消息类型处理
	switch protoId {
	case 400: // MSG_INTERNAL_MAP_ENTER_REQUEST
		ts.handleMapEnterRequest(conn, payload)
	case 404: // MSG_INTERNAL_MAP_MOVE_SYNC
		ts.handleMapMoveRequest(conn, payload)
	case 406: // MSG_INTERNAL_MAP_ATTACK_REQUEST
		ts.handleMapAttackRequest(conn, payload)
	default:
		zLog.Info("Received unknown message from GameServer", zap.Int("proto_id", protoId))
	}
}



// handleMapEnterRequest 处理玩家进入地图请求
func (ts *TCPService) handleMapEnterRequest(conn net.Conn, payload []byte) {
	var req protocol.MapEnterRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		zLog.Error("Failed to unmarshal MapEnterRequest", zap.Error(err))
		return
	}

	zLog.Info("Map enter request",
		zap.Int64("player_id", req.PlayerId),
		zap.Int64("map_id", req.MapId),
		zap.Float32("x", req.X),
		zap.Float32("y", req.Y),
		zap.Float32("z", req.Z))

	// 处理玩家进入地图
	if ts.mapService != nil {
		err := ts.mapService.HandlePlayerEnterMap(req.PlayerId, req.MapId, req.X, req.Y, req.Z)
		if err != nil {
			zLog.Error("Failed to handle player enter map", zap.Error(err))
			// 发送失败响应
			resp := &protocol.MapEnterResponse{
				Success:  false,
				ErrorMsg: err.Error(),
			}
			ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_MAP_ENTER_RESPONSE), resp)
			return
		}
	}

	// 发送成功响应
	resp := &protocol.MapEnterResponse{
		Success:  true,
		ObjectId: req.PlayerId,
		MapId:    req.MapId,
		X:        req.X,
		Y:        req.Y,
		Z:        req.Z,
	}
	ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_MAP_ENTER_RESPONSE), resp)
}

// handleMapMoveRequest 处理玩家移动请求
func (ts *TCPService) handleMapMoveRequest(conn net.Conn, payload []byte) {
	var req protocol.MapMoveRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
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

	// 处理玩家移动
	if ts.mapService != nil {
		err := ts.mapService.HandlePlayerMove(req.PlayerId, req.ObjectId, req.MapId, req.X, req.Y, req.Z)
		if err != nil {
			zLog.Error("Failed to handle player move", zap.Error(err))
		}
	}

	// 发送成功响应
	resp := &protocol.MapMoveResponse{
		Success: true,
		ObjectId: req.ObjectId,
		X:        req.X,
		Y:        req.Y,
		Z:        req.Z,
	}
	ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_MAP_MOVE_RESPONSE), resp)
}

// handleMapAttackRequest 处理玩家攻击请求
func (ts *TCPService) handleMapAttackRequest(conn net.Conn, payload []byte) {
	var req protocol.MapAttackRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		zLog.Error("Failed to unmarshal MapAttackRequest", zap.Error(err))
		return
	}

	zLog.Info("Map attack request",
		zap.Int64("player_id", req.PlayerId),
		zap.Int64("object_id", req.ObjectId),
		zap.Int64("map_id", req.MapId),
		zap.Int64("target_id", req.TargetId))

	// 处理玩家攻击
	var damage int64 = 100
	var targetHp int64 = 900

	if ts.mapService != nil {
		damage, targetHp, err := ts.mapService.HandlePlayerAttack(req.PlayerId, req.ObjectId, req.MapId, req.TargetId)
		if err != nil {
			zLog.Error("Failed to handle player attack", zap.Error(err))
		}
	}

	// 发送成功响应
	resp := &protocol.MapAttackResponse{
		Success:  true,
		TargetId: req.TargetId,
		Damage:   damage,
		TargetHp: targetHp,
	}
	ts.sendResponse(conn, int(protocol.InternalMsgId_MSG_MAP_ATTACK_RESPONSE), resp)
}

// sendResponse 发送响应消息
func (ts *TCPService) sendResponse(conn net.Conn, msgID int, msg proto.Message) {
	data, err := proto.Marshal(msg)
	if err != nil {
		zLog.Error("Failed to marshal response", zap.Error(err))
		return
	}

	// 构建zNet格式的消息头：4字节ProtoId + 4字节Version + 4字节DataSize + 4字节IsCompressed
	header := make([]byte, 16)
	protoId := msgID
	version := 1
	dataLen := len(data)
	isCompressed := 0

	// 使用大端序编码
	binary.BigEndian.PutUint32(header[:4], uint32(protoId))
	binary.BigEndian.PutUint32(header[4:8], uint32(version))
	binary.BigEndian.PutUint32(header[8:12], uint32(dataLen))
	binary.BigEndian.PutUint32(header[12:16], uint32(isCompressed))

	// 发送消息
	response := append(header, data...)
	_, err = conn.Write(response)
	if err != nil {
		zLog.Error("Failed to send response", zap.Error(err))
	}
}
