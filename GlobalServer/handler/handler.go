package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/db"
	"github.com/pzqf/zCommon/db/models"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GlobalServer/gameserverlist"
	"github.com/pzqf/zMmoServer/GlobalServer/metrics"
	"github.com/pzqf/zUtil/zCrypto"
	"go.uber.org/zap"
)

type TokenClaims struct {
	AccountID   int64  `json:"account_id"`
	AccountName string `json:"account_name"`
	jwt.RegisteredClaims
}

var jwtSecret string

// InitJWTSecret 初始化JWT密钥
func InitJWTSecret(secret string) {
	jwtSecret = secret
}

// generateToken 生成JWT token
func generateToken(accountID int64, accountName string) (string, error) {
	claims := TokenClaims{
		AccountID:   accountID,
		AccountName: accountName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// getMetricsFromContext 从Echo上下文中获取 metrics 实例
func getMetricsFromContext(c echo.Context) *metrics.Metrics {
	if m, ok := c.Get("metrics").(*metrics.Metrics); ok {
		return m
	}
	return nil
}

// HandleAccountCreate handles account creation requests
func HandleAccountCreate(c echo.Context) error {
	start := time.Now()
	var req protocol.AccountCreateRequest
	if err := c.Bind(&req); err != nil {
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusBadRequest, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "Invalid request format",
		})
	}

	// Validate request
	if req.Account == "" || req.Password == "" {
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusBadRequest, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "账号或密码不能为空",
		})
	}

	// Get DB manager
	dbMgr := db.GetMgr()
	if dbMgr == nil {
		zLog.Error("Failed to get DB manager")
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusInternalServerError, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	// Check if account exists
	account, err := dbMgr.AccountRepository.GetByName(req.Account)
	if err != nil {
		zLog.Error("Failed to check account existence", zap.Error(err))
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusInternalServerError, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	if account != nil {
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusConflict, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_ACCOUNT_ALREADY_EXISTS),
			ErrorMsg: "账号已存在",
		})
	}

	// Generate account ID using Snowflake
	accountID, err := id.GenerateAccountID()
	if err != nil {
		zLog.Error("Failed to generate account ID", zap.Error(err))
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusInternalServerError, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	now := time.Now()

	// Hash password
	hashedPassword := zCrypto.SHA256(req.Password)

	newAccount := &models.Account{
		AccountID:   int64(accountID),
		AccountName: req.Account,
		Password:    hashedPassword,
		Status:      1,
		CreatedAt:   now,
		LastLoginAt: now,
	}

	// Save to database
	createdID, err := db.GetMgr().AccountRepository.Create(newAccount)
	if err != nil || createdID <= 0 {
		zLog.Error("Failed to create account", zap.Error(err))
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusInternalServerError, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	// 记录账号注册指标
	if m := getMetricsFromContext(c); m != nil {
		m.IncrementAccountRegistrations()
		m.RecordAccountOperationTime(time.Since(start))
	}

	return c.JSON(http.StatusOK, protocol.AccountCreateResponse{
		Result:    int32(protocol.ErrorCode_ERR_SUCCESS),
		AccountId: int64(accountID),
	})
}

// HandleAccountLogin handles account login requests
func HandleAccountLogin(c echo.Context) error {
	start := time.Now()
	var req protocol.AccountLoginRequest
	if err := c.Bind(&req); err != nil {
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusBadRequest, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "Invalid request format",
		})
	}

	// Validate request
	if req.Account == "" || req.Password == "" {
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusBadRequest, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "账号或密码不能为空",
		})
	}

	// Get DB manager
	dbMgr := db.GetMgr()
	if dbMgr == nil {
		zLog.Error("Failed to get DB manager")
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusInternalServerError, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	// Get account
	account, err := dbMgr.AccountRepository.GetByName(req.Account)
	if err != nil {
		zLog.Error("Failed to get account", zap.Error(err))
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusInternalServerError, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	if account == nil {
		zLog.Info("Account not found", zap.String("account", req.Account))
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusUnauthorized, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_ACCOUNT_NOT_FOUND),
			ErrorMsg: "账号不存在",
		})
	}

	// Hash input password for comparison
	hashedPassword := zCrypto.SHA256(req.Password)

	if account.Password != hashedPassword {
		zLog.Info("Password mismatch", zap.String("account", req.Account))
		// 记录登录失败指标
		if m := getMetricsFromContext(c); m != nil {
			m.IncrementAccountLoginFailures()
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusUnauthorized, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_ACCOUNT_PASSWORD_WRONG),
			ErrorMsg: "账号或密码错误",
		})
	}

	// Update last login time
	account.LastLoginAt = time.Now()
	_, err = dbMgr.AccountRepository.Update(account)
	if err != nil {
		zLog.Error("Failed to update last login time", zap.Error(err))
	}

	// Get server list from cache (合并MySQL静态数据+Redis动态数据)
	manager := gameserverlist.GetServerListManager()
	var serverInfos []*protocol.ServerInfo
	if manager != nil {
		servers := manager.GetOnlineServers()
		for _, s := range servers {
			serverInfos = append(serverInfos, &protocol.ServerInfo{
				ServerId:       s.ServerID,
				ServerName:     s.ServerName,
				ServerType:     s.ServerType,
				GroupId:        s.GroupID,
				Address:        s.Address,
				Port:           s.Port,
				Status:         s.Status,
				OnlineCount:    s.OnlineCount,
				MaxOnlineCount: s.MaxOnlineCount,
				Region:         s.Region,
				Version:        s.Version,
			})
		}
	}

	// Generate token
	token, err := generateToken(account.AccountID, account.AccountName)
	if err != nil {
		zLog.Error("Failed to generate token", zap.Error(err))
		if m := getMetricsFromContext(c); m != nil {
			m.RecordAccountOperationTime(time.Since(start))
		}
		return c.JSON(http.StatusInternalServerError, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	// 记录登录成功指标
	if m := getMetricsFromContext(c); m != nil {
		m.IncrementAccountLogins()
		m.RecordAccountOperationTime(time.Since(start))
	}

	return c.JSON(http.StatusOK, protocol.AccountLoginResponse{
		Result:    int32(protocol.ErrorCode_ERR_SUCCESS),
		Servers:   serverInfos,
		Token:     token,
		AccountId: account.AccountID,
	})
}

// HandleGetServerList handles server list requests
func HandleGetServerList(c echo.Context) error {
	start := time.Now()
	// Get server list from cache
	manager := gameserverlist.GetServerListManager()
	if manager == nil {
		zLog.Error("Failed to get server list manager")
		if m := getMetricsFromContext(c); m != nil {
			m.RecordServerListResponseTime(time.Since(start))
		}
		return c.JSON(http.StatusOK, protocol.ServerListResponse{
			Result:  int32(protocol.ErrorCode_ERR_SUCCESS),
			Servers: []*protocol.ServerInfo{},
		})
	}

	// Get all server infos (合并静态+动态数据)
	servers := manager.GetAllServerFullInfos()

	// Prepare server info
	serverInfos := make([]*protocol.ServerInfo, 0)
	for _, s := range servers {
		serverInfos = append(serverInfos, &protocol.ServerInfo{
			ServerId:       s.ServerID,
			ServerName:     s.ServerName,
			ServerType:     s.ServerType,
			GroupId:        s.GroupID,
			Address:        s.Address,
			Port:           s.Port,
			Status:         s.Status,
			OnlineCount:    s.OnlineCount,
			MaxOnlineCount: s.MaxOnlineCount,
			Region:         s.Region,
			Version:        s.Version,
		})
	}

	// 记录服务器列表请求指标
	if m := getMetricsFromContext(c); m != nil {
		m.IncrementServerListRequests()
		m.SetGameServersCount(len(serverInfos))
		m.RecordServerListResponseTime(time.Since(start))
	}

	return c.JSON(http.StatusOK, protocol.ServerListResponse{
		Result:  int32(protocol.ErrorCode_ERR_SUCCESS),
		Servers: serverInfos,
	})
}

// HandleGetServerListByGroup handles server list by group requests
func HandleGetServerListByGroup(c echo.Context) error {
	start := time.Now()
	groupIDStr := c.Param("groupId")
	var groupID int32
	if _, err := fmt.Sscanf(groupIDStr, "%d", &groupID); err != nil {
		if m := getMetricsFromContext(c); m != nil {
			m.RecordServerListResponseTime(time.Since(start))
		}
		return c.JSON(http.StatusBadRequest, protocol.ServerListResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "Invalid group ID",
		})
	}

	// Get server list from cache by group
	manager := gameserverlist.GetServerListManager()
	if manager == nil {
		zLog.Error("Failed to get server list manager")
		if m := getMetricsFromContext(c); m != nil {
			m.RecordServerListResponseTime(time.Since(start))
		}
		return c.JSON(http.StatusInternalServerError, protocol.ServerListResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	servers := manager.GetServerFullInfosByGroup(groupID)
	var serverInfos []*protocol.ServerInfo
	for _, s := range servers {
		serverInfos = append(serverInfos, &protocol.ServerInfo{
			ServerId:       s.ServerID,
			ServerName:     s.ServerName,
			ServerType:     s.ServerType,
			GroupId:        s.GroupID,
			Address:        s.Address,
			Port:           s.Port,
			Status:         s.Status,
			OnlineCount:    s.OnlineCount,
			MaxOnlineCount: s.MaxOnlineCount,
			Region:         s.Region,
			Version:        s.Version,
		})
	}

	if m := getMetricsFromContext(c); m != nil {
		m.RecordServerListResponseTime(time.Since(start))
	}

	return c.JSON(http.StatusOK, protocol.ServerListResponse{
		Result:  int32(protocol.ErrorCode_ERR_SUCCESS),
		Servers: serverInfos,
	})
}
