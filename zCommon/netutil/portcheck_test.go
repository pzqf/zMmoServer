package netutil

import (
	"net"
	"testing"
)

// TestCheckPortOccupied_Free 未被监听的端口应判为空闲。
func TestCheckPortOccupied_Free(t *testing.T) {
	// 先占一个端口拿到其号，随即关闭 → 该端口此刻空闲。
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	if occ, _ := CheckPortOccupied(addr); occ {
		t.Fatalf("已关闭的端口 %s 应判为空闲，却判为占用", addr)
	}
	if err := EnsurePortFree(addr); err != nil {
		t.Fatalf("EnsurePortFree 对空闲端口不应报错: %v", err)
	}
}

// TestCheckPortOccupied_Busy 正在监听的端口应判为占用，且 EnsurePortFree 返回带信息的 error。
func TestCheckPortOccupied_Busy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	occ, occupant := CheckPortOccupied(addr)
	if !occ {
		t.Fatalf("正在监听的端口 %s 应判为占用", addr)
	}
	t.Logf("占用方描述: %q", occupant) // Windows 下应含本测试进程 PID/名

	if err := EnsurePortFree(addr); err == nil {
		t.Fatalf("EnsurePortFree 对被占端口应返回 error")
	} else {
		t.Logf("fail-fast error: %v", err)
	}
}

// TestCheckPortOccupied_Wildcard 0.0.0.0 监听应能被回环拨号探到。
func TestCheckPortOccupied_Wildcard(t *testing.T) {
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	_, port, _ := net.SplitHostPort(ln.Addr().String())

	if occ, _ := CheckPortOccupied("0.0.0.0:" + port); !occ {
		t.Fatalf("0.0.0.0:%s 正在监听，应判为占用", port)
	}
}
