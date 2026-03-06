package router

import (
	"sync"
)

type PacketHandler func(sessionID string, msgID uint32, data []byte) error

type PacketRouter struct {
	handlers map[uint32]PacketHandler
	mu       sync.RWMutex
}

func NewPacketRouter() *PacketRouter {
	return &PacketRouter{
		handlers: make(map[uint32]PacketHandler),
	}
}

func (pr *PacketRouter) RegisterHandler(msgID uint32, handler PacketHandler) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.handlers[msgID] = handler
}

func (pr *PacketRouter) UnregisterHandler(msgID uint32) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	delete(pr.handlers, msgID)
}

func (pr *PacketRouter) Route(sessionID string, msgID uint32, data []byte) error {
	pr.mu.RLock()
	handler, exists := pr.handlers[msgID]
	pr.mu.RUnlock()

	if !exists {
		return nil
	}

	return handler(sessionID, msgID, data)
}

func (pr *PacketRouter) HasHandler(msgID uint32) bool {
	pr.mu.RLock()
	_, exists := pr.handlers[msgID]
	pr.mu.RUnlock()

	return exists
}
