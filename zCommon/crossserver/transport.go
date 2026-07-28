package crossserver

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"go.uber.org/zap"
)

// CrossTransport 是跨服「请求-响应」传输层：把单向的 ServerRouter.SendToServer 缝成可等待应答的调用。
//
// 它补的是跨服物理路由的两个传输设计洞（2026-07-28 修）：
//  1. **msgID 上线**：Envelope 头没有 msgID 字段，此前 migration 用裸 Wrap(meta, json) 发送，
//     收侧无从知道该路由到哪个 handler。现在一律走 codec.go 的 PackMessage/UnpackMessage，
//     msgID 落在内层 BaseMessage.MsgId（与外层 zNet ProtoId 一致，见 docs/协议契约.md）。
//  2. **响应回投通道**：CrossHandlerFunc 返回响应字节，但传输是单向的，此前这些字节被丢弃，
//     request/prepare/commit 往返从没跑通。现在 HandleIncoming 收到请求后把 handler 的返回值
//     按「原路 reply + 同一 RequestID」回投，发起方经 zNet.RequestRouter 关联唤醒。
//
// 方向由 Envelope.MessageType 区分（Request/Response/Error），响应**复用请求的 msgID**——
// 关联键是 RequestID，不需要为每个响应单独分配消息号。
type CrossTransport struct {
	router        *CrossRouter
	reqRouter     *zNet.RequestRouter
	serverRouter  *ServerRouter
	localService  uint8
	localServerID int32
}

// ReplyFunc 把响应字节沿「请求来的那条连接」回投。protoID = 外层 zNet ProtoId（== 请求 msgID）。
// 服务端侧实现通常是 session.Send，客户端侧是 conn.Send。
type ReplyFunc func(protoID uint32, data []byte) error

// ErrNoReplyChannel 收到需要应答的请求、但调用方没提供回投通道。
var ErrNoReplyChannel = errors.New("cross transport: no reply channel for request")

// NewCrossTransport 创建传输层。timeout 是请求等待应答的上限（<=0 取 10s）。
// serverRouter 提供出站连接（可为 nil，纯收侧场景）；router 承载 handler 注册。
func NewCrossTransport(localService uint8, localServerID int32, router *CrossRouter, serverRouter *ServerRouter, timeout time.Duration) *CrossTransport {
	if router == nil {
		router = NewCrossRouter()
	}
	return &CrossTransport{
		router:        router,
		reqRouter:     zNet.NewRequestRouter(timeout),
		serverRouter:  serverRouter,
		localService:  localService,
		localServerID: localServerID,
	}
}

// Router 返回 handler 注册表，供各业务模块注册自己的跨服 handler。
func (t *CrossTransport) Router() *CrossRouter { return t.router }

// LocalServerID 本进程的服务器 ID（写入 Envelope.SourceServerID）。
func (t *CrossTransport) LocalServerID() int32 { return t.localServerID }

// LocalService 本进程的服务类型（ServiceTypeGame/ServiceTypeMap/...）。
func (t *CrossTransport) LocalService() uint8 { return t.localService }

// PendingCount 当前等待应答的请求数（测试/监控用）。
func (t *CrossTransport) PendingCount() int { return t.reqRouter.PendingCount() }

// ServerKey 把服务器 ID 转成 ServerRouter 的连接键。
// 约定（本传输层定稿）：**ServerRouter 的连接键 = 服务器 ID 的十进制字符串**，
// 免得注册端与发送端各拼各的键、连上了却发不出去。
func ServerKey(serverID int32) string { return strconv.FormatInt(int64(serverID), 10) }

// Call 发一条请求并等待应答。
// 返回目标 handler 的响应字节；目标 handler 返回 error 时，这里返回同文的 error（不再干等超时）。
func (t *CrossTransport) Call(ctx context.Context, targetServerID int32, targetService uint8, msgID uint32, playerID int64, payload []byte) ([]byte, error) {
	if t.serverRouter == nil {
		return nil, fmt.Errorf("cross transport: no server router")
	}

	reqID := t.reqRouter.NextRequestID()
	meta := NewRequestMeta(t.localService, t.localServerID)
	meta.RequestID = reqID
	meta.TraceID = reqID

	enveloped, err := t.pack(meta, targetServerID, targetService, msgID, playerID, payload)
	if err != nil {
		return nil, err
	}

	return t.reqRouter.SendRequest(ctx, reqID, func() error {
		return t.serverRouter.SendToServer(ServerKey(targetServerID), enveloped)
	})
}

// Notify 发一条不等应答的消息。RequestID 置 0 标明"不要回投"，收侧据此不发响应。
func (t *CrossTransport) Notify(targetServerID int32, targetService uint8, msgID uint32, playerID int64, payload []byte) error {
	if t.serverRouter == nil {
		return fmt.Errorf("cross transport: no server router")
	}

	meta := NewRequestMeta(t.localService, t.localServerID)
	meta.RequestID = 0
	enveloped, err := t.pack(meta, targetServerID, targetService, msgID, playerID, payload)
	if err != nil {
		return err
	}
	return t.serverRouter.SendToServer(ServerKey(targetServerID), enveloped)
}

func (t *CrossTransport) pack(meta Meta, targetServerID int32, targetService uint8, msgID uint32, playerID int64, payload []byte) ([]byte, error) {
	base := BuildBaseMessage(msgID, uint64(playerID), uint32(t.localServerID), 0, payload)
	return PackMessage(meta, t.localService, targetService, uint32(t.localServerID), uint32(targetServerID), base)
}

// HandleIncoming 处理一条到达的跨服线格式字节：解包后交给 Dispatch。
// reply 用于把响应回投给对端（请求类消息必须提供；纯通知可传 nil）。
func (t *CrossTransport) HandleIncoming(data []byte, reply ReplyFunc) error {
	meta, _, base, err := UnpackMessage(data)
	if err != nil {
		return fmt.Errorf("cross transport: %w", err)
	}
	if base == nil {
		return errors.New("cross transport: message missing inner base message")
	}
	return t.Dispatch(meta, base, reply)
}

// Dispatch 分派一条已解包的跨服消息。已有自己解包逻辑的服务（如 MapServer 的 tcp_service）
// 可在 switch 的 default 分支直接调它，把跨服 handler 接进既有入站链路。
//
// 三种方向：
//   - Response/Error：按 RequestID 唤醒本地等待中的 Call。
//   - Request：路由到 handler；handler 返回字节 → 回投 Response；返回 error → 回投 Error；
//     两者皆 nil → 视为通知，不回投。
func (t *CrossTransport) Dispatch(meta Meta, base *protocol.BaseMessage, reply ReplyFunc) error {
	switch meta.MessageType {
	case MessageTypeResponse:
		if !t.reqRouter.CompleteRequest(meta.RequestID, base.Data, nil) {
			zLog.Warn("Cross transport: response for unknown/expired request",
				zap.Uint64("request_id", meta.RequestID), zap.Uint32("msg_id", base.MsgId))
		}
		return nil

	case MessageTypeError:
		errMsg := string(base.Data)
		if errMsg == "" {
			errMsg = "remote handler failed"
		}
		if !t.reqRouter.CompleteRequest(meta.RequestID, nil, errors.New(errMsg)) {
			zLog.Warn("Cross transport: error response for unknown/expired request",
				zap.Uint64("request_id", meta.RequestID), zap.String("error", errMsg))
		}
		return nil
	}

	// 请求方向：路由到 handler。
	respPayload, handlerErr := t.router.Route(meta, base.MsgId, base.Data)

	// 无 RequestID 的消息是纯通知（Notify 发出的），不回投——即便 handler 有返回值。
	if meta.RequestID == 0 {
		return handlerErr
	}
	if respPayload == nil && handlerErr == nil {
		return nil
	}
	if reply == nil {
		zLog.Error("Cross transport: request needs a reply but no reply channel",
			zap.Uint64("request_id", meta.RequestID), zap.Uint32("msg_id", base.MsgId))
		return ErrNoReplyChannel
	}

	respMeta := NewResponseMetaFromRequest(meta, t.localService, t.localServerID)
	payload := respPayload
	if handlerErr != nil {
		respMeta.MessageType = MessageTypeError
		payload = []byte(handlerErr.Error())
	}

	respBase := BuildBaseMessage(base.MsgId, base.PlayerId, uint32(t.localServerID), base.MapId, payload)
	enveloped, err := PackMessage(respMeta, t.localService, meta.SourceService,
		uint32(t.localServerID), uint32(meta.SourceServerID), respBase)
	if err != nil {
		return fmt.Errorf("cross transport: pack response: %w", err)
	}
	if err := reply(base.MsgId, enveloped); err != nil {
		return fmt.Errorf("cross transport: reply: %w", err)
	}
	return handlerErr
}

// Cleanup 清理超时未回的挂起请求，须周期调用（否则超时项会积累）。
func (t *CrossTransport) Cleanup() { t.reqRouter.Cleanup() }

// StartCleanupLoop 周期 Cleanup，直到 ctx 结束。调用方自行 go 起。
func (t *CrossTransport) StartCleanupLoop(ctx context.Context, tick time.Duration) {
	if tick <= 0 {
		tick = 10 * time.Second
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.Cleanup()
		}
	}
}
