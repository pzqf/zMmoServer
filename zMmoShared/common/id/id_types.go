package id

const (
	ServerIdGroupBits    = 4
	ServerIdIndexBits    = 2
	ServerIdGroupShift   = 2
	ServerIdIndexShift   = 0
	ServerIdGroupMax     = 15
	ServerIdIndexMax     = 3
	ServerIdGroupMask    = (1 << ServerIdGroupBits) - 1
	ServerIdIndexMask    = (1 << ServerIdIndexBits) - 1
	ServerIdGroupBitMask = ServerIdGroupMask << ServerIdGroupShift
	ServerIdIndexBitMask = ServerIdIndexMask << ServerIdIndexShift
)

type ServerIdType int32

func (s ServerIdType) GetGroupID() int32 {
	return (int32(s) >> ServerIdGroupShift) & ServerIdGroupMask
}

func (s ServerIdType) GetServerIndex() int32 {
	return (int32(s) >> ServerIdIndexShift) & ServerIdIndexMask
}

func (s ServerIdType) IsValid() bool {
	groupID := s.GetGroupID()
	serverIndex := s.GetServerIndex()
	return groupID > 0 && groupID <= ServerIdGroupMax && serverIndex >= 0 && serverIndex <= ServerIdIndexMax
}

func NewServerId(groupID int32, serverIndex int32) ServerIdType {
	return ServerIdType((groupID << ServerIdGroupShift) | (serverIndex << ServerIdIndexShift))
}

type PlayerIdType int64

// MapIdType 地图唯一标识ID类型
type MapIdType int32

// RegionIdType 区域唯一标识ID类型
type RegionIdType int32

// ObjectIdType 游戏对象唯一标识ID类型
type ObjectIdType int64

// AccountIdType 账号唯一标识ID类型
type AccountIdType int64

// TeamIdType 队伍唯一标识ID类型
type TeamIdType int32

// ComboIdType 连招/组合唯一标识ID类型
type ComboIdType int64

// VisualIdType 视觉效果唯一标识ID类型
type VisualIdType int64

// LogIdType 日志唯一标识ID类型
type LogIdType int64

// ItemIdType 道具唯一标识ID类型
type ItemIdType int64

// MailIdType 邮件唯一标识ID类型
type MailIdType int64

// GuildIdType 公会唯一标识ID类型
type GuildIdType int64

// AuctionIdType 拍卖物品唯一标识ID类型
type AuctionIdType int64

// BidIdType 竞拍记录唯一标识ID类型
type BidIdType int64

// PetIdType 宠物唯一标识ID类型
type PetIdType int64

// MountIdType 坐骑唯一标识ID类型
type MountIdType int64

// AchievementIdType 成就唯一标识ID类型
type AchievementIdType int64

// ActivityIdType 活动唯一标识ID类型
type ActivityIdType int64

// DungeonIdType 副本唯一标识ID类型
type DungeonIdType int32

// InstanceIdType 副本实例唯一标识ID类型
type InstanceIdType int64

// BuffIdType Buff唯一标识ID类型
type BuffIdType int32

// BuffInstanceIdType Buff实例唯一标识ID类型
type BuffInstanceIdType int64
