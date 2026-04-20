package auth

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/client/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/proxy"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type AuthHandler struct {
	config          *config.Config
	connMgr         *connection.ClientConnMgr
	tokenManager    *TokenManager
	gameServerProxy proxy.GameServerProxy
}

func NewAuthHandler(cfg *config.Config, connMgr *connection.ClientConnMgr, tokenManager *TokenManager) *AuthHandler {
	return &AuthHandler{
		config:       cfg,
		connMgr:      connMgr,
		tokenManager: tokenManager,
	}
}

func (ah *AuthHandler) SetGameServerProxy(gsp proxy.GameServerProxy) {
	ah.gameServerProxy = gsp
}

func (ah *AuthHandler) HandleTokenVerify(session zNet.Session, tokenString string) error {
	sessionID := session.GetSid()
	zLog.Info("Handling token verify", zap.Uint64("session_id", uint64(sessionID)))

	claims, err := ah.tokenManager.ValidateToken(tokenString)
	if err != nil {
		zLog.Warn("Token validation failed", zap.Error(err))
		response := &protocol.ServerMessage{
			Result:   int32(protocol.ErrorCode_ERR_ACCOUNT_TOKEN_INVALID),
			ErrorMsg: "Invalid token",
		}
		return ah.sendResponse(session, response)
	}

	zLog.Info("Token verified successfully",
		zap.Int64("account_id", claims.AccountID),
		zap.String("account_name", claims.AccountName))

	ah.connMgr.SetAccountInfo(sessionID, id.AccountIdType(claims.AccountID), claims.AccountName)

	if ah.gameServerProxy != nil {
		notify := &protocol.AccountLoginNotify{
			SessionId:   uint64(sessionID),
			AccountId:   claims.AccountID,
			AccountName: claims.AccountName,
		}
		notifyData, err := proto.Marshal(notify)
		if err != nil {
			zLog.Error("Failed to marshal AccountLoginNotify", zap.Error(err))
		} else {
			if err := ah.gameServerProxy.SendToGameServer(sessionID, int32(protocol.SystemMsgId_MSG_SYSTEM_ACCOUNT_LOGIN_NOTIFY), notifyData); err != nil {
				zLog.Error("Failed to send AccountLoginNotify to GameServer", zap.Error(err))
			} else {
				zLog.Info("AccountLoginNotify sent to GameServer",
					zap.Uint64("session_id", uint64(sessionID)),
					zap.Int64("account_id", claims.AccountID))
			}
		}
	}

	response := &protocol.ServerMessage{
		Result:   0,
		ErrorMsg: "Success",
	}
	return ah.sendResponse(session, response)
}

func (ah *AuthHandler) sendResponse(session zNet.Session, message proto.Message) error {
	data, err := proto.Marshal(message)
	if err != nil {
		zLog.Error("Failed to marshal response", zap.Error(err))
		return err
	}

	return session.Send(zNet.ProtoIdType(protocol.SystemMsgId_MSG_SYSTEM_TOKEN_VERIFY_RESPONSE), data)
}
