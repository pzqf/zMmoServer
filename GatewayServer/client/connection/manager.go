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
	// kickFn: 关闭指定会话底层连接（GW-3 单端登录踢旧）。由 Service 注入（持 netServer）。
	kickFn func(zNet.SessionIdType)
}

// SetKickHandler 注入"按 sessionID 关闭连接"的能力，用于同账号重复登录时踢掉旧会话。
func (cm *ClientConnMgr) SetKickHandler(fn func(zNet.SessionIdType)) {
	cm.kickFn = fn
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

	// GW-3: 仅当本会话仍是该账号的"当前会话"时才清账号/名称映射。否则（该账号已被新端登录接管，
	// accountMap 指向新会话）旧连接断开会误删新会话的映射，导致新端后续按账号推送/查找全部失效。
	if session.AccountID != 0 {
		if cur, ok := cm.accountMap.Load(session.AccountID); ok && cur == sessionID {
			cm.accountMap.Delete(session.AccountID)
			if session.AccountName != "" {
				cm.nameMap.Delete(session.AccountName)
			}
		}
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

	// GW-3: 同账号多端登录——踢掉该账号此前占用的旧会话（若存在且不同）。先落新映射再踢旧，
	// 这样旧连接 OnClose→RemoveSession 时按 GW-3 的当前会话判定不会误删新映射。
	oldSession, hasOld := cm.accountMap.Load(accountID)

	session.AccountID = accountID
	session.AccountName = accountName

	// 更新账号映射
	cm.accountMap.Store(accountID, sessionID)

	// 更新名称映射
	cm.nameMap.Store(accountName, accountID)

	if hasOld && oldSession != sessionID {
		if cm.kickFn != nil {
			cm.kickFn(oldSession)
		}
		zLog.Warn("Account re-login, kicked previous session",
			zap.Int64("account_id", int64(accountID)),
			zap.Uint64("old_session", uint64(oldSession)),
			zap.Uint64("new_session", uint64(sessionID)))
	}

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
