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

	// SEC-4: 反作弊按 clientIP 追踪，不再按 sessionID（`client_<sid>`）——否则客户端一重连即换
	// sessionID、统计清零，作弊/爆包检测被轻易绕过。IP 在连接期稳定，重连仍归并同一 IP。
	allowed, reason := mh.antiCheatManager.CheckClientStatus(clientIP)
	if !allowed {
		zLog.Warn("Client rejected due to cheat detection",
			zap.String("client_ip", clientIP),
			zap.String("reason", reason))
		session.Close()
		return fmt.Errorf("cheat detected: %s", reason)
	}

	mh.antiCheatManager.RecordClientAction(clientIP, int(packet.DataSize))

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

	// 鉴权门禁（SEC-1）：TOKEN_VERIFY 与心跳/PING 已在上面处理并 return；到这里的都是
	// 业务消息，必须已通过 token 校验（会话已绑定 AccountID），否则网关沦为未认证开放中继。
	if mh.authHandler == nil || !mh.authHandler.IsAuthenticated(sessionID) {
		zLog.Warn("Unauthenticated message rejected, closing connection",
			zap.Uint64("session_id", uint64(sessionID)), zap.Int32("proto_id", protoId))
		// SEC-4: 记一次错误——未认证却发业务消息属可疑，计入该 IP 的错误率触发反作弊。
		mh.antiCheatManager.RecordError(clientIP, "unauthenticated")
		session.Close()
		return fmt.Errorf("unauthenticated session")
	}

	if mh.gameServerProxy != nil {
		err := mh.gameServerProxy.SendToGameServer(sessionID, protoId, packet.Data)
		if err != nil {
			zLog.Error("Failed to forward message to GameServer",
				zap.Error(err),
				zap.Uint64("session_id", uint64(sessionID)),
				zap.Int32("proto_id", protoId))
			mh.antiCheatManager.RecordError(clientIP, "forward_failed") // SEC-4
			return err
		}
	}

	return nil
}
