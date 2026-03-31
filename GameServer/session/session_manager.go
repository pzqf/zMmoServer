package session

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zCommon/common/id"
	"go.uber.org/zap"
)

type SessionStatus int

const (
	SessionStatusNew SessionStatus = iota
	SessionStatusLoggingIn
	SessionStatusLoggedIn
	SessionStatusSelectingServer
	SessionStatusInGame
	SessionStatusDisconnected
)

type Session struct {
	SessionID    string
	ConnectionID string
	ClientAddr   string

	AccountID    id.AccountIdType
	PlayerID     id.PlayerIdType
	GameServerID int32

	LoginType  string
	LoginTime  int64
	DeviceID   string
	DeviceInfo string

	Status         SessionStatus
	LastActiveTime int64
	HeartbeatCount int64

	ReconnectToken  string
	ReconnectExpire int64

	IP       string
	IPRegion string
	IsSecure bool
}

type SessionManager struct {
	sessions      map[string]*Session
	sessionsMu    sync.RWMutex
	accountToConn map[id.AccountIdType]string
	accountMu     sync.RWMutex
	playerToConn  map[id.PlayerIdType]string
	playerMu      sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:      make(map[string]*Session),
		accountToConn: make(map[id.AccountIdType]string),
		playerToConn:  make(map[id.PlayerIdType]string),
	}
}

func (sm *SessionManager) CreateSession(sessionID, connectionID, clientAddr string) *Session {
	sm.sessionsMu.Lock()
	defer sm.sessionsMu.Unlock()

	session := &Session{
		SessionID:      sessionID,
		ConnectionID:   connectionID,
		ClientAddr:     clientAddr,
		Status:         SessionStatusNew,
		LastActiveTime: time.Now().Unix(),
		LoginTime:      time.Now().Unix(),
		IsSecure:       true,
	}

	sm.sessions[sessionID] = session

	return session
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.sessionsMu.RLock()
	defer sm.sessionsMu.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.sessionsMu.Lock()
	defer sm.sessionsMu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		if session.AccountID > 0 {
			sm.accountMu.Lock()
			delete(sm.accountToConn, session.AccountID)
			sm.accountMu.Unlock()
		}
		if session.PlayerID > 0 {
			sm.playerMu.Lock()
			delete(sm.playerToConn, session.PlayerID)
			sm.playerMu.Unlock()
		}
		delete(sm.sessions, sessionID)
	}
}

func (sm *SessionManager) BindAccount(sessionID string, accountID id.AccountIdType) {
	sm.sessionsMu.Lock()
	session, exists := sm.sessions[sessionID]
	sm.sessionsMu.Unlock()

	if !exists {
		return
	}

	sm.accountMu.Lock()
	if oldConnID, exists := sm.accountToConn[accountID]; exists {
		sm.sessionsMu.Lock()
		if oldSession, exists := sm.sessions[oldConnID]; exists {
			oldSession.Status = SessionStatusDisconnected
		}
		sm.sessionsMu.Unlock()
	}
	sm.accountToConn[accountID] = sessionID
	sm.accountMu.Unlock()

	sm.sessionsMu.Lock()
	session.AccountID = accountID
	session.Status = SessionStatusLoggedIn
	sm.sessionsMu.Unlock()

	zLog.Info("Account bound to session", zap.Int64("account_id", int64(accountID)), zap.String("session_id", sessionID))
}

func (sm *SessionManager) BindPlayer(sessionID string, playerID id.PlayerIdType) {
	sm.sessionsMu.Lock()
	session, exists := sm.sessions[sessionID]
	sm.sessionsMu.Unlock()

	if !exists {
		return
	}

	sm.playerMu.Lock()
	if oldConnID, exists := sm.playerToConn[playerID]; exists {
		sm.sessionsMu.Lock()
		if oldSession, exists := sm.sessions[oldConnID]; exists {
			oldSession.Status = SessionStatusDisconnected
		}
		sm.sessionsMu.Unlock()
	}
	sm.playerToConn[playerID] = sessionID
	sm.playerMu.Unlock()

	sm.sessionsMu.Lock()
	session.PlayerID = playerID
	session.Status = SessionStatusInGame
	sm.sessionsMu.Unlock()

	zLog.Info("Player bound to session", zap.Int64("player_id", int64(playerID)), zap.String("session_id", sessionID))
}

func (sm *SessionManager) GetSessionByAccount(accountID id.AccountIdType) (*Session, bool) {
	sm.accountMu.RLock()
	connID, exists := sm.accountToConn[accountID]
	sm.accountMu.RUnlock()

	if !exists {
		return nil, false
	}

	return sm.GetSession(connID)
}

func (sm *SessionManager) GetSessionByPlayer(playerID id.PlayerIdType) (*Session, bool) {
	sm.playerMu.RLock()
	connID, exists := sm.playerToConn[playerID]
	sm.playerMu.RUnlock()

	if !exists {
		return nil, false
	}

	return sm.GetSession(connID)
}

func (sm *SessionManager) UpdateSessionStatus(sessionID string, status SessionStatus) {
	sm.sessionsMu.Lock()
	defer sm.sessionsMu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.Status = status
	}
}

func (sm *SessionManager) UpdateLastActive(sessionID string) {
	sm.sessionsMu.Lock()
	defer sm.sessionsMu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.LastActiveTime = time.Now().Unix()
	}
}

func (sm *SessionManager) GetSessionCount() int {
	sm.sessionsMu.RLock()
	defer sm.sessionsMu.RUnlock()

	return len(sm.sessions)
}

func (sm *SessionManager) GetOnlinePlayerCount() int {
	sm.playerMu.RLock()
	defer sm.playerMu.RUnlock()

	count := 0
	for _, session := range sm.sessions {
		if session.Status == SessionStatusInGame {
			count++
		}
	}
	return count
}

