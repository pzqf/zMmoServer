package service

import (
	"testing"

	"github.com/pzqf/zEngine/zNet"
	msgHandler "github.com/pzqf/zMmoServer/GameServer/handler/message"
)

// fakeSession 最小 zNet.Session 替身：只需要能被 laneSession 嵌入。
type fakeSession struct {
	zNet.Session
	sid zNet.SessionIdType
}

func (f fakeSession) GetSid() zNet.SessionIdType { return f.sid }

// TestLaneSession_CarriesClientSessionID ★关键回归★
//
// Gateway↔GameServer 只有一条连接承载全服玩家，"这条消息属于哪个客户端"必须**随消息走**。
// 早期实现是把它 SetObj 到共享的 Gateway 会话对象上、handler 再 GetObj 取回——
// 在"收包后立刻同步执行 handler"时没问题；一旦改成按客户端分道异步执行，
// handler 真正跑起来时该字段可能已被**另一个客户端**的消息覆盖，回包就发给错的人。
//
// 这个用例的价值在于：**单客户端跑永远发现不了**，必须两个客户端交错才暴露。
func TestLaneSession_CarriesClientSessionID(t *testing.T) {
	shared := fakeSession{sid: 999} // 共享的 Gateway 会话

	jobA := laneSession{Session: shared, clientSessionID: 1001}
	jobB := laneSession{Session: shared, clientSessionID: 2002}

	// 模拟分道后的交错执行：B 的消息先被处理，A 的后处理。
	// 两者都必须报出**自己**的客户端会话号，而不是对方的、也不是共享会话的 sid。
	if got := jobB.ClientSessionID(); got != 2002 {
		t.Fatalf("B 的客户端会话号错: got %d want 2002", got)
	}
	if got := jobA.ClientSessionID(); got != 1001 {
		t.Fatalf("A 的客户端会话号被 B 覆盖了: got %d want 1001（回包会发给错的客户端）", got)
	}

	// 底层共享会话本身不受影响（发送仍走同一条 Gateway 连接）。
	if jobA.GetSid() != 999 || jobB.GetSid() != 999 {
		t.Fatalf("底层 Gateway 会话不应被改写")
	}
}

// TestLaneSession_ImplementsCarrier laneSession 必须满足 handler 侧取值用的接口，
// 否则 getClientSessionID 会静默回落到共享会话的 sid —— 编译期发现不了，只在运行时串包。
func TestLaneSession_ImplementsCarrier(t *testing.T) {
	var s interface{} = laneSession{Session: fakeSession{sid: 1}, clientSessionID: 42}

	carrier, ok := s.(msgHandler.ClientSessionCarrier)
	if !ok {
		t.Fatalf("laneSession 必须实现 message.ClientSessionCarrier，否则 handler 取不到正确的客户端会话号")
	}
	if got := carrier.ClientSessionID(); got != 42 {
		t.Fatalf("ClientSessionID() = %d, want 42", got)
	}
}
