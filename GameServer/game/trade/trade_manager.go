// Package trade 提供 GameServer 级的玩家间交易（P2P trade）会话管理。
//
// 这是本业务引入的又一类新架构路径：**两方有状态事务 + 跨玩家一致的原子交换**。
// TradeManager 只持有会话态（双方 ID / 报价 / 确认标志），是纯粹的状态权威、不碰玩家金币；
// 真正的原子金币交换由上层 handler 在双方确认后执行（金币本身是 atomic CAS，交换带回滚）。
// 会话仅存于本 GameServer；跨服交易需 Global 层，未做。
//
// 交易安全：设置报价在**值实际变化时重置双方确认**（防最后一刻偷改价被在旧值上成交，T2）；只在值真正
// 改变时重置，事件驱动的客户端应价器仍能稳定收敛（值稳定后确认单调累积）。
package trade

import (
	"fmt"
	"sync"

	"github.com/pzqf/zCommon/common/id"
)

// Session 一笔交易会话（内部态或快照）。
type Session struct {
	A, B         id.PlayerIdType
	GoldA, GoldB int64
	ConfirmA     bool
	ConfirmB     bool
	completing   bool // 已进入成交执行（防重复 CONFIRM 导致 ExecuteSwap 跑两次→金币翻倍）
}

// TradeManager 交易会话权威。一人同时至多一笔交易。
type TradeManager struct {
	mu       sync.Mutex
	sessions map[id.PlayerIdType]*Session // A、B 都指向同一 *Session
}

func NewTradeManager() *TradeManager {
	return &TradeManager{sessions: make(map[id.PlayerIdType]*Session)}
}

// Start 发起 a↔b 交易。双方须空闲且不同。
func (tm *TradeManager) Start(a, b id.PlayerIdType) (*Session, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if a == b {
		return nil, fmt.Errorf("cannot trade with self")
	}
	if _, ok := tm.sessions[a]; ok {
		return nil, fmt.Errorf("player %d already trading", a)
	}
	if _, ok := tm.sessions[b]; ok {
		return nil, fmt.Errorf("player %d already trading", b)
	}
	s := &Session{A: a, B: b}
	tm.sessions[a] = s
	tm.sessions[b] = s
	return snapshot(s), nil
}

// SetGold 设置某方报价（>=0）。返回会话快照。
// 报价实际变化时**重置双方确认**（T2，标准交易安全）：防"一方确认后另一方偷改价"被在旧值上成交。
// 只在值真正改变时重置，避免冗余设价把确认打回、造成事件驱动应价来回震荡；值稳定后确认单调累积、正常收敛。
func (tm *TradeManager) SetGold(by id.PlayerIdType, gold int64) (*Session, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if gold < 0 {
		return nil, fmt.Errorf("gold must be >= 0")
	}
	s, ok := tm.sessions[by]
	if !ok {
		return nil, fmt.Errorf("player %d not trading", by)
	}
	if s.completing {
		return nil, fmt.Errorf("trade is completing")
	}
	if by == s.A {
		if s.GoldA != gold {
			s.GoldA = gold
			s.ConfirmA, s.ConfirmB = false, false
		}
	} else {
		if s.GoldB != gold {
			s.GoldB = gold
			s.ConfirmA, s.ConfirmB = false, false
		}
	}
	return snapshot(s), nil
}

// Confirm 某方确认。返回(快照, 双方是否都已确认, err)。
func (tm *TradeManager) Confirm(by id.PlayerIdType) (*Session, bool, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	s, ok := tm.sessions[by]
	if !ok {
		return nil, false, fmt.Errorf("player %d not trading", by)
	}
	if by == s.A {
		s.ConfirmA = true
	} else {
		s.ConfirmB = true
	}
	// 只让第一次「双方都已确认」通过 ready=true（并置 completing 锁死后续）——防重复 CONFIRM
	// 让 handler 侧的 ExecuteSwap 跑两次导致金币翻倍。锁内判定+置位，与并发 Confirm 互斥。
	ready := s.ConfirmA && s.ConfirmB && !s.completing
	if ready {
		s.completing = true
	}
	return snapshot(s), ready, nil
}

// Cancel 取消某方所在交易，清理会话。返回被取消会话的快照。
func (tm *TradeManager) Cancel(by id.PlayerIdType) (*Session, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	s, ok := tm.sessions[by]
	if !ok {
		return nil, fmt.Errorf("player %d not trading", by)
	}
	delete(tm.sessions, s.A)
	delete(tm.sessions, s.B)
	return snapshot(s), nil
}

// Finish 成交后清理会话（handler 完成金币交换后调用）。
func (tm *TradeManager) Finish(a, b id.PlayerIdType) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.sessions, a)
	delete(tm.sessions, b)
}

func snapshot(s *Session) *Session {
	cp := *s
	return &cp
}
