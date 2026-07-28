package connection

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/pzqf/zCommon/common/id"
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

// MapResponseHandler 由 MapService 实现，用于把 MapServer 的响应回填到等待中的请求。
// 在此定义接口（而非 import maps 包）以避免 maps -> connection -> maps 循环依赖。
type MapResponseHandler interface {
	HandleMapAttackResponse(requestID uint64, playerID id.PlayerIdType, targetID id.ObjectIdType,
		damage int64, targetHP int64, success bool, errorMsg string)
	// HandleAOINotify 处理 MapServer 回程的 AOI 视野事件（Phase 2.3）：eventType 为
	// crossserver.MsgInternalAOIEnter/Leave/Move，路由到 watcher 玩家 Actor 后推送客户端。
	HandleAOINotify(watcherID int64, targetID int64, mapID int32, eventType uint32, x, y, z float32)
	// HandleItemGrant 处理 MapServer 拾取授权回推（crossserver.MsgInternalItemGrant）：把物品发进玩家背包。
	// requestID 用于幂等去重（对齐 attack 407 的 inbox.TryAccept），防重复投递重复发物品。
	HandleItemGrant(requestID uint64, playerID int64, itemID, count int32)
	// HandleAttrGrant 处理 MapServer 战斗/结算属性回写（crossserver.MsgInternalAttrGrant，realm ④）：把经验/金币等
	// 持久变更加到持久化 actor。requestID 幂等去重，防同一次结算重复投递被累加两次。泛化取代原 507 ExpGrant。
	HandleAttrGrant(requestID uint64, playerID int64, changes []*protocol.AttrChange)
	// HandleCrossInstanceEnsureResponse 处理跨服实例建/取的应答（621）：payload 为
	// CrossInstanceEnsureResponse 的 protobuf 字节，按 requestID 回填到等待中的请求。
	HandleCrossInstanceEnsureResponse(requestID uint64, payload []byte)
}

type ConnectionManager struct {
	config               *config.Config
	connections          *zMap.TypedMap[string, *Connection]
	gatewayConn          net.Conn
	gatewayMu            sync.Mutex
	isConnected          atomic.Bool
	gatewayConnectedChan chan struct{}
	mapConnections       *zMap.TypedMap[int, *MapConnection]
	// mapServerConns 按 MapServer 地址去重连接：每台 MapServer 只保持一条连接。
	// 服务发现每 30s 轮询一次，若不去重会每轮对同一 MapServer 重开 TCP 并泄漏旧连接。
	mapServerConns     *zMap.TypedMap[string, *MapConnection]
	mapSessionHandler  *MapSessionHandler
	mapResponseHandler MapResponseHandler
	// crossRealmConns 跨 realm 的 MapServer 连接，**按 serverID 索引**（跨服物理路由 ②）。
	// 不能并进 mapConnections（按 mapID 索引）：跨服实例的 mapID 由各 MapServer 的实例池派生，
	// 不同 MapServer 会派生出同一个数字，按 mapID 索引必然串到错的服务器上。
	crossRealmConns *zMap.TypedMap[uint32, *MapConnection]
	// crossConnMu 串行化"建跨域连接"：两个玩家同时进同一场跨服活动时，若不串行会各建一条 TCP。
	crossConnMu sync.Mutex
}

// SetMapResponseHandler 注入 MapServer 响应回填器（通常是 *maps.MapService）。
func (cm *ConnectionManager) SetMapResponseHandler(h MapResponseHandler) {
	cm.mapResponseHandler = h
}

type MapConnection struct {
	session     zNet.Session
	addr        string
	mapIDs      []int
	isConnected bool
	closeChan   chan struct{}
	closeOnce   sync.Once
	// connectedAt 建连时刻。跨域连接的空闲回收据此留出"已连上但还没绑定玩家"的窗口，
	// 免得把某个正在进图（连上→建实例→绑定，几百毫秒）的玩家的连接给收了。
	connectedAt time.Time
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
		mapServerConns:       zMap.NewTypedMap[string, *MapConnection](),
		crossRealmConns:      zMap.NewTypedMap[uint32, *MapConnection](),
	}
	cm.mapSessionHandler = NewMapSessionHandler(cm)
	return cm
}

func (cm *ConnectionManager) ConnectToMapServer(mapServerAddr string, mapIDs []int) error {
	// 去重：同一台 MapServer 已有连接则跳过（zNet 客户端 AutoReconnect 会自愈瞬断，
	// 无需服务发现轮询每 30s 重开）。避免连接泄漏与抖动。
	if existing, ok := cm.mapServerConns.Load(mapServerAddr); ok && existing.isConnected {
		zLog.Debug("MapServer already connected, skip", zap.String("addr", mapServerAddr))
		return nil
	}

	zLog.Info("Connecting to MapServer...", zap.String("addr", mapServerAddr), zap.Ints("map_ids", mapIDs))

	_, session, err := cm.dialMapServer(mapServerAddr)
	if err != nil {
		zLog.Error("Failed to connect to MapServer", zap.String("addr", mapServerAddr), zap.Error(err))
		return err
	}

	mapConn := &MapConnection{
		session:     session,
		addr:        mapServerAddr,
		mapIDs:      mapIDs,
		isConnected: true,
		closeChan:   make(chan struct{}),
	}

	cm.mapServerConns.Store(mapServerAddr, mapConn)
	for _, mapID := range mapIDs {
		cm.mapConnections.Store(mapID, mapConn)
	}

	zLog.Info("Connected to MapServer successfully", zap.String("addr", mapServerAddr), zap.Ints("map_ids", mapIDs))

	return nil
}

// ConnectToCrossRealmMapServer 连到**别的 realm** 的一台 MapServer（跨服物理路由 ②）。
// 幂等：已连上直接返回。按 serverID 索引，不进 mapConnections（跨服实例 mapID 会撞号）。
func (cm *ConnectionManager) ConnectToCrossRealmMapServer(serverID uint32, mapServerAddr string) error {
	cm.crossConnMu.Lock()
	defer cm.crossConnMu.Unlock()

	if existing, ok := cm.crossRealmConns.Load(serverID); ok && existing.isConnected {
		return nil
	}

	client, session, err := cm.dialMapServer(mapServerAddr)
	if err != nil {
		return err
	}
	_ = client

	cm.crossRealmConns.Store(serverID, &MapConnection{
		session:     session,
		addr:        mapServerAddr,
		isConnected: true,
		closeChan:   make(chan struct{}),
		connectedAt: time.Now(),
	})

	zLog.Info("Connected to cross-realm MapServer",
		zap.Uint32("map_server_id", serverID), zap.String("addr", mapServerAddr))
	return nil
}

// SendToCrossRealmMapServer 把消息发到指定 realm 的 MapServer（按 serverID 定点，不按 mapID）。
func (cm *ConnectionManager) SendToCrossRealmMapServer(serverID uint32, protoId int, data []byte) error {
	conn, exists := cm.crossRealmConns.Load(serverID)
	if !exists || !conn.isConnected {
		return fmt.Errorf("cross-realm map server not connected: %d", serverID)
	}
	return conn.session.Send(zNet.ProtoIdType(protoId), data)
}

// IsCrossRealmConnected 是否已连上某跨 realm MapServer。
func (cm *ConnectionManager) IsCrossRealmConnected(serverID uint32) bool {
	conn, exists := cm.crossRealmConns.Load(serverID)
	return exists && conn.isConnected
}

// IdleCrossRealmServerIDs 返回"建连已超过 minAge、且不在 inUse 里"的跨域连接 serverID。
// minAge 是为了留出"已连上但玩家还没绑定"的窗口（连上→建实例→绑定通常几百毫秒），
// 否则会把正在进图的玩家的连接收掉。
func (cm *ConnectionManager) IdleCrossRealmServerIDs(inUse map[uint32]bool, minAge time.Duration) []uint32 {
	now := time.Now()
	var idle []uint32
	cm.crossRealmConns.Range(func(serverID uint32, conn *MapConnection) bool {
		if inUse[serverID] {
			return true
		}
		if now.Sub(conn.connectedAt) < minAge {
			return true
		}
		idle = append(idle, serverID)
		return true
	})
	return idle
}

// DisconnectCrossRealmMapServer 断开某跨 realm 连接（活动结束/服务器下线/空闲回收）。
func (cm *ConnectionManager) DisconnectCrossRealmMapServer(serverID uint32) {
	cm.crossConnMu.Lock()
	defer cm.crossConnMu.Unlock()

	if conn, exists := cm.crossRealmConns.LoadAndDelete(serverID); exists {
		conn.closeOnce.Do(func() {
			close(conn.closeChan)
			if conn.session != nil {
				conn.session.Close()
			}
			conn.isConnected = false
		})
		zLog.Info("Disconnected from cross-realm MapServer", zap.Uint32("map_server_id", serverID))
	}
}

// dialMapServer 建一条到 MapServer 的 zNet 客户端连接（本 realm / 跨 realm 共用）。
func (cm *ConnectionManager) dialMapServer(mapServerAddr string) (*zNet.TcpClient, zNet.Session, error) {
	host, portStr, err := net.SplitHostPort(mapServerAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid map server address %q: %w", mapServerAddr, err)
	}
	port := 0
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil || port <= 0 {
		return nil, nil, fmt.Errorf("invalid map server port %q", portStr)
	}

	client := zNet.NewTcpClient(&zNet.TcpClientConfig{
		ServerAddr:        host,
		ServerPort:        port,
		ChanSize:          1024,
		HeartbeatDuration: 30,
		MaxPacketDataSize: 1024 * 1024,
		AutoReconnect:     true,
		ReconnectDelay:    5,
		DisableEncryption: true,
	})
	client.RegisterDispatcher(cm.mapSessionHandler.Handle)

	if err := client.Connect(); err != nil {
		return nil, nil, fmt.Errorf("connect map server %s: %w", mapServerAddr, err)
	}
	session := client.GetSession()
	if session == nil {
		return nil, nil, fmt.Errorf("map server session is nil after connect: %s", mapServerAddr)
	}
	return client, session, nil
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
				cm.mapServerConns.Delete(mapConn.addr)
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

// handleMapServerPacket 处理 MapServer 回程消息。统一经共用 codec 解包
// （Envelope + protobuf CrossServerMessage），按内层 BaseMessage.MsgId（响应 InternalMsgId）
// 分派，并把结果回填到等待中的请求。见 docs/协议契约.md。
func (cm *ConnectionManager) handleMapServerPacket(session zNet.Session, packet *zNet.NetPacket) {
	protoId := int32(packet.ProtoId)

	meta, _, baseMsg, err := crossserver.UnpackMessage(packet.Data)
	if err != nil {
		zLog.Error("Invalid cross-server message from MapServer",
			zap.Int32("proto_id", protoId), zap.Error(err))
		return
	}
	if baseMsg == nil {
		zLog.Error("MapServer message missing inner base message", zap.Int32("proto_id", protoId))
		return
	}

	zLog.Debug("Received response from MapServer",
		zap.Int32("proto_id", protoId),
		zap.Uint32("msg_id", baseMsg.MsgId),
		zap.Uint64("player_id", baseMsg.PlayerId),
		zap.Uint64("request_id", meta.RequestID))

	switch baseMsg.MsgId {
	case uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_RESPONSE):
		cm.handleMapEnterResponse(meta, baseMsg)
	case uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_LEAVE_RESPONSE):
		cm.handleMapLeaveResponse(meta, baseMsg)
	case uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_MOVE_SYNC):
		cm.handleMapMoveResponse(meta, baseMsg)
	case uint32(protocol.InternalMsgId_MSG_INTERNAL_COMBAT_ACTION):
		cm.handleMapAttackResponse(meta, baseMsg)
	case crossserver.MsgInternalAOIEnter, crossserver.MsgInternalAOILeave, crossserver.MsgInternalAOIMove,
		crossserver.MsgInternalAOIAttr, crossserver.MsgInternalAOIDeath, crossserver.MsgInternalAOIBuff:
		cm.handleAOINotify(baseMsg)
	case crossserver.MsgInternalItemGrant:
		cm.handleItemGrant(meta, baseMsg)
	case crossserver.MsgInternalAttrGrant:
		cm.handleAttrGrant(meta, baseMsg)
	case uint32(protocol.InternalMsgId_MSG_INTERNAL_CROSS_INSTANCE_ENSURE_RESPONSE):
		cm.handleCrossInstanceEnsureResponse(meta, baseMsg)
	default:
		zLog.Info("Received unknown message from MapServer", zap.Uint32("msg_id", baseMsg.MsgId))
	}
}

// handleAOINotify 解 MapServer 回程的 AOI 视野事件，回填到 MapService（Phase 2.3）。
func (cm *ConnectionManager) handleAOINotify(baseMsg *protocol.BaseMessage) {
	zLog.Debug("AOI notify received from MapServer",
		zap.Uint64("watcher", baseMsg.PlayerId), zap.Uint32("evt", baseMsg.MsgId))
	if cm.mapResponseHandler == nil {
		return
	}
	watcherID := int64(baseMsg.PlayerId)
	mapID := int32(baseMsg.MapId)
	var targetID int64
	var x, y, z float32
	switch baseMsg.MsgId {
	case crossserver.MsgInternalAOIEnter:
		var n protocol.EntityEnterViewNotify
		if err := proto.Unmarshal(baseMsg.Data, &n); err != nil {
			zLog.Error("Failed to unmarshal EntityEnterViewNotify", zap.Error(err))
			return
		}
		targetID = n.EntityId
		if n.Pos != nil {
			x, y, z = n.Pos.X, n.Pos.Y, n.Pos.Z
		}
	case crossserver.MsgInternalAOILeave:
		var n protocol.EntityLeaveViewNotify
		if err := proto.Unmarshal(baseMsg.Data, &n); err != nil {
			zLog.Error("Failed to unmarshal EntityLeaveViewNotify", zap.Error(err))
			return
		}
		targetID = n.EntityId
	case crossserver.MsgInternalAOIMove:
		var n protocol.EntityMoveNotify
		if err := proto.Unmarshal(baseMsg.Data, &n); err != nil {
			zLog.Error("Failed to unmarshal EntityMoveNotify", zap.Error(err))
			return
		}
		targetID = n.EntityId
		if n.NewPos != nil {
			x, y, z = n.NewPos.X, n.NewPos.Y, n.NewPos.Z
		}
	case crossserver.MsgInternalAOIAttr:
		// 血量经 x/y 透传：x=当前血量, y=最大血量（player_aoi_handler 还原为 EntityAttrNotify）。
		var n protocol.EntityAttrNotify
		if err := proto.Unmarshal(baseMsg.Data, &n); err != nil {
			zLog.Error("Failed to unmarshal EntityAttrNotify", zap.Error(err))
			return
		}
		targetID = n.EntityId
		x, y = float32(n.CurHp), float32(n.MaxHp)
	case crossserver.MsgInternalAOIDeath:
		// killer 经 x 透传。
		var n protocol.EntityDeathNotify
		if err := proto.Unmarshal(baseMsg.Data, &n); err != nil {
			zLog.Error("Failed to unmarshal EntityDeathNotify", zap.Error(err))
			return
		}
		targetID = n.EntityId
		x = float32(n.KillerId)
	case crossserver.MsgInternalAOIBuff:
		// buff 经 x/y/z 透传：x=buffID, y=added(1/0), z=remaining_ms。
		var n protocol.EntityBuffNotify
		if err := proto.Unmarshal(baseMsg.Data, &n); err != nil {
			zLog.Error("Failed to unmarshal EntityBuffNotify", zap.Error(err))
			return
		}
		targetID = n.EntityId
		x = float32(n.BuffId)
		if n.Added {
			y = 1
		}
		z = float32(n.RemainingMs)
	default:
		return
	}
	cm.mapResponseHandler.HandleAOINotify(watcherID, targetID, mapID, baseMsg.MsgId, x, y, z)
}

// handleItemGrant 解 MapServer 拾取授权回推（crossserver.MsgInternalItemGrant），发进玩家背包。
func (cm *ConnectionManager) handleItemGrant(meta crossserver.Meta, baseMsg *protocol.BaseMessage) {
	if cm.mapResponseHandler == nil {
		return
	}
	var n protocol.ItemGrantNotify
	if err := proto.Unmarshal(baseMsg.Data, &n); err != nil {
		zLog.Error("Failed to unmarshal ItemGrantNotify", zap.Error(err))
		return
	}
	cm.mapResponseHandler.HandleItemGrant(meta.RequestID, n.PlayerId, n.ItemId, n.Count)
}

// handleAttrGrant 解 MapServer 战斗/结算属性回写（crossserver.MsgInternalAttrGrant，realm ④），加到持久化 actor。
func (cm *ConnectionManager) handleAttrGrant(meta crossserver.Meta, baseMsg *protocol.BaseMessage) {
	if cm.mapResponseHandler == nil {
		return
	}
	var n protocol.AttrGrantNotify
	if err := proto.Unmarshal(baseMsg.Data, &n); err != nil {
		zLog.Error("Failed to unmarshal AttrGrantNotify", zap.Error(err))
		return
	}
	cm.mapResponseHandler.HandleAttrGrant(meta.RequestID, n.PlayerId, n.Changes)
}

// handleCrossInstanceEnsureResponse 跨服实例建/取应答（621）：原样把内层字节按 requestID 回填，
// 由 MapService 解析（解析放在等待方，连接层不掺业务）。
func (cm *ConnectionManager) handleCrossInstanceEnsureResponse(meta crossserver.Meta, baseMsg *protocol.BaseMessage) {
	if cm.mapResponseHandler == nil {
		return
	}
	cm.mapResponseHandler.HandleCrossInstanceEnsureResponse(meta.RequestID, baseMsg.Data)
}

func (cm *ConnectionManager) handleMapEnterResponse(meta crossserver.Meta, baseMsg *protocol.BaseMessage) {
	var resp protocol.MapEnterResponse
	if err := proto.Unmarshal(baseMsg.Data, &resp); err != nil {
		zLog.Error("Failed to unmarshal MapEnterResponse", zap.Error(err))
		return
	}
	zLog.Info("Map enter response from MapServer",
		zap.Uint64("request_id", meta.RequestID),
		zap.Uint64("player_id", baseMsg.PlayerId),
		zap.Bool("success", resp.Success),
		zap.String("error", resp.ErrorMsg))
}

func (cm *ConnectionManager) handleMapLeaveResponse(meta crossserver.Meta, baseMsg *protocol.BaseMessage) {
	var resp protocol.MapLeaveResponse
	if err := proto.Unmarshal(baseMsg.Data, &resp); err != nil {
		zLog.Error("Failed to unmarshal MapLeaveResponse", zap.Error(err))
		return
	}
	zLog.Info("Map leave response from MapServer",
		zap.Uint64("request_id", meta.RequestID),
		zap.Uint64("player_id", baseMsg.PlayerId),
		zap.Bool("success", resp.Success))
}

func (cm *ConnectionManager) handleMapMoveResponse(meta crossserver.Meta, baseMsg *protocol.BaseMessage) {
	var resp protocol.MapMoveResponse
	if err := proto.Unmarshal(baseMsg.Data, &resp); err != nil {
		zLog.Error("Failed to unmarshal MapMoveResponse", zap.Error(err))
		return
	}
	zLog.Debug("Map move response from MapServer",
		zap.Uint64("request_id", meta.RequestID),
		zap.Uint64("player_id", baseMsg.PlayerId))
}

func (cm *ConnectionManager) handleMapAttackResponse(meta crossserver.Meta, baseMsg *protocol.BaseMessage) {
	var resp protocol.MapAttackResponse
	if err := proto.Unmarshal(baseMsg.Data, &resp); err != nil {
		zLog.Error("Failed to unmarshal MapAttackResponse", zap.Error(err))
		return
	}
	zLog.Info("Map attack response from MapServer",
		zap.Uint64("request_id", meta.RequestID),
		zap.Uint64("player_id", baseMsg.PlayerId),
		zap.Int64("damage", resp.Damage),
		zap.Int64("target_hp", resp.TargetHp))

	// 回填等待中的攻击请求（sendMapAttackRequest 的 pending channel）
	if cm.mapResponseHandler != nil {
		cm.mapResponseHandler.HandleMapAttackResponse(meta.RequestID,
			id.PlayerIdType(resp.PlayerId), id.ObjectIdType(resp.TargetId),
			resp.Damage, resp.TargetHp, resp.Success, resp.ErrorMsg)
	}
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
