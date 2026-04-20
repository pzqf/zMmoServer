package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/common"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type GameServerProxy interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SendToGameServer(sessionID zNet.SessionIdType, protoId int32, data []byte) error
	IsConnected() bool
}

type pendingMessage struct {
	sessionID zNet.SessionIdType
	protoId   int32
	data      []byte
}

type gameServerProxy struct {
	config        *config.Config
	clientService common.ClientServiceInterface
	tcpClient     *zNet.TcpClient
	discovery     *discovery.ServiceDiscovery
	gameServers   []*discovery.ServerInfo
	pendingQueue  []pendingMessage
	pendingMu     sync.Mutex
}

func NewGameServerProxy(cfg *config.Config, clientService common.ClientServiceInterface) GameServerProxy {
	return &gameServerProxy{
		config:        cfg,
		clientService: clientService,
	}
}

func (gsp *gameServerProxy) Start(ctx context.Context) error {
	zLog.Info("Starting GameServer proxy...")

	discovery, err := discovery.NewServiceDiscoveryWithConfig([]string{gsp.config.Etcd.Endpoints}, &gsp.config.Etcd)
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

func (gsp *gameServerProxy) IsConnected() bool {
	return gsp.tcpClient != nil && gsp.tcpClient.IsConnected()
}

func (gsp *gameServerProxy) discoverAndConnectGameServers(ctx context.Context) {
	go gsp.watchGameServerStatus(ctx)

	gsp.tryDiscoverAndConnect()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gsp.tryDiscoverAndConnect()
		}
	}
}

func (gsp *gameServerProxy) tryDiscoverAndConnect() {
	if gsp.tcpClient != nil && gsp.tcpClient.IsConnected() {
		return
	}

	gwServerID, err := id.ParseServerIDInt(int32(gsp.config.Server.ServerID))
	if err != nil {
		zLog.Error("Invalid gateway ServerID, skip discover", zap.Int("server_id", gsp.config.Server.ServerID), zap.Error(err))
		return
	}
	groupID := id.GroupIDStringFromServerID(gwServerID)
	instances, err := gsp.discovery.Discover("game", groupID)
	if err != nil {
		zLog.Error("Failed to discover GameServer", zap.Error(err))
		return
	}

	if len(instances) == 0 {
		zLog.Warn("No GameServer instances found")
		return
	}

	selectedInstance := gsp.selectGameServer(instances)
	if selectedInstance == nil {
		zLog.Warn("No suitable GameServer instance found")
		return
	}

	gsp.connectToGameServer(selectedInstance)
}

func (gsp *gameServerProxy) watchGameServerStatus(ctx context.Context) {
	gwServerID, err := id.ParseServerIDInt(int32(gsp.config.Server.ServerID))
	if err != nil {
		zLog.Error("Invalid gateway ServerID, skip watch", zap.Int("server_id", gsp.config.Server.ServerID), zap.Error(err))
		return
	}
	groupID := id.GroupIDStringFromServerID(gwServerID)

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
			gsp.handleGameServerStatusChange(event)
		}
	}
}

func (gsp *gameServerProxy) handleGameServerStatusChange(event *discovery.ServerEvent) {
	zLog.Info("Received GameServer status change event",
		zap.String("server_id", event.ServerID),
		zap.String("event_type", event.EventType),
		zap.String("status", string(event.Status)))

	targetServerID := id.ServerIDString(id.MustParseServerIDInt(int32(gsp.config.Server.ServerID)))
	if event.ServerID != targetServerID {
		return
	}

	if event.EventType == "add" || event.Status == "healthy" || event.Status == "ready" {
		if gsp.tcpClient == nil || !gsp.tcpClient.IsConnected() {
			zLog.Info("GameServer became available, triggering immediate discovery")
			go gsp.tryDiscoverAndConnect()
		}
	}

	if gsp.tcpClient != nil && gsp.tcpClient.IsConnected() {
		if event.Status == "maintenance" || event.Status == "stopped" {
			zLog.Warn("Current GameServer status changed to unhealthy, closing connection",
				zap.String("server_id", event.ServerID),
				zap.String("status", string(event.Status)))
			gsp.tcpClient.Close()
		}
	}
}

func (gsp *gameServerProxy) selectGameServer(instances []*discovery.ServerInfo) *discovery.ServerInfo {
	if len(instances) == 0 {
		return nil
	}

	targetServerID := id.ServerIDString(id.MustParseServerIDInt(int32(gsp.config.Server.ServerID)))
	zLog.Info("Selecting GameServer", zap.String("target_id", targetServerID), zap.Int("instance_count", len(instances)))
	for _, inst := range instances {
		zLog.Info("Checking instance", zap.String("inst_id", inst.ID), zap.String("address", inst.Address), zap.String("status", string(inst.Status)))
		if inst.ID == targetServerID {
			if inst.Status == "healthy" || inst.Status == "ready" {
				zLog.Info("Selected GameServer with matching ServerID",
					zap.String("server_id", inst.ID),
					zap.String("address", inst.Address),
					zap.String("status", string(inst.Status)))
				return inst
			}
		}
	}

	zLog.Warn("No matching GameServer instance found for ServerID", zap.Int("server_id", gsp.config.Server.ServerID), zap.String("target_id", targetServerID))
	return nil
}

func (gsp *gameServerProxy) connectToGameServer(instance *discovery.ServerInfo) {
	host := instance.Address
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	addr := fmt.Sprintf("%s:%d", host, instance.Port)
	zLog.Info("Connecting to GameServer", zap.String("addr", addr), zap.String("server_id", instance.ID))

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		zLog.Error("Failed to resolve GameServer address", zap.Error(err), zap.String("addr", addr))
		return
	}

	clientConfig := &zNet.TcpClientConfig{
		ServerAddr:        tcpAddr.IP.String(),
		ServerPort:        tcpAddr.Port,
		HeartbeatDuration: 30,
		MaxPacketDataSize: 1024 * 1024,
		AutoReconnect:     true,
		ReconnectDelay:    5,
		DisableEncryption: true,
	}

	gsp.tcpClient = zNet.NewTcpClient(clientConfig,
		zNet.WithClientLogger(zLog.GetStandardLogger()),
		zNet.WithClientStateCallback(func(state zNet.ClientState) {
			switch state {
			case zNet.ClientStateConnected:
				zLog.Info("Connected to GameServer", zap.String("addr", instance.Address), zap.String("server_id", instance.ID))
				gsp.flushPendingMessages()
			case zNet.ClientStateDisconnected:
				zLog.Warn("Disconnected from GameServer")
			case zNet.ClientStateReconnecting:
				zLog.Info("Reconnecting to GameServer...")
			}
		}),
	)

	gsp.tcpClient.RegisterDispatcher(gsp.handleGameServerMessage)

	err = gsp.tcpClient.Connect()
	if err != nil {
		zLog.Error("Failed to connect to GameServer", zap.Error(err), zap.String("addr", instance.Address))
		return
	}
}

func (gsp *gameServerProxy) flushPendingMessages() {
	gsp.pendingMu.Lock()
	messages := gsp.pendingQueue
	gsp.pendingQueue = nil
	gsp.pendingMu.Unlock()

	if len(messages) == 0 {
		return
	}

	zLog.Info("Flushing pending messages to GameServer", zap.Int("count", len(messages)))
	for _, msg := range messages {
		if err := gsp.sendToGameServerInternal(msg.sessionID, msg.protoId, msg.data); err != nil {
			zLog.Error("Failed to flush pending message",
				zap.Int32("proto_id", msg.protoId),
				zap.Error(err))
		}
	}
}

func (gsp *gameServerProxy) handleGameServerMessage(session zNet.Session, packet *zNet.NetPacket) error {
	zLog.Info("Received message from GameServer", zap.Uint32("proto_id", uint32(packet.ProtoId)), zap.Int("data_size", len(packet.Data)))
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
			zap.Int("server_id", gsp.config.Server.ServerID))
	}

	var crossMsg protocol.CrossServerMessage
	if err := proto.Unmarshal(data, &crossMsg); err != nil {
		zLog.Error("Failed to unmarshal cross server message", zap.Error(err), zap.Int("data_size", len(data)))
		return
	}

	baseMsg := crossMsg.Message
	if baseMsg == nil {
		zLog.Error("Cross server message has no base message")
		return
	}

	sessionID := zNet.SessionIdType(baseMsg.SessionId)
	clientData := baseMsg.Data
	msgID := baseMsg.MsgId

	zLog.Info("Received response from GameServer",
		zap.Uint32("msg_id", msgID),
		zap.Uint64("session_id", uint64(sessionID)),
		zap.Int("data_size", len(clientData)))

	err := gsp.clientService.SendToClient(sessionID, msgID, clientData)
	if err != nil {
		zLog.Error("Failed to send message to client",
			zap.Error(err),
			zap.Uint64("session_id", uint64(sessionID)))
		return
	}

	zLog.Debug("Message forwarded to client",
		zap.Uint32("msg_id", msgID),
		zap.Uint64("session_id", uint64(sessionID)))
}

func (gsp *gameServerProxy) SendToGameServer(sessionID zNet.SessionIdType, protoId int32, data []byte) error {
	if gsp.tcpClient == nil || !gsp.tcpClient.IsConnected() {
		gsp.pendingMu.Lock()
		if len(gsp.pendingQueue) < 100 {
			gsp.pendingQueue = append(gsp.pendingQueue, pendingMessage{
				sessionID: sessionID,
				protoId:   protoId,
				data:      data,
			})
			zLog.Warn("GameServer not connected, message queued",
				zap.Int32("proto_id", protoId),
				zap.Uint64("session_id", uint64(sessionID)),
				zap.Int("queue_size", len(gsp.pendingQueue)))
		} else {
			zLog.Error("Pending queue full, dropping message",
				zap.Int32("proto_id", protoId),
				zap.Int("queue_size", len(gsp.pendingQueue)))
		}
		gsp.pendingMu.Unlock()
		return fmt.Errorf("not connected to GameServer, message queued")
	}

	return gsp.sendToGameServerInternal(sessionID, protoId, data)
}

func (gsp *gameServerProxy) sendToGameServerInternal(sessionID zNet.SessionIdType, protoId int32, data []byte) error {
	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: uint64(sessionID),
		ServerId:  uint32(gsp.config.Server.ServerID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}

	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(gsp.config.Server.ServerID),
		FromService:  uint32(crossserver.ServiceTypeGateway),
		ToService:    uint32(crossserver.ServiceTypeGame),
		Message:      baseMsg,
	}

	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		zLog.Error("Failed to marshal cross server message", zap.Error(err))
		return err
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGateway, int32(gsp.config.Server.ServerID))
	wrappedData := crossserver.Wrap(meta, crossMsgData)

	err = gsp.tcpClient.Send(zNet.ProtoIdType(protoId), wrappedData)
	if err != nil {
		zLog.Error("Failed to send message to GameServer", zap.Error(err))
		return err
	}

	zLog.Info("Message sent to GameServer", zap.Uint64("session_id", uint64(sessionID)), zap.Int32("proto_id", protoId), zap.Int("data_len", len(data)))
	return nil
}
