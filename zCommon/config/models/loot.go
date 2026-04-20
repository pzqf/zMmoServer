package models

type LootGroup struct {
	LootGroupID  int32   `json:"loot_group_id"`
	Name         string  `json:"name"`
	Items        string  `json:"items"`
	DropRate     float32 `json:"drop_rate"`
	MinLevel     int32   `json:"min_level"`
	MaxLevel     int32   `json:"max_level"`
	Difficulty   string  `json:"difficulty"`
	MaxDropCount int32   `json:"max_drop_count"`
}

type LootItem struct {
	ItemID    int32   `json:"item_id"`
	CountMin  int32   `json:"count_min"`
	CountMax  int32   `json:"count_max"`
	DropRate  float32 `json:"drop_rate"`
	IsBound   bool    `json:"is_bound"`
}
