package models

// Map 地图配置
// 定义地图的基础属性，从Excel配置表加载
type Map struct {
	MapID       int32   `json:"map_id"`       // 地图ID
	Name        string  `json:"name"`         // 地图名称
	MapType     int32   `json:"map_type"`     // 地图类型（主城/野外/副本等）
	Width       int32   `json:"width"`        // 地图宽度
	Height      int32   `json:"height"`       // 地图高度
	RegionSize  int32   `json:"region_size"`  // 区域大小（用于空间分区）
	TileWidth   int32   `json:"tile_width"`   // 瓦片宽度
	TileHeight  int32   `json:"tile_height"`  // 瓦片高度
	IsInstance  bool    `json:"is_instance"`  // 是否副本
	MaxPlayers  int32   `json:"max_players"`  // 最大玩家数
	Description string  `json:"description"`  // 地图描述
	Background  string  `json:"background"`   // 背景图
	Music       string  `json:"music"`        // 背景音乐
	WeatherType string  `json:"weather_type"` // 天气类型
	MinLevel    int32   `json:"min_level"`    // 最低进入等级
	MaxLevel    int32   `json:"max_level"`    // 最高进入等级
	RespawnRate float64 `json:"respawn_rate"` // 刷新速率
}

// MapSpawnPoint 地图刷新点
// 定义怪物/NPC的刷新位置
type MapSpawnPoint struct {
	ID        int32   `json:"id"`        // 刷新点ID
	MapID     int32   `json:"map_id"`    // 所属地图ID
	Type      string  `json:"type"`      // 类型（monster/npc）
	ObjectID  int32   `json:"object_id"` // 对象ID（怪物ID或NPC ID）
	X         float64 `json:"x"`         // X坐标
	Y         float64 `json:"y"`         // Y坐标
	Z         float64 `json:"z"`         // Z坐标
	Name      string  `json:"name"`      // 名称
	Frequency int32   `json:"frequency"` // 刷新频率（秒）
	GroupID   int32   `json:"group_id"`  // 刷新组ID
}

// MapTeleportPoint 地图传送点
// 定义地图间的传送位置
type MapTeleportPoint struct {
	ID            int32   `json:"id"`             // 传送点ID
	MapID         int32   `json:"map_id"`         // 所属地图ID
	X             float64 `json:"x"`              // X坐标
	Y             float64 `json:"y"`              // Y坐标
	Z             float64 `json:"z"`              // Z坐标
	TargetMapID   int32   `json:"target_map_id"`  // 目标地图ID
	TargetX       float64 `json:"target_x"`       // 目标X坐标
	TargetY       float64 `json:"target_y"`       // 目标Y坐标
	TargetZ       float64 `json:"target_z"`       // 目标Z坐标
	Name          string  `json:"name"`           // 传送点名称
	RequiredLevel int32   `json:"required_level"` // 等级要求
	RequiredItem  int32   `json:"required_item"`  // 需要的物品ID
	IsActive      bool    `json:"is_active"`      // 是否激活
}

// MapBuilding 地图建筑
// 定义地图中的可交互建筑
type MapBuilding struct {
	ID      int32   `json:"id"`      // 建筑ID
	MapID   int32   `json:"map_id"`  // 所属地图ID
	X       float64 `json:"x"`       // X坐标
	Y       float64 `json:"y"`       // Y坐标
	Z       float64 `json:"z"`       // Z坐标
	Width   float64 `json:"width"`   // 建筑宽度
	Height  float64 `json:"height"`  // 建筑高度
	Type    string  `json:"type"`    // 建筑类型
	Name    string  `json:"name"`    // 建筑名称
	Level   int32   `json:"level"`   // 建筑等级
	HP      int32   `json:"hp"`      // 建筑生命值
	Faction int32   `json:"faction"` // 阵营
}

// MapEvent 地图事件
// 定义地图中的动态事件
type MapEvent struct {
	EventID     int32   `json:"event_id"`    // 事件ID
	MapID       int32   `json:"map_id"`      // 所属地图ID
	Type        string  `json:"type"`        // 事件类型
	Name        string  `json:"name"`        // 事件名称
	Description string  `json:"description"` // 事件描述
	X           float64 `json:"x"`           // 事件中心X坐标
	Y           float64 `json:"y"`           // 事件中心Y坐标
	Z           float64 `json:"z"`           // 事件中心Z坐标
	Radius      float64 `json:"radius"`      // 事件范围半径
	StartTime   string  `json:"start_time"`  // 开始时间
	EndTime     string  `json:"end_time"`    // 结束时间
	Duration    int32   `json:"duration"`    // 持续时间（秒）
	RewardID    int32   `json:"reward_id"`   // 奖励ID
	IsActive    bool    `json:"is_active"`   // 是否激活
}

// MapResource 地图资源点
// 定义可采集的资源点
type MapResource struct {
	ResourceID  int32   `json:"resource_id"`  // 资源点ID
	MapID       int32   `json:"map_id"`       // 所属地图ID
	Type        string  `json:"type"`         // 资源类型（矿石/草药等）
	X           float64 `json:"x"`            // X坐标
	Y           float64 `json:"y"`            // Y坐标
	Z           float64 `json:"z"`            // Z坐标
	RespawnTime int32   `json:"respawn_time"` // 刷新时间（秒）
	ItemID      int32   `json:"item_id"`      // 产出物品ID
	Quantity    int32   `json:"quantity"`     // 产出数量
	Level       int32   `json:"level"`        // 资源等级
	IsGathering bool    `json:"is_gathering"` // 是否正在采集
}
