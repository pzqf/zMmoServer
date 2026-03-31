package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"go.uber.org/zap"
)

// TokenClaims JWT声明
type TokenClaims struct {
	AccountID   int64  `json:"account_id"`
	AccountName string `json:"account_name"`
	jwt.RegisteredClaims
}

// TokenManager Token管理器
type TokenManager struct {
	secretKey   string
	tokenExpiry time.Duration
}

// NewTokenManager 创建Token管理器
func NewTokenManager(cfg *config.Config) *TokenManager {
	return &TokenManager{
		secretKey:   cfg.Server.JWTSecret,
		tokenExpiry: time.Duration(cfg.Security.TokenExpiry) * time.Second,
	}
}

// ValidateToken 验证Token
func (tm *TokenManager) ValidateToken(tokenString string) (*TokenClaims, error) {
	if tokenString == "" {
		return nil, errors.New("empty token")
	}

	// 首先尝试解析为自定义Claims格式
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(tm.secretKey), nil
	})

	if err != nil {
		// 如果解析失败，尝试解析为MapClaims格式（GlobalServer生成的Token格式）
		token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 验证签名算法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(tm.secretKey), nil
		})

		if err != nil {
			zLog.Error("Failed to parse token", zap.Error(err))
			return nil, err
		}

		// 处理MapClaims格式
		if mapClaims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// 验证Token是否过期
			exp, ok := mapClaims["exp"].(float64)
			if !ok || time.Now().Unix() > int64(exp) {
				return nil, errors.New("token expired")
			}

			// 提取账号信息
			accountID, ok := mapClaims["account_id"].(float64)
			if !ok {
				return nil, errors.New("invalid token claims")
			}

			accountName, ok := mapClaims["account_name"].(string)
			if !ok {
				return nil, errors.New("invalid token claims")
			}

			// 构建TokenClaims
			claims := &TokenClaims{
				AccountID:   int64(accountID),
				AccountName: accountName,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Unix(int64(exp), 0)),
					IssuedAt:  jwt.NewNumericDate(time.Unix(int64(mapClaims["iat"].(float64)), 0)),
				},
			}

			zLog.Info("Token validated successfully (MapClaims format)", zap.Int64("account_id", claims.AccountID))
			return claims, nil
		}
	} else if token.Valid {
		// 处理自定义Claims格式
		if claims, ok := token.Claims.(*TokenClaims); ok {
			// 验证Token是否过期
			if time.Now().Unix() > claims.ExpiresAt.Unix() {
				return nil, errors.New("token expired")
			}

			zLog.Info("Token validated successfully (TokenClaims format)", zap.Int64("account_id", claims.AccountID))
			return claims, nil
		}
	}

	return nil, errors.New("invalid token")
}

// GenerateToken 生成Token（用于测试）
func (tm *TokenManager) GenerateToken(accountID int64, accountName string) (string, error) {
	claims := &TokenClaims{
		AccountID:   accountID,
		AccountName: accountName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tm.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(tm.secretKey))
	if err != nil {
		zLog.Error("Failed to generate token", zap.Error(err))
		return "", err
	}

	return tokenString, nil
}
