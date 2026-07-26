package connection

import (
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"go.uber.org/zap"
)

// ClientHandler 客户端连接处理器
type ClientHandler struct {
	connMgr         *ClientConnMgr
	securityManager SecurityManagerInterface
	gameNotifier    GameServerNotifier
}

// SecurityManagerInterface 安全管理器接口
type SecurityManagerInterface interface {
	CheckIPAllowed(ip string) bool
	AddConnection(ip string)
	RemoveConnection(ip string)
}

// GameServerNotifier 向 GameServer 发消息的能力（由 game_server_proxy 实现）。用接口避免 connection 反向 import proxy。
type GameServerNotifier interface {
	SendToGameServer(sessionID zNet.SessionIdType, protoId int32, data []byte) error
}

// NewClientHandler 创建客户端处理器
func NewClientHandler(connMgr *ClientConnMgr, securityManager SecurityManagerInterface) *ClientHandler {
	return &ClientHandler{
		connMgr:         connMgr,
		securityManager: securityManager,
	}
}

// SetGameNotifier 注入 GameServer 通知能力（客户端断线时用来通知 GameServer 收尾，F-4）。
func (ch *ClientHandler) SetGameNotifier(n GameServerNotifier) {
	ch.gameNotifier = n
}

// OnConnect 客户端连接回调
func (ch *ClientHandler) OnConnect(session zNet.Session) {
	clientIP := session.GetClientIP()
	clientID := session.GetSid()

	zLog.Info("Client connected",
		zap.Uint64("client_id", uint64(clientID)),
		zap.String("client_ip", clientIP))

	// 检查IP是否被封禁
	if !ch.securityManager.CheckIPAllowed(clientIP) {
		zLog.Warn("Client connection rejected due to IP ban",
			zap.String("client_ip", clientIP),
			zap.Uint64("client_id", uint64(clientID)))
		session.Close()
		return
	}

	// 添加连接计数
	ch.securityManager.AddConnection(clientIP)

	// 添加会话到连接管理器
	ch.connMgr.AddSession(clientID, clientIP)
}

// OnClose 客户端断开连接回调
func (ch *ClientHandler) OnClose(session zNet.Session) {
	clientID := session.GetSid()

	// GW-1：会话已从 TcpServer 移除，此回调收到的 wrapper 的 GetClientIP() 返回 ""，
	// 直接用会导致 RemoveConnection("") → 真实 IP 的连接计数只增不减、达上限后误封整个
	// IP（自我 DoS）。从 connMgr 查回连接时存下的真实 IP（在 RemoveSession 删除之前）。
	clientIP := ""
	if info, ok := ch.connMgr.GetSessionInfo(clientID); ok {
		clientIP = info.ClientAddr
	}

	zLog.Info("Client disconnected",
		zap.Uint64("client_id", uint64(clientID)),
		zap.String("client_ip", clientIP))

	// 非正常掉线收尾（F-4）：客户端 TCP 断开但没发 LEAVE_GAME 时，主动替它向 GameServer 补一条
	// LEAVE_GAME（PlayerId=0，GS 侧按本会话 SessionId 反查玩家）。GS 的 LeaveGame 会执行
	// RunOfflineHooks（清交易/组队悬挂）+ 存盘 + 摘除 actor，避免掉线导致会话悬挂与 actor 泄漏。
	// 优雅登出已先发过 LEAVE_GAME → 此处补发时玩家已摘除，GS 侧幂等 no-op。
	// （注：玩家级重连 Reconnect 当前未接任何客户端消息=死代码，故走完整登出而非挂起等重连。）
	if ch.gameNotifier != nil {
		if err := ch.gameNotifier.SendToGameServer(clientID, int32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME), nil); err != nil {
			zLog.Warn("Notify GameServer on client disconnect failed",
				zap.Uint64("client_id", uint64(clientID)), zap.Error(err))
		}
	}

	// 移除连接计数（用真实 IP）
	ch.securityManager.RemoveConnection(clientIP)

	// 从连接管理器中移除会话
	ch.connMgr.RemoveSession(clientID)
}

// OnReceive 客户端消息接收回调
func (ch *ClientHandler) OnReceive(session zNet.Session, packet *zNet.NetPacket) {
	// 消息处理由消息处理器负责
}

// OnError 客户端错误回调
func (ch *ClientHandler) OnError(session zNet.Session, err error) {
	clientIP := session.GetClientIP()
	clientID := session.GetSid()

	zLog.Warn("Client error",
		zap.Uint64("client_id", uint64(clientID)),
		zap.String("client_ip", clientIP),
		zap.Error(err))

	// 移除连接计数
	ch.securityManager.RemoveConnection(clientIP)

	// 从连接管理器中移除会话
	ch.connMgr.RemoveSession(clientID)
}
