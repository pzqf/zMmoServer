package connection

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// MapRequestHandler 抽象地图业务处理，避免 connection 与 maps 包循环依赖
type MapRequestHandler interface {
	HandlePlayerEnterMap(playerID int64, mapID int64, x, y, z float32) error
	HandlePlayerMove(playerID, objectID, mapID int64, x, y, z float32) error
	HandlePlayerAttack(playerID, objectID, mapID, targetID int64) (int64, int64, error)
}

type ConnectionManager struct {
	config          *config.Config
	gameConnections *zMap.TypedMap[string, *GameConnection] // GameServer ID -> GameServer连接
	mapGameMapping  *zMap.TypedMap[int, string]             // 地图ID -> GameServer ID
	isConnected     bool
	mapHandler      MapRequestHandler
}

type GameConnection struct {
	conn         net.Conn
	gameServerID string
	isConnected  bool
	sendChan     chan []byte
	closeChan    chan struct{}
	closeOnce    sync.Once
}

func NewConnectionManager(cfg *config.Config) *ConnectionManager {
	return &ConnectionManager{
		config:          cfg,
		gameConnections: zMap.NewTypedMap[string, *GameConnection](),
		mapGameMapping:  zMap.NewTypedMap[int, string](),
		isConnected:     false,
	}
}

// SetMapHandler 设置地图请求处理器
func (cm *ConnectionManager) SetMapHandler(handler MapRequestHandler) {
	cm.mapHandler = handler
}

func (cm *ConnectionManager) ConnectToGameServer(gameServerID string, addr string) error {
	// 已存在连接时，仅在连接健康时复用；否则先清理再重连
	if existing, exists := cm.gameConnections.Load(gameServerID); exists {
		if existing != nil && existing.isConnected {
			return nil
		}
		if existing != nil {
			existing.closeOnce.Do(func() {
				close(existing.closeChan)
				if existing.conn != nil {
					_ = existing.conn.Close()
				}
				existing.isConnected = false
			})
		}
		cm.gameConnections.Delete(gameServerID)
	}

	zLog.Info("Connecting to GameServer...", zap.String("game_server_id", gameServerID), zap.String("addr", addr))

	conn, err := net.DialTimeout("tcp", addr, time.Duration(cm.config.GameServer.GameServerConnectTimeout)*time.Second)
	if err != nil {
		zLog.Error("Failed to connect to GameServer", zap.Error(err))
		return err
	}

	gameConn := &GameConnection{
		conn:         conn,
		gameServerID: gameServerID,
		isConnected:  true,
		sendChan:     make(chan []byte, 100),
		closeChan:    make(chan struct{}),
	}

	cm.gameConnections.Store(gameServerID, gameConn)
	cm.isConnected = true
	zLog.Info("Connected to GameServer successfully", zap.String("game_server_id", gameServerID))

	// 启动消息处理
	go cm.receiveFromGameServer(gameConn)
	go gameConn.sendLoop()

	return nil
}

func (cm *ConnectionManager) DisconnectFromGameServer(gameServerID string) {
	if gameConn, exists := cm.gameConnections.LoadAndDelete(gameServerID); exists {
		gameConn.closeOnce.Do(func() {
			close(gameConn.closeChan)
			if gameConn.conn != nil {
				gameConn.conn.Close()
			}
			gameConn.isConnected = false
		})
		zLog.Info("Disconnected from GameServer", zap.String("game_server_id", gameServerID))
	}

	// 清理 map->game 路由中的失效映射
	cm.mapGameMapping.Range(func(mapID int, mappedGameServerID string) bool {
		if mappedGameServerID == gameServerID {
			cm.mapGameMapping.Delete(mapID)
		}
		return true
	})
}

// GetConnectedGameServerIDs 获取当前已连接的 GameServer ID 列表
func (cm *ConnectionManager) GetConnectedGameServerIDs() []string {
	ids := make([]string, 0)
	cm.gameConnections.Range(func(gameServerID string, gameConn *GameConnection) bool {
		ids = append(ids, gameServerID)
		return true
	})
	return ids
}

func (cm *ConnectionManager) RegisterMapToGameServer(mapID int, gameServerID string) {
	cm.mapGameMapping.Store(mapID, gameServerID)
	zLog.Info("Registered map to GameServer", zap.Int("map_id", mapID), zap.String("game_server_id", gameServerID))
}

func (cm *ConnectionManager) SendToGameServer(gameServerID string, data []byte) error {
	gameConn, exists := cm.gameConnections.Load(gameServerID)

	if !exists || !gameConn.isConnected {
		return fmt.Errorf("game server not connected: %s", gameServerID)
	}

	select {
	case gameConn.sendChan <- data:
		return nil
	default:
		return fmt.Errorf("game server send channel full")
	}
}

func (cm *ConnectionManager) SendToGameServerByMap(mapID int, data []byte) error {
	gameServerID, exists := cm.mapGameMapping.Load(mapID)

	if !exists {
		return fmt.Errorf("no game server registered for map: %d", mapID)
	}

	return cm.SendToGameServer(gameServerID, data)
}

func (cm *ConnectionManager) receiveFromGameServer(gameConn *GameConnection) {
	buffer := make([]byte, 4096)
	conn := gameConn.conn
	var pendingData []byte

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	for {
		select {
		case <-gameConn.closeChan:
			return
		default:
			n, err := conn.Read(buffer)
			if err != nil {
				zLog.Error("Failed to read from GameServer", zap.String("game_server_id", gameConn.gameServerID), zap.Error(err))
				cm.DisconnectFromGameServer(gameConn.gameServerID)
				return
			}

			// 重置读取超时
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			if n > 0 {
				// 打印读取到的数据长度和前几个字节，用于调试
				zLog.Debug("Received data from GameServer",
					zap.String("game_server_id", gameConn.gameServerID),
					zap.Int("length", n),
					zap.ByteString("data", buffer[:min(n, 16)]))

				// 将新读取的数据添加到待处理数据中
				pendingData = append(pendingData, buffer[:n]...)
				// 处理待处理数据
				cm.processPendingData(gameConn, &pendingData)
			}
		}
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// processPendingData 处理待处理数据
func (cm *ConnectionManager) processPendingData(gameConn *GameConnection, pendingData *[]byte) {
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
			zap.String("game_server_id", gameConn.gameServerID),
			zap.Int("proto_id", protoId),
			zap.Int("version", version),
			zap.Int("data_len", dataLen),
			zap.Int("is_compressed", isCompressed))

		if dataLen > 1024*1024 {
			zLog.Error("Message too long from GameServer", zap.String("game_server_id", gameConn.gameServerID), zap.Int("length", dataLen))
			// 丢弃此消息，继续处理下一个消息
			*pendingData = (*pendingData)[16:]
			continue
		}

		// 计算总消息长度：16字节头部 + 数据长度
		totalLen := 16 + dataLen
		if len(*pendingData) < totalLen {
			// 数据不足，等待更多数据
			zLog.Debug("Insufficient data",
				zap.String("game_server_id", gameConn.gameServerID),
				zap.Int("available", len(*pendingData)),
				zap.Int("required", totalLen))
			break
		}

		// 提取完整的消息
		message := (*pendingData)[:totalLen]
		// 从待处理数据中移除已处理的消息
		*pendingData = (*pendingData)[totalLen:]

		// 处理消息
		cm.handleGameServerMessage(gameConn, message)
	}
}

func (cm *ConnectionManager) handleGameServerMessage(gameConn *GameConnection, data []byte) {
	// 打印原始数据，用于调试
	zLog.Debug("Raw message data",
		zap.String("game_server_id", gameConn.gameServerID),
		zap.ByteString("data", data))

	// 解析zNet格式的消息
	if len(data) < 16 {
		zLog.Error("Invalid zNet message format from GameServer", zap.String("game_server_id", gameConn.gameServerID), zap.Int("size", len(data)))
		return
	}

	// 解析zNet消息
	protoId := int(binary.BigEndian.Uint32(data[:4]))
	version := int(binary.BigEndian.Uint32(data[4:8]))
	dataLen := int(binary.BigEndian.Uint32(data[8:12]))
	isCompressed := int(binary.BigEndian.Uint32(data[12:16]))

	zLog.Debug("zNet message parsed",
		zap.String("game_server_id", gameConn.gameServerID),
		zap.Int("proto_id", protoId),
		zap.Int("version", version),
		zap.Int("data_len", dataLen),
		zap.Int("is_compressed", isCompressed))

	// 检查数据长度
	if dataLen > 1024*1024 {
		zLog.Error("Message too long from GameServer", zap.String("game_server_id", gameConn.gameServerID), zap.Int("length", dataLen))
		return
	}

	// 检查总消息长度
	totalLen := 16 + dataLen
	if len(data) < totalLen {
		zLog.Error("Insufficient data from GameServer", zap.String("game_server_id", gameConn.gameServerID), zap.Int("actual", len(data)), zap.Int("expected", totalLen))
		return
	}

	// 提取数据部分并解析跨服信封（兼容旧格式）
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
			zap.Int("proto_id", protoId),
			zap.Int("server_id", cm.config.Server.ServerID))
	}

	// 根据消息类型处理
	switch protoId {
	case 400: // MSG_INTERNAL_MAP_ENTER_REQUEST
		// 处理玩家进入地图请求
		cm.handleMapEnterRequest(gameConn, payload, &meta)
	case 404: // MSG_INTERNAL_MAP_MOVE_SYNC
		// 处理玩家移动同步
		cm.handleMapMoveRequest(gameConn, payload, &meta)
	case 406: // MSG_INTERNAL_MAP_ATTACK_REQUEST
		// 处理玩家攻击请求
		cm.handleMapAttackRequest(gameConn, payload, &meta)
	default:
		zLog.Info("Received unknown message from GameServer", zap.String("game_server_id", gameConn.gameServerID), zap.Int("proto_id", protoId))
	}
}

// handleMapEnterRequest 处理玩家进入地图请求
func (cm *ConnectionManager) handleMapEnterRequest(gameConn *GameConnection, payload []byte, reqMeta *crossserver.Meta) {
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

	if cm.mapHandler == nil {
		resp := &protocol.MapEnterResponse{
			Success:  false,
			ErrorMsg: "map handler not configured",
			ObjectId: req.PlayerId,
			MapId:    req.MapId,
		}
		cm.sendGameServerResponse(gameConn, 401, resp, reqMeta)
		return
	}

	if err := cm.mapHandler.HandlePlayerEnterMap(req.PlayerId, req.MapId, req.X, req.Y, req.Z); err != nil {
		resp := &protocol.MapEnterResponse{
			Success:  false,
			ErrorMsg: err.Error(),
			ObjectId: req.PlayerId,
			MapId:    req.MapId,
		}
		cm.sendGameServerResponse(gameConn, 401, resp, reqMeta)
		return
	}

	resp := &protocol.MapEnterResponse{
		Success:  true,
		ObjectId: req.PlayerId,
		MapId:    req.MapId,
		X:        req.X,
		Y:        req.Y,
		Z:        req.Z,
	}

	cm.sendGameServerResponse(gameConn, 401, resp, reqMeta) // MSG_INTERNAL_MAP_ENTER_RESPONSE
}

// handleMapMoveRequest 处理玩家移动请求
func (cm *ConnectionManager) handleMapMoveRequest(gameConn *GameConnection, payload []byte, reqMeta *crossserver.Meta) {
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

	if cm.mapHandler == nil {
		resp := &protocol.MapMoveResponse{
			Success:  false,
			ErrorMsg: "map handler not configured",
			PlayerId: req.PlayerId,
		}
		cm.sendGameServerResponse(gameConn, 405, resp, reqMeta)
		return
	}

	if err := cm.mapHandler.HandlePlayerMove(req.PlayerId, req.ObjectId, req.MapId, req.X, req.Y, req.Z); err != nil {
		resp := &protocol.MapMoveResponse{
			Success:  false,
			ErrorMsg: err.Error(),
			PlayerId: req.PlayerId,
		}
		cm.sendGameServerResponse(gameConn, 405, resp, reqMeta)
		return
	}

	resp := &protocol.MapMoveResponse{
		Success:  true,
		PlayerId: req.PlayerId,
		X:        req.X,
		Y:        req.Y,
		Z:        req.Z,
	}

	cm.sendGameServerResponse(gameConn, 405, resp, reqMeta) // MSG_INTERNAL_MAP_MOVE_RESPONSE
}

// handleMapAttackRequest 处理玩家攻击请求
func (cm *ConnectionManager) handleMapAttackRequest(gameConn *GameConnection, payload []byte, reqMeta *crossserver.Meta) {
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

	if cm.mapHandler == nil {
		resp := &protocol.MapAttackResponse{
			Success:  false,
			ErrorMsg: "map handler not configured",
			PlayerId: req.PlayerId,
			TargetId: req.TargetId,
		}
		cm.sendGameServerResponse(gameConn, 407, resp, reqMeta)
		return
	}

	damage, targetHP, err := cm.mapHandler.HandlePlayerAttack(req.PlayerId, req.ObjectId, req.MapId, req.TargetId)
	if err != nil {
		resp := &protocol.MapAttackResponse{
			Success:  false,
			ErrorMsg: err.Error(),
			PlayerId: req.PlayerId,
			TargetId: req.TargetId,
		}
		cm.sendGameServerResponse(gameConn, 407, resp, reqMeta)
		return
	}

	resp := &protocol.MapAttackResponse{
		Success:  true,
		PlayerId: req.PlayerId,
		TargetId: req.TargetId,
		Damage:   damage,
		TargetHp: targetHP,
	}

	cm.sendGameServerResponse(gameConn, 407, resp, reqMeta) // MSG_INTERNAL_MAP_ATTACK_RESPONSE
}

// sendGameServerResponse 发送响应到GameServer
func (cm *ConnectionManager) sendGameServerResponse(gameConn *GameConnection, msgID int, msg proto.Message, reqMeta *crossserver.Meta) {
	data, err := proto.Marshal(msg)
	if err != nil {
		zLog.Error("Failed to marshal response", zap.Error(err))
		return
	}

	if reqMeta != nil {
		respMeta := crossserver.NewResponseMetaFromRequest(*reqMeta, crossserver.ServiceTypeMap, int32(cm.config.Server.ServerID))
		data = crossserver.Wrap(respMeta, data)
		zLog.Debug("Sending cross-server envelope to GameServer",
			zap.Uint64("trace_id", respMeta.TraceID),
			zap.Uint64("request_id", respMeta.RequestID),
			zap.Int("proto_id", msgID),
			zap.Int("server_id", cm.config.Server.ServerID))
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
	err = cm.SendToGameServer(gameConn.gameServerID, response)
	if err != nil {
		zLog.Error("Failed to send response to GameServer", zap.Error(err))
	}
}

func (gc *GameConnection) sendLoop() {
	var batch []byte
	var batchSize int
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case data := <-gc.sendChan:
			batch = append(batch, data...)
			batchSize += len(data)
			// 当批次达到一定大小或时间到时发送
			if batchSize >= 4096 {
				gc.sendBatch(batch)
				batch = nil
				batchSize = 0
			}
		case <-ticker.C:
			if batchSize > 0 {
				gc.sendBatch(batch)
				batch = nil
				batchSize = 0
			}
		case <-gc.closeChan:
			// 发送剩余数据
			if batchSize > 0 {
				gc.sendBatch(batch)
			}
			return
		}
	}
}

func (gc *GameConnection) sendBatch(batch []byte) {
	if gc.conn != nil {
		_, err := gc.conn.Write(batch)
		if err != nil {
			zLog.Error("Failed to send batch to GameServer", zap.String("game_server_id", gc.gameServerID), zap.Error(err))
			gc.isConnected = false
		}
	}
}
