package crossserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

type RPCRequest struct {
	MsgID     uint32      `json:"msg_id"`
	RequestID uint64      `json:"request_id"`
	Payload   interface{} `json:"payload"`
}

type RPCResponse struct {
	RequestID uint64      `json:"request_id"`
	Payload   interface{} `json:"payload"`
	Error     string      `json:"error,omitempty"`
}

type RPCHandlerFunc func(ctx context.Context, requestID uint64, payload []byte) (interface{}, error)

type RPCEndpoint struct {
	serviceType uint8
	msgID       uint32
	handler     RPCHandlerFunc
	timeout     time.Duration
	router      *CrossRouter
	reqRouter   *RequestRouter
}

func NewRPCEndpoint(serviceType uint8, msgID uint32, handler RPCHandlerFunc) *RPCEndpoint {
	return &RPCEndpoint{
		serviceType: serviceType,
		msgID:       msgID,
		handler:     handler,
		timeout:     10 * time.Second,
	}
}

func (e *RPCEndpoint) WithTimeout(timeout time.Duration) *RPCEndpoint {
	e.timeout = timeout
	return e
}

func (e *RPCEndpoint) WithRouter(router *CrossRouter) *RPCEndpoint {
	e.router = router
	return e
}

func (e *RPCEndpoint) WithRequestRouter(reqRouter *RequestRouter) *RPCEndpoint {
	e.reqRouter = reqRouter
	return e
}

func (e *RPCEndpoint) Register() {
	if e.router == nil {
		zLog.Error("Cannot register RPC endpoint without CrossRouter",
			zap.Uint8("service", e.serviceType),
			zap.Uint32("msg_id", e.msgID))
		return
	}

	e.router.RegisterHandler(e.serviceType, e.msgID, func(meta Meta, payload []byte) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
		defer cancel()

		result, err := e.handler(ctx, meta.RequestID, payload)
		if err != nil {
			resp := RPCResponse{
				RequestID: meta.RequestID,
				Error:     err.Error(),
			}
			respBytes, _ := json.Marshal(resp)
			return respBytes, nil
		}

		resp := RPCResponse{
			RequestID: meta.RequestID,
			Payload:   result,
		}
		respBytes, _ := json.Marshal(resp)
		return respBytes, nil
	})

	zLog.Info("RPC endpoint registered",
		zap.Uint8("service", e.serviceType),
		zap.Uint32("msg_id", e.msgID))
}

func (e *RPCEndpoint) Call(ctx context.Context, serverRouter *ServerRouter, targetServerID string, payload interface{}) (interface{}, error) {
	if e.reqRouter == nil {
		return nil, fmt.Errorf("request router not configured")
	}

	reqID := e.reqRouter.NextRequestID()

	req := RPCRequest{
		MsgID:     e.msgID,
		RequestID: reqID,
		Payload:   payload,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal rpc request: %w", err)
	}

	meta := NewRequestMeta(e.serviceType, 0)
	meta.RequestID = reqID
	enveloped := Wrap(meta, reqBytes)

	result, err := e.reqRouter.SendRequest(ctx, reqID, func() error {
		return serverRouter.SendToServer(targetServerID, enveloped)
	})

	if err != nil {
		return nil, fmt.Errorf("rpc call: %w", err)
	}

	var resp RPCResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal rpc response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("rpc error: %s", resp.Error)
	}

	return resp.Payload, nil
}

func (e *RPCEndpoint) CallToService(ctx context.Context, serverRouter *ServerRouter, targetServiceType uint8, payload interface{}) error {
	req := RPCRequest{
		MsgID:   e.msgID,
		Payload: payload,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal rpc request: %w", err)
	}

	meta := NewRequestMeta(e.serviceType, 0)
	enveloped := Wrap(meta, reqBytes)

	return serverRouter.SendToService(targetServiceType, enveloped)
}

type RPCService struct {
	router    *CrossRouter
	reqRouter *RequestRouter
	srvRouter *ServerRouter
	endpoints []*RPCEndpoint
}

func NewRPCService(router *CrossRouter, reqRouter *RequestRouter, srvRouter *ServerRouter) *RPCService {
	return &RPCService{
		router:    router,
		reqRouter: reqRouter,
		srvRouter: srvRouter,
		endpoints: make([]*RPCEndpoint, 0),
	}
}

func (rs *RPCService) Register(serviceType uint8, msgID uint32, handler RPCHandlerFunc, opts ...func(*RPCEndpoint)) *RPCEndpoint {
	ep := NewRPCEndpoint(serviceType, msgID, handler)
	ep.WithRouter(rs.router)
	ep.WithRequestRouter(rs.reqRouter)

	for _, opt := range opts {
		opt(ep)
	}

	ep.Register()
	rs.endpoints = append(rs.endpoints, ep)
	return ep
}

func (rs *RPCService) Call(ctx context.Context, serviceType uint8, msgID uint32, targetServerID string, payload interface{}) (interface{}, error) {
	for _, ep := range rs.endpoints {
		if ep.serviceType == serviceType && ep.msgID == msgID {
			return ep.Call(ctx, rs.srvRouter, targetServerID, payload)
		}
	}
	return nil, fmt.Errorf("no rpc endpoint for service=%d msg_id=%d", serviceType, msgID)
}

func (rs *RPCService) EndpointCount() int {
	return len(rs.endpoints)
}
