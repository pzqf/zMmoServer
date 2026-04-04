package connection

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

// ClientConnMgr 客户端连接管理器
type ClientConnMgr struct {
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

// NewClientConnMgr 创建客户端连接管理器
func NewClientConnMgr() *ClientConnMgr {
	return &ClientConnMgr{
		sessions:   zMap.NewTypedMap[zNet.SessionIdType, *SessionInfo](),
		accountMap: zMap.NewTypedMap[id.AccountIdType, zNet.SessionIdType](),
		nameMap:    zMap.NewTypedMap[string, id.AccountIdType](),
	}
}

// AddSession 添加会话
func (cm *ClientConnMgr) AddSession(sessionID zNet.SessionIdType, clientAddr string) {
	cm.sessions.Store(sessionID, &SessionInfo{
		SessionID:  sessionID,
		ClientAddr: clientAddr,
	})

	zLog.Info("Session added", zap.Uint64("session_id", uint64(sessionID)), zap.String("client_addr", clientAddr))
}

// RemoveSession 移除会话
func (cm *ClientConnMgr) RemoveSession(sessionID zNet.SessionIdType) {
	session, exists := cm.sessions.Load(sessionID)
	if !exists {
		return
	}

	// 从账号映射中移除
	if session.AccountID != 0 {
		cm.accountMap.Delete(session.AccountID)
	}

	// 从名称映射中移除
	if session.AccountName != "" {
		cm.nameMap.Delete(session.AccountName)
	}

	cm.sessions.Delete(sessionID)

	zLog.Info("Session removed", zap.Uint64("session_id", uint64(sessionID)))
}

// SetAccountInfo 设置账号信息
func (cm *ClientConnMgr) SetAccountInfo(sessionID zNet.SessionIdType, accountID id.AccountIdType, accountName string) {
	session, exists := cm.sessions.Load(sessionID)
	if !exists {
		return
	}

	session.AccountID = accountID
	session.AccountName = accountName

	// 更新账号映射
	cm.accountMap.Store(accountID, sessionID)

	// 更新名称映射
	cm.nameMap.Store(accountName, accountID)

	zLog.Info("Account info set", 
		zap.Uint64("session_id", uint64(sessionID)), 
		zap.Int64("account_id", int64(accountID)), 
		zap.String("account_name", accountName))
}

// GetSessionInfo 获取会话信息
func (cm *ClientConnMgr) GetSessionInfo(sessionID zNet.SessionIdType) (*SessionInfo, bool) {
	return cm.sessions.Load(sessionID)
}

// GetSessionByAccountID 根据账号ID获取会话ID
func (cm *ClientConnMgr) GetSessionByAccountID(accountID id.AccountIdType) (zNet.SessionIdType, bool) {
	return cm.accountMap.Load(accountID)
}

// GetAccountIDByName 根据账号名称获取账号ID
func (cm *ClientConnMgr) GetAccountIDByName(accountName string) (id.AccountIdType, bool) {
	return cm.nameMap.Load(accountName)
}

// GetAccountID 获取会话的账号ID
func (cm *ClientConnMgr) GetAccountID(sessionID zNet.SessionIdType) (int64, bool) {
	session, exists := cm.sessions.Load(sessionID)
	if !exists {
		return 0, false
	}
	return int64(session.AccountID), true
}

// GetConnectionCount 获取连接数
func (cm *ClientConnMgr) GetConnectionCount() int {
	return int(cm.sessions.Len())
}

// GetAccountCount 获取账号数
func (cm *ClientConnMgr) GetAccountCount() int {
	return int(cm.accountMap.Len())
}
