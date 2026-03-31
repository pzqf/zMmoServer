package connection

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Connection struct {
	conn        net.Conn
	connID      string
	playerID    int64
	accountID   int64
	isConnected bool
	lastActive  time.Time
	sendChan    chan []byte
	closeChan   chan struct{}
	closeOnce   sync.Once
}

type ConnectionManager struct {
	config               *config.Config
	connections          *zMap.TypedMap[string, *Connection]
	gatewayConn          net.Conn
	gatewayMu            sync.Mutex
	isConnected          atomic.Bool
	gatewayConnectedChan chan struct{}
	mapConnections       *zMap.TypedMap[int, *MapConnection] // 地图ID -> MapServer连接
}

type MapConnection struct {
	conn        net.Conn
	mapIDs      []int
	isConnected bool
	sendChan    chan []byte
	closeChan   chan struct{}
	closeOnce   sync.Once
}

func NewConnectionManager(cfg *config.Config) *ConnectionManager {
	return &ConnectionManager{
		config:               cfg,
		connections:          zMap.NewTypedMap[string, *Connection](),
		isConnected:          atomic.Bool{},
		gatewayConnectedChan: make(chan struct{}),
		mapConnections:       zMap.NewTypedMap[int, *MapConnection](),
	}
}

func (cm *ConnectionManager) ConnectToMapServer(mapServerAddr string, mapIDs []int) error {
	zLog.Info("Connecting to MapServer...", zap.String("addr", mapServerAddr), zap.Ints("map_ids", mapIDs))

	conn, err := net.DialTimeout("tcp", mapServerAddr, 10*time.Second)
	if err != nil {
		zLog.Error("Failed to connect to MapServer", zap.Error(err))
		return err
	}

	mapConn := &MapConnection{
		conn:        conn,
		mapIDs:      mapIDs,
		isConnected: true,
		sendChan:    make(chan []byte, 100),
		closeChan:   make(chan struct{}),
	}

	// 注册地图ID到连接的映射
	for _, mapID := range mapIDs {
		cm.mapConnections.Store(mapID, mapConn)
	}

	zLog.Info("Connected to MapServer successfully", zap.String("addr", mapServerAddr), zap.Ints("map_ids", mapIDs))

	// 启动消息处理
	go cm.receiveFromMapServer(mapConn)
	go mapConn.sendLoop()

	return nil
}

func (cm *ConnectionManager) DisconnectFromMapServer(mapIDs []int) {
	for _, mapID := range mapIDs {
		if mapConn, exists := cm.mapConnections.LoadAndDelete(mapID); exists {
			mapConn.closeOnce.Do(func() {
				close(mapConn.closeChan)
				if mapConn.conn != nil {
					mapConn.conn.Close()
				}
				mapConn.isConnected = false
			})
			zLog.Info("Disconnected from MapServer for map", zap.Int("map_id", mapID))
		}
	}
}

func (cm *ConnectionManager) SendToMap(mapID int, data []byte) error {
	mapConn, exists := cm.mapConnections.Load(mapID)

	if !exists || !mapConn.isConnected {
		return fmt.Errorf("map server not connected for map %d", mapID)
	}

	select {
	case mapConn.sendChan <- data:
		return nil
	default:
		return fmt.Errorf("map server send channel full")
	}
}

func (cm *ConnectionManager) receiveFromMapServer(mapConn *MapConnection) {
	buffer := make([]byte, 4096)
	conn := mapConn.conn

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	for {
		select {
		case <-mapConn.closeChan:
			return
		default:
			n, err := conn.Read(buffer)
			if err != nil {
				zLog.Error("Failed to read from MapServer", zap.Error(err))
				// 标记连接为断开
				mapConn.isConnected = false
				return
			}

			// 重置读取超时
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			if n > 0 {
				cm.handleMapServerMessage(mapConn, buffer[:n])
			}
		}
	}
}

func (cm *ConnectionManager) handleMapServerMessage(mapConn *MapConnection, data []byte) {
	// 解析消息格式：长度前缀 + 消息ID + 数据
	if len(data) < 8 {
		zLog.Error("Invalid message format from MapServer", zap.Int("size", len(data)))
		return
	}

	// 解析长度
	length := binary.BigEndian.Uint32(data[:4])
	if length > 1024*1024 {
		zLog.Error("Message too long from MapServer", zap.Uint32("length", length))
		return
	}

	if len(data) < int(length) {
		zLog.Error("Insufficient data from MapServer", zap.Int("actual", len(data)), zap.Uint32("expected", length))
		return
	}

	// 解析消息ID
	msgID := binary.BigEndian.Uint32(data[4:8])

	// 解析消息内容
	var msg protocol.ClientMessage
	if err := proto.Unmarshal(data[8:length], &msg); err != nil {
		zLog.Error("Failed to unmarshal message", zap.Error(err))
		return
	}

	// 根据消息类型处理
	switch msgID {
	case uint32(protocol.MapMsgId_MSG_MAP_MOVE):
		// 处理玩家移动
		cm.handlePlayerMoveFromMap(&msg)
	default:
		zLog.Info("Received unknown message from MapServer", zap.Uint32("msg_id", msgID))
	}
}

func (cm *ConnectionManager) handlePlayerMoveFromMap(msg *protocol.ClientMessage) {
	zLog.Info("Handling player move from MapServer", zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家移动逻辑
}

func (mc *MapConnection) sendLoop() {
	for {
		select {
		case data := <-mc.sendChan:
			if mc.conn != nil {
				_, err := mc.conn.Write(data)
				if err != nil {
					zLog.Error("Failed to send to MapServer", zap.Error(err))
					mc.isConnected = false
				}
			}
		case <-mc.closeChan:
			return
		}
	}
}

func (cm *ConnectionManager) SendToGateway(data []byte) error {
	cm.gatewayMu.Lock()
	defer cm.gatewayMu.Unlock()

	if !cm.isConnected.Load() || cm.gatewayConn == nil {
		return fmt.Errorf("gateway not connected")
	}

	_, err := cm.gatewayConn.Write(data)
	if err != nil {
		zLog.Error("Failed to send to Gateway", zap.Error(err))
		cm.isConnected.Store(false)
		return err
	}

	return nil
}

func (cm *ConnectionManager) receiveFromGateway() {
	buffer := make([]byte, 4096)

	for {
		cm.gatewayMu.Lock()
		if !cm.isConnected.Load() || cm.gatewayConn == nil {
			cm.gatewayMu.Unlock()
			break
		}
		conn := cm.gatewayConn
		cm.gatewayMu.Unlock()

		n, err := conn.Read(buffer)
		if err != nil {
			zLog.Error("Failed to read from Gateway", zap.Error(err))
			cm.isConnected.Store(false)
			break
		}

		if n > 0 {
			cm.handleGatewayMessage(buffer[:n])
		}
	}
}

func (cm *ConnectionManager) handleGatewayMessage(data []byte) {
	// 解析消息格式：长度前缀 + 消息ID + 数据
	if len(data) < 8 {
		zLog.Error("Invalid message format from Gateway", zap.Int("size", len(data)))
		return
	}

	// 解析长度
	length := binary.BigEndian.Uint32(data[:4])
	if length > 1024*1024 {
		zLog.Error("Message too long from Gateway", zap.Uint32("length", length))
		return
	}

	if len(data) < int(length) {
		zLog.Error("Insufficient data from Gateway", zap.Int("actual", len(data)), zap.Uint32("expected", length))
		return
	}

	// 解析消息ID
	msgID := binary.BigEndian.Uint32(data[4:8])

	// 解析消息内容
	var msg protocol.ClientMessage
	if err := proto.Unmarshal(data[8:length], &msg); err != nil {
		zLog.Error("Failed to unmarshal message", zap.Error(err))
		return
	}

	// 根据消息类型处理
	switch msgID {
	case uint32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME):
		// 处理玩家登录
		cm.handlePlayerLogin(&msg)
	case uint32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME):
		// 处理玩家登出
		cm.handlePlayerLogout(&msg)
	case uint32(protocol.MapMsgId_MSG_MAP_MOVE):
		// 处理玩家移动
		cm.handlePlayerMove(&msg)
	default:
		zLog.Info("Received unknown message from Gateway", zap.Uint32("msg_id", msgID))
	}
}

func (cm *ConnectionManager) handlePlayerLogin(msg *protocol.ClientMessage) {
	zLog.Info("Handling player login", zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家登录逻辑
}

func (cm *ConnectionManager) handlePlayerLogout(msg *protocol.ClientMessage) {
	zLog.Info("Handling player logout", zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家登出逻辑
}

func (cm *ConnectionManager) handlePlayerMove(msg *protocol.ClientMessage) {
	zLog.Info("Handling player move", zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家移动逻辑
}

func (cm *ConnectionManager) AddConnection(connID string, conn net.Conn) *Connection {
	connection := &Connection{
		conn:        conn,
		connID:      connID,
		isConnected: true,
		lastActive:  time.Now(),
		sendChan:    make(chan []byte, 100),
		closeChan:   make(chan struct{}),
	}

	cm.connections.Store(connID, connection)
	go connection.sendLoop()

	return connection
}

func (cm *ConnectionManager) RemoveConnection(connID string) {
	if conn, exists := cm.connections.LoadAndDelete(connID); exists {
		conn.closeOnce.Do(func() {
			close(conn.closeChan)
			if conn.conn != nil {
				conn.conn.Close()
			}
			conn.isConnected = false
		})
	}
}

func (cm *ConnectionManager) GetConnection(connID string) (*Connection, bool) {
	return cm.connections.Load(connID)
}

func (cm *ConnectionManager) GetConnectionCount() int {
	return int(cm.connections.Len())
}

func (cm *ConnectionManager) Broadcast(data []byte) {
	cm.connections.Range(func(connID string, conn *Connection) bool {
		if conn.isConnected {
			select {
			case conn.sendChan <- data:
			default:
				zLog.Warn("Connection send channel full, dropping message", zap.String("conn_id", conn.connID))
			}
		}
		return true
	})
}

func (c *Connection) Send(data []byte) error {
	if !c.isConnected {
		return nil
	}

	select {
	case c.sendChan <- data:
		c.lastActive = time.Now()
		return nil
	default:
		return nil
	}
}

func (c *Connection) sendLoop() {
	var batch []byte
	var batchSize int
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case data := <-c.sendChan:
			batch = append(batch, data...)
			batchSize += len(data)
			// 当批次达到一定大小或时间到时发送
			if batchSize >= 4096 {
				c.sendBatch(batch)
				batch = nil
				batchSize = 0
			}
		case <-ticker.C:
			if batchSize > 0 {
				c.sendBatch(batch)
				batch = nil
				batchSize = 0
			}
		case <-c.closeChan:
			// 发送剩余数据
			if batchSize > 0 {
				c.sendBatch(batch)
			}
			return
		}
	}
}

func (c *Connection) sendBatch(batch []byte) {
	if c.conn != nil {
		_, err := c.conn.Write(batch)
		if err != nil {
			zLog.Error("Failed to send batch to connection", zap.String("conn_id", c.connID), zap.Error(err))
			c.isConnected = false
		}
	}
}

func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		close(c.closeChan)
		if c.conn != nil {
			c.conn.Close()
		}
		c.isConnected = false
	})
}

func (c *Connection) SetPlayerID(playerID int64) {
	c.playerID = playerID
}

func (c *Connection) GetPlayerID() int64 {
	return c.playerID
}

func (c *Connection) SetAccountID(accountID int64) {
	c.accountID = accountID
}

func (c *Connection) GetAccountID() int64 {
	return c.accountID
}

func (c *Connection) UpdateLastActive() {
	c.lastActive = time.Now()
}

func (c *Connection) GetLastActive() time.Time {
	return c.lastActive
}

func (c *Connection) IsConnected() bool {
	return c.isConnected
}

// SetGatewayConnection 设置Gateway连接
func (cm *ConnectionManager) SetGatewayConnection(conn net.Conn) {
	cm.gatewayMu.Lock()
	oldConn := cm.gatewayConn
	oldStatus := cm.isConnected.Load()
	cm.gatewayConn = conn
	cm.isConnected.Store(conn != nil)
	cm.gatewayMu.Unlock()

	if conn != nil {
		// 发送连接成功信号
		select {
		case cm.gatewayConnectedChan <- struct{}{}:
		default:
			// 通道已满，忽略
		}
		zLog.Info("Gateway connection set successfully")
	} else if oldStatus {
		// 连接断开
		if oldConn != nil {
			oldConn.Close()
		}
		zLog.Info("Gateway connection reset")
	}
}

// IsGatewayConnected 检查Gateway是否连接
func (cm *ConnectionManager) IsGatewayConnected() bool {
	return cm.isConnected.Load()
}

// GatewayConnectedChan 获取Gateway连接成功的通道
func (cm *ConnectionManager) GatewayConnectedChan() <-chan struct{} {
	return cm.gatewayConnectedChan
}

// GetGatewayConn 获取Gateway连接
func (cm *ConnectionManager) GetGatewayConn() net.Conn {
	cm.gatewayMu.Lock()
	defer cm.gatewayMu.Unlock()

	return cm.gatewayConn
}
