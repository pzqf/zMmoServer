package models

type Dungeon struct {
	DungeonID       int32  `json:"dungeon_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Type            int32  `json:"type"`
	Difficulty      int32  `json:"difficulty"`
	MinLevel        int32  `json:"min_level"`
	MaxLevel        int32  `json:"max_level"`
	MinPlayers      int32  `json:"min_players"`
	MaxPlayers      int32  `json:"max_players"`
	TimeLimit       int32  `json:"time_limit"`
	DailyLimit      int32  `json:"daily_limit"`
	MapID           int32  `json:"map_id"`
	RewardExp       int64  `json:"reward_exp"`
	RewardGold      int64  `json:"reward_gold"`
	RewardItemIDs   string `json:"reward_item_ids"`
	EntryCostType   int32  `json:"entry_cost_type"`
	EntryCostAmount int32  `json:"entry_cost_amount"`
	WaveCount       int32  `json:"wave_count"`
	BossID          int32  `json:"boss_id"`
	IsOpen          bool   `json:"is_open"`
}

type DungeonWave struct {
	WaveID      int32  `json:"wave_id"`
	DungeonID   int32  `json:"dungeon_id"`
	WaveIndex   int32  `json:"wave_index"`
	MonsterIDs  string `json:"monster_ids"`
	MonsterCount int32 `json:"monster_count"`
	SpawnDelay  int32  `json:"spawn_delay"`
	IsBoss      bool   `json:"is_boss"`
}
