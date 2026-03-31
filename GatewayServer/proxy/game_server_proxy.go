package proxy

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/message"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/connection"
	gwService "github.com/pzqf/zMmoServer/GatewayServer/service"
	"go.uber.org/zap"
)

type GameServerProxy interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SendToGameServer(sessionID zNet.SessionIdType, protoId int32, data []byte) error
}

type gameServerProxy struct {
	config      *config.Config
	tcpService  *gwService.TCPService
	connManager *connection.ConnectionManager
	tcpClient   *zNet.TcpClient
	discovery   *discovery.ServiceDiscovery
	gameServers []*discovery.ServerInfo
}

func NewGameServerProxy(cfg *config.Config, tcpService *gwService.TCPService, connManager *connection.ConnectionManager) GameServerProxy {
	return &gameServerProxy{
		config:      cfg,
		tcpService:  tcpService,
		connManager: connManager,
	}
}

func (gsp *gameServerProxy) Start(ctx context.Context) error {
	zLog.Info("Starting GameServer proxy...")

	// 初始化服务发现
	discovery, err := discovery.NewServiceDiscovery([]string{gsp.config.Etcd.Endpoints})
	if err != nil {
		return fmt.Errorf("failed to create service discovery: %w", err)
	}
	gsp.discovery = discovery

	go gsp.discoverAndConnectGameServers(ctx)

	return nil
}

func (gsp *gameServerProxy) Stop(ctx context.Context) error {
	zLog.Info("Stopping GameServer proxy...")

	if gsp.tcpClient != nil {
		gsp.tcpClient.Close()
	}

	return nil
}

func (gsp *gameServerProxy) discoverAndConnectGameServers(ctx context.Context) {
	// 启动服务状态监控
	go gsp.watchGameServerStatus(ctx)

	// 定期发现GameServer
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Gateway group 从 6 位 ServerID 解析得到
			gwServerID, err := id.ParseServerIDInt(int32(gsp.config.Server.ServerID))
			if err != nil {
				zLog.Error("Invalid gateway ServerID, skip discover", zap.Int("server_id", gsp.config.Server.ServerID), zap.Error(err))
				continue
			}
			groupID := id.GroupIDStringFromServerID(gwServerID)
			instances, err := gsp.discovery.Discover("game", groupID)
			if err != nil {
				zLog.Error("Failed to discover GameServer", zap.Error(err))
				continue
			}

			if len(instances) == 0 {
				zLog.Warn("No GameServer instances found")
				continue
			}

			// 选择GameServer（基于健康状态和负载）
			selectedInstance := gsp.selectGameServer(instances)
			if selectedInstance == nil {
				zLog.Warn("No suitable GameServer instance found")
				continue
			}

			// 检查是否已经连接到该GameServer
			if gsp.tcpClient != nil && gsp.tcpClient.IsConnected() {
				// 已经连接，不需要重新连接
				continue
			}

			// 连接到选中的GameServer
			gsp.connectToGameServer(selectedInstance)
		}
	}
}

// watchGameServerStatus 监控GameServer状态变化
func (gsp *gameServerProxy) watchGameServerStatus(ctx context.Context) {
	// Gateway group 从 6 位 ServerID 解析得到
	gwServerID, err := id.ParseServerIDInt(int32(gsp.config.Server.ServerID))
	if err != nil {
		zLog.Error("Invalid gateway ServerID, skip watch", zap.Int("server_id", gsp.config.Server.ServerID), zap.Error(err))
		return
	}
	groupID := id.GroupIDStringFromServerID(gwServerID)

	// 启动服务状态监控
	eventChan, err := gsp.discovery.Watch("game", groupID)
	if err != nil {
		zLog.Error("Failed to start watching GameServer status", zap.Error(err))
		return
	}

	zLog.Info("Started watching GameServer status changes")

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventChan:
			if !ok {
				zLog.Warn("GameServer status watch channel closed")
				return
			}

			// 处理服务状态变化
			gsp.handleGameServerStatusChange(event)
		}
	}
}

// handleGameServerStatusChange 处理GameServer状态变化
func (gsp *gameServerProxy) handleGameServerStatusChange(event *discovery.ServerEvent) {
	zLog.Info("Received GameServer status change event",
		zap.String("server_id", event.ServerID),
		zap.String("event_type", event.EventType),
		zap.String("status", string(event.Status)))

	// 如果当前连接的GameServer状态变为不健康，重新选择服务器
	if gsp.tcpClient != nil && gsp.tcpClient.IsConnected() {
		// 这里可以根据实际情况判断是否需要重新连接
		// 例如，如果当前连接的服务器状态变为maintenance或stopped，需要重新选择
		if event.Status == discovery.ServerStatusMaintenance || event.Status == discovery.ServerStatusStopped {
			zLog.Warn("Current GameServer status changed to unhealthy, will reconnect",
				zap.String("server_id", event.ServerID),
				zap.String("status", string(event.Status)))
			// 关闭当前连接，触发重新连接
			gsp.tcpClient.Close()
		}
	}
}

func (gsp *gameServerProxy) selectGameServer(instances []*discovery.ServerInfo) *discovery.ServerInfo {
	if len(instances) == 0 {
		return nil
	}

	// 首先尝试根据配置的GameServerID选择
	targetGameServerID, err := id.ParseServerIDInt(int32(gsp.config.GameServer.GameServerID))
	if err != nil {
		zLog.Error("Invalid target GameServerID", zap.Int("game_server_id", gsp.config.GameServer.GameServerID), zap.Error(err))
		// 如果配置的ID无效，尝试选择健康状态的服务器
	} else {
		// 尝试找到配置的目标服务器
		targetInt := int32(targetGameServerID)
		for _, inst := range instances {
			// 使用ID字段作为服务器ID
			if inst.ID == fmt.Sprintf("%d", gsp.config.GameServer.GameServerID) {
				// 检查服务器状态是否健康
				if inst.Status == discovery.ServerStatusHealthy || inst.Status == discovery.ServerStatusReady {
					return inst
				}
			}
		}
		zLog.Warn("No matching healthy GameServer instance for target", zap.Int32("target_game_server_id", targetInt))
	}

	// 如果没有找到配置的服务器或配置的服务器不健康，选择最健康的服务器
	var bestInstance *discovery.ServerInfo
	var bestScore float64

	for _, inst := range instances {
		// 跳过不健康的服务器
		if inst.Status != discovery.ServerStatusHealthy && inst.Status != discovery.ServerStatusReady {
			continue
		}

		// 计算服务器评分：负载越低，玩家数越少，评分越高
		score := 100.0
		if inst.Load > 0 {
			score -= inst.Load * 10
		}
		score -= float64(inst.Players) * 0.1

		// 选择评分最高的服务器
		if bestInstance == nil || score > bestScore {
			bestInstance = inst
			bestScore = score
		}
	}

	if bestInstance != nil {
		zLog.Info("Selected GameServer based on health and load",
			zap.String("server_id", bestInstance.ID),
			zap.String("address", bestInstance.Address),
			zap.Float64("load", bestInstance.Load),
			zap.Int("players", bestInstance.Players),
			zap.String("status", string(bestInstance.Status)))
		return bestInstance
	}

	// 如果没有健康的服务器，返回第一个服务器
	if len(instances) > 0 {
		zLog.Warn("No healthy GameServer instances found, returning first available",
			zap.String("server_id", instances[0].ID),
			zap.String("status", string(instances[0].Status)))
		return instances[0]
	}

	return nil
}

func (gsp *gameServerProxy) connectToGameServer(instance *discovery.ServerInfo) {
	// 解析GameServer地址
	tcpAddr, err := net.ResolveTCPAddr("tcp", instance.Address)
	if err != nil {
		zLog.Error("Failed to resolve GameServer address", zap.Error(err), zap.String("addr", instance.Address))
		return
	}

	// 创建TcpClient配置
	clientConfig := &zNet.TcpClientConfig{
		ServerAddr:        tcpAddr.IP.String(),
		ServerPort:        tcpAddr.Port,
		HeartbeatDuration: 30,
		MaxPacketDataSize: 1024 * 1024,
		AutoReconnect:     true,
		ReconnectDelay:    5,
		DisableEncryption: true, // 服务器之间禁用加密，提高响应速度
	}

	// 创建TcpClient
	gsp.tcpClient = zNet.NewTcpClient(clientConfig,
		zNet.WithClientLogger(zLog.GetStandardLogger()),
		zNet.WithClientStateCallback(func(state zNet.ClientState) {
			switch state {
			case zNet.ClientStateConnected:
				zLog.Info("Connected to GameServer", zap.String("addr", instance.Address), zap.String("server_id", instance.ID))
			case zNet.ClientStateDisconnected:
				zLog.Warn("Disconnected from GameServer")
			case zNet.ClientStateReconnecting:
				zLog.Info("Reconnecting to GameServer...")
			}
		}),
	)

	// 注册消息处理器
	gsp.tcpClient.RegisterDispatcher(gsp.handleGameServerMessage)

	// 连接到GameServer
	err = gsp.tcpClient.Connect()
	if err != nil {
		zLog.Error("Failed to connect to GameServer", zap.Error(err), zap.String("addr", instance.Address))
		return
	}
}

func (gsp *gameServerProxy) handleGameServerMessage(session zNet.Session, packet *zNet.NetPacket) error {
	// 处理GameServer消息
	gsp.processGameServerMessage(packet.Data)
	return nil
}

func (gsp *gameServerProxy) processGameServerMessage(data []byte) {
	meta, payload, wrapped, unwrapErr := crossserver.Unwrap(data)
	if unwrapErr != nil {
		zLog.Error("Invalid cross-server envelope from GameServer", zap.Error(unwrapErr))
		return
	}
	if payload != nil {
		data = payload
	}
	if wrapped {
		zLog.Debug("Received cross-server envelope from GameServer",
			zap.Uint64("trace_id", meta.TraceID),
			zap.Uint64("request_id", meta.RequestID),
			zap.Int32("proto_id", 0),
			zap.Int("server_id", gsp.config.Server.ServerID))
	}

	// 使用消息解码
	msg, err := message.Decode(data)
	if err != nil {
		zLog.Error("Failed to decode message from GameServer", zap.Error(err))
		return
	}

	// 检查消息长度
	if gsp.tcpClient != nil && uint32(gsp.tcpClient.GetMaxPacketDataSize()) < msg.Header.Length {
		zLog.Error("Message too long from GameServer",
			zap.Uint32("length", msg.Header.Length),
			zap.Int32("max", gsp.tcpClient.GetMaxPacketDataSize()))
		return
	}

	// 处理内部消息
	if msg.Header.MsgID == uint32(protocol.InternalMsgId_MSG_INTERNAL_SERVICE_HEARTBEAT) {
		// 处理心跳消息
		var heartbeatReq protocol.ServiceHeartbeatRequest
		if err := proto.Unmarshal(msg.Data, &heartbeatReq); err != nil {
			zLog.Error("Failed to unmarshal heartbeat request", zap.Error(err))
			return
		}
		zLog.Debug("Received heartbeat from GameServer",
			zap.Int32("server_id", heartbeatReq.ServerId),
			zap.Int32("online_count", heartbeatReq.OnlineCount))
		return
	}

	// 解析消息内容，提取sessionID
	var clientMsg protocol.ClientMessage
	if err := proto.Unmarshal(msg.Data, &clientMsg); err != nil {
		zLog.Error("Failed to unmarshal client message", zap.Error(err))
		return
	}

	// 查找对应的客户端连接
	sessionID := zNet.SessionIdType(clientMsg.SessionId)

	// 转发消息给客户端
	err = gsp.tcpService.SendToClient(sessionID, data)
	if err != nil {
		zLog.Error("Failed to send message to client",
			zap.Error(err),
			zap.Uint64("session_id", uint64(sessionID)))
		return
	}

	zLog.Debug("Message forwarded to client",
		zap.Uint32("msg_id", msg.Header.MsgID),
		zap.Uint64("session_id", uint64(sessionID)))
}

func (gsp *gameServerProxy) SendToGameServer(sessionID zNet.SessionIdType, protoId int32, data []byte) error {
	if gsp.tcpClient == nil || !gsp.tcpClient.IsConnected() {
		return fmt.Errorf("not connected to GameServer")
	}

	// 编码消息
	encodedMsg, err := message.Encode(uint32(protoId), data)
	if err != nil {
		zLog.Error("Failed to encode message", zap.Error(err))
		return err
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeUnknown, int32(gsp.config.Server.ServerID))
	encodedMsg = crossserver.Wrap(meta, encodedMsg)
	zLog.Debug("Sending cross-server envelope to GameServer",
		zap.Uint64("trace_id", meta.TraceID),
		zap.Uint64("request_id", meta.RequestID),
		zap.Int32("proto_id", protoId),
		zap.Int("server_id", gsp.config.Server.ServerID))

	// 使用Send方法发送消息
	err = gsp.tcpClient.Send(protoId, encodedMsg)
	if err != nil {
		zLog.Error("Failed to send message to GameServer", zap.Error(err))
		return err
	}

	zLog.Info("Message sent to GameServer", zap.Uint64("session_id", uint64(sessionID)), zap.Int32("proto_id", protoId), zap.Int("data_len", len(data)))
	return nil
}
