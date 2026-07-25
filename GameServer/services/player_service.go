package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
	"go.uber.org/zap"
)

type PlayerInfo struct {
	PlayerID   id.PlayerIdType `json:"player_id"`
	Name       string          `json:"name"`
	PlayerName string          `json:"player_name"`
	Level      int             `json:"level"`
	Exp        int64           `json:"exp"`
	Gold       int64           `json:"gold"`
	Diamond    int64           `json:"diamond"`
	Sex        int             `json:"sex"`
	Age        int             `json:"age"`
	VipLevel   int             `json:"vip_level"`
	CreateTime int64           `json:"create_time"`
}

type OnlinePlayer struct {
	PlayerInfo
	SessionID    string
	MapID        int32
	X, Y         float32
	Status       int
	LastSaveTime time.Time
	mu           sync.RWMutex
}

type PlayerService struct {
	playerDAO     *dao.PlayerDAO
	connector     connector.DBConnector // 背包/技能持久化直接走原生 SQL（对齐真实表结构，可为 nil→降级不持久）
	snowflake     *id.Snowflake
	onlinePlayers map[id.PlayerIdType]*OnlinePlayer
	onlineMu      sync.RWMutex
	saveInterval  time.Duration
	isRunning     bool
	saveChan      chan id.PlayerIdType
	stopChan      chan struct{}
}

func NewPlayerService(playerDAO *dao.PlayerDAO, dbConnector connector.DBConnector) *PlayerService {
	snowflake, err := id.NewSnowflake(1, 1)
	if err != nil {
		zLog.Error("Failed to create snowflake", zap.Error(err))
		snowflake = nil
	}

	ps := &PlayerService{
		playerDAO:     playerDAO,
		connector:     dbConnector,
		snowflake:     snowflake,
		onlinePlayers: make(map[id.PlayerIdType]*OnlinePlayer),
		saveInterval:  30 * time.Second,
		saveChan:      make(chan id.PlayerIdType, 1000),
		stopChan:      make(chan struct{}),
	}

	ps.ensureAuxSchema()
	go ps.saveLoop()

	return ps
}

// —— 背包 / 技能持久化（业务层建设 2026-07-25）——
// 直接对齐**真实表结构**走原生 SQL（现有 dao.PlayerItemDAO/PlayerSkillDAO 的列名与线上表不符、
// 且模型被别处复用不便改，故不用之）。真实列：
//   player_items (id auto, player_id, item_id, count, position, is_equipped, created_at, updated_at)
//   player_skills(id auto, player_id, skill_id, level, exp, is_active, created_at, updated_at)
// 存盘用「先清后插」使 DB 与内存一致；id 自增，item_id/skill_id 为逻辑ID无唯一约束，跨玩家不冲突。

// PersistItem 一条背包持久化记录（item_id=物品配置ID, slot=背包槽）。
type PersistItem struct {
	ConfigID int32
	Count    int32
	Slot     int32
}

// PersistSkill 一条技能持久化记录。
type PersistSkill struct {
	SkillID int32
	Level   int32
	Exp     int64
}

func (ps *PlayerService) LoadPlayerItems(playerID int64) ([]PersistItem, error) {
	if ps.connector == nil {
		return nil, nil
	}
	rows, err := ps.connector.QuerySync("SELECT item_id, count, position FROM player_items WHERE player_id = ?", playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PersistItem
	for rows.Next() {
		var it PersistItem
		if err := rows.Scan(&it.ConfigID, &it.Count, &it.Slot); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (ps *PlayerService) SavePlayerItems(playerID int64, items []PersistItem) error {
	if ps.connector == nil {
		return nil
	}
	if _, err := ps.connector.ExecSync("DELETE FROM player_items WHERE player_id = ?", playerID); err != nil {
		return err
	}
	now := time.Now()
	for _, it := range items {
		if _, err := ps.connector.ExecSync(
			"INSERT INTO player_items (player_id, item_id, count, position, is_equipped, created_at, updated_at) VALUES (?, ?, ?, ?, 0, ?, ?)",
			playerID, it.ConfigID, it.Count, it.Slot, now, now); err != nil {
			return err
		}
	}
	return nil
}

func (ps *PlayerService) LoadPlayerSkills(playerID int64) ([]PersistSkill, error) {
	if ps.connector == nil {
		return nil, nil
	}
	rows, err := ps.connector.QuerySync("SELECT skill_id, level, exp FROM player_skills WHERE player_id = ?", playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PersistSkill
	for rows.Next() {
		var sk PersistSkill
		if err := rows.Scan(&sk.SkillID, &sk.Level, &sk.Exp); err != nil {
			return nil, err
		}
		out = append(out, sk)
	}
	return out, rows.Err()
}

func (ps *PlayerService) SavePlayerSkills(playerID int64, skills []PersistSkill) error {
	if ps.connector == nil {
		return nil
	}
	if _, err := ps.connector.ExecSync("DELETE FROM player_skills WHERE player_id = ?", playerID); err != nil {
		return err
	}
	now := time.Now()
	for _, sk := range skills {
		if _, err := ps.connector.ExecSync(
			"INSERT INTO player_skills (player_id, skill_id, level, exp, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, 1, ?, ?)",
			playerID, sk.SkillID, sk.Level, sk.Exp, now, now); err != nil {
			return err
		}
	}
	return nil
}

// ensureAuxSchema 建仓库表（player_warehouse 无现成表）。幂等，启动时调一次。
func (ps *PlayerService) ensureAuxSchema() {
	if ps.connector == nil {
		return
	}
	_, err := ps.connector.ExecSync(`CREATE TABLE IF NOT EXISTS player_warehouse (
		id BIGINT NOT NULL AUTO_INCREMENT,
		player_id BIGINT NOT NULL,
		item_id INT NOT NULL,
		count INT NOT NULL,
		position INT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		PRIMARY KEY (id),
		KEY idx_player_id (player_id)
	)`)
	if err != nil {
		zLog.Warn("ensure player_warehouse table failed", zap.Error(err))
	}
}

func (ps *PlayerService) LoadPlayerWarehouse(playerID int64) ([]PersistItem, error) {
	if ps.connector == nil {
		return nil, nil
	}
	rows, err := ps.connector.QuerySync("SELECT item_id, count, position FROM player_warehouse WHERE player_id = ?", playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PersistItem
	for rows.Next() {
		var it PersistItem
		if err := rows.Scan(&it.ConfigID, &it.Count, &it.Slot); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (ps *PlayerService) SavePlayerWarehouse(playerID int64, items []PersistItem) error {
	if ps.connector == nil {
		return nil
	}
	if _, err := ps.connector.ExecSync("DELETE FROM player_warehouse WHERE player_id = ?", playerID); err != nil {
		return err
	}
	now := time.Now()
	for _, it := range items {
		if _, err := ps.connector.ExecSync(
			"INSERT INTO player_warehouse (player_id, item_id, count, position, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			playerID, it.ConfigID, it.Count, it.Slot, now, now); err != nil {
			return err
		}
	}
	return nil
}

// —— 邮件（业务层建设 2026-07-25）：离线持久异步投递 + 领取。用现成 player_mails 表；附件仅金币（JSON）——

type mailAttachment struct {
	Gold int64 `json:"gold"`
}

// MailRow 一封邮件（列表用）。
type MailRow struct {
	MailID    int64
	Sender    string
	Title     string
	Content   string
	Gold      int64
	IsClaimed bool
}

// SendMail 发一封邮件给 toPlayerID（收件人在线与否均可——落库即投递）。
func (ps *PlayerService) SendMail(toPlayerID int64, sender, title, content string, gold int64) error {
	if ps.connector == nil {
		return fmt.Errorf("no db")
	}
	att, _ := json.Marshal(mailAttachment{Gold: gold})
	now := time.Now()
	_, err := ps.connector.ExecSync(
		"INSERT INTO player_mails (player_id, sender, title, content, is_read, is_claimed, attachments, created_at, updated_at) VALUES (?, ?, ?, ?, 0, 0, ?, ?, ?)",
		toPlayerID, sender, title, content, string(att), now, now)
	return err
}

// ListMail 拉取某玩家的邮件（新→旧，最多 50 封）。
func (ps *PlayerService) ListMail(playerID int64) ([]MailRow, error) {
	if ps.connector == nil {
		return nil, nil
	}
	rows, err := ps.connector.QuerySync(
		"SELECT id, sender, title, content, attachments, is_claimed FROM player_mails WHERE player_id = ? ORDER BY id DESC LIMIT 50", playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MailRow
	for rows.Next() {
		var m MailRow
		var attStr string
		var claimed int
		if err := rows.Scan(&m.MailID, &m.Sender, &m.Title, &m.Content, &attStr, &claimed); err != nil {
			return nil, err
		}
		var att mailAttachment
		_ = json.Unmarshal([]byte(attStr), &att)
		m.Gold = att.Gold
		m.IsClaimed = claimed != 0
		out = append(out, m)
	}
	return out, rows.Err()
}

// ClaimMail 领取邮件附件：原子标记已领（WHERE is_claimed=0 防重复领），返回到账金币。
func (ps *PlayerService) ClaimMail(playerID, mailID int64) (int64, bool, error) {
	if ps.connector == nil {
		return 0, false, fmt.Errorf("no db")
	}
	res, err := ps.connector.ExecSync(
		"UPDATE player_mails SET is_claimed = 1, is_read = 1, updated_at = ? WHERE id = ? AND player_id = ? AND is_claimed = 0",
		time.Now(), mailID, playerID)
	if err != nil {
		return 0, false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, false, nil // 不存在/非本人/已领
	}
	// 读附件金币
	rows, err := ps.connector.QuerySync("SELECT attachments FROM player_mails WHERE id = ?", mailID)
	if err != nil {
		return 0, true, err
	}
	defer rows.Close()
	var gold int64
	if rows.Next() {
		var attStr string
		if err := rows.Scan(&attStr); err == nil {
			var att mailAttachment
			_ = json.Unmarshal([]byte(attStr), &att)
			gold = att.Gold
		}
	}
	return gold, true, nil
}

func (ps *PlayerService) Stop() {
	ps.isRunning = false
	close(ps.stopChan)

	ps.onlineMu.RLock()
	for _, p := range ps.onlinePlayers {
		ps.savePlayer(p)
	}
	ps.onlineMu.RUnlock()
}

func (ps *PlayerService) saveLoop() {
	ticker := time.NewTicker(ps.saveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ps.stopChan:
			ps.onlineMu.RLock()
			for _, p := range ps.onlinePlayers {
				ps.savePlayer(p)
			}
			ps.onlineMu.RUnlock()
			return
		case playerID := <-ps.saveChan:
			ps.onlineMu.RLock()
			if p, ok := ps.onlinePlayers[playerID]; ok {
				ps.savePlayer(p)
			}
			ps.onlineMu.RUnlock()
		case <-ticker.C:
			ps.onlineMu.RLock()
			for _, p := range ps.onlinePlayers {
				if time.Since(p.LastSaveTime) >= ps.saveInterval {
					ps.savePlayer(p)
				}
			}
			ps.onlineMu.RUnlock()
		}
	}
}

func (ps *PlayerService) savePlayer(p *OnlinePlayer) {
	p.mu.RLock()
	player := &models.Player{
		PlayerID:   int64(p.PlayerID),
		PlayerName: p.PlayerName,
		Level:      p.Level,
		Experience: p.Exp,
		Gold:       p.Gold,
		Diamond:    p.Diamond,
		Sex:        p.Sex,
		Age:        p.Age,
		VipLevel:   p.VipLevel,
		UpdatedAt:  time.Now(),
	}
	p.mu.RUnlock()

	updated, err := ps.playerDAO.UpdatePlayer(player)
	if err != nil {
		zLog.Error("Failed to save player", zap.Int64("player_id", int64(p.PlayerID)), zap.Error(err))
		return
	}
	if updated {
		p.LastSaveTime = time.Now()
		zLog.Debug("Player saved", zap.Int64("player_id", int64(p.PlayerID)))
	}
}

func (ps *PlayerService) GetPlayerList(accountID id.AccountIdType) ([]PlayerInfo, error) {
	players, err := ps.playerDAO.GetPlayersByAccountID(int64(accountID))
	if err != nil {
		zLog.Error("Failed to get player list", zap.Error(err))
		return nil, err
	}

	result := make([]PlayerInfo, 0, len(players))
	for _, player := range players {
		result = append(result, PlayerInfo{
			PlayerID:   id.PlayerIdType(player.PlayerID),
			Name:       player.PlayerName,
			PlayerName: player.PlayerName,
			Level:      player.Level,
			Exp:        player.Experience,
			Gold:       player.Gold,
			Diamond:    player.Diamond,
			Sex:        player.Sex,
			Age:        player.Age,
			VipLevel:   player.VipLevel,
			CreateTime: player.CreatedAt.Unix(),
		})
	}
	return result, nil
}

func (ps *PlayerService) GetPlayerByID(playerID id.PlayerIdType) (*PlayerInfo, error) {
	player, err := ps.playerDAO.GetPlayerByID(int64(playerID))
	if err != nil {
		zLog.Error("Failed to get player", zap.Error(err))
		return nil, err
	}
	if player == nil {
		return nil, nil
	}

	return &PlayerInfo{
		PlayerID:   id.PlayerIdType(player.PlayerID),
		Name:       player.PlayerName,
		PlayerName: player.PlayerName,
		Level:      player.Level,
		Exp:        player.Experience,
		Gold:       player.Gold,
		Diamond:    player.Diamond,
		Sex:        player.Sex,
		Age:        player.Age,
		VipLevel:   player.VipLevel,
		CreateTime: player.CreatedAt.Unix(),
	}, nil
}

func (ps *PlayerService) CreatePlayer(accountID id.AccountIdType, playerName string, sex int32, age int32) (id.PlayerIdType, error) {
	existing, err := ps.playerDAO.GetPlayerByName(playerName)
	if err != nil {
		zLog.Error("Failed to check player name", zap.Error(err))
		return 0, err
	}
	if existing != nil {
		return 0, fmt.Errorf("player name already exists")
	}

	var playerID id.PlayerIdType
	if ps.snowflake != nil {
		generatedID, err := ps.snowflake.NextID()
		if err != nil {
			zLog.Error("Failed to generate player ID", zap.Error(err))
			return 0, err
		}
		playerID = id.PlayerIdType(generatedID)
	} else {
		playerID = id.PlayerIdType(time.Now().UnixNano())
	}

	now := time.Now()
	player := &models.Player{
		PlayerID:     int64(playerID),
		AccountID:    int64(accountID),
		PlayerName:   playerName,
		Sex:          int(sex),
		Age:          int(age),
		Level:        1,
		Experience:   0,
		Gold:         1000,
		Diamond:      100,
		VipLevel:     0,
		LastLoginAt:  now,
		LastLogoutAt: now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if _, err := ps.playerDAO.CreatePlayer(player); err != nil {
		zLog.Error("Failed to create player", zap.Error(err))
		return 0, err
	}

	zLog.Info("Player created successfully", zap.Int64("player_id", int64(playerID)), zap.String("player_name", playerName))
	return playerID, nil
}

func (ps *PlayerService) PlayerLogin(playerID id.PlayerIdType) error {
	ps.playerDAO.UpdatePlayerLastLogin(int64(playerID), time.Now())
	zLog.Info("Player logged in", zap.Int64("player_id", int64(playerID)))
	return nil
}

func (ps *PlayerService) PlayerLogout(playerID id.PlayerIdType) error {
	ps.onlineMu.Lock()
	if p, ok := ps.onlinePlayers[playerID]; ok {
		ps.savePlayer(p)
		delete(ps.onlinePlayers, playerID)
	}
	ps.onlineMu.Unlock()

	ps.playerDAO.UpdatePlayerLastLogout(int64(playerID), time.Now())
	zLog.Info("Player logged out", zap.Int64("player_id", int64(playerID)))
	return nil
}

func (ps *PlayerService) AddToOnline(playerID id.PlayerIdType, sessionID string, info *PlayerInfo) {
	ps.onlineMu.Lock()
	defer ps.onlineMu.Unlock()

	ps.onlinePlayers[playerID] = &OnlinePlayer{
		PlayerInfo:   *info,
		SessionID:    sessionID,
		Status:       1,
		LastSaveTime: time.Now(),
	}
	zLog.Info("Player added to online", zap.Int64("player_id", int64(playerID)), zap.String("session_id", sessionID))
}

func (ps *PlayerService) RemoveFromOnline(playerID id.PlayerIdType) {
	ps.onlineMu.Lock()
	defer ps.onlineMu.Unlock()

	if p, ok := ps.onlinePlayers[playerID]; ok {
		ps.savePlayer(p)
		delete(ps.onlinePlayers, playerID)
		zLog.Info("Player removed from online", zap.Int64("player_id", int64(playerID)))
	}
}

func (ps *PlayerService) GetOnlinePlayer(playerID id.PlayerIdType) (*OnlinePlayer, bool) {
	ps.onlineMu.RLock()
	defer ps.onlineMu.RUnlock()
	p, ok := ps.onlinePlayers[playerID]
	return p, ok
}

func (ps *PlayerService) IsOnline(playerID id.PlayerIdType) bool {
	ps.onlineMu.RLock()
	defer ps.onlineMu.RUnlock()
	_, ok := ps.onlinePlayers[playerID]
	return ok
}

func (ps *PlayerService) UpdatePlayerPosition(playerID id.PlayerIdType, mapID int32, x, y float32) {
	ps.onlineMu.RLock()
	if p, ok := ps.onlinePlayers[playerID]; ok {
		p.mu.Lock()
		p.MapID = mapID
		p.X = x
		p.Y = y
		p.mu.Unlock()
	}
	ps.onlineMu.RUnlock()
}

func (ps *PlayerService) UpdatePlayerGold(playerID id.PlayerIdType, goldDelta int64) error {
	ps.onlineMu.RLock()
	p, ok := ps.onlinePlayers[playerID]
	ps.onlineMu.RUnlock()

	if !ok {
		return fmt.Errorf("player not online")
	}

	p.mu.Lock()
	p.Gold += goldDelta
	if p.Gold < 0 {
		p.Gold = 0
	}
	p.mu.Unlock()

	select {
	case ps.saveChan <- playerID:
	default:
	}

	return nil
}

func (ps *PlayerService) UpdatePlayerDiamond(playerID id.PlayerIdType, diamondDelta int64) error {
	ps.onlineMu.RLock()
	p, ok := ps.onlinePlayers[playerID]
	ps.onlineMu.RUnlock()

	if !ok {
		return fmt.Errorf("player not online")
	}

	p.mu.Lock()
	p.Diamond += diamondDelta
	if p.Diamond < 0 {
		p.Diamond = 0
	}
	p.mu.Unlock()

	select {
	case ps.saveChan <- playerID:
	default:
	}

	return nil
}

func (ps *PlayerService) UpdatePlayerExp(playerID id.PlayerIdType, expDelta int64) (newLevel int, levelUp bool) {
	ps.onlineMu.RLock()
	p, ok := ps.onlinePlayers[playerID]
	ps.onlineMu.RUnlock()

	if !ok {
		return 0, false
	}

	p.mu.Lock()
	p.Exp += expDelta
	oldLevel := p.Level
	newLevel = p.Level

	expForNextLevel := int64(newLevel * 100)
	for p.Exp >= expForNextLevel && expForNextLevel > 0 {
		newLevel++
		expForNextLevel = int64(newLevel * 100)
	}

	if newLevel > oldLevel {
		p.Level = newLevel
		levelUp = true
		zLog.Info("Player leveled up", zap.Int64("player_id", int64(playerID)), zap.Int("old_level", oldLevel), zap.Int("new_level", newLevel))
	}
	p.mu.Unlock()

	if levelUp {
		select {
		case ps.saveChan <- playerID:
		default:
		}
	}

	return newLevel, levelUp
}

// SyncOnlinePlayerData 同步在线玩家数据（从 Actor 层同步到 Service 层）
func (ps *PlayerService) SyncOnlinePlayerData(playerID id.PlayerIdType, level int, exp, gold, diamond int64) {
	ps.onlineMu.RLock()
	p, ok := ps.onlinePlayers[playerID]
	ps.onlineMu.RUnlock()

	if !ok {
		return
	}

	p.mu.Lock()
	p.Level = level
	p.Exp = exp
	p.Gold = gold
	p.Diamond = diamond
	p.mu.Unlock()
}
