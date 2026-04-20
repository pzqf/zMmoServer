package player

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	playerservice "github.com/pzqf/zMmoServer/GameServer/services"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"go.uber.org/zap"
)

type LoginService struct {
	playerManager  *PlayerManager
	playerService  *playerservice.PlayerService
	sessionManager *session.SessionManager
}

func NewLoginService(playerManager *PlayerManager, playerService *playerservice.PlayerService, sessionManager *session.SessionManager) *LoginService {
	return &LoginService{
		playerManager:  playerManager,
		playerService:  playerService,
		sessionManager: sessionManager,
	}
}

// EnterGame 玩家进入游戏完整流程
func (ls *LoginService) EnterGame(sessionID string, playerID id.PlayerIdType) error {
	sess, exists := ls.sessionManager.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if ls.playerManager.HasPlayer(playerID) {
		existingPlayer, _ := ls.playerManager.GetPlayer(playerID)
		if existingPlayer != nil {
			ls.sessionManager.BindPlayer(sessionID, playerID)
			zLog.Info("Player already online, rebind session",
				zap.Int64("player_id", int64(playerID)),
				zap.String("session_id", sessionID))
			return nil
		}
	}

	info, err := ls.playerService.GetPlayerByID(playerID)
	if err != nil {
		return fmt.Errorf("get player info failed: %w", err)
	}
	if info == nil {
		return fmt.Errorf("player not found: %d", playerID)
	}

	accountID := sess.AccountID
	if accountID == 0 {
		accountID = id.AccountIdType(playerID)
	}

	p := NewPlayer(playerID, accountID, info.PlayerName)
	if ls.playerManager.mapOp != nil {
		p.SetMapOperator(ls.playerManager.mapOp)
	}
	if ls.playerManager.clientSender != nil {
		p.SetSessionInfo(sessionID, ls.playerManager.clientSender)
	}

	if err := ls.playerManager.AddPlayer(p); err != nil {
		return fmt.Errorf("add player to manager failed: %w", err)
	}

	ls.sessionManager.BindPlayer(sessionID, playerID)

	if err := ls.playerService.PlayerLogin(playerID); err != nil {
		zLog.Warn("Failed to update player login time", zap.Error(err))
	}

	ls.playerService.AddToOnline(playerID, sessionID, info)

	zLog.Info("Player entered game successfully",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("account_id", int64(accountID)),
		zap.String("session_id", sessionID))

	return nil
}

// LeaveGame 玩家离开游戏完整流程
func (ls *LoginService) LeaveGame(playerID id.PlayerIdType) error {
	if !ls.playerManager.HasPlayer(playerID) {
		zLog.Warn("Player not online, skip leave", zap.Int64("player_id", int64(playerID)))
		return nil
	}

	p, err := ls.playerManager.GetPlayer(playerID)
	if err != nil {
		zLog.Warn("Player not found in manager", zap.Int64("player_id", int64(playerID)))
		return nil
	}

	if p.GetCurrentMapID() != 0 && ls.playerManager.mapOp != nil {
		if err := ls.playerManager.mapOp.LeaveMap(playerID, p.GetCurrentMapID()); err != nil {
			zLog.Warn("Failed to leave map during logout",
				zap.Int64("player_id", int64(playerID)),
				zap.Error(err))
		}
	}

	ls.syncPlayerDataToService(p)

	sess, exists := ls.sessionManager.GetSessionByPlayer(playerID)
	if exists {
		ls.sessionManager.UpdateSessionStatus(sess.SessionID, session.SessionStatusDisconnected)
	}

	if err := ls.playerService.PlayerLogout(playerID); err != nil {
		zLog.Warn("Failed to logout player from service", zap.Error(err))
	}

	if err := ls.playerManager.RemovePlayer(playerID); err != nil {
		zLog.Error("Failed to remove player from manager", zap.Error(err))
		return err
	}

	zLog.Info("Player left game successfully", zap.Int64("player_id", int64(playerID)))
	return nil
}

// syncPlayerDataToService 将 PlayerActor 中的数据同步到 PlayerService 的 OnlinePlayer
func (ls *LoginService) syncPlayerDataToService(p *Player) {
	playerID := p.GetPlayerID()
	attrs := p.GetAttrs()
	if attrs == nil {
		return
	}

	ls.playerService.SyncOnlinePlayerData(
		playerID,
		int(attrs.GetLevel()),
		attrs.GetExp(),
		attrs.GetGold(),
		attrs.GetDiamond(),
	)
}

// GetPlayerList 获取账号下的角色列表
func (ls *LoginService) GetPlayerList(accountID id.AccountIdType) ([]playerservice.PlayerInfo, error) {
	return ls.playerService.GetPlayerList(accountID)
}

// CreatePlayer 创建角色
func (ls *LoginService) CreatePlayer(accountID id.AccountIdType, name string, sex int32, age int32) (id.PlayerIdType, error) {
	return ls.playerService.CreatePlayer(accountID, name, sex, age)
}

// GenerateReconnectToken 生成断线重连 Token
func (ls *LoginService) GenerateReconnectToken(playerID id.PlayerIdType) (string, error) {
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate reconnect token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	sess, exists := ls.sessionManager.GetSessionByPlayer(playerID)
	if !exists {
		return "", fmt.Errorf("session not found for player: %d", playerID)
	}

	sess.ReconnectToken = token
	sess.ReconnectExpire = time.Now().Add(5 * time.Minute).Unix()

	zLog.Info("Reconnect token generated",
		zap.Int64("player_id", int64(playerID)),
		zap.String("token", token))

	return token, nil
}

// Reconnect 断线重连
func (ls *LoginService) Reconnect(newSessionID string, playerID id.PlayerIdType, reconnectToken string) error {
	oldSess, exists := ls.sessionManager.GetSessionByPlayer(playerID)
	if !exists {
		return fmt.Errorf("player session not found: %d", playerID)
	}

	if oldSess.ReconnectToken != reconnectToken {
		return fmt.Errorf("invalid reconnect token")
	}

	if time.Now().Unix() > oldSess.ReconnectExpire {
		return fmt.Errorf("reconnect token expired")
	}

	if !ls.playerManager.HasPlayer(playerID) {
		return fmt.Errorf("player not online: %d", playerID)
	}

	ls.sessionManager.BindPlayer(newSessionID, playerID)
	ls.sessionManager.UpdateSessionStatus(newSessionID, session.SessionStatusInGame)

	p, err := ls.playerManager.GetPlayer(playerID)
	if err != nil {
		return fmt.Errorf("get player failed: %w", err)
	}

	if err := ls.playerManager.ResumePlayer(playerID); err != nil {
		zLog.Warn("Failed to resume player lifecycle", zap.Error(err))
	}

	zLog.Info("Player reconnected successfully",
		zap.Int64("player_id", int64(playerID)),
		zap.String("new_session_id", newSessionID),
		zap.Int32("current_map", int32(p.GetCurrentMapID())))

	return nil
}

// OnDisconnect 玩家断线处理（不立即登出，保留 Actor 等待重连）
func (ls *LoginService) OnDisconnect(playerID id.PlayerIdType) error {
	if !ls.playerManager.HasPlayer(playerID) {
		return nil
	}

	if err := ls.playerManager.SuspendPlayer(playerID); err != nil {
		zLog.Warn("Failed to suspend player lifecycle", zap.Error(err))
	}

	_, _ = ls.GenerateReconnectToken(playerID)

	zLog.Info("Player disconnected, waiting for reconnect",
		zap.Int64("player_id", int64(playerID)))

	return nil
}
