package models

// Monster 怪物配置结构
type Monster struct {
	MonsterID    int32   `json:"monster_id"`
	Name         string  `json:"name"`
	Level        int32   `json:"level"`
	HP           int32   `json:"hp"`
	MP           int32   `json:"mp"`
	Attack       int32   `json:"attack"`
	Defense      int32   `json:"defense"`
	Speed        int32   `json:"speed"`
	Exp          int32   `json:"exp"`
	DropItemRate float32 `json:"drop_item_rate"`
	DropItems    string  `json:"drop_items"`
	LootGroupID  int32   `json:"loot_group_id"`
	RespawnTime  int32   `json:"respawn_time"`
	AIType       string  `json:"ai_type"`
	Difficulty   string  `json:"difficulty"`
}
