package id

import (
	"fmt"
	"strconv"
)

const (
	ServerIDGroupMax = 9999
	ServerIDIndexMin = 1
	ServerIDIndexMax = 99
)

type ServerIdType int32

func (s ServerIdType) GetGroupID() int32 {
	return int32(s) / 100
}

func (s ServerIdType) GetServerIndex() int32 {
	return int32(s) % 100
}

func (s ServerIdType) IsValid() bool {
	groupID := s.GetGroupID()
	serverIndex := s.GetServerIndex()
	return groupID > 0 && groupID <= ServerIDGroupMax && serverIndex >= ServerIDIndexMin && serverIndex <= ServerIDIndexMax
}

func NewServerId(groupID int32, serverIndex int32) ServerIdType {
	return ServerIdType(groupID*100 + serverIndex)
}

// BuildServerID 生成6位服务器ID（GroupID(4位)+ServerIndex(2位)）
func BuildServerID(groupID int32, serverIndex int32) (ServerIdType, error) {
	if groupID <= 0 || groupID > ServerIDGroupMax {
		return 0, fmt.Errorf("invalid group_id %d, expected 1-%d", groupID, ServerIDGroupMax)
	}
	if serverIndex < ServerIDIndexMin || serverIndex > ServerIDIndexMax {
		return 0, fmt.Errorf("invalid server_index %d, expected %d-%d", serverIndex, ServerIDIndexMin, ServerIDIndexMax)
	}
	return NewServerId(groupID, serverIndex), nil
}

// ParseServerIDStr 解析6位字符串格式的服务器ID（如 000101）
func ParseServerIDStr(serverID string) (ServerIdType, error) {
	if len(serverID) != 6 {
		return 0, fmt.Errorf("invalid server_id length %d, expected 6", len(serverID))
	}
	idValue, err := strconv.Atoi(serverID)
	if err != nil {
		return 0, fmt.Errorf("invalid server_id %q: %w", serverID, err)
	}
	parsed := ServerIdType(idValue)
	if !parsed.IsValid() {
		return 0, fmt.Errorf("invalid server_id %q", serverID)
	}
	return parsed, nil
}

// ParseServerIDInt 解析整型服务器ID（如 101 -> 000101）
func ParseServerIDInt(serverID int32) (ServerIdType, error) {
	parsed := ServerIdType(serverID)
	if !parsed.IsValid() {
		return 0, fmt.Errorf("invalid server_id %d", serverID)
	}
	return parsed, nil
}

// MustParseServerIDInt 解析整型服务器ID，失败返回0
func MustParseServerIDInt(serverID int32) ServerIdType {
	parsed, err := ParseServerIDInt(serverID)
	if err != nil {
		return 0
	}
	return parsed
}

// ServerIDString 将服务器ID格式化为6位字符串（如 000101）
func ServerIDString(serverID ServerIdType) string {
	return fmt.Sprintf("%06d", int32(serverID))
}

// GroupIDStringFromServerID 从服务器ID获取组ID字符串
func GroupIDStringFromServerID(serverID ServerIdType) string {
	return fmt.Sprintf("%d", serverID.GetGroupID())
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

// QuestIdType 任务唯一标识ID类型
type QuestIdType int32
