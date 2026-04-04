package auth

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GatewayServer/client/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	config       *config.Config
	connMgr      *connection.ClientConnMgr
	tokenManager *TokenManager
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(cfg *config.Config, connMgr *connection.ClientConnMgr, tokenManager *TokenManager) *AuthHandler {
	return &AuthHandler{
		config:       cfg,
		connMgr:      connMgr,
		tokenManager: tokenManager,
	}
}

// HandleTokenVerify 处理Token验证
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

	response := &protocol.ServerMessage{
		Result:   0, // 临时使用0表示成功
		ErrorMsg: "Success",
	}
	return ah.sendResponse(session, response)
}

// sendResponse 发送响应
func (ah *AuthHandler) sendResponse(session zNet.Session, message proto.Message) error {
	data, err := proto.Marshal(message)
	if err != nil {
		zLog.Error("Failed to marshal response", zap.Error(err))
		return err
	}

	return session.Send(1, data) // 临时使用1表示登录响应
}
