package maps

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoServer/GameServer/game/object"
	"github.com/pzqf/zMmoServer/GameServer/net/protolayer"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// MapService 地图服务
type MapService struct {
	config        *config.Config
	protocol      protolayer.Protocol
	maps          map[id.MapIdType]*Map
	mapsMu        sync.RWMutex
	mapServerAddr string
	mapServerConn net.Conn // 与MapServer的连接
	connMu        sync.Mutex
}

// NewMapService 创建地图服务
func NewMapService(cfg *config.Config, protocol protolayer.Protocol) *MapService {
	return &MapService{
		config:        cfg,
		protocol:      protocol,
		maps:          make(map[id.MapIdType]*Map),
		mapServerAddr: cfg.MapServer.MapServerAddr,
	}
}

// Start 启动地图服务
func (ms *MapService) Start(ctx context.Context) error {
	zLog.Info("Starting MapService...")

	// 加载地图
	ms.loadMaps()

	// 连接到MapServer
	err := ms.connectToMapServer()
	if err != nil {
		zLog.Error("Failed to connect to MapServer", zap.Error(err))
		// 继续启动，MapServer连接失败不影响服务启动
	}

	zLog.Info("MapService started successfully")
	return nil
}

// connectToMapServer 连接到MapServer
func (ms *MapService) connectToMapServer() error {
	zLog.Info("Connecting to MapServer...", zap.String("addr", ms.mapServerAddr))

	conn, err := net.Dial("tcp", ms.mapServerAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to MapServer: %w", err)
	}

	ms.connMu.Lock()
	ms.mapServerConn = conn
	ms.connMu.Unlock()

	zLog.Info("Connected to MapServer", zap.String("addr", ms.mapServerAddr))

	// 发送认证请求
	return ms.sendMapServerAuthRequest(conn)
}

// sendMapServerAuthRequest 发送MapServer认证请求
func (ms *MapService) sendMapServerAuthRequest(conn net.Conn) error {
	req := &protocol.MapServerAuthRequest{
		MapServerId: "MapServer-1",
		Version:     "1.0.0",
	}

	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %w", err)
	}

	// 构建zNet格式的消息头：4字节ProtoId + 4字节Version + 4字节DataSize + 4字节IsCompressed
	header := make([]byte, 16)
	protoId := 400 // MSG_INTERNAL_MAP_ENTER_REQUEST
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
		return fmt.Errorf("failed to send auth request: %w", err)
	}

	// 读取响应
	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	// 解析zNet格式的响应
	if n < 16 {
		return fmt.Errorf("invalid response length: %d", n)
	}

	respProtoId := int(binary.BigEndian.Uint32(buffer[:4]))
	_ = int(binary.BigEndian.Uint32(buffer[4:8]))
	respLen := int(binary.BigEndian.Uint32(buffer[8:12]))
	_ = int(binary.BigEndian.Uint32(buffer[12:16]))

	if respProtoId != 401 { // MSG_INTERNAL_MAP_ENTER_RESPONSE
		return fmt.Errorf("unexpected response message ID: %d", respProtoId)
	}

	if n < 16+respLen {
		return fmt.Errorf("response length mismatch: expected %d, got %d", 16+respLen, n)
	}

	var resp protocol.MapServerAuthResponse
	if err := proto.Unmarshal(buffer[16:16+respLen], &resp); err != nil {
		return fmt.Errorf("failed to unmarshal auth response: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("MapServer auth failed: %s", resp.ErrorMsg)
	}

	zLog.Info("MapServer auth successful", zap.String("game_server_id", resp.GameServerId))
	return nil
}

// Stop 停止地图服务
func (ms *MapService) Stop(ctx context.Context) error {
	zLog.Info("Stopping MapService...")

	// 清理地图
	ms.mapsMu.Lock()
	ms.maps = make(map[id.MapIdType]*Map)
	ms.mapsMu.Unlock()

	zLog.Info("MapService stopped")
	return nil
}

// loadMaps 加载地图
func (ms *MapService) loadMaps() {
	// 加载新手村地图
	mapID := id.MapIdType(1001)
	mapName := "新手村"
	width, height := float32(500), float32(500)

	ms.mapsMu.Lock()
	ms.maps[mapID] = NewMap(mapID, 1001, mapName, width, height)
	ms.mapsMu.Unlock()

	zLog.Info("Map loaded", zap.Int32("map_id", int32(mapID)), zap.String("name", mapName))
}

// GetMap 获取地图
func (ms *MapService) GetMap(mapID id.MapIdType) (*Map, error) {
	ms.mapsMu.RLock()
	defer ms.mapsMu.RUnlock()

	if m, exists := ms.maps[mapID]; exists {
		return m, nil
	}

	return nil, fmt.Errorf("map not found: %d", mapID)
}

// HandlePlayerEnterMap 处理玩家进入地图
func (ms *MapService) HandlePlayerEnterMap(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	// 先在本地处理
	// 获取地图
	m, err := ms.GetMap(mapID)
	if err != nil {
		return err
	}

	// 创建玩家游戏对象
	playerObj := object.NewGameObjectWithType(id.ObjectIdType(playerID), "player", common.GameObjectTypePlayer)
	playerObj.SetPosition(pos)

	// 添加玩家到地图
	m.AddPlayer(playerID, playerObj)

	// 向MapServer发送进入地图请求
	err = ms.sendMapEnterRequest(playerID, mapID, pos)
	if err != nil {
		zLog.Warn("Failed to send map enter request to MapServer", zap.Error(err))
		// 继续执行，MapServer通信失败不影响本地处理
	}

	zLog.Info("Player entered map",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)),
		zap.Float32("x", pos.X),
		zap.Float32("y", pos.Y),
		zap.Float32("z", pos.Z))

	return nil
}

// sendMapEnterRequest 发送进入地图请求到MapServer
func (ms *MapService) sendMapEnterRequest(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	ms.connMu.Lock()
	conn := ms.mapServerConn
	ms.connMu.Unlock()

	if conn == nil {
		// 尝试重新连接
		err := ms.connectToMapServer()
		if err != nil {
			return fmt.Errorf("no connection to MapServer: %w", err)
		}

		ms.connMu.Lock()
		conn = ms.mapServerConn
		ms.connMu.Unlock()

		if conn == nil {
			return fmt.Errorf("failed to connect to MapServer")
		}
	}

	req := &protocol.MapEnterRequest{
		PlayerId:     int64(playerID),
		SessionId:    0, // 暂时设为0
		MapId:        int64(mapID),
		X:            pos.X,
		Y:            pos.Y,
		Z:            pos.Z,
		GameServerId: 1,        // 暂时设为1
		PlayerData:   []byte{}, // 暂时为空
	}

	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal map enter request: %w", err)
	}

	// 构建zNet格式的消息头：4字节ProtoId + 4字节Version + 4字节DataSize + 4字节IsCompressed
	header := make([]byte, 16)
	protoId := 400 // MSG_INTERNAL_MAP_ENTER_REQUEST
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
		// 连接可能已断开，重置连接
		ms.connMu.Lock()
		ms.mapServerConn = nil
		ms.connMu.Unlock()
		return fmt.Errorf("failed to send map enter request: %w", err)
	}

	// 读取响应
	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		// 连接可能已断开，重置连接
		ms.connMu.Lock()
		ms.mapServerConn = nil
		ms.connMu.Unlock()
		return fmt.Errorf("failed to read map enter response: %w", err)
	}

	// 解析zNet格式的响应
	if n < 16 {
		return fmt.Errorf("invalid response length: %d", n)
	}

	respProtoId := int(binary.BigEndian.Uint32(buffer[:4]))
	_ = int(binary.BigEndian.Uint32(buffer[4:8]))
	respLen := int(binary.BigEndian.Uint32(buffer[8:12]))
	_ = int(binary.BigEndian.Uint32(buffer[12:16]))

	if respProtoId != 401 { // MSG_INTERNAL_MAP_ENTER_RESPONSE
		return fmt.Errorf("unexpected response message ID: %d", respProtoId)
	}

	if n < 16+respLen {
		return fmt.Errorf("response length mismatch: expected %d, got %d", 16+respLen, n)
	}

	var resp protocol.MapEnterResponse
	if err := proto.Unmarshal(buffer[16:16+respLen], &resp); err != nil {
		return fmt.Errorf("failed to unmarshal map enter response: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("MapServer enter map failed: %s", resp.ErrorMsg)
	}

	zLog.Info("MapServer enter map successful",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("object_id", resp.ObjectId),
		zap.Int64("map_id", resp.MapId))

	return nil
}

// HandlePlayerLeaveMap 处理玩家离开地图
func (ms *MapService) HandlePlayerLeaveMap(playerID id.PlayerIdType, mapID id.MapIdType) error {
	// 获取地图
	m, err := ms.GetMap(mapID)
	if err != nil {
		return err
	}

	// 从地图移除玩家
	m.RemovePlayer(playerID)

	zLog.Info("Player left map",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)))

	return nil
}

// HandlePlayerMove 处理玩家移动
func (ms *MapService) HandlePlayerMove(playerID id.PlayerIdType, mapID id.MapIdType, targetPos common.Vector3) error {
	// 先在本地处理
	// 获取地图
	m, err := ms.GetMap(mapID)
	if err != nil {
		return err
	}

	// 获取玩家对象
	playerObj := m.GetObject(id.ObjectIdType(playerID))
	if playerObj == nil {
		return fmt.Errorf("player not found in map: %d", playerID)
	}

	// 移动玩家
	err = m.MoveObject(playerObj, targetPos)
	if err != nil {
		return err
	}

	// 向MapServer发送移动请求
	err = ms.sendMapMoveRequest(playerID, id.ObjectIdType(playerID), mapID, targetPos)
	if err != nil {
		zLog.Warn("Failed to send map move request to MapServer", zap.Error(err))
		// 继续执行，MapServer通信失败不影响本地处理
	}

	zLog.Debug("Player moved",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)),
		zap.Float32("x", targetPos.X),
		zap.Float32("y", targetPos.Y),
		zap.Float32("z", targetPos.Z))

	return nil
}

// sendMapMoveRequest 发送移动请求到MapServer
func (ms *MapService) sendMapMoveRequest(playerID id.PlayerIdType, objectID id.ObjectIdType, mapID id.MapIdType, pos common.Vector3) error {
	ms.connMu.Lock()
	conn := ms.mapServerConn
	ms.connMu.Unlock()

	if conn == nil {
		// 尝试重新连接
		err := ms.connectToMapServer()
		if err != nil {
			return fmt.Errorf("no connection to MapServer: %w", err)
		}

		ms.connMu.Lock()
		conn = ms.mapServerConn
		ms.connMu.Unlock()

		if conn == nil {
			return fmt.Errorf("failed to connect to MapServer")
		}
	}

	req := &protocol.MapMoveRequest{
		PlayerId: int64(playerID),
		ObjectId: int64(objectID),
		MapId:    int64(mapID),
		X:        pos.X,
		Y:        pos.Y,
		Z:        pos.Z,
	}

	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal map move request: %w", err)
	}

	// 构建zNet格式的消息头：4字节ProtoId + 4字节Version + 4字节DataSize + 4字节IsCompressed
	header := make([]byte, 16)
	protoId := 404 // MSG_INTERNAL_MAP_MOVE_SYNC
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
		// 连接可能已断开，重置连接
		ms.connMu.Lock()
		ms.mapServerConn = nil
		ms.connMu.Unlock()
		return fmt.Errorf("failed to send map move request: %w", err)
	}

	// 读取响应
	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		// 连接可能已断开，重置连接
		ms.connMu.Lock()
		ms.mapServerConn = nil
		ms.connMu.Unlock()
		return fmt.Errorf("failed to read map move response: %w", err)
	}

	// 解析zNet格式的响应
	if n < 16 {
		return fmt.Errorf("invalid response length: %d", n)
	}

	respProtoId := int(binary.BigEndian.Uint32(buffer[:4]))
	_ = int(binary.BigEndian.Uint32(buffer[4:8]))
	respLen := int(binary.BigEndian.Uint32(buffer[8:12]))
	_ = int(binary.BigEndian.Uint32(buffer[12:16]))

	if respProtoId != 405 { // MSG_INTERNAL_MAP_MOVE_RESPONSE
		return fmt.Errorf("unexpected response message ID: %d", respProtoId)
	}

	if n < 16+respLen {
		return fmt.Errorf("response length mismatch: expected %d, got %d", 16+respLen, n)
	}

	var resp protocol.MapMoveResponse
	if err := proto.Unmarshal(buffer[16:16+respLen], &resp); err != nil {
		return fmt.Errorf("failed to unmarshal map move response: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("MapServer move failed")
	}

	zLog.Debug("MapServer move successful",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("resp_player_id", resp.PlayerId))

	return nil
}

// HandlePlayerAttack 处理玩家攻击
func (ms *MapService) HandlePlayerAttack(playerID id.PlayerIdType, mapID id.MapIdType, targetID id.ObjectIdType) (int64, int64, error) {
	// 先在本地处理
	// 获取地图
	m, err := ms.GetMap(mapID)
	if err != nil {
		return 0, 0, err
	}

	// 获取玩家对象
	playerObj := m.GetObject(id.ObjectIdType(playerID))
	if playerObj == nil {
		return 0, 0, fmt.Errorf("player not found in map: %d", playerID)
	}

	// 获取目标对象
	targetObj := m.GetObject(targetID)
	if targetObj == nil {
		return 0, 0, fmt.Errorf("target not found in map: %d", targetID)
	}

	// 检查目标类型是否为怪物
	if targetObj.GetType() != common.GameObjectTypeMonster {
		return 0, 0, fmt.Errorf("target is not a monster: %d", targetID)
	}

	// 向MapServer发送攻击请求
	damage, targetHP, err := ms.sendMapAttackRequest(playerID, id.ObjectIdType(playerID), mapID, targetID)
	if err != nil {
		zLog.Warn("Failed to send map attack request to MapServer", zap.Error(err))
		// 继续执行，MapServer通信失败不影响本地处理
		// 使用默认伤害值
		damage = 10
		targetHP = 90
	}

	zLog.Info("Player attacked monster",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("target_id", int64(targetID)),
		zap.Int32("map_id", int32(mapID)),
		zap.Int64("damage", damage),
		zap.Int64("target_hp", targetHP))

	// 这里可以添加战斗逻辑

	return damage, targetHP, nil
}

// sendMapAttackRequest 发送攻击请求到MapServer
func (ms *MapService) sendMapAttackRequest(playerID id.PlayerIdType, objectID id.ObjectIdType, mapID id.MapIdType, targetID id.ObjectIdType) (int64, int64, error) {
	ms.connMu.Lock()
	conn := ms.mapServerConn
	ms.connMu.Unlock()

	if conn == nil {
		// 尝试重新连接
		err := ms.connectToMapServer()
		if err != nil {
			return 0, 0, fmt.Errorf("no connection to MapServer: %w", err)
		}

		ms.connMu.Lock()
		conn = ms.mapServerConn
		ms.connMu.Unlock()

		if conn == nil {
			return 0, 0, fmt.Errorf("failed to connect to MapServer")
		}
	}

	req := &protocol.MapAttackRequest{
		PlayerId: int64(playerID),
		ObjectId: int64(objectID),
		MapId:    int64(mapID),
		TargetId: int64(targetID),
	}

	data, err := proto.Marshal(req)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal map attack request: %w", err)
	}

	// 构建zNet格式的消息头：4字节ProtoId + 4字节Version + 4字节DataSize + 4字节IsCompressed
	header := make([]byte, 16)
	protoId := 406 // MSG_INTERNAL_MAP_ATTACK_REQUEST
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
		// 连接可能已断开，重置连接
		ms.connMu.Lock()
		ms.mapServerConn = nil
		ms.connMu.Unlock()
		return 0, 0, fmt.Errorf("failed to send map attack request: %w", err)
	}

	// 读取响应
	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		// 连接可能已断开，重置连接
		ms.connMu.Lock()
		ms.mapServerConn = nil
		ms.connMu.Unlock()
		return 0, 0, fmt.Errorf("failed to read map attack response: %w", err)
	}

	// 解析zNet格式的响应
	if n < 16 {
		return 0, 0, fmt.Errorf("invalid response length: %d", n)
	}

	respProtoId := int(binary.BigEndian.Uint32(buffer[:4]))
	_ = int(binary.BigEndian.Uint32(buffer[4:8]))
	respLen := int(binary.BigEndian.Uint32(buffer[8:12]))
	_ = int(binary.BigEndian.Uint32(buffer[12:16]))

	if respProtoId != 407 { // MSG_INTERNAL_MAP_ATTACK_RESPONSE
		return 0, 0, fmt.Errorf("unexpected response message ID: %d", respProtoId)
	}

	if n < 16+respLen {
		return 0, 0, fmt.Errorf("response length mismatch: expected %d, got %d", 16+respLen, n)
	}

	var resp protocol.MapAttackResponse
	if err := proto.Unmarshal(buffer[16:16+respLen], &resp); err != nil {
		return 0, 0, fmt.Errorf("failed to unmarshal map attack response: %w", err)
	}

	if !resp.Success {
		return 0, 0, fmt.Errorf("MapServer attack failed")
	}

	zLog.Debug("MapServer attack successful",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("target_id", resp.TargetId),
		zap.Int64("damage", resp.Damage),
		zap.Int64("target_hp", resp.TargetHp))

	return resp.Damage, resp.TargetHp, nil
}

// SendMapEnterResponse 发送进入地图响应
func (ms *MapService) SendMapEnterResponse(conn interface{}, playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	resp := &protocol.MapEnterResponse{
		Success:  true,
		ObjectId: int64(playerID),
		MapId:    int64(mapID),
		X:        pos.X,
		Y:        pos.Y,
		Z:        pos.Z,
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		return err
	}

	// 这里需要根据实际的连接类型发送消息
	// 暂时返回nil
	_ = data
	return nil
}
