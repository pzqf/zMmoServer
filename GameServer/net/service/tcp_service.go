package service

import (
	"context"
	"fmt"
	"net"
	"os"
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
	messageRouter     *msgHandler.Router
	isRunning         bool
	// gatewaySession: 已接受的 Gateway 连接会话（1:1），供服务端主动推送（如 AOI 视野）
	// 经此会话 Send，与请求-响应同一 zNet 通道/分帧。收到 Gateway 消息时更新。
	gatewaySession atomic.Value
	wg             sync.WaitGroup
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
	itemHandler := msgHandler.NewItemHandler(ts.playerManager, int32(ts.config.Server.ServerID))
	skillHandler := msgHandler.NewSkillHandler(ts.playerManager, int32(ts.config.Server.ServerID))
	chatHandler := msgHandler.NewChatHandler(ts.playerManager, int32(ts.config.Server.ServerID))

	ts.messageRouter.RegisterHandler(int32(protocol.SystemMsgId_MSG_SYSTEM_ACCOUNT_LOGIN_NOTIFY), systemHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME), playerHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.PlayerMsgId_MSG_PLAYER_CREATE), playerHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME), playerHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.MapMsgId_MSG_MAP_ENTER), mapHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.MapMsgId_MSG_MAP_LEAVE), mapHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.MapMsgId_MSG_MAP_MOVE), mapHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.MapMsgId_MSG_MAP_ATTACK), mapHandler)
	// 物品 / 仓库（业务层建设 2026-07-25）
	ts.messageRouter.RegisterHandler(int32(protocol.ItemMsgId_MSG_ITEM_LIST), itemHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.ItemMsgId_MSG_ITEM_USE), itemHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.ItemMsgId_MSG_ITEM_MOVE), itemHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.ItemMsgId_MSG_ITEM_PICKUP), itemHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.ItemMsgId_MSG_WAREHOUSE_LIST), itemHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.ItemMsgId_MSG_WAREHOUSE_STORE), itemHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.ItemMsgId_MSG_WAREHOUSE_RETRIEVE), itemHandler)
	// 技能（业务层建设 2026-07-25）
	ts.messageRouter.RegisterHandler(int32(protocol.SkillMsgId_MSG_SKILL_LIST), skillHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.SkillMsgId_MSG_SKILL_LEARN), skillHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.SkillMsgId_MSG_SKILL_UPGRADE), skillHandler)
	ts.messageRouter.RegisterHandler(int32(protocol.SkillMsgId_MSG_SKILL_CAST), skillHandler)
	// 聊天（业务层建设 2026-07-25）：世界频道扇出
	ts.messageRouter.RegisterHandler(int32(protocol.ChatMsgId_MSG_CHAT_SEND), chatHandler)
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

// handleConnectionMessage 处理来自客户端的消息
func (ts *TCPService) handleConnectionMessage(session zNet.Session, packet *zNet.NetPacket) error {
	// 处理消息
	sessionID := session.GetSid()
	zLog.Info("Received message", zap.Uint64("session_id", uint64(sessionID)), zap.Int32("proto_id", int32(packet.ProtoId)), zap.Int("data_size", len(packet.Data)))

	// 记录 Gateway 会话，供服务端主动推送（AOI 视野等）复用同一 zNet 通道
	ts.gatewaySession.Store(session)

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
