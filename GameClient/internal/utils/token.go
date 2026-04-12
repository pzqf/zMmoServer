package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims JWT声明
type TokenClaims struct {
	AccountID   int64  `json:"account_id"`
	AccountName string `json:"account_name"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT token
func GenerateToken(accountID int64, accountName, secretKey string) (string, error) {
	claims := &TokenClaims{
		AccountID:   accountID,
		AccountName: accountName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
