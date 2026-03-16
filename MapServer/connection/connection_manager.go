package connection

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type ConnectionManager struct {
	config          *config.Config
	gameConnections map[string]*GameConnection // GameServer ID -> GameServer连接
	gameConnMu      sync.RWMutex
	mapGameMapping  map[int]string // 地图ID -> GameServer ID
	mapMu           sync.RWMutex
	isConnected     bool
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
		gameConnections: make(map[string]*GameConnection),
		mapGameMapping:  make(map[int]string),
		isConnected:     false,
	}
}

func (cm *ConnectionManager) ConnectToGameServer(gameServerID string, addr string) error {
	cm.gameConnMu.Lock()
	defer cm.gameConnMu.Unlock()

	// 检查是否已连接
	if _, exists := cm.gameConnections[gameServerID]; exists {
		return nil
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

	cm.gameConnections[gameServerID] = gameConn
	cm.isConnected = true
	zLog.Info("Connected to GameServer successfully", zap.String("game_server_id", gameServerID))

	// 启动消息处理
	go cm.receiveFromGameServer(gameConn)
	go gameConn.sendLoop()

	return nil
}

func (cm *ConnectionManager) DisconnectFromGameServer(gameServerID string) {
	cm.gameConnMu.Lock()
	defer cm.gameConnMu.Unlock()

	if gameConn, exists := cm.gameConnections[gameServerID]; exists {
		gameConn.closeOnce.Do(func() {
			close(gameConn.closeChan)
			if gameConn.conn != nil {
				gameConn.conn.Close()
			}
			gameConn.isConnected = false
		})
		delete(cm.gameConnections, gameServerID)
		zLog.Info("Disconnected from GameServer", zap.String("game_server_id", gameServerID))
	}
}

func (cm *ConnectionManager) RegisterMapToGameServer(mapID int, gameServerID string) {
	cm.mapMu.Lock()
	defer cm.mapMu.Unlock()

	cm.mapGameMapping[mapID] = gameServerID
	zLog.Info("Registered map to GameServer", zap.Int("map_id", mapID), zap.String("game_server_id", gameServerID))
}

func (cm *ConnectionManager) SendToGameServer(gameServerID string, data []byte) error {
	cm.gameConnMu.RLock()
	gameConn, exists := cm.gameConnections[gameServerID]
	cm.gameConnMu.RUnlock()

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
	cm.mapMu.RLock()
	gameServerID, exists := cm.mapGameMapping[mapID]
	cm.mapMu.RUnlock()

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
				// 标记连接为断开
				gameConn.isConnected = false
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

	// 解析zNet消息头
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

	// 提取数据部分
	payload := data[16:totalLen]

	// 根据消息类型处理
	switch protoId {
	case 400: // MSG_INTERNAL_MAP_ENTER_REQUEST
		// 处理玩家进入地图请求
		cm.handleMapEnterRequest(gameConn, payload)
	case 404: // MSG_INTERNAL_MAP_MOVE_SYNC
		// 处理玩家移动同步
		cm.handleMapMoveRequest(gameConn, payload)
	case 406: // MSG_INTERNAL_MAP_ATTACK_REQUEST
		// 处理玩家攻击请求
		cm.handleMapAttackRequest(gameConn, payload)
	default:
		zLog.Info("Received unknown message from GameServer", zap.String("game_server_id", gameConn.gameServerID), zap.Int("proto_id", protoId))
	}
}



// handleMapEnterRequest 处理玩家进入地图请求
func (cm *ConnectionManager) handleMapEnterRequest(gameConn *GameConnection, payload []byte) {
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

	// 这里可以添加地图进入逻辑

	// 发送成功响应
	resp := &protocol.MapEnterResponse{
		Success:  true,
		ObjectId: req.PlayerId,
		MapId:    req.MapId,
		X:        req.X,
		Y:        req.Y,
		Z:        req.Z,
	}

	cm.sendGameServerResponse(gameConn, 401, resp) // MSG_INTERNAL_MAP_ENTER_RESPONSE
}

// handleMapMoveRequest 处理玩家移动请求
func (cm *ConnectionManager) handleMapMoveRequest(gameConn *GameConnection, payload []byte) {
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

	// 这里可以添加移动逻辑

	// 发送成功响应
	resp := &protocol.MapMoveResponse{
		Success:  true,
		PlayerId: req.PlayerId,
		X:        req.X,
		Y:        req.Y,
		Z:        req.Z,
	}

	cm.sendGameServerResponse(gameConn, 405, resp) // MSG_INTERNAL_MAP_MOVE_RESPONSE
}

// handleMapAttackRequest 处理玩家攻击请求
func (cm *ConnectionManager) handleMapAttackRequest(gameConn *GameConnection, payload []byte) {
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

	// 这里可以添加攻击逻辑

	// 发送成功响应
	resp := &protocol.MapAttackResponse{
		Success:  true,
		PlayerId: req.PlayerId,
		TargetId: req.TargetId,
		Damage:   10, // 模拟伤害
		TargetHp: 90, // 模拟目标剩余血量
	}

	cm.sendGameServerResponse(gameConn, 407, resp) // MSG_INTERNAL_MAP_ATTACK_RESPONSE
}



// sendGameServerResponse 发送响应到GameServer
func (cm *ConnectionManager) sendGameServerResponse(gameConn *GameConnection, msgID int, msg proto.Message) {
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
	err = cm.SendToGameServer(gameConn.gameServerID, response)
	if err != nil {
		zLog.Error("Failed to send response to GameServer", zap.Error(err))
	}
}

func (gc *GameConnection) sendLoop() {
	for {
		select {
		case data := <-gc.sendChan:
			if gc.conn != nil {
				_, err := gc.conn.Write(data)
				if err != nil {
					zLog.Error("Failed to send to GameServer", zap.String("game_server_id", gc.gameServerID), zap.Error(err))
					gc.isConnected = false
				}
			}
		case <-gc.closeChan:
			return
		}
	}
}




