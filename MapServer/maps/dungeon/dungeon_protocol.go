package dungeon

import "github.com/pzqf/zCommon/common/id"

const (
	MsgDungeonCreateRequest    uint32 = 40001
	MsgDungeonCreateResponse   uint32 = 40002
	MsgDungeonEnterRequest     uint32 = 40003
	MsgDungeonEnterResponse    uint32 = 40004
	MsgDungeonLeaveRequest     uint32 = 40005
	MsgDungeonLeaveResponse    uint32 = 40006
	MsgDungeonStartRequest     uint32 = 40007
	MsgDungeonStartResponse    uint32 = 40008
	MsgDungeonCompleteNotify   uint32 = 40009
	MsgDungeonWaveComplete     uint32 = 40010
	MsgDungeonWaveStart        uint32 = 40011
	MsgDungeonPlayerDeath      uint32 = 40012
	MsgDungeonMonsterKilled    uint32 = 40013
	MsgDungeonRewardNotify     uint32 = 40014
	MsgDungeonFailNotify       uint32 = 40015
	MsgDungeonDestroyRequest   uint32 = 40016
	MsgDungeonDestroyResponse  uint32 = 40017
	MsgDungeonQueryRequest     uint32 = 40018
	MsgDungeonQueryResponse    uint32 = 40019
)

type DungeonCreateRequest struct {
	DungeonID id.DungeonIdType `json:"dungeon_id"`
	LeaderID  id.PlayerIdType  `json:"leader_id"`
	Players   []id.PlayerIdType `json:"players"`
}

type DungeonCreateResponse struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	DungeonID  id.DungeonIdType  `json:"dungeon_id"`
	MapID      id.MapIdType      `json:"map_id"`
	Success    bool              `json:"success"`
	Error      string            `json:"error,omitempty"`
}

type DungeonEnterRequest struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	PlayerID   id.PlayerIdType  `json:"player_id"`
}

type DungeonEnterResponse struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	PlayerID   id.PlayerIdType  `json:"player_id"`
	MapID      id.MapIdType      `json:"map_id"`
	Success    bool              `json:"success"`
	Error      string            `json:"error,omitempty"`
}

type DungeonLeaveRequest struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	PlayerID   id.PlayerIdType  `json:"player_id"`
}

type DungeonLeaveResponse struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	PlayerID   id.PlayerIdType  `json:"player_id"`
	Success    bool              `json:"success"`
	Error      string            `json:"error,omitempty"`
}

type DungeonStartRequest struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
}

type DungeonStartResponse struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	Success    bool              `json:"success"`
	Error      string            `json:"error,omitempty"`
}

type DungeonCompleteNotify struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	DungeonID  id.DungeonIdType  `json:"dungeon_id"`
	IsSuccess  bool              `json:"is_success"`
	ClearTime  int32             `json:"clear_time"`
	Exp        int64             `json:"exp"`
	Gold       int64             `json:"gold"`
}

type DungeonWaveStart struct {
	InstanceID  id.InstanceIdType `json:"instance_id"`
	WaveIndex   int32             `json:"wave_index"`
	MonsterIDs  []int32           `json:"monster_ids"`
	IsBoss      bool              `json:"is_boss"`
}

type DungeonWaveComplete struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	WaveIndex  int32             `json:"wave_index"`
	NextWave   int32             `json:"next_wave"`
	IsFinal    bool              `json:"is_final"`
}

type DungeonPlayerDeathNotify struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	PlayerID   id.PlayerIdType  `json:"player_id"`
	AllDead    bool              `json:"all_dead"`
}

type DungeonMonsterKilledNotify struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	KillerID   id.PlayerIdType  `json:"killer_id"`
	MonsterID  int32             `json:"monster_id"`
	WaveKills  int32             `json:"wave_kills"`
}

type DungeonRewardNotify struct {
	InstanceID  id.InstanceIdType             `json:"instance_id"`
	DungeonID   id.DungeonIdType              `json:"dungeon_id"`
	Exp         int64                         `json:"exp"`
	Gold        int64                         `json:"gold"`
	PlayerStats map[id.PlayerIdType]*PlayerDungeonStats `json:"player_stats"`
}

type DungeonFailNotify struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	DungeonID  id.DungeonIdType  `json:"dungeon_id"`
	Reason     string            `json:"reason"`
}

type DungeonDestroyRequest struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
}

type DungeonDestroyResponse struct {
	InstanceID id.InstanceIdType `json:"instance_id"`
	Success    bool              `json:"success"`
	Error      string            `json:"error,omitempty"`
}

type DungeonQueryRequest struct {
	PlayerID id.PlayerIdType  `json:"player_id"`
}

type DungeonQueryResponse struct {
	PlayerID id.PlayerIdType                  `json:"player_id"`
	Records  map[id.DungeonIdType]*PlayerDungeonRecord `json:"records"`
}
