package handler

import (
	"reflect"
	"runtime"

	"github.com/pzqf/zCommon/net/router"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"go.uber.org/zap"
)

type Context struct {
	Session zNet.Session
	ProtoId int32
	Data    []byte
}

type HandlerFunc func(ctx *Context) error

type MessageHandler struct {
	router      *router.PacketRouter
	middlewares []MiddlewareFunc
}

type MiddlewareFunc func(ctx *Context, next HandlerFunc) error

func NewMessageHandler(router *router.PacketRouter) *MessageHandler {
	return &MessageHandler{
		router: router,
	}
}

func (mh *MessageHandler) Use(middleware MiddlewareFunc) {
	mh.middlewares = append(mh.middlewares, middleware)
}

func (mh *MessageHandler) Handle(cmd int32, handler HandlerFunc) {
	name := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()

	wrappedHandler := mh.wrapMiddleware(handler)

	mh.router.RegisterDataHandler(cmd, func(session zNet.Session, protoId int32, data []byte) error {
		ctx := &Context{
			Session: session,
			ProtoId: protoId,
			Data:    data,
		}
		return wrappedHandler(ctx)
	})

	zLog.Debug("Registered message handler",
		zap.Int32("cmd", cmd),
		zap.String("handler", name))
}

func (mh *MessageHandler) HandlePacket(cmd int32, handler router.HandlerFunc) {
	mh.router.RegisterHandler(cmd, handler)
}

func (mh *MessageHandler) wrapMiddleware(handler HandlerFunc) HandlerFunc {
	for i := len(mh.middlewares) - 1; i >= 0; i-- {
		middleware := mh.middlewares[i]
		originalHandler := handler
		handler = func(ctx *Context) error {
			return middleware(ctx, originalHandler)
		}
	}
	return handler
}

func LoggingMiddleware(ctx *Context, next HandlerFunc) error {
	zLog.Debug("Handling message",
		zap.Int32("proto_id", ctx.ProtoId),
		zap.Uint64("session_id", uint64(ctx.Session.GetSid())))
	err := next(ctx)
	if err != nil {
		zLog.Error("Message handler error",
			zap.Int32("proto_id", ctx.ProtoId),
			zap.Error(err))
	}
	return err
}

func RecoveryMiddleware(ctx *Context, next HandlerFunc) error {
	defer func() {
		if r := recover(); r != nil {
			zLog.Error("Message handler panic",
				zap.Int32("proto_id", ctx.ProtoId),
				zap.Any("recover", r))
		}
	}()
	return next(ctx)
}
