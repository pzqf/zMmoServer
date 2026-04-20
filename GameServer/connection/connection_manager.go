package connection

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
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
	mapConnections       *zMap.TypedMap[int, *MapConnection]
	mapSessionHandler    *MapSessionHandler
}

type MapConnection struct {
	session     zNet.Session
	mapIDs      []int
	isConnected bool
	closeChan   chan struct{}
	closeOnce   sync.Once
}

type MapSessionHandler struct {
	cm *ConnectionManager
}

func NewMapSessionHandler(cm *ConnectionManager) *MapSessionHandler {
	return &MapSessionHandler{cm: cm}
}

func (h *MapSessionHandler) Handle(session zNet.Session, packet *zNet.NetPacket) error {
	h.cm.handleMapServerPacket(session, packet)
	return nil
}

func NewConnectionManager(cfg *config.Config) *ConnectionManager {
	cm := &ConnectionManager{
		config:               cfg,
		connections:          zMap.NewTypedMap[string, *Connection](),
		isConnected:          atomic.Bool{},
		gatewayConnectedChan: make(chan struct{}),
		mapConnections:       zMap.NewTypedMap[int, *MapConnection](),
	}
	cm.mapSessionHandler = NewMapSessionHandler(cm)
	return cm
}

func (cm *ConnectionManager) ConnectToMapServer(mapServerAddr string, mapIDs []int) error {
	zLog.Info("Connecting to MapServer...", zap.String("addr", mapServerAddr), zap.Ints("map_ids", mapIDs))

	host, portStr, err := net.SplitHostPort(mapServerAddr)
	if err != nil {
		zLog.Error("Invalid MapServer address", zap.String("addr", mapServerAddr), zap.Error(err))
		return fmt.Errorf("invalid map server address: %w", err)
	}
	port := 30001
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		zLog.Error("Invalid MapServer port", zap.String("port", portStr), zap.Error(err))
		return fmt.Errorf("invalid map server port: %w", err)
	}

	tcpConfig := &zNet.TcpClientConfig{
		ServerAddr:        host,
		ServerPort:        port,
		ChanSize:          1024,
		HeartbeatDuration: 30,
		MaxPacketDataSize: 1024 * 1024,
		AutoReconnect:     true,
		ReconnectDelay:    5,
		DisableEncryption: true,
	}

	client := zNet.NewTcpClient(tcpConfig)
	client.RegisterDispatcher(cm.mapSessionHandler.Handle)

	if err := client.Connect(); err != nil {
		zLog.Error("Failed to connect to MapServer", zap.Error(err))
		return err
	}

	session := client.GetSession()
	if session == nil {
		zLog.Error("MapServer session is nil after connect")
		return fmt.Errorf("map server session is nil")
	}

	mapConn := &MapConnection{
		session:     session,
		mapIDs:      mapIDs,
		isConnected: true,
		closeChan:   make(chan struct{}),
	}

	for _, mapID := range mapIDs {
		cm.mapConnections.Store(mapID, mapConn)
	}

	zLog.Info("Connected to MapServer successfully", zap.String("addr", mapServerAddr), zap.Ints("map_ids", mapIDs))

	return nil
}

func (cm *ConnectionManager) DisconnectFromMapServer(mapIDs []int) {
	for _, mapID := range mapIDs {
		if mapConn, exists := cm.mapConnections.LoadAndDelete(mapID); exists {
			mapConn.closeOnce.Do(func() {
				close(mapConn.closeChan)
				if mapConn.session != nil {
					mapConn.session.Close()
				}
				mapConn.isConnected = false
			})
			zLog.Info("Disconnected from MapServer for map", zap.Int("map_id", mapID))
		}
	}
}

func (cm *ConnectionManager) SendToMap(mapID int, protoId int, data []byte) error {
	mapConn, exists := cm.mapConnections.Load(mapID)
	if !exists || !mapConn.isConnected {
		return fmt.Errorf("map server not connected for map %d", mapID)
	}

	return mapConn.session.Send(zNet.ProtoIdType(protoId), data)
}

func (cm *ConnectionManager) handleMapServerPacket(session zNet.Session, packet *zNet.NetPacket) {
	protoId := int32(packet.ProtoId)
	data := packet.Data

	meta, payload, wrapped, unwrapErr := crossserver.Unwrap(data)
	if unwrapErr != nil {
		zLog.Error("Invalid cross-server envelope from MapServer", zap.Error(unwrapErr))
		return
	}
	if wrapped {
		zLog.Debug("Received cross-server envelope from MapServer",
			zap.Uint64("trace_id", meta.TraceID),
			zap.Uint64("request_id", meta.RequestID),
			zap.Int32("proto_id", protoId))
	}

	var crossMsg crossserver.CrossServerMessage
	if err := json.Unmarshal(payload, &crossMsg); err != nil {
		zLog.Error("Failed to unmarshal cross server message from MapServer", zap.Error(err))
		return
	}

	baseMsg := crossMsg.Message

	zLog.Info("Received response from MapServer",
		zap.Int32("proto_id", protoId),
		zap.Uint32("msg_id", baseMsg.MsgID),
		zap.Uint64("player_id", baseMsg.PlayerID),
		zap.Uint32("from_server_id", crossMsg.FromServerID))

	switch baseMsg.MsgID {
	case uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_RESPONSE):
		cm.handleMapEnterResponse(&baseMsg)
	case uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_LEAVE_RESPONSE):
		cm.handleMapLeaveResponse(&baseMsg)
	case uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_MOVE_SYNC):
		cm.handleMapMoveResponse(&baseMsg)
	case uint32(protocol.InternalMsgId_MSG_INTERNAL_COMBAT_ACTION):
		cm.handleMapAttackResponse(&baseMsg)
	default:
		zLog.Info("Received unknown message from MapServer", zap.Uint32("msg_id", baseMsg.MsgID))
	}
}

func (cm *ConnectionManager) handleMapEnterResponse(baseMsg *crossserver.BaseMessage) {
	zLog.Info("Map enter response from MapServer",
		zap.Uint64("player_id", baseMsg.PlayerID),
		zap.Int("data_size", len(baseMsg.Data)))
}

func (cm *ConnectionManager) handleMapLeaveResponse(baseMsg *crossserver.BaseMessage) {
	zLog.Info("Map leave response from MapServer",
		zap.Uint64("player_id", baseMsg.PlayerID))
}

func (cm *ConnectionManager) handleMapMoveResponse(baseMsg *crossserver.BaseMessage) {
	zLog.Debug("Map move response from MapServer",
		zap.Uint64("player_id", baseMsg.PlayerID))
}

func (cm *ConnectionManager) handleMapAttackResponse(baseMsg *crossserver.BaseMessage) {
	zLog.Info("Map attack response from MapServer",
		zap.Uint64("player_id", baseMsg.PlayerID))
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

func (cm *ConnectionManager) SetGatewayConnection(conn net.Conn) {
	cm.gatewayMu.Lock()
	oldConn := cm.gatewayConn
	oldStatus := cm.isConnected.Load()
	cm.gatewayConn = conn
	cm.isConnected.Store(conn != nil)
	cm.gatewayMu.Unlock()

	if conn != nil {
		select {
		case cm.gatewayConnectedChan <- struct{}{}:
		default:
		}
		zLog.Info("Gateway connection set successfully")
	} else if oldStatus {
		if oldConn != nil {
			oldConn.Close()
		}
		zLog.Info("Gateway connection reset")
	}
}

func (cm *ConnectionManager) IsGatewayConnected() bool {
	return cm.isConnected.Load()
}

func (cm *ConnectionManager) GatewayConnectedChan() <-chan struct{} {
	return cm.gatewayConnectedChan
}

func (cm *ConnectionManager) GetGatewayConn() net.Conn {
	cm.gatewayMu.Lock()
	defer cm.gatewayMu.Unlock()

	return cm.gatewayConn
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
