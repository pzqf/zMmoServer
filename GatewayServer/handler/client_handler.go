package handler

import (
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/auth"
	"go.uber.org/zap"
)

// ClientHandler 客户端连接处理器
type ClientHandler struct {
	session          zNet.Session
	securityManager  *auth.SecurityManager
	antiCheatManager *auth.AntiCheatManager
}

// NewClientHandler 创建客户端处理器
func NewClientHandler(securityManager *auth.SecurityManager, antiCheatManager *auth.AntiCheatManager) *ClientHandler {
	return &ClientHandler{
		securityManager:  securityManager,
		antiCheatManager: antiCheatManager,
	}
}

// OnConnect 连接建立时调用
func (ch *ClientHandler) OnConnect(session zNet.Session) {
	ch.session = session

	// 获取客户端标识符（使用会话ID）
	clientID := getClientIdentifier(session)

	// 检查防作弊状态
	allowed, reason := ch.antiCheatManager.CheckClientStatus(clientID)
	if !allowed {
		zLog.Warn("Client connection rejected due to cheat detection",
			zap.String("client_id", clientID),
			zap.String("reason", reason))
		session.Close()
		return
	}

	zLog.Info("Client connected",
		zap.Uint64("session_id", uint64(session.GetSid())),
		zap.String("client_id", clientID))
}

// OnClose 连接关闭时调用
func (ch *ClientHandler) OnClose(session zNet.Session) {
	zLog.Info("Client disconnected", zap.Uint64("session_id", uint64(session.GetSid())))
}

// OnReceive 接收到消息时调用
func (ch *ClientHandler) OnReceive(session zNet.Session, packet *zNet.NetPacket) {
	// 获取客户端标识符
	clientID := getClientIdentifier(session)

	// 记录客户端行为
	ch.antiCheatManager.RecordClientAction(clientID, int(packet.DataSize))

	zLog.Debug("Received packet",
		zap.Uint64("session_id", uint64(session.GetSid())),
		zap.Int32("proto_id", packet.ProtoId),
		zap.Int32("data_size", packet.DataSize),
		zap.String("client_id", clientID))

	// TODO: 实现消息处理逻辑
	// 这里需要将消息转发给 GameServer
}

// OnError 发生错误时调用
func (ch *ClientHandler) OnError(session zNet.Session, err error) {
	// 获取客户端标识符
	clientID := getClientIdentifier(session)

	// 记录错误
	ch.antiCheatManager.RecordError(clientID, err.Error())

	zLog.Error("Client error",
		zap.Uint64("session_id", uint64(session.GetSid())),
		zap.String("client_id", clientID),
		zap.Error(err))
}

// getClientIdentifier 获取客户端标识符
func getClientIdentifier(session zNet.Session) string {
	return fmt.Sprintf("client_%d", session.GetSid())
}
