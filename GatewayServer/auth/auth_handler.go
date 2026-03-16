package auth

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/token"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type AuthHandler struct {
	config       *config.Config
	connManager  *connection.ConnectionManager
	tokenManager *token.TokenManager
}

func NewAuthHandler(cfg *config.Config, connManager *connection.ConnectionManager,
	tokenManager *token.TokenManager) *AuthHandler {
	return &AuthHandler{
		config:       cfg,
		connManager:  connManager,
		tokenManager: tokenManager,
	}
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

	ah.connManager.SetAccountInfo(sessionID, id.AccountIdType(claims.AccountID), claims.AccountName)

	response := &protocol.ServerMessage{
		Result: int32(protocol.ErrorCode_ERR_SUCCESS),
	}
	return ah.sendResponse(session, response)
}

func (ah *AuthHandler) sendResponse(session zNet.Session, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		zLog.Error("Failed to marshal response", zap.Error(err))
		return err
	}

	return session.Send(0, data)
}
