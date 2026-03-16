package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GlobalServer/metrics"
	"github.com/pzqf/zMmoServer/GlobalServer/serverstatus"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/db"
	"github.com/pzqf/zMmoShared/db/models"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
)

var jwtSecret string

// InitJWTSecret 初始化JWT密钥
func InitJWTSecret(secret string) {
	jwtSecret = secret
}

// getMetricsFromContext 从 Echo 上下文中获取 metrics 实例
func getMetricsFromContext(c echo.Context) *metrics.Metrics {
	if m, ok := c.Get("metrics").(*metrics.Metrics); ok {
		return m
	}
	return nil
}

// 生成JWT token
func generateToken(accountID int64, accountName string) (string, error) {
	// 创建token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"account_id":   accountID,
		"account_name": accountName,
		"exp":          time.Now().Add(time.Hour * 24 * 7).Unix(), // 7天过期
		"iat":          time.Now().Unix(),
	})

	// 签名token
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// HandleAccountCreate handles account creation requests
func HandleAccountCreate(c echo.Context) error {
	var req protocol.AccountCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "Invalid request format",
		})
	}

	// Validate request
	if req.Account == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "账号或密码不能为空",
		})
	}

	// Get DB manager
	dbMgr := db.GetMgr()
	if dbMgr == nil {
		zLog.Error("Failed to get DB manager")
		return c.JSON(http.StatusInternalServerError, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	// Check if account exists
	account, err := dbMgr.AccountRepository.GetByName(req.Account)
	if err != nil {
		zLog.Error("Failed to check account existence", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	if account != nil {
		return c.JSON(http.StatusConflict, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_ACCOUNT_ALREADY_EXISTS),
			ErrorMsg: "账号已存在",
		})
	}

	// Generate account ID using Snowflake
	accountID, err := id.GenerateAccountID()
	if err != nil {
		zLog.Error("Failed to generate account ID", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	now := time.Now()

	newAccount := &models.Account{
		AccountID:   int64(accountID),
		AccountName: req.Account,
		Password:    req.Password,
		Status:      1,
		CreatedAt:   now,
		LastLoginAt: now,
	}

	// Save to database
	createdID, err := db.GetMgr().AccountRepository.Create(newAccount)
	if err != nil || createdID <= 0 {
		zLog.Error("Failed to create account", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, protocol.AccountCreateResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	// 记录账号注册指标
	if m := getMetricsFromContext(c); m != nil {
		m.IncrementAccountRegistrations()
	}

	return c.JSON(http.StatusOK, protocol.AccountCreateResponse{
		Result:    int32(protocol.ErrorCode_ERR_SUCCESS),
		AccountId: int64(accountID),
	})
}

// HandleAccountLogin handles account login requests
func HandleAccountLogin(c echo.Context) error {
	var req protocol.AccountLoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "Invalid request format",
		})
	}

	// Validate request
	if req.Account == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "账号或密码不能为空",
		})
	}

	// Get DB manager
	dbMgr := db.GetMgr()
	if dbMgr == nil {
		zLog.Error("Failed to get DB manager")
		return c.JSON(http.StatusInternalServerError, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	// Get account
	account, err := dbMgr.AccountRepository.GetByName(req.Account)
	if err != nil {
		zLog.Error("Failed to get account", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	if account == nil {
		zLog.Info("Account not found", zap.String("account", req.Account))
		return c.JSON(http.StatusUnauthorized, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_ACCOUNT_NOT_FOUND),
			ErrorMsg: "账号不存在",
		})
	}

	if account.Password != req.Password {
		zLog.Info("Password mismatch", zap.String("account", req.Account))
		// 记录登录失败指标
		if m := getMetricsFromContext(c); m != nil {
			m.IncrementAccountLoginFailures()
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
	manager := serverstatus.GetManager()
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
		return c.JSON(http.StatusInternalServerError, protocol.AccountLoginResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	// 记录登录成功指标
	if m := getMetricsFromContext(c); m != nil {
		m.IncrementAccountLogins()
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
	// Get server list from cache
	manager := serverstatus.GetManager()
	if manager == nil {
		zLog.Error("Failed to get server status manager")
		return c.JSON(http.StatusOK, protocol.ServerListResponse{
			Result:  int32(protocol.ErrorCode_ERR_SUCCESS),
			Servers: []*protocol.ServerInfo{},
		})
	}

	// Get all server infos (合并静态+动态数据)
	servers := manager.GetAllServerInfos()

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
	}

	return c.JSON(http.StatusOK, protocol.ServerListResponse{
		Result:  int32(protocol.ErrorCode_ERR_SUCCESS),
		Servers: serverInfos,
	})
}

// HandleGetServerListByGroup handles server list by group requests
func HandleGetServerListByGroup(c echo.Context) error {
	groupIDStr := c.Param("groupId")

	var groupID int32
	if _, err := fmt.Sscanf(groupIDStr, "%d", &groupID); err != nil {
		return c.JSON(http.StatusBadRequest, protocol.ServerListResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "Invalid group ID",
		})
	}

	// Get server list from cache by group
	manager := serverstatus.GetManager()
	if manager == nil {
		zLog.Error("Failed to get server status manager")
		return c.JSON(http.StatusInternalServerError, protocol.ServerListResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	servers := manager.GetServerInfosByGroup(groupID)

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

	return c.JSON(http.StatusOK, protocol.ServerListResponse{
		Result:  int32(protocol.ErrorCode_ERR_SUCCESS),
		Servers: serverInfos,
	})
}

// HandleServerRegister handles server register requests
// 注意：现在服务器注册只更新Redis动态数据，不写入MySQL
func HandleServerRegister(c echo.Context) error {
	var req protocol.ServerRegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, protocol.ServerRegisterResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "Invalid request format",
		})
	}

	// Get server cache manager
	manager := serverstatus.GetManager()
	if manager == nil {
		zLog.Error("Failed to get server status manager")
		return c.JSON(http.StatusInternalServerError, protocol.ServerRegisterResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	// 检查静态配置中是否存在该服务器
	staticServer := manager.GetServerInfo(req.ServerId)
	if staticServer == nil {
		zLog.Error("Server not found in static config", zap.Int32("serverId", req.ServerId))
		return c.JSON(http.StatusNotFound, protocol.ServerRegisterResponse{
			Result:   int32(protocol.ErrorCode_ERR_NOT_FOUND),
			ErrorMsg: "服务器未在配置中注册，请联系运维",
		})
	}

	// 更新服务器状态到Redis
	status := &serverstatus.ServerStatus{
		ServerID:      req.ServerId,
		Address:       req.Address,
		Port:          req.Port,
		Status:        1, // Online
		OnlineCount:   0,
		Version:       req.Version,
		LastHeartbeat: time.Now(),
	}

	if err := manager.UpdateServerStatus(status); err != nil {
		zLog.Error("Failed to update server status", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, protocol.ServerRegisterResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	zLog.Info("Game server registered",
		zap.Int32("serverId", req.ServerId),
		zap.String("serverName", req.ServerName),
		zap.String("address", req.Address),
		zap.Int32("port", req.Port),
	)

	return c.JSON(http.StatusOK, protocol.ServerRegisterResponse{
		Result: int32(protocol.ErrorCode_ERR_SUCCESS),
	})
}

// HandleServerHeartbeat handles server heartbeat requests
// 注意：现在心跳只更新Redis动态数据
func HandleServerHeartbeat(c echo.Context) error {
	var req protocol.ServerHeartbeatRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, protocol.ServerHeartbeatResponse{
			Result:   int32(protocol.ErrorCode_ERR_INVALID_PARAM),
			ErrorMsg: "Invalid request format",
		})
	}

	// Get server cache manager
	manager := serverstatus.GetManager()
	if manager == nil {
		zLog.Error("Failed to get server status manager")
		return c.JSON(http.StatusInternalServerError, protocol.ServerHeartbeatResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	// 获取当前服务器状态
	currentStatus, err := manager.GetServerStatus(req.ServerId)
	if err != nil {
		zLog.Error("Failed to get server status", zap.Error(err), zap.Int32("serverId", req.ServerId))
		return c.JSON(http.StatusInternalServerError, protocol.ServerHeartbeatResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	if currentStatus == nil {
		zLog.Error("Server not registered", zap.Int32("serverId", req.ServerId))
		return c.JSON(http.StatusNotFound, protocol.ServerHeartbeatResponse{
			Result:   int32(protocol.ErrorCode_ERR_NOT_FOUND),
			ErrorMsg: "服务器未注册",
		})
	}

	// 更新服务器状态
	currentStatus.OnlineCount = req.OnlineCount
	if req.Status > 0 {
		currentStatus.Status = req.Status
	}
	if req.Version != "" {
		currentStatus.Version = req.Version
	}
	currentStatus.LastHeartbeat = time.Now()

	if err := manager.UpdateServerStatus(currentStatus); err != nil {
		zLog.Error("Failed to update server status", zap.Error(err), zap.Int32("serverId", req.ServerId))
		return c.JSON(http.StatusInternalServerError, protocol.ServerHeartbeatResponse{
			Result:   int32(protocol.ErrorCode_ERR_UNKNOWN),
			ErrorMsg: "服务器错误",
		})
	}

	return c.JSON(http.StatusOK, protocol.ServerHeartbeatResponse{
		Result: int32(protocol.ErrorCode_ERR_SUCCESS),
	})
}
