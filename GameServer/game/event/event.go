package event

import (
	"github.com/pzqf/zEngine/zEvent"
	"github.com/pzqf/zMmoShared/common/id"
)

// 游戏业务相关的事件类型定义
const (
	// 玩家相关事件
	EventPlayerLogin  zEvent.EventType = 1001
	EventPlayerLogout zEvent.EventType = 1002
	EventPlayerExpAdd zEvent.EventType = 1003
	EventPlayerLevelUp zEvent.EventType = 1004

	// 货币相关事件
	EventPlayerGoldAdd    zEvent.EventType = 1010
	EventPlayerGoldReduce zEvent.EventType = 1011
	EventPlayerDiamondAdd zEvent.EventType = 1012
	EventPlayerDiamondReduce zEvent.EventType = 1013

	// 物品相关事件
	EventPlayerItemAdd    zEvent.EventType = 1020
	EventPlayerItemRemove zEvent.EventType = 1021
	EventPlayerItemUse    zEvent.EventType = 1022

	// 战斗相关事件
	EventPlayerDamage    zEvent.EventType = 1030
	EventPlayerHeal      zEvent.EventType = 1031
	EventPlayerDeath     zEvent.EventType = 1032
	EventPlayerRespawn   zEvent.EventType = 1033

	// 地图相关事件
	EventPlayerEnterMap zEvent.EventType = 1040
	EventPlayerLeaveMap zEvent.EventType = 1041
	EventPlayerMove     zEvent.EventType = 1042

	// 社交相关事件
	EventPlayerJoinGuild  zEvent.EventType = 1050
	EventPlayerLeaveGuild zEvent.EventType = 1051
	EventPlayerJoinTeam   zEvent.EventType = 1052
	EventPlayerLeaveTeam  zEvent.EventType = 1053
	EventPlayerAddFriend  zEvent.EventType = 1054
	EventPlayerRemoveFriend zEvent.EventType = 1055

	// 公会相关事件
	EventGuildCreate   zEvent.EventType = 1060
	EventGuildDissolve zEvent.EventType = 1061
	EventGuildKick     zEvent.EventType = 1062
	EventGuildPromote  zEvent.EventType = 1063
	EventGuildNotice   zEvent.EventType = 1064

	// 技能相关事件
	EventPlayerSkillUnlock  zEvent.EventType = 1070
	EventPlayerSkillLearn   zEvent.EventType = 1071
	EventPlayerSkillUpgrade zEvent.EventType = 1072
	EventPlayerSkillUse     zEvent.EventType = 1073

	// 任务相关事件
	EventQuestAccept        zEvent.EventType = 1080
	EventQuestProgress      zEvent.EventType = 1081
	EventQuestComplete      zEvent.EventType = 1082
	EventQuestReadyToComplete zEvent.EventType = 1083
	EventQuestSubmit        zEvent.EventType = 1084
	EventQuestAbandon       zEvent.EventType = 1085

	// 邮件相关事件
	EventMailReceived zEvent.EventType = 1090
	EventMailRead     zEvent.EventType = 1091
	EventMailClaimed  zEvent.EventType = 1092

	// 聊天相关事件
	EventChatMessage zEvent.EventType = 1100

	// 拍卖相关事件
	EventAuctionBid    zEvent.EventType = 1110
	EventAuctionSold    zEvent.EventType = 1111
	EventAuctionCancel zEvent.EventType = 1112

	// 排行榜相关事件
	EventRankUpdate zEvent.EventType = 1120
)

// PlayerEventData 基础玩家事件数据
type PlayerEventData struct {
	PlayerID id.PlayerIdType
}

// PlayerExpEventData 玩家经验变化事件数据
type PlayerExpEventData struct {
	PlayerID  id.PlayerIdType
	Exp       int64
	TotalExp  int64
}

// PlayerLevelUpEventData 玩家升级事件数据
type PlayerLevelUpEventData struct {
	PlayerID   id.PlayerIdType
	OldLevel   int32
	NewLevel   int32
}

// PlayerGoldEventData 玩家金币变化事件数据
type PlayerGoldEventData struct {
	PlayerID  id.PlayerIdType
	Amount    int64
	TotalGold int64
}

// PlayerDiamondEventData 玩家钻石变化事件数据
type PlayerDiamondEventData struct {
	PlayerID     id.PlayerIdType
	Amount       int64
	TotalDiamond int64
}

// PlayerItemEventData 玩家物品事件数据
type PlayerItemEventData struct {
	PlayerID id.PlayerIdType
	ItemID   id.ItemIdType
	ItemCfgID int32
	Count    int32
	Slot     int32
}

// PlayerDamageEventData 玩家受伤事件数据
type PlayerDamageEventData struct {
	PlayerID   id.PlayerIdType
	AttackerID id.ObjectIdType
	Damage     int32
	CurrentHP  int32
}

// PlayerHealEventData 玩家治疗事件数据
type PlayerHealEventData struct {
	PlayerID  id.PlayerIdType
	HealerID  id.ObjectIdType
	HealAmount int32
	CurrentHP int32
}

// PlayerMapEventData 玩家地图事件数据
type PlayerMapEventData struct {
	PlayerID id.PlayerIdType
	MapID    id.MapIdType
	PosX     float32
	PosY     float32
	PosZ     float32
}

// PlayerGuildEventData 玩家公会事件数据
type PlayerGuildEventData struct {
	PlayerID id.PlayerIdType
	GuildID  id.GuildIdType
}

// PlayerTeamEventData 玩家队伍事件数据
type PlayerTeamEventData struct {
	PlayerID id.PlayerIdType
	TeamID   id.TeamIdType
}

// PlayerFriendEventData 玩家好友事件数据
type PlayerFriendEventData struct {
	PlayerID id.PlayerIdType
	FriendID id.PlayerIdType
}

// FriendEventData 好友事件数据
type FriendEventData struct {
	PlayerID id.PlayerIdType
	FriendID id.PlayerIdType
}

// PlayerSkillEventData 玩家技能事件数据
type PlayerSkillEventData struct {
	PlayerID      id.PlayerIdType
	SkillConfigID int32
	SkillName     string
	Level         int32
	OldLevel      int32
	TargetID      id.ObjectIdType
}

// QuestEventData 任务事件数据
type QuestEventData struct {
	PlayerID id.PlayerIdType
	QuestID  int32
	Progress int32
}

// MailEventData 邮件事件数据
type MailEventData struct {
	PlayerID id.PlayerIdType
	MailID   id.MailIdType
}

// ChatMessageEventData 聊天消息事件数据
type ChatMessageEventData struct {
	PlayerID   id.PlayerIdType
	PlayerName string
	Channel    int32
	Content    string
	TargetID   id.PlayerIdType
	MapID      id.MapIdType
}

// GuildEventData 公会事件数据
type GuildEventData struct {
	PlayerID  id.PlayerIdType
	GuildID   id.GuildIdType
	GuildName string
}

// AuctionEventData 拍卖事件数据
type AuctionEventData struct {
	AuctionID id.AuctionIdType
	SellerID  id.PlayerIdType
	BuyerID   id.PlayerIdType
	Price      int64
}

// RankEventData 排行榜事件数据
type RankEventData struct {
	PlayerID id.PlayerIdType
	RankType int32
	Rank     int32
}

// NewEvent 创建游戏事件
func NewEvent(eventType zEvent.EventType, source interface{}, data interface{}) *zEvent.Event {
	return zEvent.NewEvent(eventType, source, data)
}

// GetGlobalEventBus 获取全局事件总线实例
func GetGlobalEventBus() *zEvent.EventBus {
	return zEvent.GetGlobalEventBus()
}

// Subscribe 订阅事件
func Subscribe(eventType zEvent.EventType, handler zEvent.EventHandler) {
	GetGlobalEventBus().Subscribe(eventType, handler)
}

// Publish 发布事件
func Publish(event *zEvent.Event) {
	GetGlobalEventBus().Publish(event)
}

// PublishSync 同步发布事件
func PublishSync(event *zEvent.Event) {
	GetGlobalEventBus().PublishSync(event)
}
