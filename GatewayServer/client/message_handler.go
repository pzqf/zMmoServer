package client

import (
	"fmt"
	"strconv"

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

	// 反作弊的统计**主体**（subject）：已认证会话按**账号**，未认证阶段（拿不到账号）才按 IP。
	//
	// 演变：最早按 sessionID —— 客户端一重连即换 ID、统计清零，检测可被轻易绕过（SEC-4 修）；
	// 于是改成按 clientIP —— 重连不清零了，但**同一 IP 上的所有玩家被并成一份统计**：
	// 网吧/公司/家庭多设备、尤其运营商 CGNAT 下成千上万玩家共用一个出口 IP，动作数叠加后
	// 轻易越过阈值，累计 3 次高危就拒绝**该 IP 上所有人**的新连接——一人超标、全楼连坐。
	// 账号同时满足两个要求：重连稳定（不被绕过）+ 玩家间独立（不连坐）。
	subject := mh.cheatSubject(sessionID, clientIP)

	allowed, reason := mh.antiCheatManager.CheckClientStatus(subject)
	if !allowed {
		zLog.Warn("Client rejected due to cheat detection",
			zap.String("subject", subject),
			zap.String("client_ip", clientIP),
			zap.String("reason", reason))
		session.Close()
		return fmt.Errorf("cheat detected: %s", reason)
	}

	mh.antiCheatManager.RecordClientAction(subject, int(packet.DataSize))

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
		// SEC-4: 记一次错误——未认证却发业务消息属可疑，计入其错误率触发反作弊。
		mh.antiCheatManager.RecordError(subject, "unauthenticated")
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
			mh.antiCheatManager.RecordError(subject, "forward_failed") // SEC-4
			return err
		}
	}

	return nil
}

// cheatSubject 反作弊统计主体：认证后按账号（`acct:<id>`），未认证按 IP（`ip:<addr>`）。
//
// 未认证阶段只有 token 校验那几个包，量极小；一旦认证成功即切到账号维度，
// 于是同一 IP 上的不同玩家各记各的，互不连坐。
func (mh *MessageHandler) cheatSubject(sessionID zNet.SessionIdType, clientIP string) string {
	if mh.authHandler != nil {
		if accountID := mh.authHandler.AccountOf(sessionID); accountID != 0 {
			return "acct:" + strconv.FormatInt(accountID, 10)
		}
	}
	return "ip:" + clientIP
}
