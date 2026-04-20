package message

import (
	"fmt"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type SystemHandler struct {
	sessionManager *session.SessionManager
}

func NewSystemHandler(sessionManager *session.SessionManager) *SystemHandler {
	return &SystemHandler{
		sessionManager: sessionManager,
	}
}

func (h *SystemHandler) Handle(session zNet.Session, protoId int32, data []byte) error {
	switch protoId {
	case int32(protocol.SystemMsgId_MSG_SYSTEM_ACCOUNT_LOGIN_NOTIFY):
		return h.handleAccountLoginNotify(data)
	default:
		zLog.Warn("Unknown system message", zap.Int32("proto_id", protoId))
		return nil
	}
}

func (h *SystemHandler) handleAccountLoginNotify(data []byte) error {
	var notify protocol.AccountLoginNotify
	if err := proto.Unmarshal(data, &notify); err != nil {
		zLog.Error("Failed to unmarshal AccountLoginNotify", zap.Error(err))
		return err
	}

	sessionID := fmt.Sprintf("%d", notify.SessionId)
	accountID := id.AccountIdType(notify.AccountId)
	accountName := notify.AccountName

	zLog.Info("Account login notify received",
		zap.String("session_id", sessionID),
		zap.Int64("account_id", int64(accountID)),
		zap.String("account_name", accountName))

	sess, exists := h.sessionManager.GetSession(sessionID)
	if !exists {
		sess = h.sessionManager.CreateSession(sessionID, "", "")
	}

	sess.AccountID = accountID
	sess.Status = session.SessionStatusLoggedIn

	h.sessionManager.BindAccount(sessionID, accountID)

	zLog.Info("Account bound to session in GameServer",
		zap.String("session_id", sessionID),
		zap.Int64("account_id", int64(accountID)))

	return nil
}
