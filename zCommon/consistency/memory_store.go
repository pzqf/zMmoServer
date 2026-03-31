package consistency

import (
	"sync"
	"time"
)

type OutboxMessage struct {
	RequestID     uint64
	Topic         string
	TargetMapID   int
	ProtoID       int
	Payload       []byte
	Sent          bool
	Attempts      int
	LastError     string
	DeadLetter    bool
	CreatedAt     time.Time
	LastAttemptAt time.Time
}

type OutboxStore interface {
	Add(msg OutboxMessage)
	MarkSent(requestID uint64)
	MarkAttempt(requestID uint64, err error)
	MarkDeadLetter(requestID uint64, reason string)
	ListPending(limit int) []OutboxMessage
	ListDeadLetters(limit int) []OutboxMessage
	CountPending() int
	CountDeadLetters() int
	PurgeDeadLetters(olderThan time.Duration) int
}

type InboxStore interface {
	TryAccept(requestID uint64) bool
}

type MemoryOutbox struct {
	mu       sync.Mutex
	messages map[uint64]OutboxMessage
}

func NewMemoryOutbox() *MemoryOutbox {
	return &MemoryOutbox{
		messages: make(map[uint64]OutboxMessage),
	}
}

func (m *MemoryOutbox) Add(msg OutboxMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	m.messages[msg.RequestID] = msg
}

func (m *MemoryOutbox) MarkSent(requestID uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[requestID]
	if !ok {
		return
	}
	msg.Sent = true
	m.messages[requestID] = msg
}

func (m *MemoryOutbox) MarkAttempt(requestID uint64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[requestID]
	if !ok {
		return
	}
	msg.Attempts++
	msg.LastAttemptAt = time.Now()
	if err != nil {
		msg.LastError = err.Error()
	}
	m.messages[requestID] = msg
}

func (m *MemoryOutbox) ListPending(limit int) []OutboxMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 100
	}
	out := make([]OutboxMessage, 0, limit)
	for _, msg := range m.messages {
		if msg.Sent || msg.DeadLetter {
			continue
		}
		out = append(out, msg)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func (m *MemoryOutbox) MarkDeadLetter(requestID uint64, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[requestID]
	if !ok {
		return
	}
	msg.DeadLetter = true
	msg.LastError = reason
	msg.LastAttemptAt = time.Now()
	m.messages[requestID] = msg
}

func (m *MemoryOutbox) ListDeadLetters(limit int) []OutboxMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 100
	}
	out := make([]OutboxMessage, 0, limit)
	for _, msg := range m.messages {
		if !msg.DeadLetter {
			continue
		}
		out = append(out, msg)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func (m *MemoryOutbox) CountPending() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, msg := range m.messages {
		if msg.Sent || msg.DeadLetter {
			continue
		}
		count++
	}
	return count
}

func (m *MemoryOutbox) CountDeadLetters() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, msg := range m.messages {
		if msg.DeadLetter {
			count++
		}
	}
	return count
}

func (m *MemoryOutbox) PurgeDeadLetters(olderThan time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	removed := 0
	for requestID, msg := range m.messages {
		if !msg.DeadLetter {
			continue
		}
		ageBase := msg.LastAttemptAt
		if ageBase.IsZero() {
			ageBase = msg.CreatedAt
		}
		if olderThan <= 0 || now.Sub(ageBase) >= olderThan {
			delete(m.messages, requestID)
			removed++
		}
	}
	return removed
}

type MemoryInbox struct {
	mu   sync.Mutex
	seen map[uint64]struct{}
}

func NewMemoryInbox() *MemoryInbox {
	return &MemoryInbox{
		seen: make(map[uint64]struct{}),
	}
}

func (m *MemoryInbox) TryAccept(requestID uint64) bool {
	if requestID == 0 {
		return true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.seen[requestID]; ok {
		return false
	}
	m.seen[requestID] = struct{}{}
	return true
}
