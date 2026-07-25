package handler

import (
	"testing"
	"time"
)

// TestAccountLockout 验证 SEC-6：窗口内失败达阈值即锁定；成功清零；窗口过期计数重置。
func TestAccountLockout(t *testing.T) {
	l := newAccountLoginLockout(3, 100*time.Millisecond, 200*time.Millisecond)
	const acc = "victim"

	// 未失败前不锁。
	if locked, _ := l.lockedFor(acc); locked {
		t.Fatal("should not be locked initially")
	}

	// 达阈值前不锁。
	l.recordFailure(acc)
	l.recordFailure(acc)
	if locked, _ := l.lockedFor(acc); locked {
		t.Fatal("should not be locked below threshold")
	}

	// 第 3 次失败 → 锁定。
	l.recordFailure(acc)
	if locked, remain := l.lockedFor(acc); !locked || remain <= 0 {
		t.Fatalf("should be locked after threshold: locked=%v remain=%v", locked, remain)
	}

	// 锁定期过后解锁。
	time.Sleep(220 * time.Millisecond)
	if locked, _ := l.lockedFor(acc); locked {
		t.Fatal("should unlock after lockDuration")
	}

	// 成功清零：两次失败后成功，再两次失败不应触发锁定（计数已被清）。
	l.recordFailure(acc)
	l.recordFailure(acc)
	l.recordSuccess(acc)
	l.recordFailure(acc)
	l.recordFailure(acc)
	if locked, _ := l.lockedFor(acc); locked {
		t.Fatal("recordSuccess should have reset the failure count")
	}
}

// TestAccountLockout_WindowReset 窗口过期后失败计数重置，不会跨窗累计触发锁定。
func TestAccountLockout_WindowReset(t *testing.T) {
	l := newAccountLoginLockout(3, 40*time.Millisecond, time.Second)
	const acc = "slow"

	l.recordFailure(acc)
	l.recordFailure(acc)
	time.Sleep(60 * time.Millisecond) // 超过窗口
	l.recordFailure(acc)              // 新窗口第 1 次
	if locked, _ := l.lockedFor(acc); locked {
		t.Fatal("failures across expired window must not accumulate to lock")
	}
}
