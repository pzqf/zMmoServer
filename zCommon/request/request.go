package request

import (
	"context"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

type DedupStore struct {
	requests map[string]time.Time
	mutex    sync.RWMutex
	expire   time.Duration
	stopCh   chan struct{}
}

func NewDedupStore(expire time.Duration) *DedupStore {
	s := &DedupStore{
		requests: make(map[string]time.Time),
		expire:   expire,
		stopCh:   make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

func (s *DedupStore) TryAccept(key string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.requests[key]; exists {
		zLog.Debug("Duplicate request detected", zap.String("key", key))
		return false
	}

	s.requests[key] = time.Now()
	return true
}

func (s *DedupStore) Remove(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.requests, key)
}

func (s *DedupStore) GetSize() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.requests)
}

func (s *DedupStore) Stop() {
	close(s.stopCh)
}

func (s *DedupStore) cleanupLoop() {
	ticker := time.NewTicker(s.expire / 2)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

func (s *DedupStore) cleanup() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for key, timestamp := range s.requests {
		if now.Sub(timestamp) > s.expire {
			delete(s.requests, key)
		}
	}
}

type TimeoutManager struct {
	requests map[string]context.CancelFunc
	mutex    sync.RWMutex
}

func NewTimeoutManager() *TimeoutManager {
	return &TimeoutManager{
		requests: make(map[string]context.CancelFunc),
	}
}

func (m *TimeoutManager) AddRequest(key string, timeout time.Duration, callback func()) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	m.mutex.Lock()
	m.requests[key] = cancel
	m.mutex.Unlock()

	go func() {
		<-ctx.Done()
		m.mutex.Lock()
		delete(m.requests, key)
		m.mutex.Unlock()

		if ctx.Err() == context.DeadlineExceeded {
			zLog.Warn("Request timeout", zap.String("key", key))
			if callback != nil {
				callback()
			}
		}
	}()

	cleanup := func() {
		m.mutex.Lock()
		if c, exists := m.requests[key]; exists {
			c()
			delete(m.requests, key)
		}
		m.mutex.Unlock()
	}

	return ctx, cleanup
}

func (m *TimeoutManager) RemoveRequest(key string) {
	m.mutex.Lock()
	if cancel, exists := m.requests[key]; exists {
		cancel()
		delete(m.requests, key)
	}
	m.mutex.Unlock()
}

func (m *TimeoutManager) GetRequestCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.requests)
}
