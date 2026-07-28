package crossserver

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

var errBoom = errors.New("boom: handler exploded")

// ---------------------------------------------------------------------------
// loopbackNet：把两个 CrossTransport 用「内存连接」接起来的测试网。
//
// 它只替掉最底层的字节投递（真环境是 zNet 会话），**线格式、msgID、Envelope、请求-响应关联
// 全部走生产代码**——所以这套测试能真实证明"控制面握手跑得通"，而不是绕开传输层假装打通。
// 投递是异步的（另起 goroutine），正是为了逼真地考验 RequestRouter 的关联：响应从别的
// goroutine 回来，发起方必须被正确唤醒。断言前调 drain() 等所有投递落定。
// ---------------------------------------------------------------------------

type loopbackNet struct {
	mu   sync.Mutex
	wg   sync.WaitGroup
	errs []error
}

type loopbackNode struct {
	net       *loopbackNet
	serverID  int32
	service   uint8
	transport *CrossTransport
	router    *ServerRouter
}

func newLoopbackNet() *loopbackNet { return &loopbackNet{} }

func (ln *loopbackNet) addNode(serverID int32, service uint8) *loopbackNode {
	sr := NewServerRouter()
	return &loopbackNode{
		net:       ln,
		serverID:  serverID,
		service:   service,
		transport: NewCrossTransport(service, serverID, NewCrossRouter(), sr, 3*time.Second),
		router:    sr,
	}
}

// link 双向连通两个节点（各自在对方的 ServerRouter 里登记一条连接）。
func (ln *loopbackNet) link(a, b *loopbackNode) {
	ln.connect(a, b)
	ln.connect(b, a)
}

func (ln *loopbackNet) connect(from, to *loopbackNode) {
	from.router.RegisterConnection(&ServerConnection{
		ServerID:    ServerKey(to.serverID),
		ServiceType: to.service,
		Connected:   true,
		SendFunc: func(data []byte) error {
			ln.deliver(to, from, data)
			return nil
		},
	})
}

func (ln *loopbackNet) deliver(to, from *loopbackNode, data []byte) {
	ln.wg.Add(1)
	go func() {
		defer ln.wg.Done()
		// reply：把响应沿"消息来的那条连接"投回发送方。
		reply := func(protoID uint32, resp []byte) error {
			ln.deliver(from, to, resp)
			return nil
		}
		if err := to.transport.HandleIncoming(data, reply); err != nil {
			ln.mu.Lock()
			ln.errs = append(ln.errs, err)
			ln.mu.Unlock()
		}
	}()
}

// drain 等所有在途投递处理完（含回投的响应）。
func (ln *loopbackNet) drain() { ln.wg.Wait() }

func (ln *loopbackNet) errors() []error {
	ln.mu.Lock()
	defer ln.mu.Unlock()
	return append([]error(nil), ln.errs...)
}

// ---------------------------------------------------------------------------

const (
	testMsgEcho   uint32 = 9101
	testMsgUpper  uint32 = 9102
	testMsgBoom   uint32 = 9103
	testMsgNotify uint32 = 9104
)

// TestCrossTransport_RoutesByMsgID 传输层最核心的回归守护：**msgID 真的上线了**。
// 旧实现用裸 Wrap(meta, json) 发送，Envelope 头没有 msgID 字段，收侧无从知道该进哪个 handler，
// 于是控制面从没跑通。这里注册两个 msgID 不同的 handler，断言各自只被自己那条消息命中。
func TestCrossTransport_RoutesByMsgID(t *testing.T) {
	net := newLoopbackNet()
	a := net.addNode(101, ServiceTypeGame)
	b := net.addNode(202, ServiceTypeMap)
	net.link(a, b)

	var echoHits, upperHits int
	var mu sync.Mutex
	b.transport.Router().RegisterHandlerAllServices(testMsgEcho, func(meta Meta, payload []byte) ([]byte, error) {
		mu.Lock()
		echoHits++
		mu.Unlock()
		return payload, nil
	})
	b.transport.Router().RegisterHandlerAllServices(testMsgUpper, func(meta Meta, payload []byte) ([]byte, error) {
		mu.Lock()
		upperHits++
		mu.Unlock()
		return []byte(strings.ToUpper(string(payload))), nil
	})

	ctx := context.Background()

	resp, err := a.transport.Call(ctx, b.serverID, b.service, testMsgEcho, 5001, []byte("ping"))
	if err != nil {
		t.Fatalf("call echo: %v", err)
	}
	if string(resp) != "ping" {
		t.Fatalf("echo payload mangled: %q", resp)
	}

	resp, err = a.transport.Call(ctx, b.serverID, b.service, testMsgUpper, 5001, []byte("ping"))
	if err != nil {
		t.Fatalf("call upper: %v", err)
	}
	if string(resp) != "PING" {
		t.Fatalf("expected upper handler, got %q（msgID 没被正确路由）", resp)
	}

	net.drain()
	mu.Lock()
	defer mu.Unlock()
	if echoHits != 1 || upperHits != 1 {
		t.Fatalf("每个 handler 应各命中 1 次, echo=%d upper=%d", echoHits, upperHits)
	}
	if a.transport.PendingCount() != 0 {
		t.Fatalf("应答已到，不应还有挂起请求: %d", a.transport.PendingCount())
	}
}

// TestCrossTransport_MetaCarriesSourceAndRequestID handler 能拿到发起方身份与 RequestID（关联键）。
func TestCrossTransport_MetaCarriesSourceAndRequestID(t *testing.T) {
	net := newLoopbackNet()
	a := net.addNode(101, ServiceTypeGame)
	b := net.addNode(202, ServiceTypeMap)
	net.link(a, b)

	var got Meta
	var mu sync.Mutex
	b.transport.Router().RegisterHandlerAllServices(testMsgEcho, func(meta Meta, payload []byte) ([]byte, error) {
		mu.Lock()
		got = meta
		mu.Unlock()
		return []byte("ok"), nil
	})

	if _, err := a.transport.Call(context.Background(), b.serverID, b.service, testMsgEcho, 1, []byte("x")); err != nil {
		t.Fatalf("call: %v", err)
	}
	net.drain()

	mu.Lock()
	defer mu.Unlock()
	if got.SourceServerID != a.serverID {
		t.Fatalf("source server id = %d, want %d", got.SourceServerID, a.serverID)
	}
	if got.SourceService != ServiceTypeGame {
		t.Fatalf("source service = %d, want %d", got.SourceService, ServiceTypeGame)
	}
	if got.RequestID == 0 {
		t.Fatalf("请求必须带非零 RequestID，否则响应无从关联")
	}
	if got.MessageType != MessageTypeRequest {
		t.Fatalf("message type = %d, want request", got.MessageType)
	}
}

// TestCrossTransport_HandlerErrorPropagates handler 报错 → 发起方立刻拿到同文错误，而不是干等到超时。
func TestCrossTransport_HandlerErrorPropagates(t *testing.T) {
	net := newLoopbackNet()
	a := net.addNode(101, ServiceTypeGame)
	b := net.addNode(202, ServiceTypeMap)
	net.link(a, b)

	b.transport.Router().RegisterHandlerAllServices(testMsgBoom, func(meta Meta, payload []byte) ([]byte, error) {
		return nil, errBoom
	})

	start := time.Now()
	_, err := a.transport.Call(context.Background(), b.serverID, b.service, testMsgBoom, 1, nil)
	if err == nil {
		t.Fatalf("expected error from remote handler")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("错误原因应回投给发起方, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("不应等到超时才失败, 用了 %v", elapsed)
	}
	net.drain()
}

// TestCrossTransport_UnknownMsgIDFails 没注册 handler 的 msgID → 发起方拿到明确错误（而非静默丢弃 + 超时）。
func TestCrossTransport_UnknownMsgIDFails(t *testing.T) {
	net := newLoopbackNet()
	a := net.addNode(101, ServiceTypeGame)
	b := net.addNode(202, ServiceTypeMap)
	net.link(a, b)

	_, err := a.transport.Call(context.Background(), b.serverID, b.service, 60001, 1, nil)
	if err == nil {
		t.Fatalf("expected error for unrouted msg id")
	}
	if !strings.Contains(err.Error(), "no handler") {
		t.Fatalf("错误应说明没有 handler, got %v", err)
	}
	net.drain()
}

// TestCrossTransport_NotifyDoesNotReply 通知类消息（RequestID=0）即便 handler 有返回值也不回投，
// 避免对端凭空收到一条无人等待的响应。
func TestCrossTransport_NotifyDoesNotReply(t *testing.T) {
	net := newLoopbackNet()
	a := net.addNode(101, ServiceTypeGame)
	b := net.addNode(202, ServiceTypeMap)
	net.link(a, b)

	done := make(chan []byte, 1)
	b.transport.Router().RegisterHandlerAllServices(testMsgNotify, func(meta Meta, payload []byte) ([]byte, error) {
		if meta.RequestID != 0 {
			t.Errorf("Notify 发出的消息 RequestID 应为 0, got %d", meta.RequestID)
		}
		done <- payload
		return []byte("this response must be dropped"), nil
	})

	if err := a.transport.Notify(b.serverID, b.service, testMsgNotify, 7, []byte("fire")); err != nil {
		t.Fatalf("notify: %v", err)
	}

	select {
	case payload := <-done:
		if string(payload) != "fire" {
			t.Fatalf("payload = %q", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("通知未送达")
	}

	net.drain()
	if errs := net.errors(); len(errs) != 0 {
		t.Fatalf("通知不应产生投递错误: %v", errs)
	}
	if a.transport.PendingCount() != 0 {
		t.Fatalf("通知不该登记挂起请求: %d", a.transport.PendingCount())
	}
}

// TestCrossTransport_CallTimesOutWhenPeerSilent 对端不应答 → Call 超时返回，且挂起项被清掉。
func TestCrossTransport_CallTimesOutWhenPeerSilent(t *testing.T) {
	net := newLoopbackNet()
	a := net.addNode(101, ServiceTypeGame)
	b := net.addNode(202, ServiceTypeMap)
	net.link(a, b)

	// handler 返回 nil,nil = 不回投（当作通知处理），发起方只能等超时。
	b.transport.Router().RegisterHandlerAllServices(testMsgEcho, func(meta Meta, payload []byte) ([]byte, error) {
		return nil, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	_, err := a.transport.Call(ctx, b.serverID, b.service, testMsgEcho, 1, nil)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	net.drain()
	a.transport.Cleanup()
	if a.transport.PendingCount() != 0 {
		t.Fatalf("超时后不应残留挂起项: %d", a.transport.PendingCount())
	}
}
