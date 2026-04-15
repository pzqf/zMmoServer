package message

import (
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zCommon/net/router"
)

type Handler interface {
	Handle(session zNet.Session, protoId int32, data []byte) error
}

type Router struct {
	inner *router.PacketRouter
}

func NewRouter() *Router {
	return &Router{
		inner: router.NewPacketRouter(),
	}
}

func (r *Router) RegisterHandler(protoId int32, handler Handler) {
	r.inner.RegisterDataHandler(protoId, func(session zNet.Session, pid int32, data []byte) error {
		return handler.Handle(session, pid, data)
	})
}

func (r *Router) Handle(session zNet.Session, protoId int32, data []byte) error {
	return r.inner.HandleData(session, protoId, data)
}
