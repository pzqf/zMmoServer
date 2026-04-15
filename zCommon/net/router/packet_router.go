package router

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type HandlerFunc func(session zNet.Session, packet *zNet.NetPacket) error

type DataHandlerFunc func(session zNet.Session, protoId int32, data []byte) error

type PacketRouter struct {
	handlers     *zMap.TypedMap[int32, HandlerFunc]
	dataHandlers *zMap.TypedMap[int32, DataHandlerFunc]
}

func NewPacketRouter() *PacketRouter {
	return &PacketRouter{
		handlers:     zMap.NewTypedMap[int32, HandlerFunc](),
		dataHandlers: zMap.NewTypedMap[int32, DataHandlerFunc](),
	}
}

func (pr *PacketRouter) RegisterHandler(cmd int32, handler HandlerFunc) {
	pr.handlers.Store(cmd, handler)
	zLog.Debug("Registered packet handler", zap.Int32("cmd", cmd))
}

func (pr *PacketRouter) RegisterDataHandler(cmd int32, handler DataHandlerFunc) {
	pr.dataHandlers.Store(cmd, handler)
	zLog.Debug("Registered data handler", zap.Int32("cmd", cmd))
}

func (pr *PacketRouter) UnregisterHandler(cmd int32) {
	pr.handlers.Delete(cmd)
	pr.dataHandlers.Delete(cmd)
	zLog.Debug("Unregistered handler", zap.Int32("cmd", cmd))
}

func (pr *PacketRouter) Route(session zNet.Session, packet *zNet.NetPacket) error {
	if handler, exists := pr.handlers.Load(int32(packet.ProtoId)); exists {
		return handler(session, packet)
	}

	if dataHandler, exists := pr.dataHandlers.Load(int32(packet.ProtoId)); exists {
		return dataHandler(session, int32(packet.ProtoId), packet.Data)
	}

	zLog.Warn("No handler found for command", zap.Int32("cmd", int32(packet.ProtoId)))
	return nil
}

func (pr *PacketRouter) HandleData(session zNet.Session, protoId int32, data []byte) error {
	if dataHandler, exists := pr.dataHandlers.Load(protoId); exists {
		return dataHandler(session, protoId, data)
	}

	if handler, exists := pr.handlers.Load(protoId); exists {
		packet := &zNet.NetPacket{
			ProtoId: zNet.ProtoIdType(protoId),
			Data:    data,
		}
		return handler(session, packet)
	}

	zLog.Warn("No handler found for command", zap.Int32("cmd", protoId))
	return nil
}

func (pr *PacketRouter) HasHandler(cmd int32) bool {
	if _, exists := pr.handlers.Load(cmd); exists {
		return true
	}
	_, exists := pr.dataHandlers.Load(cmd)
	return exists
}

func (pr *PacketRouter) HandlerCount() int {
	return int(pr.handlers.Len() + pr.dataHandlers.Len())
}
