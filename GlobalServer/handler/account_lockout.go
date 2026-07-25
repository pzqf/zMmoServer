package handler

import (
	"sync"
	"time"
)

// accountLoginLockout 账号级登录失败锁定（SEC-6）。此前只有 IP 限速，缺账号级爆破防护——
// 攻击者换 IP 即可对单个账号无限试密码。此处按账号累计失败次数，窗口内超阈值临时锁定该账号。
//
// 注：内存态、仅单实例有效；多 GlobalServer 实例需换 Redis 共享计数（见 docs/审查发现.md SEC-6）。
type accountLoginLockout struct {
	mu           sync.Mutex
	records      map[string]*lockRecord
	maxFails     int
	window       time.Duration
	lockDuration time.Duration
	cleanupCap   int // records 超过此数时机会性清理，限制内存增长
}

type lockRecord struct {
	fails       int
	windowStart time.Time
	lockedUntil time.Time
}

// loginLockout 全局单例：5 分钟窗口内失败 5 次 → 锁 15 分钟。
var loginLockout = newAccountLoginLockout(5, 5*time.Minute, 15*time.Minute)

func newAccountLoginLockout(maxFails int, window, lockDuration time.Duration) *accountLoginLockout {
	return &accountLoginLockout{
		records:      make(map[string]*lockRecord),
		maxFails:     maxFails,
		window:       window,
		lockDuration: lockDuration,
		cleanupCap:   4096,
	}
}

// lockedFor 返回该账号当前是否被锁及剩余锁定时长。
func (l *accountLoginLockout) lockedFor(account string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	r := l.records[account]
	if r == nil {
		return false, 0
	}
	if now := time.Now(); now.Before(r.lockedUntil) {
		return true, r.lockedUntil.Sub(now)
	}
	return false, 0
}

// recordFailure 记一次登录失败；窗口内累计超阈值则锁定该账号。
func (l *accountLoginLockout) recordFailure(account string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.records) > l.cleanupCap {
		l.cleanupLocked()
	}

	now := time.Now()
	r := l.records[account]
	if r == nil || now.Sub(r.windowStart) > l.window {
		r = &lockRecord{windowStart: now}
		l.records[account] = r
	}
	r.fails++
	if r.fails >= l.maxFails {
		r.lockedUntil = now.Add(l.lockDuration)
		r.fails = 0
		r.windowStart = now
	}
}

// recordSuccess 成功登录后清除该账号的失败记录。
func (l *accountLoginLockout) recordSuccess(account string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.records, account)
}

// cleanupLocked 清理既不在锁定期、窗口也已过期的记录（调用方须持锁）。
func (l *accountLoginLockout) cleanupLocked() {
	now := time.Now()
	for acc, r := range l.records {
		if now.After(r.lockedUntil) && now.Sub(r.windowStart) > l.window {
			delete(l.records, acc)
		}
	}
}
