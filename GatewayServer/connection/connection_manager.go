package connection

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

// ConnectionManager 连接管理器
type ConnectionManager struct {
	sessions   *zMap.TypedMap[zNet.SessionIdType, *SessionInfo]
	accountMap *zMap.TypedMap[id.AccountIdType, zNet.SessionIdType]
	nameMap    *zMap.TypedMap[string, id.AccountIdType]
}

// SessionInfo 会话信息
type SessionInfo struct {
	SessionID   zNet.SessionIdType
	AccountID   id.AccountIdType
	AccountName string
	ClientAddr  string
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		sessions:   zMap.NewTypedMap[zNet.SessionIdType, *SessionInfo](),
		accountMap: zMap.NewTypedMap[id.AccountIdType, zNet.SessionIdType](),
		nameMap:    zMap.NewTypedMap[string, id.AccountIdType](),
	}
}

// AddSession 添加会话
func (cm *ConnectionManager) AddSession(sessionID zNet.SessionIdType, clientAddr string) {
	cm.sessions.Store(sessionID, &SessionInfo{
		SessionID:  sessionID,
		ClientAddr: clientAddr,
	})

	zLog.Info("Session added", zap.Uint64("session_id", uint64(sessionID)), zap.String("client_addr", clientAddr))
}

// RemoveSession 移除会话
func (cm *ConnectionManager) RemoveSession(sessionID zNet.SessionIdType) {
	session, exists := cm.sessions.Load(sessionID)
	if !exists {
		return
	}

	if session.AccountID > 0 {
		cm.accountMap.Delete(session.AccountID)
		cm.nameMap.Delete(session.AccountName)
	}

	cm.sessions.Delete(sessionID)

	zLog.Info("Session removed", zap.Uint64("session_id", uint64(sessionID)))
}

// SetAccountInfo 设置账号信息
func (cm *ConnectionManager) SetAccountInfo(sessionID zNet.SessionIdType, accountID id.AccountIdType, accountName string) {
	session, exists := cm.sessions.Load(sessionID)
	if !exists {
		zLog.Warn("Session not found when setting account info", zap.Uint64("session_id", uint64(sessionID)))
		return
	}

	// 清除旧的映射
	if session.AccountID > 0 {
		cm.accountMap.Delete(session.AccountID)
		cm.nameMap.Delete(session.AccountName)
	}

	// 创建新的 SessionInfo
	newSession := &SessionInfo{
		SessionID:   sessionID,
		AccountID:   accountID,
		AccountName: accountName,
		ClientAddr:  session.ClientAddr,
	}

	// 存储新的 SessionInfo
	cm.sessions.Store(sessionID, newSession)

	// 设置新的映射
	cm.accountMap.Store(accountID, sessionID)
	cm.nameMap.Store(accountName, accountID)

	zLog.Info("Account info set",
		zap.Uint64("session_id", uint64(sessionID)),
		zap.Int64("account_id", int64(accountID)),
		zap.String("account_name", accountName))
}

// GetSessionInfo 获取会话信息
func (cm *ConnectionManager) GetSessionInfo(sessionID zNet.SessionIdType) (*SessionInfo, bool) {
	return cm.sessions.Load(sessionID)
}

// GetSessionByAccountID 根据账号ID获取会话
func (cm *ConnectionManager) GetSessionByAccountID(accountID id.AccountIdType) (zNet.SessionIdType, bool) {
	return cm.accountMap.Load(accountID)
}

// GetAccountIDByName 根据账号名称获取账号ID
func (cm *ConnectionManager) GetAccountIDByName(accountName string) (id.AccountIdType, bool) {
	return cm.nameMap.Load(accountName)
}

// GetAccountID 获取账号ID
func (cm *ConnectionManager) GetAccountID(sessionID zNet.SessionIdType) (int64, bool) {
	session, exists := cm.sessions.Load(sessionID)
	if !exists {
		return 0, false
	}

	return int64(session.AccountID), true
}

// GetConnectionCount 获取连接数量
func (cm *ConnectionManager) GetConnectionCount() int {
	return int(cm.sessions.Len())
}

// GetAccountCount 获取账号数量
func (cm *ConnectionManager) GetAccountCount() int {
	return int(cm.accountMap.Len())
}

