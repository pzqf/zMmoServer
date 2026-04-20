package service

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/consistency"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/message"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	"github.com/pzqf/zMmoServer/GameServer/game/maps"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"github.com/pzqf/zMmoServer/GameServer/handler"
	msgHandler "github.com/pzqf/zMmoServer/GameServer/handler/message"
	"github.com/pzqf/zMmoServer/GameServer/net/protolayer"
	playerservice "github.com/pzqf/zMmoServer/GameServer/services"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// TCPService TCP服务
type TCPService struct {
	config            *config.Config
	connManager       *connection.ConnectionManager
	sessionManager    *session.SessionManager
	playerManager     *player.PlayerManager
	playerService     *playerservice.PlayerService
	playerHandler     *handler.PlayerHandler
	mapService        *maps.MapService
	loginService      *player.LoginService
	protocol          protolayer.Protocol
	tcpServer         *zNet.TcpServer
	mapServerListener net.Listener
	messageRouter     *msgHandler.Router
	isRunning         bool
	wg                sync.WaitGroup
	gatewayInbox      consistency.InboxStore
	dedupeHits        atomic.Uint64
	onDedupeHit       func(total uint64)
}

func NewTCPService(cfg *config.Config, connManager *connection.ConnectionManager, sessionManager *session.SessionManager, playerManager *player.PlayerManager, playerService *playerservice.PlayerService, playerHandler *handler.PlayerHandler, mapService *maps.MapService, loginService *player.LoginService, protocol protolayer.Protocol) *TCPService {
	return &TCPService{
		config:         cfg,
		connManager:    connManager,
		sessionManager: sessionManager,
		playerManager:  playerManager,
		playerService:  playerService,
		playerHandler:  playerHandler,
		mapService:     mapService,
		loginService:   loginService,
		protocol:       protocol,
		isRunning:      false,
		gatewayInbox:   consistency.NewMemoryInbox(),
		messageRouter:  msgHandler.NewRouter(),
	}
}

func (ts *TCPService) initMessageRouter() {
	if ts.messageRouter == nil {
		return
	}

	playerHandler := msgHandler.NewPlayerHandler(ts.sessionManager, ts.playerManager, ts.playerService, ts.loginService, int32(ts.config.Server.ServerID))
	mapHandler := msgHandler.NewMapHandler(ts.mapService, ts.playerManager, int32(ts.config.Server.ServerID))
	systemHandler := msgHandler.NewSystemHandler(ts.sessionManager)

	ts.messageRouter.RegisterHandler(int32(protocol.SystemMsgId_MSG_SYSTEM_ACCOUNT_LOGIN_NOTIFY), systemHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME), playerHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.PlayerMsgId_MSG_PLAYER_CREATE), playerHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME), playerHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.MapMsgId_MSG_MAP_ENTER), mapHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.MapMsgId_MSG_MAP_LEAVE), mapHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.MapMsgId_MSG_MAP_MOVE), mapHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.MapMsgId_MSG_MAP_ATTACK), mapHandler)
}

func (ts *TCPService) Name() string {
	return "TCPService"
}

// SetOnGatewayDedupeHit 设置Gateway去重命中回调（用于实时指标上报）
func (ts *TCPService) SetOnGatewayDedupeHit(cb func(total uint64)) {
	ts.onDedupeHit = cb
}

func (ts *TCPService) Start(ctx context.Context) error {
	if ts.isRunning {
		return nil
	}

	zLog.Info("Starting TCP service...", zap.String("addr", ts.config.Server.ListenAddr))

	ts.initMessageRouter()

	// 启动地图服务
	if ts.mapService != nil {
		if err := ts.mapService.Start(ctx); err != nil {
			return fmt.Errorf("failed to start map service: %v", err)
		}
	}

	// 创建zNet.TcpServer配置（用于Gateway连接）
	tcpConfig := &zNet.TcpConfig{
		ListenAddress:       ts.config.Server.ListenAddr,
		MaxClientCount:      ts.config.Server.MaxConnections,
		HeartbeatDuration:   ts.config.Server.HeartbeatInterval,
		ChanSize:            ts.config.Server.ChanSize,
		MaxPacketDataSize:   int32(ts.config.Server.MaxPacketDataSize),
		UseWorkerPool:       ts.config.Server.UseWorkerPool,
		WorkerPoolSize:      ts.config.Server.WorkerPoolSize,
		WorkerQueueSize:     ts.config.Server.WorkerQueueSize,
		DisableEncryption:   ts.config.Server.DisableEncryption,
		EnableKeyRotation:   ts.config.Server.EnableKeyRotation,
		KeyRotationInterval: time.Duration(ts.config.Server.KeyRotationInterval) * time.Second,
		MaxHistoryKeys:      ts.config.Server.MaxHistoryKeys,
		EnableSequenceCheck: ts.config.Server.EnableSequenceCheck,
		SequenceWindowSize:  ts.config.Server.SequenceWindowSize,
		TimestampTolerance:  ts.config.Server.TimestampTolerance,
	}

	// 创建zNet.TcpServer（用于Gateway连接）
	ts.tcpServer = zNet.NewTcpServer(tcpConfig, zNet.WithLogger(zLog.GetStandardLogger()))

	// 注册消息处理器
	ts.tcpServer.RegisterDispatcher(ts.handleConnectionMessage)

	// 启动服务
	err := ts.tcpServer.Start()
	if err != nil {
		return fmt.Errorf("failed to start TCP service: %v", err)
	}

	// 启动MapServer连接监听（使用单独的端口）
	mapServerPort := 20002 // MapServer连接端口
	mapServerAddr := fmt.Sprintf("0.0.0.0:%d", mapServerPort)
	zLog.Info("Starting MapServer connection listener...", zap.String("addr", mapServerAddr))

	listener, err := net.Listen("tcp", mapServerAddr)
	if err != nil {
		zLog.Error("Failed to start MapServer listener", zap.Error(err))
		// 继续启动，MapServer监听失败不影响主要功能
	} else {
		ts.mapServerListener = listener
		go ts.acceptMapServerConnections()
	}

	ts.isRunning = true

	zLog.Info("TCP service started successfully", zap.String("addr", ts.config.Server.ListenAddr))

	return nil
}

func (ts *TCPService) Stop(ctx context.Context) error {
	if !ts.isRunning {
		return nil
	}

	zLog.Info("Stopping TCP service...")

	if ts.tcpServer != nil {
		ts.tcpServer.Close()
	}

	// 停止MapServer监听
	if ts.mapServerListener != nil {
		ts.mapServerListener.Close()
	}

	// 停止地图服务
	if ts.mapService != nil {
		if err := ts.mapService.Stop(ctx); err != nil {
			zLog.Error("Failed to stop map service", zap.Error(err))
		}
	}

	ts.isRunning = false

	zLog.Info("TCP service stopped")

	return nil
}

// acceptMapServerConnections 接受MapServer的连接
func (ts *TCPService) acceptMapServerConnections() {
	if ts.mapServerListener == nil {
		return
	}

	zLog.Info("Accepting MapServer connections...")

	for {
		conn, err := ts.mapServerListener.Accept()
		if err != nil {
			zLog.Error("Failed to accept MapServer connection", zap.Error(err))
			break
		}

		zLog.Info("MapServer connected", zap.String("addr", conn.RemoteAddr().String()))

		// 处理MapServer连接
		go ts.handleMapServerConnection(conn)
	}
}

// handleMapServerConnection 处理MapServer连接
func (ts *TCPService) handleMapServerConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 4096)
	var pendingData []byte

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			zLog.Error("Failed to read from MapServer", zap.Error(err))
			break
		}

		if n > 0 {
			// 将新读取的数据添加到待处理数据中
			pendingData = append(pendingData, buffer[:n]...)
			// 处理待处理数据
			ts.processMapServerData(conn, &pendingData)
		}
	}

	zLog.Info("MapServer connection closed", zap.String("addr", conn.RemoteAddr().String()))
}

// processMapServerData 处理MapServer数据
func (ts *TCPService) processMapServerData(conn net.Conn, pendingData *[]byte) {
	for {
		// 检查是否有足够的数据来解析zNet消息头（16字节）
		if len(*pendingData) < 16 {
			// 数据不足，等待更多数据
			break
		}

		// 解析zNet格式的消息头
		// zNet消息格式：4字节ProtoId + 4字节Version + 4字节DataSize + 4字节IsCompressed
		_ = int(binary.BigEndian.Uint32((*pendingData)[:4]))  // protoId
		_ = int(binary.BigEndian.Uint32((*pendingData)[4:8])) // version
		dataLen := int(binary.BigEndian.Uint32((*pendingData)[8:12]))
		_ = int(binary.BigEndian.Uint32((*pendingData)[12:16])) // isCompressed

		if dataLen > 1024*1024 {
			zLog.Error("Message too long from MapServer", zap.Int("length", dataLen))
			// 丢弃此消息，继续处理下一个消息
			*pendingData = (*pendingData)[16:]
			continue
		}

		// 计算总消息长度：16字节头部 + 数据长度
		totalLen := 16 + dataLen
		if len(*pendingData) < totalLen {
			// 数据不足，等待更多数据
			break
		}

		// 提取完整的消息
		message := (*pendingData)[:totalLen]
		// 从待处理数据中移除已处理的消息
		*pendingData = (*pendingData)[totalLen:]

		// 处理消息
		ts.handleMapServerMessage(conn, message)
	}
}

// handleMapServerMessage 处理来自MapServer的消息
func (ts *TCPService) handleMapServerMessage(conn net.Conn, data []byte) {
	// 解析zNet格式的消息
	if len(data) < 16 {
		zLog.Error("Invalid zNet message format from MapServer", zap.Int("size", len(data)))
		return
	}

	// 解析zNet消息头
	protoId := int(binary.BigEndian.Uint32(data[:4]))
	_ = int(binary.BigEndian.Uint32(data[4:8])) // version
	dataLen := int(binary.BigEndian.Uint32(data[8:12]))
	_ = int(binary.BigEndian.Uint32(data[12:16])) // isCompressed

	// 检查数据长度
	if dataLen > 1024*1024 {
		zLog.Error("Message too long from MapServer", zap.Int("length", dataLen))
		return
	}

	// 检查总消息长度
	totalLen := 16 + dataLen
	if len(data) < totalLen {
		zLog.Error("Insufficient data from MapServer", zap.Int("actual", len(data)), zap.Int("expected", totalLen))
		return
	}

	// 提取数据部分并解析跨服信封（兼容未包裹旧格式）
	rawPayload := data[16:totalLen]
	meta, payload, wrapped, unwrapErr := crossserver.Unwrap(rawPayload)
	if unwrapErr != nil {
		zLog.Error("Invalid cross-server envelope from MapServer", zap.Error(unwrapErr))
		return
	}
	if wrapped {
		zLog.Debug("Received cross-server envelope from MapServer",
			zap.Uint64("trace_id", meta.TraceID),
			zap.Uint64("request_id", meta.RequestID),
			zap.Int("proto_id", protoId),
			zap.Int("server_id", ts.config.Server.ServerID))
	}

	// 解析跨服务器消息
	crossMsg := &protocol.CrossServerMessage{}
	if err := proto.Unmarshal(payload, crossMsg); err != nil {
		zLog.Error("Failed to unmarshal cross server message from MapServer", zap.Error(err))
		return
	}

	// 提取基础消息
	baseMsg := crossMsg.Message

	// 提取数据部分
	actualPayload := baseMsg.GetData()

	// 获取消息ID
	forwardProtoId := int32(baseMsg.GetMsgId())

	playerID := id.PlayerIdType(baseMsg.GetPlayerId())

	switch forwardProtoId {
	case int32(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE):
		// 处理地图进入响应
		var resp protocol.ClientMapEnterResponse
		if err := proto.Unmarshal(actualPayload, &resp); err != nil {
			zLog.Error("Failed to unmarshal map enter response", zap.Error(err))
			return
		}
		zLog.Info("Received map enter response from MapServer",
			zap.Uint64("player_id", baseMsg.GetPlayerId()),
			zap.Int32("map_id", resp.MapId),
			zap.Int32("result", resp.Result))

		// 转发到Gateway
		if err := ts.forwardToGateway(playerID, forwardProtoId, actualPayload); err != nil {
			zLog.Error("Failed to forward map enter response to Gateway", zap.Error(err))
		}

	case int32(protocol.MapMsgId_MSG_MAP_LEAVE_RESPONSE):
		// 处理地图离开响应
		var resp protocol.ClientMapLeaveResponse
		if err := proto.Unmarshal(actualPayload, &resp); err != nil {
			zLog.Error("Failed to unmarshal map leave response", zap.Error(err))
			return
		}
		zLog.Info("Received map leave response from MapServer",
			zap.Uint64("player_id", baseMsg.GetPlayerId()),
			zap.Int32("result", resp.Result))

		// 转发到Gateway
		if err := ts.forwardToGateway(playerID, forwardProtoId, actualPayload); err != nil {
			zLog.Error("Failed to forward map leave response to Gateway", zap.Error(err))
		}

	case int32(protocol.MapMsgId_MSG_MAP_MOVE_RESPONSE):
		// 处理地图移动响应
		var resp protocol.ClientMapMoveResponse
		if err := proto.Unmarshal(actualPayload, &resp); err != nil {
			zLog.Error("Failed to unmarshal map move response", zap.Error(err))
			return
		}
		zLog.Info("Received map move response from MapServer",
			zap.Uint64("player_id", baseMsg.GetPlayerId()),
			zap.Int32("result", resp.Result))

		// 转发到Gateway
		if err := ts.forwardToGateway(playerID, forwardProtoId, actualPayload); err != nil {
			zLog.Error("Failed to forward map move response to Gateway", zap.Error(err))
		}

	case int32(protocol.MapMsgId_MSG_MAP_ATTACK_RESPONSE):
		// 处理地图攻击响应
		var resp protocol.ClientMapAttackResponse
		if err := proto.Unmarshal(actualPayload, &resp); err != nil {
			zLog.Error("Failed to unmarshal map attack response", zap.Error(err))
			return
		}
		zLog.Info("Received map attack response from MapServer",
			zap.Uint64("player_id", baseMsg.GetPlayerId()),
			zap.Int64("target_id", resp.TargetId),
			zap.Int32("result", resp.Result),
			zap.Int64("damage", resp.Damage),
			zap.Int64("target_hp", resp.TargetHp))

		// 转发到Gateway
		if err := ts.forwardToGateway(playerID, forwardProtoId, actualPayload); err != nil {
			zLog.Error("Failed to forward map attack response to Gateway", zap.Error(err))
		}

	default:
		zLog.Info("Received unknown message from MapServer", zap.Int32("proto_id", forwardProtoId))
	}
}

// forwardToGateway 转发消息到Gateway
func (ts *TCPService) forwardToGateway(playerID id.PlayerIdType, protoId int32, data []byte) error {
	if ts.sessionManager == nil {
		return fmt.Errorf("session manager not set")
	}

	sess, exists := ts.sessionManager.GetSessionByPlayer(playerID)
	if !exists {
		return fmt.Errorf("session not found for player %d", playerID)
	}

	sessionID := sess.SessionID
	sessionIDUint, err := strconv.ParseUint(sessionID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid session ID format: %s", sessionID)
	}

	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: sessionIDUint,
		PlayerId:  uint64(playerID),
		ServerId:  uint32(ts.config.Server.ServerID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}

	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(ts.config.Server.ServerID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeGateway),
		Message:      baseMsg,
	}

	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal cross server message: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ts.config.Server.ServerID))
	wrappedData := crossserver.Wrap(meta, crossMsgData)

	if ts.connManager == nil {
		return fmt.Errorf("connection manager not set")
	}

	return ts.connManager.SendToGateway(wrappedData)
}

// handleConnectionMessage 处理来自客户端的消息
func (ts *TCPService) handleConnectionMessage(session zNet.Session, packet *zNet.NetPacket) error {
	// 处理消息
	sessionID := session.GetSid()
	zLog.Info("Received message", zap.Uint64("session_id", uint64(sessionID)), zap.Int32("proto_id", int32(packet.ProtoId)), zap.Int("data_size", len(packet.Data)))

	// 暂时统一处理，后续需要区分Gateway和MapServer连接
	// 注意：MapServer连接应该使用单独的处理逻辑，而不是通过zNet
	ts.handleGatewayMessage(session, int32(packet.ProtoId), packet.Data)

	return nil
}

// isGatewayConnection 检查是否是Gateway连接
func (ts *TCPService) isGatewayConnection(addr string) bool {
	// 检查是否在Kubernetes环境中
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		// 在Kubernetes环境中，由于网络策略隔离，所有内部连接都是可信的
		// 或者通过Gateway服务名称验证
		gatewayService := os.Getenv("GATEWAY_SERVICE_NAME")
		if gatewayService != "" {
			// 通过DNS解析服务名称获取IP列表
			addrs, err := net.LookupHost(gatewayService)
			if err == nil {
				// 提取客户端IP
				clientIP := addr
				if idx := strings.LastIndex(addr, ":"); idx != -1 {
					clientIP = addr[:idx]
				}
				// 检查客户端IP是否在服务IP列表中
				for _, serviceAddr := range addrs {
					if serviceAddr == clientIP {
						return true
					}
				}
			}
		}
		// 如果服务名称验证失败或未配置，在Kubernetes环境中默认允许所有连接
		// 实际生产环境应该结合网络策略使用
		return true
	}

	// 在非Kubernetes环境中，使用传统的IP地址验证
	clientIP := addr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		clientIP = addr[:idx]
	}
	return clientIP == ts.config.Gateway.GatewayAddr
}

// handleGatewayMessage 处理Gateway消息
func (ts *TCPService) handleGatewayMessage(session zNet.Session, protoId int32, data []byte) {
	meta, envelopePayload, wrapped, unwrapErr := crossserver.Unwrap(data)
	if unwrapErr != nil {
		zLog.Error("Invalid cross-server envelope from Gateway", zap.Error(unwrapErr))
		return
	}
	if envelopePayload != nil {
		data = envelopePayload
	}
	if wrapped {
		zLog.Debug("Received cross-server envelope from Gateway",
			zap.Uint64("trace_id", meta.TraceID),
			zap.Uint64("request_id", meta.RequestID),
			zap.Int32("proto_id", protoId),
			zap.Int("server_id", ts.config.Server.ServerID))
		if shouldDeduplicateGatewayProto(protoId) && !ts.gatewayInbox.TryAccept(meta.RequestID) {
			total := ts.dedupeHits.Add(1)
			if ts.onDedupeHit != nil {
				ts.onDedupeHit(total)
			}
			zLog.Warn("Duplicate gateway request ignored",
				zap.Uint64("request_id", meta.RequestID),
				zap.Int32("proto_id", protoId))
			return
		}
	}

	crossMsg := &protocol.CrossServerMessage{}
	if err := proto.Unmarshal(data, crossMsg); err != nil {
		zLog.Error("Failed to unmarshal cross server message", zap.Error(err), zap.Int("data_size", len(data)))
		return
	}

	payload := crossMsg.Message.GetData()
	actualProtoId := int32(crossMsg.Message.GetMsgId())
	clientSessionID := zNet.SessionIdType(crossMsg.Message.GetSessionId())

	zLog.Info("Received message from Gateway",
		zap.Int32("outer_proto_id", protoId),
		zap.Int32("inner_proto_id", actualProtoId),
		zap.Uint64("client_session_id", uint64(clientSessionID)),
		zap.Int("data_size", len(payload)))

	if tcpSess, ok := session.(*zNet.TcpServerSession); ok {
		tcpSess.SetObj(clientSessionID)
	}

	if ts.messageRouter != nil {
		if err := ts.messageRouter.Handle(session, actualProtoId, payload); err != nil {
			zLog.Error("Failed to handle message via router",
				zap.Int32("proto_id", actualProtoId),
				zap.Error(err))
		}
		return
	}

	zLog.Warn("No message router configured", zap.Int32("proto_id", protoId))
}

// GetGatewayDedupeHits 返回Gateway重复请求去重命中次数
func (ts *TCPService) GetGatewayDedupeHits() uint64 {
	return ts.dedupeHits.Load()
}

func shouldDeduplicateGatewayProto(protoId int32) bool {
	switch protoId {
	case int32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME),
		int32(protocol.PlayerMsgId_MSG_PLAYER_CREATE),
		int32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME),
		int32(protocol.MapMsgId_MSG_MAP_ENTER),
		int32(protocol.MapMsgId_MSG_MAP_MOVE),
		int32(protocol.MapMsgId_MSG_MAP_ATTACK):
		return true
	default:
		return false
	}
}

// handlePlayerLoginFromGateway 处理从Gateway来的玩家登录消息
func (ts *TCPService) handlePlayerLoginFromGateway(msg *protocol.ClientMessage) {
	zLog.Info("Handling player login from Gateway", zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家登录逻辑
}

// handlePlayerLogoutFromGateway 处理从Gateway来的玩家登出消息
func (ts *TCPService) handlePlayerLogoutFromGateway(msg *protocol.ClientMessage) {
	zLog.Info("Handling player logout from Gateway", zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家登出逻辑
}

// handlePlayerMoveFromGateway 处理从Gateway来的玩家移动消息
func (ts *TCPService) handlePlayerMoveFromGateway(msg *protocol.ClientMessage) {
	zLog.Info("Handling player move from Gateway", zap.Uint64("session_id", uint64(msg.SessionId)))
	// 实现玩家移动逻辑
}

func (ts *TCPService) handleMessage(sessionID string, conn net.Conn, data []byte) {
	session, exists := ts.sessionManager.GetSession(sessionID)
	if !exists {
		zLog.Warn("Session not found", zap.String("session_id", sessionID))
		return
	}

	ts.sessionManager.UpdateLastActive(sessionID)

	zLog.Info("Received message", zap.String("session_id", sessionID), zap.Int("data_len", len(data)))

	msgID, payload, err := ts.protocol.Decode(data)
	if err != nil {
		zLog.Error("Failed to decode message", zap.Error(err))
		return
	}

	// 检查玩家是否已登录
	if session.PlayerID != 0 {
		// 玩家已登录，通过PlayerManager路由消息
		msg := &player.PlayerMessage{
			Source: player.SourceGateway,
			Type:   player.MessageType(msgID),
			Data:   payload,
		}

		if err := ts.playerManager.RouteMessage(session.PlayerID, msg); err != nil {
			zLog.Error("Failed to route player message", zap.Error(err), zap.Int64("player_id", int64(session.PlayerID)))
			// 路由失败，直接处理
			ts.processMessage(session, conn, msgID, payload)
		}
	} else {
		// 玩家未登录，直接处理（登录、创建角色等）
		ts.processMessage(session, conn, msgID, payload)
	}
}

func (ts *TCPService) processMessage(sess *session.Session, conn net.Conn, msgID uint32, payload []byte) {
	zLog.Info("Processing message", zap.String("session_id", sess.SessionID), zap.Uint32("msg_id", msgID))

	var response proto.Message
	var err error

	switch msgID {
	case 1003:
		response, err = ts.handlePlayerLogin(sess, payload)
	case 1004:
		response, err = ts.handlePlayerCreate(sess, payload)
	case 1005:
		response, err = ts.handlePlayerSelect(sess, payload)
	case 1006:
		response, err = ts.handlePlayerLogout(sess)
	default:
		zLog.Warn("Unknown message ID", zap.String("session_id", sess.SessionID), zap.Uint32("msg_id", msgID))
		return
	}

	if err != nil {
		zLog.Error("Failed to process message", zap.Error(err))
		return
	}

	if response != nil {
		ts.sendResponse(conn, msgID, response)
	}
}

func (ts *TCPService) handlePlayerLogin(sess *session.Session, payload []byte) (*protocol.PlayerLoginResponse, error) {
	zLog.Info("Handling player login", zap.String("session_id", sess.SessionID))

	var req protocol.PlayerLoginRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		zLog.Error("Failed to unmarshal player login request", zap.Error(err))
		return nil, err
	}

	sess.AccountID = id.AccountIdType(req.PlayerId)
	sess.Status = session.SessionStatusLoggedIn

	response, err := ts.playerHandler.HandlePlayerLogin(sess.SessionID, id.AccountIdType(req.PlayerId))
	if err != nil {
		zLog.Error("Failed to handle player login", zap.Error(err))
		return nil, err
	}

	zLog.Info("Player login handled", zap.Int64("account_id", int64(req.PlayerId)))
	return response, nil
}

func (ts *TCPService) handlePlayerCreate(sess *session.Session, payload []byte) (*protocol.PlayerCreateResponse, error) {
	zLog.Info("Handling player create", zap.String("session_id", sess.SessionID))

	var req protocol.PlayerCreateRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		zLog.Error("Failed to unmarshal player create request", zap.Error(err))
		return nil, err
	}

	response, err := ts.playerHandler.HandlePlayerCreate(sess.SessionID, id.AccountIdType(sess.AccountID), req.Name, req.Sex, req.Age)
	if err != nil {
		zLog.Error("Failed to handle player create", zap.Error(err))
		return nil, err
	}

	zLog.Info("Player create handled", zap.String("player_name", req.Name))
	return response, nil
}

func (ts *TCPService) handlePlayerSelect(sess *session.Session, payload []byte) (*protocol.CommonResponse, error) {
	zLog.Info("Handling player select", zap.String("session_id", sess.SessionID))

	var req protocol.PlayerLoginRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		zLog.Error("Failed to unmarshal player select request", zap.Error(err))
		return nil, err
	}

	response, err := ts.playerHandler.HandlePlayerSelect(sess.SessionID, id.PlayerIdType(req.PlayerId))
	if err != nil {
		zLog.Error("Failed to handle player select", zap.Error(err))
		return nil, err
	}

	sess.Status = session.SessionStatusInGame

	zLog.Info("Player select handled", zap.Int64("player_id", int64(req.PlayerId)))
	return response, nil
}

func (ts *TCPService) handlePlayerLogout(sess *session.Session) (*protocol.CommonResponse, error) {
	zLog.Info("Handling player logout", zap.String("session_id", sess.SessionID))

	response, err := ts.playerHandler.HandlePlayerLogout(sess.SessionID)
	if err != nil {
		zLog.Error("Failed to handle player logout", zap.Error(err))
		return nil, err
	}

	return response, nil
}

// SendResponse 发送响应消息
func (ts *TCPService) SendResponse(conn net.Conn, msgID uint32, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		zLog.Error("Failed to marshal response", zap.Error(err))
		return err
	}

	// 使用message包编码消息
	packet, err := message.Encode(msgID, data)
	if err != nil {
		zLog.Error("Failed to encode response", zap.Error(err))
		return err
	}

	_, err = conn.Write(packet)
	if err != nil {
		zLog.Error("Failed to send response", zap.Error(err))
		return err
	}

	zLog.Info("Response sent", zap.Uint32("msg_id", msgID), zap.Int("data_len", len(packet)))
	return nil
}

func (ts *TCPService) sendResponse(conn net.Conn, msgID uint32, msg proto.Message) error {
	return ts.SendResponse(conn, msgID, msg)
}

// SendResponseToGateway 发送响应给Gateway，使用新的消息结构
func (ts *TCPService) SendResponseToGateway(session zNet.Session, protoId int32, sessionID uint64, playerID uint64, data []byte) error {
	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: sessionID,
		PlayerId:  playerID,
		ServerId:  uint32(ts.config.Server.ServerID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}

	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(ts.config.Server.ServerID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeGateway),
		Message:      baseMsg,
	}

	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		zLog.Error("Failed to marshal cross server message", zap.Error(err))
		return err
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ts.config.Server.ServerID))
	wrappedData := crossserver.Wrap(meta, crossMsgData)

	err = session.Send(zNet.ProtoIdType(protoId), wrappedData)
	if err != nil {
		zLog.Error("Failed to send response to Gateway", zap.Error(err))
		return err
	}

	return nil
}

func generateSessionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
