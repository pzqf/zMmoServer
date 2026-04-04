package connection

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"go.uber.org/zap"
)

// ClientHandler 客户端连接处理器
type ClientHandler struct {
	connMgr         *ClientConnMgr
	securityManager SecurityManagerInterface
}

// SecurityManagerInterface 安全管理器接口
type SecurityManagerInterface interface {
	CheckIPAllowed(ip string) bool
	AddConnection(ip string)
	RemoveConnection(ip string)
}

// NewClientHandler 创建客户端处理器
func NewClientHandler(connMgr *ClientConnMgr, securityManager SecurityManagerInterface) *ClientHandler {
	return &ClientHandler{
		connMgr:         connMgr,
		securityManager: securityManager,
	}
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
	clientIP := session.GetClientIP()
	clientID := session.GetSid()

	zLog.Info("Client disconnected",
		zap.Uint64("client_id", uint64(clientID)),
		zap.String("client_ip", clientIP))

	// 移除连接计数
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
