package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"go.uber.org/zap"
)

// TokenClaims JWT Token声明
type TokenClaims struct {
	AccountID   int64  `json:"account_id"`
	AccountName string `json:"account_name"`
	jwt.RegisteredClaims
}

// TokenManager Token管理器
type TokenManager struct {
	config *config.Config
}

// NewTokenManager 创建Token管理器
func NewTokenManager(cfg *config.Config) *TokenManager {
	return &TokenManager{
		config: cfg,
	}
}

// GenerateToken 生成Token
func (tm *TokenManager) GenerateToken(accountID int64, accountName string) (string, error) {
	claims := TokenClaims{
		AccountID:   accountID,
		AccountName: accountName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(tm.config.Security.TokenExpiry) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(tm.config.Server.JWTSecret))
	if err != nil {
		zLog.Error("Failed to generate token", zap.Error(err))
		return "", err
	}

	zLog.Info("Token generated", zap.Int64("account_id", accountID), zap.String("account_name", accountName))
	return tokenString, nil
}

// ValidateToken 验证Token
func (tm *TokenManager) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(tm.config.Server.JWTSecret), nil
	})

	if err != nil {
		zLog.Warn("Token validation failed", zap.Error(err))
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		zLog.Info("Token validated", zap.Int64("account_id", claims.AccountID), zap.String("account_name", claims.AccountName))
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
