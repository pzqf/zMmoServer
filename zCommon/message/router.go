package message

import (
	"errors"
	"github.com/pzqf/zEngine/zNet"
)

// MessageHandler 消息处理函数类型
type MessageHandler func(session zNet.Session, msg *Message) error

// Router 消息路由器
type Router struct {
	handlers map[uint32]MessageHandler
}

// NewRouter 创建新的消息路由器
func NewRouter() *Router {
	return &Router{
		handlers: make(map[uint32]MessageHandler),
	}
}

// RegisterHandler 注册消息处理函数
func (r *Router) RegisterHandler(msgID uint32, handler MessageHandler) {
	r.handlers[msgID] = handler
}

// HandleMessage 处理消息
func (r *Router) HandleMessage(session zNet.Session, data []byte) error {
	msg, err := Decode(data)
	if err != nil {
		return err
	}
	
	handler, ok := r.handlers[msg.Header.MsgID]
	if !ok {
		return errors.New("no handler found for message ID")
	}
	
	return handler(session, msg)
}