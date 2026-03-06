package handler

import (
	"github.com/pzqf/zEngine/zLog"
	playerservice "github.com/pzqf/zMmoServer/GameServer/service"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/protocol"
	"go.uber.org/zap"
)

type PlayerHandler struct {
	sessionManager *session.SessionManager
	playerService  *playerservice.PlayerService
}

func NewPlayerHandler(sessionManager *session.SessionManager, playerService *playerservice.PlayerService) *PlayerHandler {
	return &PlayerHandler{
		sessionManager: sessionManager,
		playerService:  playerService,
	}
}

func (ph *PlayerHandler) HandlePlayerLogin(sessionID string, accountID id.AccountIdType) (*protocol.PlayerLoginResponse, error) {
	zLog.Info("Handling player login", zap.String("session_id", sessionID), zap.Int64("account_id", int64(accountID)))

	ph.sessionManager.BindAccount(sessionID, accountID)

	playerList, err := ph.playerService.GetPlayerList(accountID)
	if err != nil {
		zLog.Error("Failed to get player list", zap.Error(err))
		return &protocol.PlayerLoginResponse{
			Success:  false,
			ErrorMsg: "Failed to get player list",
		}, err
	}

	if len(playerList) == 0 {
		return &protocol.PlayerLoginResponse{
			Success:  true,
			PlayerId: 0,
		}, nil
	}

	player := playerList[0]

	zLog.Info("Player login handled", zap.Int64("account_id", int64(accountID)), zap.Int64("player_id", int64(player.PlayerID)))

	return &protocol.PlayerLoginResponse{
		Success:  true,
		PlayerId: int64(player.PlayerID),
		Name:     player.Name,
		Level:    int32(player.Level),
		Gold:     player.Gold,
	}, nil
}

func (ph *PlayerHandler) HandlePlayerSelect(sessionID string, playerID id.PlayerIdType) (*protocol.Response, error) {
	zLog.Info("Handling player select", zap.String("session_id", sessionID), zap.Int64("player_id", int64(playerID)))

	ph.sessionManager.BindPlayer(sessionID, playerID)

	err := ph.playerService.PlayerLogin(playerID)
	if err != nil {
		zLog.Error("Failed to login player", zap.Error(err))
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Failed to login player",
		}, err
	}

	player, err := ph.playerService.GetPlayerByID(playerID)
	if err != nil {
		zLog.Error("Failed to get player info", zap.Error(err))
		return &protocol.Response{
			Result:   1,
			ErrorMsg: "Failed to get player info",
		}, err
	}

	if player != nil {
		ph.playerService.AddToOnline(playerID, sessionID, player)
	}

	ph.sessionManager.UpdateSessionStatus(sessionID, session.SessionStatusInGame)

	zLog.Info("Player selected and logged in", zap.Int64("player_id", int64(playerID)))

	return &protocol.Response{
		Result: 0,
	}, nil
}

func (ph *PlayerHandler) HandlePlayerCreate(sessionID string, accountID id.AccountIdType, playerName string, sex int32, age int32) (*protocol.PlayerCreateResponse, error) {
	zLog.Info("Handling player create", zap.String("session_id", sessionID), zap.Int64("account_id", int64(accountID)), zap.String("player_name", playerName))

	playerID, err := ph.playerService.CreatePlayer(accountID, playerName, sex, age)
	if err != nil {
		zLog.Error("Failed to create player", zap.Error(err))
		return &protocol.PlayerCreateResponse{
			Success:  false,
			ErrorMsg: "Failed to create player",
		}, err
	}

	ph.sessionManager.BindPlayer(sessionID, playerID)

	player, err := ph.playerService.GetPlayerByID(playerID)
	if err != nil {
		zLog.Error("Failed to get player", zap.Error(err))
		return &protocol.PlayerCreateResponse{
			Success: true,
			Player:  nil,
		}, nil
	}

	zLog.Info("Player created successfully", zap.Int64("player_id", int64(playerID)))

	return &protocol.PlayerCreateResponse{
		Success: true,
		Player: &protocol.PlayerInfo{
			PlayerId: int64(player.PlayerID),
			Name:     player.PlayerName,
			Level:    int32(player.Level),
			Sex:      int32(player.Sex),
			Age:      int32(player.Age),
		},
	}, nil
}

func (ph *PlayerHandler) HandlePlayerLogout(sessionID string) (*protocol.Response, error) {
	zLog.Info("Handling player logout", zap.String("session_id", sessionID))

	session, exists := ph.sessionManager.GetSession(sessionID)
	if !exists {
		return &protocol.Response{
			Result: 0,
		}, nil
	}

	if session.PlayerID > 0 {
		ph.playerService.RemoveFromOnline(session.PlayerID)

		err := ph.playerService.PlayerLogout(session.PlayerID)
		if err != nil {
			zLog.Error("Failed to logout player", zap.Error(err))
		}
	}

	ph.sessionManager.RemoveSession(sessionID)

	zLog.Info("Player logged out", zap.String("session_id", sessionID))

	return &protocol.Response{
		Result: 0,
	}, nil
}
