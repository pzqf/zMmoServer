package connection

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoShared/protocol"
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
	connections          map[string]*Connection
	connectionsMu        sync.RWMutex
	gatewayConn          net.Conn
	gatewayMu            sync.Mutex
	isConnected          bool
	gatewayConnectedChan chan struct{}
	mapConnections       map[int]*MapConnection // 地图ID -> MapServer连接
	mapConnMu            sync.RWMutex
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
		connections:          make(map[string]*Connection),
		isConnected:          false,
		gatewayConnectedChan: make(chan struct{}),
		mapConnections:       make(map[int]*MapConnection),
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
	cm.mapConnMu.Lock()
	for _, mapID := range mapIDs {
		cm.mapConnections[mapID] = mapConn
	}
	cm.mapConnMu.Unlock()

	zLog.Info("Connected to MapServer successfully", zap.String("addr", mapServerAddr), zap.Ints("map_ids", mapIDs))

	// 启动消息处理
	go cm.receiveFromMapServer(mapConn)
	go mapConn.sendLoop()

	return nil
}

func (cm *ConnectionManager) DisconnectFromMapServer(mapIDs []int) {
	cm.mapConnMu.Lock()
	defer cm.mapConnMu.Unlock()

	for _, mapID := range mapIDs {
		if mapConn, exists := cm.mapConnections[mapID]; exists {
			mapConn.closeOnce.Do(func() {
				close(mapConn.closeChan)
				if mapConn.conn != nil {
					mapConn.conn.Close()
				}
				mapConn.isConnected = false
			})
			delete(cm.mapConnections, mapID)
			zLog.Info("Disconnected from MapServer for map", zap.Int("map_id", mapID))
		}
	}
}

func (cm *ConnectionManager) SendToMap(mapID int, data []byte) error {
	cm.mapConnMu.RLock()
	mapConn, exists := cm.mapConnections[mapID]
	cm.mapConnMu.RUnlock()

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
		cm.handlePlayerMoveFromMap(msg)
	default:
		zLog.Info("Received unknown message from MapServer", zap.Uint32("msg_id", msgID))
	}
}

func (cm *ConnectionManager) handlePlayerMoveFromMap(msg protocol.ClientMessage) {
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

	if !cm.isConnected || cm.gatewayConn == nil {
		return fmt.Errorf("gateway not connected")
	}

	_, err := cm.gatewayConn.Write(data)
	if err != nil {
		zLog.Error("Failed to send to Gateway", zap.Error(err))
		cm.isConnected = false
		return err
	}

	return nil
}

func (cm *ConnectionManager) receiveFromGateway() {
	buffer := make([]byte, 4096)

	for {
		cm.gatewayMu.Lock()
		if !cm.isConnected || cm.gatewayConn == nil {
			cm.gatewayMu.Unlock()
			break
		}
		conn := cm.gatewayConn
		cm.gatewayMu.Unlock()

		n, err := conn.Read(buffer)
		if err != nil {
			zLog.Error("Failed to read from Gateway", zap.Error(err))
			cm.gatewayMu.Lock()
			cm.isConnected = false
			cm.gatewayMu.Unlock()
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
		cm.handlePlayerLogin(msg)
	case uint32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME):
		// 处理玩家登出
		cm.handlePlayerLogout(msg)
	case uint32(protocol.MapMsgId_MSG_MAP_MOVE):
		// 处理玩家移动
		cm.handlePlayerMove(msg)
	default:
		zLog.Info("Received unknown message from Gateway", zap.Uint32("msg_id", msgID))
	}
}

func (cm *ConnectionManager) handlePlayerLogin(msg protocol.ClientMessage) {
	zLog.Info("Handling player login", zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家登录逻辑
}

func (cm *ConnectionManager) handlePlayerLogout(msg protocol.ClientMessage) {
	zLog.Info("Handling player logout", zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家登出逻辑
}

func (cm *ConnectionManager) handlePlayerMove(msg protocol.ClientMessage) {
	zLog.Info("Handling player move", zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家移动逻辑
}

func (cm *ConnectionManager) AddConnection(connID string, conn net.Conn) *Connection {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()

	connection := &Connection{
		conn:        conn,
		connID:      connID,
		isConnected: true,
		lastActive:  time.Now(),
		sendChan:    make(chan []byte, 100),
		closeChan:   make(chan struct{}),
	}

	cm.connections[connID] = connection
	go connection.sendLoop()

	return connection
}

func (cm *ConnectionManager) RemoveConnection(connID string) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()

	if conn, exists := cm.connections[connID]; exists {
		conn.closeOnce.Do(func() {
			close(conn.closeChan)
			if conn.conn != nil {
				conn.conn.Close()
			}
			conn.isConnected = false
		})
		delete(cm.connections, connID)
	}
}

func (cm *ConnectionManager) GetConnection(connID string) (*Connection, bool) {
	cm.connectionsMu.RLock()
	defer cm.connectionsMu.RUnlock()

	conn, exists := cm.connections[connID]
	return conn, exists
}

func (cm *ConnectionManager) GetConnectionCount() int {
	cm.connectionsMu.RLock()
	defer cm.connectionsMu.RUnlock()

	return len(cm.connections)
}

func (cm *ConnectionManager) Broadcast(data []byte) {
	cm.connectionsMu.RLock()
	defer cm.connectionsMu.RUnlock()

	for _, conn := range cm.connections {
		if conn.isConnected {
			select {
			case conn.sendChan <- data:
			default:
				zLog.Warn("Connection send channel full, dropping message", zap.String("conn_id", conn.connID))
			}
		}
	}
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
	for {
		select {
		case data := <-c.sendChan:
			if c.conn != nil {
				_, err := c.conn.Write(data)
				if err != nil {
					zLog.Error("Failed to send to connection", zap.String("conn_id", c.connID), zap.Error(err))
					c.isConnected = false
				}
			}
		case <-c.closeChan:
			return
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
	oldStatus := cm.isConnected
	cm.gatewayConn = conn
	cm.isConnected = (conn != nil)
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
	cm.gatewayMu.Lock()
	defer cm.gatewayMu.Unlock()

	return cm.isConnected
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
