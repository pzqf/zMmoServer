package crossserver

import (
	"encoding/json"
	"fmt"
	"time"
)

type PlayerMigrationData struct {
	PlayerID     int64  `json:"player_id"`
	AccountID    int64  `json:"account_id"`
	PlayerName   string `json:"player_name"`
	Level        int32  `json:"level"`
	Exp          int64  `json:"exp"`
	Gold         int64  `json:"gold"`
	Diamond      int64  `json:"diamond"`
	VipLevel     int32  `json:"vip_level"`
	VipExp       int32  `json:"vip_exp"`
	Class        int32  `json:"class"`
	Sex          int32  `json:"sex"`
	Health       int32  `json:"health"`
	MaxHealth    int32  `json:"max_health"`
	Mana         int32  `json:"mana"`
	MaxMana      int32  `json:"max_mana"`
	Strength     int32  `json:"strength"`
	Agility      int32  `json:"agility"`
	Intelligence int32  `json:"intelligence"`
	Stamina      int32  `json:"stamina"`
	Spirit       int32  `json:"spirit"`
	SkillPoints  int32  `json:"skill_points"`
	AttrPoints   int32  `json:"attr_points"`
	Items        []int32 `json:"items"`
	Skills       []int32 `json:"skills"`
	Buffs        []BuffMigrationData `json:"buffs"`
	Debuffs      []DebuffMigrationData `json:"debuffs"`
}

type BuffMigrationData struct {
	ID        int32             `json:"id"`
	Name      string            `json:"name"`
	DurationMs int64            `json:"duration_ms"`
	RemainingMs int64           `json:"remaining_ms"`
	Effects   []EffectMigrationData `json:"effects"`
}

type DebuffMigrationData struct {
	ID        int32             `json:"id"`
	Name      string            `json:"name"`
	DurationMs int64            `json:"duration_ms"`
	RemainingMs int64           `json:"remaining_ms"`
	Effects   []EffectMigrationData `json:"effects"`
}

type EffectMigrationData struct {
	Attribute string `json:"attribute"`
	Value     int32  `json:"value"`
}

type MapPlayerMigrationData struct {
	ObjectID  int64   `json:"object_id"`
	PlayerID  int64   `json:"player_id"`
	Name      string  `json:"name"`
	PositionX float64 `json:"position_x"`
	PositionY float64 `json:"position_y"`
	PositionZ float64 `json:"position_z"`
	MapID     int32   `json:"map_id"`
	Status    int32   `json:"status"`
}

type MigrationDataWrapper struct {
	Version    int64                `json:"version"`
	PlayerData PlayerMigrationData  `json:"player_data"`
	MapData    *MapPlayerMigrationData `json:"map_data,omitempty"`
	Timestamp  int64                `json:"timestamp"`
}

func SerializePlayerData(playerData *PlayerMigrationData, mapData *MapPlayerMigrationData) ([]byte, []byte, error) {
	wrapper := MigrationDataWrapper{
		Version:    1,
		PlayerData: *playerData,
		MapData:    mapData,
		Timestamp:  time.Now().UnixMilli(),
	}

	playerBytes, err := json.Marshal(wrapper)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal migration data: %w", err)
	}

	var mapBytes []byte
	if mapData != nil {
		mapBytes, err = json.Marshal(mapData)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal map data: %w", err)
		}
	}

	return playerBytes, mapBytes, nil
}

func DeserializePlayerData(playerBytes []byte) (*MigrationDataWrapper, error) {
	var wrapper MigrationDataWrapper
	if err := json.Unmarshal(playerBytes, &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal migration data: %w", err)
	}
	return &wrapper, nil
}

func DeserializeMapData(mapBytes []byte) (*MapPlayerMigrationData, error) {
	if len(mapBytes) == 0 {
		return nil, nil
	}

	var mapData MapPlayerMigrationData
	if err := json.Unmarshal(mapBytes, &mapData); err != nil {
		return nil, fmt.Errorf("unmarshal map data: %w", err)
	}
	return &mapData, nil
}
