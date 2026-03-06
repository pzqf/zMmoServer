package connection

import (
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

type ConnectionManager struct {
	sessions   map[zNet.SessionIdType]*SessionInfo
	accountMap map[id.AccountIdType]zNet.SessionIdType
	nameMap    map[string]id.AccountIdType
	mutex      sync.RWMutex
}

type SessionInfo struct {
	SessionID   zNet.SessionIdType
	AccountID   id.AccountIdType
	AccountName string
	ClientAddr  string
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		sessions:   make(map[zNet.SessionIdType]*SessionInfo),
		accountMap: make(map[id.AccountIdType]zNet.SessionIdType),
		nameMap:    make(map[string]id.AccountIdType),
	}
}

func (cm *ConnectionManager) AddSession(sessionID zNet.SessionIdType, clientAddr string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.sessions[sessionID] = &SessionInfo{
		SessionID:  sessionID,
		ClientAddr: clientAddr,
	}

	zLog.Info("Session added", zap.Uint64("session_id", uint64(sessionID)), zap.String("client_addr", clientAddr))
}

func (cm *ConnectionManager) RemoveSession(sessionID zNet.SessionIdType) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return
	}

	if session.AccountID > 0 {
		delete(cm.accountMap, session.AccountID)
		delete(cm.nameMap, session.AccountName)
	}

	delete(cm.sessions, sessionID)

	zLog.Info("Session removed", zap.Uint64("session_id", uint64(sessionID)))
}

func (cm *ConnectionManager) SetAccountInfo(sessionID zNet.SessionIdType, accountID id.AccountIdType, accountName string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		zLog.Warn("Session not found when setting account info", zap.Uint64("session_id", uint64(sessionID)))
		return
	}

	// 清除旧的映射
	if session.AccountID > 0 {
		delete(cm.accountMap, session.AccountID)
		delete(cm.nameMap, session.AccountName)
	}

	// 设置新的映射
	session.AccountID = accountID
	session.AccountName = accountName
	cm.accountMap[accountID] = sessionID
	cm.nameMap[accountName] = accountID

	zLog.Info("Account info set",
		zap.Uint64("session_id", uint64(sessionID)),
		zap.Int64("account_id", int64(accountID)),
		zap.String("account_name", accountName))
}

func (cm *ConnectionManager) GetSessionInfo(sessionID zNet.SessionIdType) (*SessionInfo, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	session, exists := cm.sessions[sessionID]
	return session, exists
}

func (cm *ConnectionManager) GetSessionByAccountID(accountID id.AccountIdType) (zNet.SessionIdType, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	sessionID, exists := cm.accountMap[accountID]
	return sessionID, exists
}

func (cm *ConnectionManager) GetAccountIDByName(accountName string) (id.AccountIdType, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	accountID, exists := cm.nameMap[accountName]
	return accountID, exists
}

func (cm *ConnectionManager) GetAccountID(sessionID zNet.SessionIdType) (int64, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return 0, false
	}

	return int64(session.AccountID), true
}

func (cm *ConnectionManager) GetSessionCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return len(cm.sessions)
}

func (cm *ConnectionManager) GetConnectionCount() int {
	return cm.GetSessionCount()
}

func (cm *ConnectionManager) GetAccountCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return len(cm.accountMap)
}
