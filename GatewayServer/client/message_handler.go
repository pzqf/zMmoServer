package client

import (
	"fmt"

	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/client/auth"
	"github.com/pzqf/zMmoServer/GatewayServer/client/security"
	"github.com/pzqf/zMmoServer/GatewayServer/proxy"
	"go.uber.org/zap"
)

type MessageHandler struct {
	ipManager        *security.IPManager
	antiCheatManager *security.AntiCheatManager
	gameServerProxy  proxy.GameServerProxy
	authHandler      *auth.AuthHandler
}

func NewMessageHandler(ipManager *security.IPManager, antiCheatManager *security.AntiCheatManager, gameServerProxy proxy.GameServerProxy, authHandler *auth.AuthHandler) *MessageHandler {
	return &MessageHandler{
		ipManager:        ipManager,
		antiCheatManager: antiCheatManager,
		gameServerProxy:  gameServerProxy,
		authHandler:      authHandler,
	}
}

func (mh *MessageHandler) HandleMessage(session zNet.Session, packet *zNet.NetPacket) error {
	clientIP := session.GetClientIP()
	sessionID := session.GetSid()

	if !mh.ipManager.CheckIPAllowed(clientIP) {
		zLog.Warn("IP is banned, closing connection", zap.String("client_ip", clientIP))
		session.Close()
		return fmt.Errorf("IP banned")
	}

	clientID := fmt.Sprintf("client_%d", sessionID)
	allowed, reason := mh.antiCheatManager.CheckClientStatus(clientID)
	if !allowed {
		zLog.Warn("Client rejected due to cheat detection",
			zap.String("client_id", clientID),
			zap.String("reason", reason))
		session.Close()
		return fmt.Errorf("cheat detected: %s", reason)
	}

	mh.antiCheatManager.RecordClientAction(clientID, int(packet.DataSize))

	protoId := int32(packet.ProtoId)

	if protoId == int32(protocol.SystemMsgId_MSG_SYSTEM_TOKEN_VERIFY) {
		if mh.authHandler != nil {
			tokenString := string(packet.Data)
			if err := mh.authHandler.HandleTokenVerify(session, tokenString); err != nil {
				zLog.Error("Token verify failed", zap.Error(err), zap.Uint64("session_id", uint64(sessionID)))
				return err
			}
		}
		return nil
	}

	if protoId == int32(protocol.SystemMsgId_MSG_SYSTEM_HEARTBEAT) || protoId == int32(protocol.SystemMsgId_MSG_SYSTEM_PING) {
		return nil
	}

	if mh.gameServerProxy != nil {
		err := mh.gameServerProxy.SendToGameServer(sessionID, protoId, packet.Data)
		if err != nil {
			zLog.Error("Failed to forward message to GameServer",
				zap.Error(err),
				zap.Uint64("session_id", uint64(sessionID)),
				zap.Int32("proto_id", protoId))
			return err
		}
	}

	return nil
}
