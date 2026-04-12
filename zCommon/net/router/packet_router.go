package router

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"go.uber.org/zap"
)

// 定义处理器函数类型
type HandlerFunc func(session zNet.Session, packet *zNet.NetPacket) error

// 定义路由处理器映射表
type HandlerTable map[int32]HandlerFunc

// PacketRouter 数据包路由器
type PacketRouter struct {
	handlers HandlerTable
}

// NewPacketRouter 创建一个新的数据包路由器
func NewPacketRouter() *PacketRouter {
	return &PacketRouter{
		handlers: make(HandlerTable),
	}
}

// RegisterHandler 注册一个消息处理器
func (pr *PacketRouter) RegisterHandler(cmd int32, handler HandlerFunc) {
	pr.handlers[cmd] = handler
	zLog.Debug("Registered handler", zap.Int32("cmd", cmd))
}

// UnregisterHandler 注销一个消息处理器
func (pr *PacketRouter) UnregisterHandler(cmd int32) {
	delete(pr.handlers, cmd)
	zLog.Debug("Unregistered handler", zap.Int32("cmd", cmd))
}

// Route 路由数据包到相应的处理程序
func (pr *PacketRouter) Route(session zNet.Session, packet *zNet.NetPacket) error {
	// 查找对应的处理函数
	handler, exists := pr.handlers[int32(packet.ProtoId)]
	if !exists {
		zLog.Warn("No handler found for command", zap.Int32("cmd", int32(packet.ProtoId)))
		return nil
	}

	// 执行处理函数
	return handler(session, packet)
}
