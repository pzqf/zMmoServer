package crossserver

import (
	"fmt"
	"testing"
)

// requestID 同时是**幂等去重键**：收侧（MapServer 的进图/攻击去重、GameServer 的响应去重、
// 迁移 inbox）都按裸 requestID 判重放。各进程的序号都从 1 开始递增，若不混入 serverID，
// 两个 realm 的 GameServer 必然发出同号请求 —— 目标 MapServer 会把第二个 realm 玩家的进图
// 当成重放直接丢弃，那个玩家永远进不去跨服实例。
//
// 这不是推演：2026-07-28 的跨 realm 多进程 E2E 就是这么红的
//（map_server 日志 "Duplicate map enter request ignored, request_id=4"）。

// TestComposeRequestID_UniqueAcrossServers 不同服务器的同序号请求必须得到不同的 requestID。
func TestComposeRequestID_UniqueAcrossServers(t *testing.T) {
	seen := make(map[uint64]string)
	for _, serverID := range []int32{101, 201, 102, 999999} {
		for seq := uint64(1); seq <= 8; seq++ {
			id := ComposeRequestID(serverID, seq)
			key := fmt.Sprintf("server=%d seq=%d", serverID, seq)
			if prev, dup := seen[id]; dup {
				t.Fatalf("requestID 冲突：%s 与 %s 都得到 %d（跨 realm 会被误判为重放）", key, prev, id)
			}
			seen[id] = key
		}
	}
}

// TestComposeRequestID_MonotonicPerServer 同一服务器内仍单调递增（便于按序排查日志）。
func TestComposeRequestID_MonotonicPerServer(t *testing.T) {
	prev := uint64(0)
	for seq := uint64(1); seq <= 200; seq++ {
		id := ComposeRequestID(101, seq)
		if id <= prev {
			t.Fatalf("同服务器内应单调递增: seq=%d id=%d prev=%d", seq, id, prev)
		}
		prev = id
	}
}

// TestComposeRequestID_ZeroServerKeepsLegacyBehaviour serverID=0（未指定）时退化为纯进程内序号。
func TestComposeRequestID_ZeroServerKeepsLegacyBehaviour(t *testing.T) {
	if got := ComposeRequestID(0, 12345); got != 12345 {
		t.Fatalf("serverID=0 应原样返回序号, got %d", got)
	}
}

// TestNewRequestMeta_CarriesServerScopedID 生产入口产出的 requestID 必须已带 serverID 前缀。
func TestNewRequestMeta_CarriesServerScopedID(t *testing.T) {
	a := NewRequestMeta(ServiceTypeGame, 101)
	b := NewRequestMeta(ServiceTypeGame, 201)

	if a.RequestID>>40 != 101 {
		t.Fatalf("高位应为 serverID 101, got %d", a.RequestID>>40)
	}
	if b.RequestID>>40 != 201 {
		t.Fatalf("高位应为 serverID 201, got %d", b.RequestID>>40)
	}
	if a.RequestID == b.RequestID {
		t.Fatalf("不同 realm 的请求 ID 不得相同: %d", a.RequestID)
	}
	if a.TraceID != a.RequestID {
		t.Fatalf("TraceID 应与 RequestID 一致（同一次请求的两个视角）")
	}
}

// TestNewRequestMeta_SurvivesEnvelopeRoundTrip 组合后的 requestID 必须能原样穿过 Envelope。
func TestNewRequestMeta_SurvivesEnvelopeRoundTrip(t *testing.T) {
	meta := NewRequestMeta(ServiceTypeGame, 201)
	wrapped := Wrap(meta, []byte("payload"))

	got, payload, ok, err := Unwrap(wrapped)
	if err != nil || !ok {
		t.Fatalf("Unwrap 失败: ok=%v err=%v", ok, err)
	}
	if got.RequestID != meta.RequestID {
		t.Fatalf("requestID 穿线格式后变了: want %d got %d", meta.RequestID, got.RequestID)
	}
	if string(payload) != "payload" {
		t.Fatalf("payload 损坏: %q", payload)
	}
}
