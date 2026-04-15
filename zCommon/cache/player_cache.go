package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zCommon/redis"
	"go.uber.org/zap"
)

const (
	KeyPrefixPlayer     = "player"
	KeyPrefixPlayerMap  = "player_map"
	KeyPrefixPlayerItem = "player_item"
	KeyPrefixPlayerBuff = "player_buff"
	KeyPrefixOnline     = "online"
	KeyPrefixSession    = "session"

	DefaultPlayerTTL   = 30 * time.Minute
	DefaultOnlineTTL   = 5 * time.Minute
	DefaultSessionTTL  = 10 * time.Minute
)

type PlayerCacheData struct {
	PlayerID     int64  `json:"player_id"`
	AccountID    int64  `json:"account_id"`
	PlayerName   string `json:"player_name"`
	Level        int32  `json:"level"`
	Exp          int64  `json:"exp"`
	Gold         int64  `json:"gold"`
	Diamond      int64  `json:"diamond"`
	VipLevel     int32  `json:"vip_level"`
	ServerID     int32  `json:"server_id"`
	LastLoginAt  int64  `json:"last_login_at"`
}

type PlayerMapCacheData struct {
	PlayerID     int64 `json:"player_id"`
	MapID        int32 `json:"map_id"`
	MapServerID  uint32 `json:"map_server_id"`
	GameServerID uint32 `json:"game_server_id"`
}

type SessionCacheData struct {
	SessionID   uint64 `json:"session_id"`
	PlayerID    int64  `json:"player_id"`
	GameServerID uint32 `json:"game_server_id"`
	GatewayID   uint32 `json:"gateway_id"`
	ConnectedAt int64  `json:"connected_at"`
}

type PlayerCacheConfig struct {
	PlayerTTL   time.Duration
	OnlineTTL   time.Duration
	SessionTTL  time.Duration
}

func DefaultPlayerCacheConfig() PlayerCacheConfig {
	return PlayerCacheConfig{
		PlayerTTL:  DefaultPlayerTTL,
		OnlineTTL:  DefaultOnlineTTL,
		SessionTTL: DefaultSessionTTL,
	}
}

type PlayerCache struct {
	client *redis.Client
	config PlayerCacheConfig
}

func NewPlayerCache(client *redis.Client, config PlayerCacheConfig) *PlayerCache {
	return &PlayerCache{
		client: client,
		config: config,
	}
}

func (pc *PlayerCache) SetPlayer(ctx context.Context, data *PlayerCacheData) error {
	key := fmt.Sprintf("%s:%d", KeyPrefixPlayer, data.PlayerID)
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal player cache data: %w", err)
	}

	if err := pc.client.Set(key, string(bytes), pc.config.PlayerTTL); err != nil {
		return fmt.Errorf("redis set player: %w", err)
	}

	zLog.Debug("Player cache set",
		zap.Int64("player_id", data.PlayerID),
		zap.Duration("ttl", pc.config.PlayerTTL))
	return nil
}

func (pc *PlayerCache) GetPlayer(ctx context.Context, playerID int64) (*PlayerCacheData, error) {
	key := fmt.Sprintf("%s:%d", KeyPrefixPlayer, playerID)
	val, err := pc.client.Get(key)
	if err != nil {
		return nil, fmt.Errorf("redis get player: %w", err)
	}

	var data PlayerCacheData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("unmarshal player cache data: %w", err)
	}

	return &data, nil
}

func (pc *PlayerCache) DeletePlayer(ctx context.Context, playerID int64) error {
	key := fmt.Sprintf("%s:%d", KeyPrefixPlayer, playerID)
	if err := pc.client.Del(key); err != nil {
		return fmt.Errorf("redis del player: %w", err)
	}
	return nil
}

func (pc *PlayerCache) SetPlayerMap(ctx context.Context, data *PlayerMapCacheData) error {
	key := fmt.Sprintf("%s:%d", KeyPrefixPlayerMap, data.PlayerID)
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal player map cache data: %w", err)
	}

	if err := pc.client.Set(key, string(bytes), pc.config.PlayerTTL); err != nil {
		return fmt.Errorf("redis set player map: %w", err)
	}
	return nil
}

func (pc *PlayerCache) GetPlayerMap(ctx context.Context, playerID int64) (*PlayerMapCacheData, error) {
	key := fmt.Sprintf("%s:%d", KeyPrefixPlayerMap, playerID)
	val, err := pc.client.Get(key)
	if err != nil {
		return nil, fmt.Errorf("redis get player map: %w", err)
	}

	var data PlayerMapCacheData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("unmarshal player map cache data: %w", err)
	}
	return &data, nil
}

func (pc *PlayerCache) DeletePlayerMap(ctx context.Context, playerID int64) error {
	key := fmt.Sprintf("%s:%d", KeyPrefixPlayerMap, playerID)
	if err := pc.client.Del(key); err != nil {
		return fmt.Errorf("redis del player map: %w", err)
	}
	return nil
}

func (pc *PlayerCache) SetOnline(ctx context.Context, playerID int64, serverID int32) error {
	key := fmt.Sprintf("%s:%d", KeyPrefixOnline, playerID)
	val := fmt.Sprintf("%d", serverID)
	if err := pc.client.Set(key, val, pc.config.OnlineTTL); err != nil {
		return fmt.Errorf("redis set online: %w", err)
	}
	return nil
}

func (pc *PlayerCache) IsOnline(ctx context.Context, playerID int64) (bool, int32, error) {
	key := fmt.Sprintf("%s:%d", KeyPrefixOnline, playerID)
	val, err := pc.client.Get(key)
	if err != nil {
		return false, 0, nil
	}

	var serverID int32
	fmt.Sscanf(val, "%d", &serverID)
	return true, serverID, nil
}

func (pc *PlayerCache) SetOffline(ctx context.Context, playerID int64) error {
	key := fmt.Sprintf("%s:%d", KeyPrefixOnline, playerID)
	if err := pc.client.Del(key); err != nil {
		return fmt.Errorf("redis del online: %w", err)
	}
	return nil
}

func (pc *PlayerCache) SetSession(ctx context.Context, data *SessionCacheData) error {
	key := fmt.Sprintf("%s:%d", KeyPrefixSession, data.SessionID)
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal session cache data: %w", err)
	}

	if err := pc.client.Set(key, string(bytes), pc.config.SessionTTL); err != nil {
		return fmt.Errorf("redis set session: %w", err)
	}
	return nil
}

func (pc *PlayerCache) GetSession(ctx context.Context, sessionID uint64) (*SessionCacheData, error) {
	key := fmt.Sprintf("%s:%d", KeyPrefixSession, sessionID)
	val, err := pc.client.Get(key)
	if err != nil {
		return nil, fmt.Errorf("redis get session: %w", err)
	}

	var data SessionCacheData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("unmarshal session cache data: %w", err)
	}
	return &data, nil
}

func (pc *PlayerCache) DeleteSession(ctx context.Context, sessionID uint64) error {
	key := fmt.Sprintf("%s:%d", KeyPrefixSession, sessionID)
	if err := pc.client.Del(key); err != nil {
		return fmt.Errorf("redis del session: %w", err)
	}
	return nil
}

func (pc *PlayerCache) RefreshPlayerTTL(ctx context.Context, playerID int64) error {
	key := fmt.Sprintf("%s:%d", KeyPrefixPlayer, playerID)
	if err := pc.client.Expire(key, pc.config.PlayerTTL); err != nil {
		return fmt.Errorf("redis expire player: %w", err)
	}

	mapKey := fmt.Sprintf("%s:%d", KeyPrefixPlayerMap, playerID)
	pc.client.Expire(mapKey, pc.config.PlayerTTL)

	onlineKey := fmt.Sprintf("%s:%d", KeyPrefixOnline, playerID)
	pc.client.Expire(onlineKey, pc.config.OnlineTTL)

	return nil
}

func (pc *PlayerCache) UpdatePlayerField(ctx context.Context, playerID int64, field string, value interface{}) error {
	key := fmt.Sprintf("%s:%d", KeyPrefixPlayer, playerID)
	return pc.client.HSet(key, field, value)
}

func (pc *PlayerCache) GetPlayerField(ctx context.Context, playerID int64, field string) (string, error) {
	key := fmt.Sprintf("%s:%d", KeyPrefixPlayer, playerID)
	return pc.client.HGet(key, field)
}
