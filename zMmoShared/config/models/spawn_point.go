package models

type SpawnPointType int32

const (
	SpawnPointTypeMonster SpawnPointType = 1
	SpawnPointTypeNPC     SpawnPointType = 2
)

type SpawnPoint struct {
	SpawnID      int32          `json:"spawn_id"`      // 刷新点ID
	MapID        int32          `json:"map_id"`        // 所属地图ID
	MonsterID    int32          `json:"monster_id"`    // 怪物ID（NPC时为NPC ID）
	SpawnType    SpawnPointType `json:"spawn_type"`    // 刷新类型（1=怪物，2=NPC）
	PosX         float32        `json:"pos_x"`         // X坐标
	PosY         float32        `json:"pos_y"`         // Y坐标
	PosZ         float32        `json:"pos_z"`         // Z坐标
	MaxCount     int32          `json:"max_count"`     // 最大刷新数量
	SpawnInterval int32         `json:"spawn_interval"` // 刷新间隔（秒）
	Radius       float32        `json:"radius"`        // 刷新半径
	PatrolRange  float32        `json:"patrol_range"`  // 巡逻范围
}
