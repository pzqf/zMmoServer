package player

import (
	"fmt"

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
	ls := &LoginService{
		playerManager:  playerManager,
		playerService:  playerService,
		sessionManager: sessionManager,
	}
	// 注册"周期存盘前 actor→online 同步"回调（F-5）：让 saveLoop 落库实时进度而非登录快照，
	// 崩溃至多丢一个存盘周期（30s）的增益，而非整场会话。
	playerService.SetSyncProvider(ls.syncAllOnline)
	return ls
}

// syncAllOnline 把所有在线玩家 actor 的最新进度推进 PlayerService 的 online 缓存（供周期存盘前调用）。
// 只读 actor 的原子属性 + 写 online 缓存（各自加锁），与登出路径的 syncPlayerDataToService 同源、安全。
func (ls *LoginService) syncAllOnline() {
	for _, p := range ls.playerManager.GetAllPlayers() {
		ls.syncPlayerDataToService(p)
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
	// 从 DB 载入玩家属性到 actor。此前遗漏——actor 的等级/经验/金币/钻石恒为 0，与 DB 真值不符；
	// 更严重的是登出时 syncPlayerDataToService 会把 actor 的 0 写回 online 缓存、再由 savePlayer
	// 落库，等于每次登出把玩家金币清零（数据丢失）。载入后读写与持久化才一致。
	if a := p.GetAttrs(); a != nil {
		a.SetLevel(int32(info.Level))
		a.SetExp(info.Exp)
		a.SetGold(info.Gold)
		a.SetDiamond(info.Diamond)
		// VipLevel 与金币同型：漏载入则 actor 恒 0，一旦将来有 actor 侧 vip 写者，登出 sync 会把 0
		// 覆盖回缓存再落库（与曾发生的金币清零同因）。此处补齐，防潜伏零值覆盖（F-10）。
		a.SetVipLevel(int32(info.VipLevel))
	}
	// 从 DB 载入背包/技能到 actor（在 AddPlayer 前，actor 尚未处理消息 → 无并发）。
	loadPlayerAssets(ls.playerService, p, playerID)

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
	// 把 actor 的背包/技能写回 DB（登出存盘）。
	savePlayerAssets(ls.playerService, p, playerID)
	// 统一清理该玩家的跨玩家会话（交易/组队等注册的下线钩子），避免悬挂→重登被锁（T3/M1）。
	ls.playerManager.RunOfflineHooks(playerID)

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

// PersistAllOnline 关服时把所有在线玩家落库（属性 sync→缓存 + 背包/技能/仓库存盘）+ 清理跨玩家会话。
// 由 OnBeforeStop 在停网络前调用（此时玩家 actor 仍在）；players 表由随后的 PlayerService 存盘写入。
// 弥补 F-3：此前关服不 sync actor、不存背包/技能/仓库 → 优雅重启丢在线玩家资产与会话增益。
func (ls *LoginService) PersistAllOnline() {
	players := ls.playerManager.GetAllPlayers()
	for _, p := range players {
		pid := p.GetPlayerID()
		ls.syncPlayerDataToService(p)
		savePlayerAssets(ls.playerService, p, pid)
		ls.playerManager.RunOfflineHooks(pid)
	}
	ls.playerService.FlushAllOnline() // 把 sync 后的 online 缓存写回 players 表（属性/金币）
	zLog.Info("Persisted all online players on shutdown", zap.Int("count", len(players)))
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

// 注：玩家级断线重连（Reconnect/GenerateReconnectToken/OnDisconnect + 会话 ReconnectToken/Expire +
// PlayerManager.Suspend/ResumePlayer）曾作为脚手架存在，但全仓 0 调用者、从未接任何客户端消息，且
// 「挂起等重连」缺回收器会永久泄漏 actor。非正常掉线现由网关 OnClose→补发 LEAVE_GAME→完整登出收口
// （见 F-4），故该套死代码已整体删除，消除「接线即爆」隐患。将来若要做真重连，重新按需实现即可。
