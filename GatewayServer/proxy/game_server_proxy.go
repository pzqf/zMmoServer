package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"google.golang.org/protobuf/proto"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/message"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/common"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"go.uber.org/zap"
)

type GameServerProxy interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SendToGameServer(sessionID zNet.SessionIdType, protoId int32, data []byte) error
}

type gameServerProxy struct {
	config        *config.Config
	clientService common.ClientServiceInterface
	tcpClient     *zNet.TcpClient
	discovery     *discovery.ServiceDiscovery
	gameServers   []*discovery.ServerInfo
}

func NewGameServerProxy(cfg *config.Config, clientService common.ClientServiceInterface) GameServerProxy {
	return &gameServerProxy{
		config:        cfg,
		clientService: clientService,
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

	// 只关注与当前Gateway ServerID相同的GameServer
	targetServerID := fmt.Sprintf("%d", gsp.config.Server.ServerID)
	if event.ServerID != targetServerID {
		return
	}

	// 如果当前连接的GameServer状态变为不健康，关闭连接
	if gsp.tcpClient != nil && gsp.tcpClient.IsConnected() {
		if event.Status == "maintenance" || event.Status == "stopped" {
			zLog.Warn("Current GameServer status changed to unhealthy, closing connection",
				zap.String("server_id", event.ServerID),
				zap.String("status", string(event.Status)))
			// 关闭当前连接，等待下一次发现周期重新连接
			gsp.tcpClient.Close()
		}
	}
}

func (gsp *gameServerProxy) selectGameServer(instances []*discovery.ServerInfo) *discovery.ServerInfo {
	if len(instances) == 0 {
		return nil
	}

	// 只选择与Gateway ServerID相同的GameServer
	targetServerID := fmt.Sprintf("%d", gsp.config.Server.ServerID)
	for _, inst := range instances {
		// 使用ID字段作为服务器ID
		if inst.ID == targetServerID {
			// 检查服务器状态是否健康
			if inst.Status == "healthy" || inst.Status == "ready" {
				zLog.Info("Selected GameServer with matching ServerID",
					zap.String("server_id", inst.ID),
					zap.String("address", inst.Address),
					zap.String("status", string(inst.Status)))
				return inst
			}
		}
	}

	zLog.Warn("No matching GameServer instance found for ServerID", zap.Int("server_id", gsp.config.Server.ServerID))
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

	// 解析跨服务器消息
	var crossMsg crossserver.CrossServerMessage
	if err := json.Unmarshal(msg.Data, &crossMsg); err != nil {
		zLog.Error("Failed to unmarshal cross server message", zap.Error(err))
		return
	}

	// 提取基础消息
	baseMsg := crossMsg.Message

	// 查找对应的客户端连接
	sessionID := zNet.SessionIdType(baseMsg.SessionID)

	// 转发消息给客户端，使用baseMsg.Data作为实际消息数据
	err = gsp.clientService.SendToClient(sessionID, baseMsg.Data)
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

	// 创建基础消息
	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: uint64(sessionID),
		ServerId:  uint32(gsp.config.Server.ServerID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}

	// 创建跨服务器消息
	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(gsp.config.Server.ServerID),
		FromService:  uint32(crossserver.ServiceTypeGateway),
		ToService:    uint32(crossserver.ServiceTypeGame),
		Message:      baseMsg,
	}

	// 使用Protocol Buffers序列化消息
	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		zLog.Error("Failed to marshal cross server message", zap.Error(err))
		return err
	}

	// 编码消息
	encodedMsg, err := message.Encode(uint32(protoId), crossMsgData)
	if err != nil {
		zLog.Error("Failed to encode message", zap.Error(err))
		return err
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGateway, int32(gsp.config.Server.ServerID))
	encodedMsg = crossserver.Wrap(meta, encodedMsg)
	zLog.Debug("Sending cross-server envelope to GameServer",
		zap.Uint64("trace_id", meta.TraceID),
		zap.Uint64("request_id", meta.RequestID),
		zap.Int32("proto_id", protoId),
		zap.Int("server_id", gsp.config.Server.ServerID))

	// 使用Send方法发送消息
	err = gsp.tcpClient.Send(zNet.ProtoIdType(protoId), encodedMsg)
	if err != nil {
		zLog.Error("Failed to send message to GameServer", zap.Error(err))
		return err
	}

	zLog.Info("Message sent to GameServer", zap.Uint64("session_id", uint64(sessionID)), zap.Int32("proto_id", protoId), zap.Int("data_len", len(data)))
	return nil
}
