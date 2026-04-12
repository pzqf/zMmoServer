package client

import (
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/client/security"
	"github.com/pzqf/zMmoServer/GatewayServer/proxy"
	"go.uber.org/zap"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	ipManager        *security.IPManager
	antiCheatManager *security.AntiCheatManager
	gameServerProxy  proxy.GameServerProxy
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(ipManager *security.IPManager, antiCheatManager *security.AntiCheatManager, gameServerProxy proxy.GameServerProxy) *MessageHandler {
	return &MessageHandler{
		ipManager:        ipManager,
		antiCheatManager: antiCheatManager,
		gameServerProxy:  gameServerProxy,
	}
}

// HandleMessage 处理客户端消息
func (mh *MessageHandler) HandleMessage(session zNet.Session, packet *zNet.NetPacket) error {
	// 处理客户端消息
	clientIP := session.GetClientIP()
	sessionID := session.GetSid()

	// 检查IP是否被封禁
	if !mh.ipManager.CheckIPAllowed(clientIP) {
		zLog.Warn("IP is banned, closing connection", zap.String("client_ip", clientIP))
		session.Close()
		return fmt.Errorf("IP banned")
	}

	// 检查防作弊状态
	clientID := fmt.Sprintf("client_%d", sessionID)
	allowed, reason := mh.antiCheatManager.CheckClientStatus(clientID)
	if !allowed {
		zLog.Warn("Client rejected due to cheat detection",
			zap.String("client_id", clientID),
			zap.String("reason", reason))
		session.Close()
		return fmt.Errorf("cheat detected: %s", reason)
	}

	// 记录客户端行为
	mh.antiCheatManager.RecordClientAction(clientID, int(packet.DataSize))

	// 转发消息到GameServer
	if mh.gameServerProxy != nil {
		err := mh.gameServerProxy.SendToGameServer(sessionID, int32(packet.ProtoId), packet.Data)
		if err != nil {
			zLog.Error("Failed to forward message to GameServer",
				zap.Error(err),
				zap.Uint64("session_id", uint64(sessionID)),
				zap.Int32("proto_id", int32(packet.ProtoId)))
			return err
		}
	}

	return nil
}
