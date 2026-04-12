package message

import (
	"github.com/pzqf/zEngine/zNet"
)

// Handler 消息处理器接口
type Handler interface {
	Handle(session zNet.Session, protoId int32, data []byte) error
}

// Router 消息路由器
type Router struct {
	handlers map[int32]Handler
}

// NewRouter 创建消息路由器
func NewRouter() *Router {
	return &Router{
		handlers: make(map[int32]Handler),
	}
}

// RegisterHandler 注册消息处理器
func (r *Router) RegisterHandler(protoId int32, handler Handler) {
	r.handlers[protoId] = handler
}

// Handle 处理消息
func (r *Router) Handle(session zNet.Session, protoId int32, data []byte) error {
	handler, exists := r.handlers[protoId]
	if !exists {
		// 处理未注册的消息
		return nil
	}

	return handler.Handle(session, protoId, data)
}
