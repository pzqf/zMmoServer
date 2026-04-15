package crossserver

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type CrossHandlerFunc func(meta Meta, payload []byte) ([]byte, error)

type CrossRouter struct {
	handlers *zMap.TypedMap[uint8, *serviceRouter]
	running  atomic.Bool
	logger   *zap.Logger
}

type serviceRouter struct {
	handlers *zMap.TypedMap[uint32, CrossHandlerFunc]
}

func newServiceRouter() *serviceRouter {
	return &serviceRouter{
		handlers: zMap.NewTypedMap[uint32, CrossHandlerFunc](),
	}
}

func NewCrossRouter() *CrossRouter {
	return &CrossRouter{
		handlers: zMap.NewTypedMap[uint8, *serviceRouter](),
		logger:   zLog.GetLogger(),
	}
}

func (cr *CrossRouter) RegisterHandler(serviceType uint8, msgID uint32, handler CrossHandlerFunc) {
	sr, exists := cr.handlers.Load(serviceType)
	if !exists {
		sr = newServiceRouter()
		cr.handlers.Store(serviceType, sr)
	}
	sr.handlers.Store(msgID, handler)
	cr.logger.Debug("Cross-router handler registered",
		zap.Uint8("service", serviceType),
		zap.Uint32("msg_id", msgID))
}

func (cr *CrossRouter) UnregisterHandler(serviceType uint8, msgID uint32) {
	sr, exists := cr.handlers.Load(serviceType)
	if !exists {
		return
	}
	sr.handlers.Delete(msgID)
}

func (cr *CrossRouter) Route(meta Meta, msgID uint32, payload []byte) ([]byte, error) {
	sr, exists := cr.handlers.Load(meta.SourceService)
	if !exists {
		return nil, fmt.Errorf("no handler for service type %d", meta.SourceService)
	}

	handler, exists := sr.handlers.Load(msgID)
	if !exists {
		return nil, fmt.Errorf("no handler for msg_id %d from service %d", msgID, meta.SourceService)
	}

	return handler(meta, payload)
}

func (cr *CrossRouter) HasHandler(serviceType uint8, msgID uint32) bool {
	sr, exists := cr.handlers.Load(serviceType)
	if !exists {
		return false
	}
	_, exists = sr.handlers.Load(msgID)
	return exists
}

type ServerConnection struct {
	Conn       interface{}
	ServerID   string
	ServiceType uint8
	Address    string
	Connected  bool
	SendFunc   func(data []byte) error
}

type ServerRouter struct {
	connections *zMap.TypedMap[string, *ServerConnection]
	serviceMap  *zMap.TypedMap[uint8, *zMap.TypedMap[string, *ServerConnection]]
	mapToServer *zMap.TypedMap[int32, string]
	running     atomic.Bool
	mu          sync.Mutex
}

func NewServerRouter() *ServerRouter {
	return &ServerRouter{
		connections: zMap.NewTypedMap[string, *ServerConnection](),
		serviceMap:  zMap.NewTypedMap[uint8, *zMap.TypedMap[string, *ServerConnection]](),
		mapToServer: zMap.NewTypedMap[int32, string](),
	}
}

func (sr *ServerRouter) RegisterConnection(conn *ServerConnection) {
	sr.connections.Store(conn.ServerID, conn)

	serviceConns, exists := sr.serviceMap.Load(conn.ServiceType)
	if !exists {
		serviceConns = zMap.NewTypedMap[string, *ServerConnection]()
		sr.serviceMap.Store(conn.ServiceType, serviceConns)
	}
	serviceConns.Store(conn.ServerID, conn)

	zLog.Info("Server connection registered",
		zap.String("server_id", conn.ServerID),
		zap.Uint8("service_type", conn.ServiceType),
		zap.String("address", conn.Address))
}

func (sr *ServerRouter) UnregisterConnection(serverID string) {
	if conn, exists := sr.connections.LoadAndDelete(serverID); exists {
		if serviceConns, ok := sr.serviceMap.Load(conn.ServiceType); ok {
			serviceConns.Delete(serverID)
		}

		sr.mapToServer.Range(func(mapID int32, sid string) bool {
			if sid == serverID {
				sr.mapToServer.Delete(mapID)
			}
			return true
		})

		zLog.Info("Server connection unregistered",
			zap.String("server_id", serverID))
	}
}

func (sr *ServerRouter) RegisterMapServer(mapID int32, serverID string) {
	sr.mapToServer.Store(mapID, serverID)
	zLog.Debug("Map registered to server",
		zap.Int32("map_id", mapID),
		zap.String("server_id", serverID))
}

func (sr *ServerRouter) SendToServer(serverID string, data []byte) error {
	conn, exists := sr.connections.Load(serverID)
	if !exists || !conn.Connected {
		return fmt.Errorf("server not connected: %s", serverID)
	}
	return conn.SendFunc(data)
}

func (sr *ServerRouter) SendToService(serviceType uint8, data []byte) error {
	serviceConns, exists := sr.serviceMap.Load(serviceType)
	if !exists {
		return fmt.Errorf("no servers for service type %d", serviceType)
	}

	var lastErr error
	serviceConns.Range(func(serverID string, conn *ServerConnection) bool {
		if conn.Connected {
			if err := conn.SendFunc(data); err != nil {
				lastErr = err
			}
		}
		return true
	})
	return lastErr
}

func (sr *ServerRouter) SendToMapServer(mapID int32, data []byte) error {
	serverID, exists := sr.mapToServer.Load(mapID)
	if !exists {
		return fmt.Errorf("no server for map %d", mapID)
	}
	return sr.SendToServer(serverID, data)
}

func (sr *ServerRouter) GetConnection(serverID string) (*ServerConnection, bool) {
	return sr.connections.Load(serverID)
}

func (sr *ServerRouter) GetMapServerConnection(mapID int32) (*ServerConnection, bool) {
	serverID, exists := sr.mapToServer.Load(mapID)
	if !exists {
		return nil, false
	}
	return sr.connections.Load(serverID)
}

func (sr *ServerRouter) GetConnectionsByService(serviceType uint8) []*ServerConnection {
	serviceConns, exists := sr.serviceMap.Load(serviceType)
	if !exists {
		return nil
	}

	var result []*ServerConnection
	serviceConns.Range(func(serverID string, conn *ServerConnection) bool {
		result = append(result, conn)
		return true
	})
	return result
}

func (sr *ServerRouter) ConnectionCount() int {
	return int(sr.connections.Len())
}

type PendingResponse struct {
	ch       chan *ResponseResult
	deadline time.Time
}

type ResponseResult struct {
	Data  []byte
	Error error
}

type RequestRouter struct {
	pending   *zMap.TypedMap[uint64, *PendingResponse]
	nextReqID atomic.Uint64
	timeout   time.Duration
	cleanupMu sync.Mutex
}

func NewRequestRouter(timeout time.Duration) *RequestRouter {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &RequestRouter{
		pending: zMap.NewTypedMap[uint64, *PendingResponse](),
		timeout: timeout,
	}
}

func (rr *RequestRouter) NextRequestID() uint64 {
	return rr.nextReqID.Add(1)
}

func (rr *RequestRouter) RegisterPending(requestID uint64) <-chan *ResponseResult {
	ch := make(chan *ResponseResult, 1)
	rr.pending.Store(requestID, &PendingResponse{
		ch:       ch,
		deadline: time.Now().Add(rr.timeout),
	})
	return ch
}

func (rr *RequestRouter) CompleteRequest(requestID uint64, data []byte, err error) bool {
	pending, exists := rr.pending.LoadAndDelete(requestID)
	if !exists {
		return false
	}

	result := &ResponseResult{Data: data, Error: err}
	select {
	case pending.ch <- result:
	default:
	}
	return true
}

func (rr *RequestRouter) SendRequest(ctx context.Context, requestID uint64, sendFn func() error) ([]byte, error) {
	respCh := rr.RegisterPending(requestID)

	if err := sendFn(); err != nil {
		rr.pending.Delete(requestID)
		return nil, fmt.Errorf("send request failed: %w", err)
	}

	select {
	case result := <-respCh:
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Data, nil
	case <-ctx.Done():
		rr.pending.Delete(requestID)
		return nil, ctx.Err()
	case <-time.After(rr.timeout):
		rr.pending.Delete(requestID)
		return nil, fmt.Errorf("request %d timed out after %v", requestID, rr.timeout)
	}
}

func (rr *RequestRouter) Cleanup() {
	rr.cleanupMu.Lock()
	defer rr.cleanupMu.Unlock()

	now := time.Now()
	var expired []uint64
	rr.pending.Range(func(reqID uint64, p *PendingResponse) bool {
		if now.After(p.deadline) {
			expired = append(expired, reqID)
		}
		return true
	})

	for _, reqID := range expired {
		if p, exists := rr.pending.LoadAndDelete(reqID); exists {
			select {
			case p.ch <- &ResponseResult{Error: fmt.Errorf("request expired")}:
			default:
			}
		}
	}

	if len(expired) > 0 {
		zLog.Debug("Cleaned up expired pending requests",
			zap.Int("count", len(expired)))
	}
}

func (rr *RequestRouter) PendingCount() int {
	return int(rr.pending.Len())
}
