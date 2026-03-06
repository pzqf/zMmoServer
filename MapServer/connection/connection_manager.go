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
	config          *config.Config
	connections     map[string]*Connection
	connectionsMu   sync.RWMutex
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
		connections:     make(map[string]*Connection),
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
				cm.handleGameServerMessage(gameConn, buffer[:n])
			}
		}
	}
}

func (cm *ConnectionManager) handleGameServerMessage(gameConn *GameConnection, data []byte) {
	// 解析消息格式：长度前缀 + 消息ID + 数据
	if len(data) < 8 {
		zLog.Error("Invalid message format from GameServer", zap.String("game_server_id", gameConn.gameServerID), zap.Int("size", len(data)))
		return
	}

	// 解析长度
	length := binary.BigEndian.Uint32(data[:4])
	if length > 1024*1024 {
		zLog.Error("Message too long from GameServer", zap.String("game_server_id", gameConn.gameServerID), zap.Uint32("length", length))
		return
	}

	if len(data) < int(length) {
		zLog.Error("Insufficient data from GameServer", zap.String("game_server_id", gameConn.gameServerID), zap.Int("actual", len(data)), zap.Uint32("expected", length))
		return
	}

	// 解析消息ID
	msgID := binary.BigEndian.Uint32(data[4:8])

	// 解析消息内容
	var msg protocol.Message
	if err := proto.Unmarshal(data[8:length], &msg); err != nil {
		zLog.Error("Failed to unmarshal message", zap.String("game_server_id", gameConn.gameServerID), zap.Error(err))
		return
	}

	// 根据消息类型处理
	switch msgID {
	case protocol.MsgIdPlayerLogin:
		// 处理玩家登录
		cm.handlePlayerLogin(gameConn, msg)
	case protocol.MsgIdPlayerLogout:
		// 处理玩家登出
		cm.handlePlayerLogout(gameConn, msg)
	case protocol.MsgIdPlayerMove:
		// 处理玩家移动
		cm.handlePlayerMove(gameConn, msg)
	default:
		zLog.Info("Received unknown message from GameServer", zap.String("game_server_id", gameConn.gameServerID), zap.Uint32("msg_id", msgID))
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

func (cm *ConnectionManager) handlePlayerLogin(gameConn *GameConnection, msg protocol.Message) {
	zLog.Info("Handling player login", zap.String("game_server_id", gameConn.gameServerID), zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家登录逻辑
}

func (cm *ConnectionManager) handlePlayerLogout(gameConn *GameConnection, msg protocol.Message) {
	zLog.Info("Handling player logout", zap.String("game_server_id", gameConn.gameServerID), zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家登出逻辑
}

func (cm *ConnectionManager) handlePlayerMove(gameConn *GameConnection, msg protocol.Message) {
	zLog.Info("Handling player move", zap.String("game_server_id", gameConn.gameServerID), zap.Uint64("session_id", uint64(msg.SessionId)))
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
