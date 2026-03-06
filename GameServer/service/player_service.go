package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/db/dao"
	"github.com/pzqf/zMmoShared/db/models"
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
	snowflake     *id.Snowflake
	onlinePlayers map[id.PlayerIdType]*OnlinePlayer
	onlineMu      sync.RWMutex
	saveInterval  time.Duration
	isRunning     bool
	saveChan      chan id.PlayerIdType
	stopChan      chan struct{}
}

func NewPlayerService(playerDAO *dao.PlayerDAO) *PlayerService {
	snowflake, err := id.NewSnowflake(1, 1)
	if err != nil {
		zLog.Error("Failed to create snowflake", zap.Error(err))
		snowflake = nil
	}

	ps := &PlayerService{
		playerDAO:     playerDAO,
		snowflake:     snowflake,
		onlinePlayers: make(map[id.PlayerIdType]*OnlinePlayer),
		saveInterval:  30 * time.Second,
		saveChan:      make(chan id.PlayerIdType, 1000),
		stopChan:      make(chan struct{}),
	}

	go ps.saveLoop()

	return ps
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

	ps.playerDAO.UpdatePlayer(player, func(updated bool, err error) {
		if err != nil {
			zLog.Error("Failed to save player", zap.Int64("player_id", int64(p.PlayerID)), zap.Error(err))
			return
		}
		if updated {
			p.LastSaveTime = time.Now()
			zLog.Debug("Player saved", zap.Int64("player_id", int64(p.PlayerID)))
		}
	})
}

func (ps *PlayerService) GetPlayerList(accountID id.AccountIdType) ([]PlayerInfo, error) {
	var result []PlayerInfo
	var err error

	ps.playerDAO.GetPlayersByAccountID(int64(accountID), func(players []*models.Player, e error) {
		if e != nil {
			err = e
			return
		}

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
	})

	if err != nil {
		zLog.Error("Failed to get player list", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (ps *PlayerService) GetPlayerByID(playerID id.PlayerIdType) (*PlayerInfo, error) {
	var result *PlayerInfo
	var err error

	ps.playerDAO.GetPlayerByID(int64(playerID), func(player *models.Player, e error) {
		if e != nil {
			err = e
			return
		}
		if player == nil {
			return
		}

		result = &PlayerInfo{
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
		}
	})

	if err != nil {
		zLog.Error("Failed to get player", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (ps *PlayerService) CreatePlayer(accountID id.AccountIdType, playerName string, sex int32, age int32) (id.PlayerIdType, error) {
	var nameExists bool
	var nameErr error
	ps.playerDAO.GetPlayerByName(playerName, func(player *models.Player, e error) {
		if e != nil {
			nameErr = e
			return
		}
		nameExists = player != nil
	})

	if nameErr != nil {
		zLog.Error("Failed to check player name", zap.Error(nameErr))
		return 0, nameErr
	}

	if nameExists {
		zLog.Error("Player name already exists", zap.String("player_name", playerName))
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
		PlayerID:   int64(playerID),
		AccountID:  int64(accountID),
		PlayerName: playerName,
		Sex:        int(sex),
		Age:        int(age),
		Level:      1,
		Experience: 0,
		Gold:       1000, // 初始金币
		Diamond:    100,  // 初始钻石
		VipLevel:   0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	var err error
	ps.playerDAO.CreatePlayer(player, func(id int64, e error) {
		if e != nil {
			err = e
		}
	})

	if err != nil {
		zLog.Error("Failed to create player", zap.Error(err))
		return 0, err
	}

	zLog.Info("Player created successfully", zap.Int64("player_id", int64(playerID)), zap.String("player_name", playerName))
	return playerID, nil
}

func (ps *PlayerService) PlayerLogin(playerID id.PlayerIdType) error {
	// 更新玩家登录状态
	now := time.Now()
	ps.playerDAO.UpdatePlayerLastLogin(int64(playerID), now)
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

	now := time.Now()
	ps.playerDAO.UpdatePlayerLastLogout(int64(playerID), now)
	zLog.Info("Player logged out", zap.Int64("player_id", int64(playerID)))
	return nil
}

func (ps *PlayerService) AddToOnline(playerID id.PlayerIdType, sessionID string, info *PlayerInfo) {
	ps.onlineMu.Lock()
	defer ps.onlineMu.Unlock()

	onlinePlayer := &OnlinePlayer{
		PlayerInfo:   *info,
		SessionID:    sessionID,
		Status:       1,
		LastSaveTime: time.Now(),
	}
	ps.onlinePlayers[playerID] = onlinePlayer
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
