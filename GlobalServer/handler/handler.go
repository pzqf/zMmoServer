package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/db"
	"github.com/pzqf/zMmoShared/db/models"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
)

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
	tokenString, err := token.SignedString([]byte("zMmoServerSecretKey"))
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
			Success:  false,
			ErrorMsg: "Invalid request format",
		})
	}

	zLog.Info("Received account create request", zap.String("account", req.Account))

	// Validate request
	if req.Account == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, protocol.AccountCreateResponse{
			Success:  false,
			ErrorMsg: "账号或密码不能为空",
		})
	}

	// Get DB manager
	dbMgr := db.GetMgr()
	if dbMgr == nil {
		zLog.Error("Failed to get DB manager")
		return c.JSON(http.StatusInternalServerError, protocol.AccountCreateResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}

	// Check if account exists
	account, err := dbMgr.AccountRepository.GetByName(req.Account)
	if err != nil {
		zLog.Error("Failed to check account existence", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, protocol.AccountCreateResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}

	if account != nil {
		return c.JSON(http.StatusConflict, protocol.AccountCreateResponse{
			Success:  false,
			ErrorMsg: "账号已存在",
		})
	}

	// Generate account ID and create account
	accountID := time.Now().UnixNano() / 1000000
	now := time.Now()

	newAccount := &models.Account{
		AccountID:   accountID,
		AccountName: req.Account,
		Password:    req.Password,
		Status:      1,
		CreatedAt:   now,
		LastLoginAt: now,
	}

	// Save to database
	id, err := db.GetMgr().AccountRepository.Create(newAccount)
	if err != nil || id <= 0 {
		zLog.Error("Failed to create account", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, protocol.AccountCreateResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}

	return c.JSON(http.StatusOK, protocol.AccountCreateResponse{
		Success: true,
	})
}

// HandleAccountLogin handles account login requests
func HandleAccountLogin(c echo.Context) error {
	var req protocol.AccountLoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, protocol.AccountLoginResponse{
			Success:  false,
			ErrorMsg: "Invalid request format",
		})
	}

	zLog.Info("Received account login request", zap.String("account", req.Account))

	// Validate request
	if req.Account == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, protocol.AccountLoginResponse{
			Success:  false,
			ErrorMsg: "账号或密码不能为空",
		})
	}

	// Get DB manager
	dbMgr := db.GetMgr()
	if dbMgr == nil {
		zLog.Error("Failed to get DB manager")
		return c.JSON(http.StatusInternalServerError, protocol.AccountLoginResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}

	// Get account
	account, err := dbMgr.AccountRepository.GetByName(req.Account)
	if err != nil {
		zLog.Error("Failed to get account", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, protocol.AccountLoginResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}

	if account == nil {
		zLog.Info("Account not found", zap.String("account", req.Account))
		return c.JSON(http.StatusUnauthorized, protocol.AccountLoginResponse{
			Success:  false,
			ErrorMsg: "账号不存在",
		})
	}

	zLog.Info("Account found", zap.Int64("account_id", account.AccountID), zap.String("account_name", account.AccountName), zap.String("password", account.Password))
	zLog.Info("Login request password", zap.String("password", req.Password))

	if account.Password != req.Password {
		zLog.Info("Password mismatch", zap.String("account", req.Account))
		return c.JSON(http.StatusUnauthorized, protocol.AccountLoginResponse{
			Success:  false,
			ErrorMsg: "账号或密码错误",
		})
	}

	zLog.Info("Password matched", zap.String("account", req.Account))

	// Update last login time
	zLog.Info("Updating last login time...")
	account.LastLoginAt = time.Now()
	_, err = dbMgr.AccountRepository.Update(account)
	if err != nil {
		zLog.Error("Failed to update last login time", zap.Error(err))
	} else {
		zLog.Info("Last login time updated successfully")
	}

	// Get server list
	zLog.Info("Getting server list...")
	servers, err := dbMgr.GameServerRepository.GetAll()
	var serverInfos []*protocol.ServerInfo
	if err != nil {
		zLog.Error("Failed to get server list", zap.Error(err))
		// 使用默认服务器
		serverInfos = []*protocol.ServerInfo{
			{
				ServerId:       1,
				ServerName:     "测试服务器",
				ServerType:     "game",
				GroupId:        1,
				Address:        "127.0.0.1",
				Port:           8081,
				Status:         1,
				OnlineCount:    0,
				MaxOnlineCount: 1000,
				Region:         "cn",
				Version:        "1.0.0",
			},
		}
	} else {
		// Prepare server info
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
		// 如果没有服务器，使用默认服务器
		if len(serverInfos) == 0 {
			serverInfos = []*protocol.ServerInfo{
				{
					ServerId:       1,
					ServerName:     "测试服务器",
					ServerType:     "game",
					GroupId:        1,
					Address:        "127.0.0.1",
					Port:           8081,
					Status:         1,
					OnlineCount:    0,
					MaxOnlineCount: 1000,
					Region:         "cn",
					Version:        "1.0.0",
				},
			}
		}
	}
	zLog.Info("Server list retrieved successfully", zap.Int("server_count", len(serverInfos)))

	// Generate token
	zLog.Info("Generating token...", zap.Int64("account_id", account.AccountID), zap.String("account_name", account.AccountName))
	token, err := generateToken(account.AccountID, account.AccountName)
	if err != nil {
		zLog.Error("Failed to generate token", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, protocol.AccountLoginResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}
	zLog.Info("Token generated successfully", zap.String("token", token))

	zLog.Info("Returning login response", zap.Bool("success", true), zap.Int("server_count", len(serverInfos)), zap.String("token", token))
	return c.JSON(http.StatusOK, protocol.AccountLoginResponse{
		Success: true,
		Servers: serverInfos,
		Token:   token,
	})
}

// HandleGetServerList handles server list requests
func HandleGetServerList(c echo.Context) error {
	zLog.Info("Received get server list request")

	// Get DB manager
	dbMgr := db.GetMgr()
	if dbMgr == nil {
		zLog.Error("Failed to get DB manager")
		// 返回默认服务器
		return c.JSON(http.StatusOK, protocol.ServerListResponse{
			Success: true,
			Servers: []*protocol.ServerInfo{
				{
					ServerId:       1,
					ServerName:     "测试服务器",
					ServerType:     "game",
					GroupId:        1,
					Address:        "127.0.0.1",
					Port:           8081,
					Status:         1,
					OnlineCount:    0,
					MaxOnlineCount: 1000,
					Region:         "cn",
					Version:        "1.0.0",
				},
			},
		})
	}

	// Get all game servers
	servers, err := dbMgr.GameServerRepository.GetAll()
	if err != nil {
		zLog.Error("Failed to get game server list", zap.Error(err))
		// 返回默认服务器
		return c.JSON(http.StatusOK, protocol.ServerListResponse{
			Success: true,
			Servers: []*protocol.ServerInfo{
				{
					ServerId:       1,
					ServerName:     "测试服务器",
					ServerType:     "game",
					GroupId:        1,
					Address:        "127.0.0.1",
					Port:           8081,
					Status:         1,
					OnlineCount:    0,
					MaxOnlineCount: 1000,
					Region:         "cn",
					Version:        "1.0.0",
				},
			},
		})
	}

	// Prepare server info
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

	// 如果没有服务器，返回默认服务器
	if len(serverInfos) == 0 {
		return c.JSON(http.StatusOK, protocol.ServerListResponse{
			Success: true,
			Servers: []*protocol.ServerInfo{
				{
					ServerId:       1,
					ServerName:     "测试服务器",
					ServerType:     "game",
					GroupId:        1,
					Address:        "127.0.0.1",
					Port:           8081,
					Status:         1,
					OnlineCount:    0,
					MaxOnlineCount: 1000,
					Region:         "cn",
					Version:        "1.0.0",
				},
			},
		})
	}

	return c.JSON(http.StatusOK, protocol.ServerListResponse{
		Success: true,
		Servers: serverInfos,
	})
}

// HandleGetServerListByGroup handles server list by group requests
func HandleGetServerListByGroup(c echo.Context) error {
	groupIDStr := c.Param("groupId")
	zLog.Info("Received get server list by group request", zap.String("groupId", groupIDStr))

	var groupID int32
	if _, err := fmt.Sscanf(groupIDStr, "%d", &groupID); err != nil {
		return c.JSON(http.StatusBadRequest, protocol.ServerListResponse{
			Success:  false,
			ErrorMsg: "Invalid group ID",
		})
	}

	dbMgr := db.GetMgr()
	if dbMgr == nil {
		zLog.Error("Failed to get DB manager")
		return c.JSON(http.StatusInternalServerError, protocol.ServerListResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}

	servers, err := dbMgr.GameServerRepository.GetByGroupID(groupID)
	if err != nil {
		zLog.Error("Failed to get game server list by group", zap.Error(err), zap.Int32("groupId", groupID))
		return c.JSON(http.StatusInternalServerError, protocol.ServerListResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}

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
		Success: true,
		Servers: serverInfos,
	})
}

// HandleServerRegister handles server register requests
func HandleServerRegister(c echo.Context) error {
	var req protocol.ServerRegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, protocol.ServerRegisterResponse{
			Success:  false,
			ErrorMsg: "Invalid request format",
		})
	}

	zLog.Info("Received server register request", zap.String("serverName", req.ServerName))

	// Get DB manager
	dbMgr := db.GetMgr()
	if dbMgr == nil {
		zLog.Error("Failed to get DB manager")
		return c.JSON(http.StatusInternalServerError, protocol.ServerRegisterResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}

	// Create game server
	gameServer := &models.GameServer{
		ServerID:       req.ServerId,
		ServerName:     req.ServerName,
		ServerType:     req.ServerType,
		GroupID:        req.GroupId,
		Address:        req.Address,
		Port:           req.Port,
		Status:         1, // Online
		OnlineCount:    0,
		MaxOnlineCount: req.MaxOnlineCount,
		Region:         req.Region,
		Version:        req.Version,
		LastHeartbeat:  time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Save to database
	id, err := dbMgr.GameServerRepository.Create(gameServer)
	if err != nil || id <= 0 {
		zLog.Error("Failed to register game server", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, protocol.ServerRegisterResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}

	return c.JSON(http.StatusOK, protocol.ServerRegisterResponse{
		Success:  true,
		ServerId: id,
	})
}

// HandleServerHeartbeat handles server heartbeat requests
func HandleServerHeartbeat(c echo.Context) error {
	var req protocol.ServerHeartbeatRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, protocol.ServerHeartbeatResponse{
			Success:  false,
			ErrorMsg: "Invalid request format",
		})
	}

	zLog.Info("Received server heartbeat request", zap.Int32("serverId", req.ServerId))

	// Get DB manager
	dbMgr := db.GetMgr()
	if dbMgr == nil {
		zLog.Error("Failed to get DB manager")
		return c.JSON(http.StatusInternalServerError, protocol.ServerHeartbeatResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}

	// Get game server
	server, err := dbMgr.GameServerRepository.GetByID(req.ServerId)
	if err != nil || server == nil {
		zLog.Error("Failed to get game server", zap.Error(err), zap.Int32("serverId", req.ServerId))
		return c.JSON(http.StatusNotFound, protocol.ServerHeartbeatResponse{
			Success:  false,
			ErrorMsg: "服务器不存在",
		})
	}

	// Update server info
	server.OnlineCount = req.OnlineCount
	if req.Status > 0 {
		server.Status = req.Status
	}
	server.LastHeartbeat = time.Now()
	server.UpdatedAt = time.Now()

	// Save to database
	_, err = dbMgr.GameServerRepository.Update(server)
	if err != nil {
		zLog.Error("Failed to update game server", zap.Error(err), zap.Int32("serverId", req.ServerId))
		return c.JSON(http.StatusInternalServerError, protocol.ServerHeartbeatResponse{
			Success:  false,
			ErrorMsg: "服务器错误",
		})
	}

	return c.JSON(http.StatusOK, protocol.ServerHeartbeatResponse{
		Success: true,
	})
}
