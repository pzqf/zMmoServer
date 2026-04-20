package player

import (
	"time"

	"github.com/pzqf/zEngine/zActor"
	"github.com/pzqf/zCommon/common/id"
)

type MessageType uint32

const (
	// 网络协议映射 (0-999) - 与 Protobuf PlayerMsgId/MapMsgId 对应
	MsgNetEnterGame MessageType = 202 // MSG_PLAYER_ENTER_GAME
	MsgNetLeaveGame MessageType = 204 // MSG_PLAYER_LEAVE_GAME
	MsgNetMapEnter  MessageType = 1200 // MSG_MAP_ENTER
	MsgNetMapLeave  MessageType = 1202 // MSG_MAP_LEAVE
	MsgNetMapMove   MessageType = 1204 // MSG_MAP_MOVE
	MsgNetMapAttack MessageType = 1206 // MSG_MAP_ATTACK

	// 资源相关 (10000-10999)
	MsgAddGold       MessageType = 10001
	MsgDeductGold    MessageType = 10002
	MsgAddDiamond    MessageType = 10003
	MsgDeductDiamond MessageType = 10004

	// 物品相关 (11000-11999)
	MsgAddItem     MessageType = 11001
	MsgRemoveItem  MessageType = 11002
	MsgUseItem     MessageType = 11003
	MsgEquipItem   MessageType = 11004
	MsgUnequipItem MessageType = 11005

	// 技能相关 (12000-12999)
	MsgLearnSkill   MessageType = 12001
	MsgUseSkill     MessageType = 12002
	MsgUpgradeSkill MessageType = 12003

	// 任务相关 (13000-13999)
	MsgAcceptQuest   MessageType = 13001
	MsgCompleteQuest MessageType = 13002

	// Buff相关 (14000-14999)
	MsgAddBuff    MessageType = 14001
	MsgRemoveBuff MessageType = 14002

	// 社交相关 (15000-15999)
	MsgAddFriend    MessageType = 15001
	MsgRemoveFriend MessageType = 15002

	// 队伍相关 (16000-16999)
	MsgCreateTeam MessageType = 16001
	MsgJoinTeam   MessageType = 16002
	MsgLeaveTeam  MessageType = 16003

	// AOI 视野同步 (17000-17999)
	MsgAOIEnterView MessageType = 17001
	MsgAOILeaveView MessageType = 17002
	MsgAOIMove      MessageType = 17003
)

type MessageSource int

const (
	SourceGateway   MessageSource = iota
	SourceMapServer
	SourceAuction
)

type NetEnterGameRequest struct {
	PlayerID id.PlayerIdType
}

type NetLeaveGameRequest struct {
	PlayerID id.PlayerIdType
}

type NetMapEnterRequest struct {
	PlayerID id.PlayerIdType
	MapID    id.MapIdType
	PosX     float32
	PosY     float32
	PosZ     float32
}

type NetMapLeaveRequest struct {
	PlayerID id.PlayerIdType
	MapID    id.MapIdType
}

type NetMapMoveRequest struct {
	PlayerID id.PlayerIdType
	MapID    id.MapIdType
	PosX     float32
	PosY     float32
	PosZ     float32
}

type NetMapAttackRequest struct {
	PlayerID id.PlayerIdType
	MapID    id.MapIdType
	TargetID id.ObjectIdType
}

type NetResponse struct {
	ProtoId int32
	Data    []byte
}

type GoldRequest struct {
	Amount int64
}

type DiamondRequest struct {
	Amount int64
}

type AddItemRequest struct {
	ItemID    id.ItemIdType
	ItemCount int32
}

type RemoveItemRequest struct {
	ItemID    id.ItemIdType
	ItemCount int32
	Slot      int32
}

type UseItemRequest struct {
	ItemID id.ItemIdType
	Slot   int32
}

type EquipRequest struct {
	Slot      int32
	EquipSlot int32
}

type SkillRequest struct {
	SkillID int32
}

type BuffRequest struct {
	BuffID   int32
	Duration time.Duration
}

type QuestRequest struct {
	QuestID int64
}

type AddFriendRequest struct {
	FriendID id.PlayerIdType
}

type RemoveFriendRequest struct {
	FriendID id.PlayerIdType
}

type CreateTeamRequest struct {
	TeamName string
}

type JoinTeamRequest struct {
	TeamID id.TeamIdType
}

type LeaveTeamRequest struct{}

type AOIViewRequest struct {
	WatcherID id.PlayerIdType
	TargetID  int64
	MapID     id.MapIdType
	PosX      float32
	PosY      float32
	PosZ      float32
	OldPosX   float32
	OldPosY   float32
	OldPosZ   float32
}

type BaseResponse struct {
	Success bool
	Error   string
}

type GoldResponse struct {
	BaseResponse
	CurrentGold int64
}

type DiamondResponse struct {
	BaseResponse
	CurrentDiamond int64
}

type ItemResponse struct {
	BaseResponse
	ItemID    id.ItemIdType
	ItemCount int32
	Slot      int32
}

type EquipResponse struct {
	BaseResponse
	EquipSlot int32
}

type SkillResponse struct {
	BaseResponse
	SkillID int32
}

type BuffResponse struct {
	BaseResponse
	BuffID int32
}

type QuestResponse struct {
	BaseResponse
	QuestID int64
}

type FriendResponse struct {
	BaseResponse
	FriendID id.PlayerIdType
}

type TeamResponse struct {
	BaseResponse
	TeamID id.TeamIdType
}

type PlayerMessage struct {
	zActor.BaseActorMessage
	Source   MessageSource
	Type     MessageType
	Data     interface{}
	Callback chan interface{}
}

func NewGoldRequest(amount int64) *GoldRequest {
	return &GoldRequest{Amount: amount}
}

func NewDiamondRequest(amount int64) *DiamondRequest {
	return &DiamondRequest{Amount: amount}
}

func NewAddItemRequest(itemID id.ItemIdType, count int32) *AddItemRequest {
	return &AddItemRequest{ItemID: itemID, ItemCount: count}
}

func NewRemoveItemRequest(itemID id.ItemIdType, count int32, slot int32) *RemoveItemRequest {
	return &RemoveItemRequest{ItemID: itemID, ItemCount: count, Slot: slot}
}

func NewUseItemRequest(itemID id.ItemIdType, slot int32) *UseItemRequest {
	return &UseItemRequest{ItemID: itemID, Slot: slot}
}

func NewSkillRequest(skillID int32) *SkillRequest {
	return &SkillRequest{SkillID: skillID}
}

func NewBuffRequest(buffID int32, duration time.Duration) *BuffRequest {
	return &BuffRequest{BuffID: buffID, Duration: duration}
}

func NewQuestRequest(questID int64) *QuestRequest {
	return &QuestRequest{QuestID: questID}
}

func NewEquipRequest(slot int32, equipSlot int32) *EquipRequest {
	return &EquipRequest{Slot: slot, EquipSlot: equipSlot}
}

func NewPlayerMessage(playerID id.PlayerIdType, source MessageSource, msgType MessageType, data interface{}) *PlayerMessage {
	return &PlayerMessage{
		BaseActorMessage: zActor.BaseActorMessage{ActorID: int64(playerID)},
		Source:           source,
		Type:             msgType,
		Data:             data,
	}
}

func NewPlayerMessageWithCallback(playerID id.PlayerIdType, source MessageSource, msgType MessageType, data interface{}) (*PlayerMessage, chan interface{}) {
	callback := make(chan interface{}, 1)
	msg := &PlayerMessage{
		BaseActorMessage: zActor.BaseActorMessage{ActorID: int64(playerID)},
		Source:           source,
		Type:             msgType,
		Data:             data,
		Callback:         callback,
	}
	return msg, callback
}

func NewAddGoldMessage(playerID id.PlayerIdType, source MessageSource, amount int64) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgAddGold, NewGoldRequest(amount))
}

func NewDeductGoldMessage(playerID id.PlayerIdType, source MessageSource, amount int64) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgDeductGold, NewGoldRequest(amount))
}

func NewAddDiamondMessage(playerID id.PlayerIdType, source MessageSource, amount int64) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgAddDiamond, NewDiamondRequest(amount))
}

func NewDeductDiamondMessage(playerID id.PlayerIdType, source MessageSource, amount int64) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgDeductDiamond, NewDiamondRequest(amount))
}

func NewAddItemMessage(playerID id.PlayerIdType, source MessageSource, itemID id.ItemIdType, count int32) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgAddItem, NewAddItemRequest(itemID, count))
}

func NewRemoveItemMessage(playerID id.PlayerIdType, source MessageSource, itemID id.ItemIdType, count int32, slot int32) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgRemoveItem, NewRemoveItemRequest(itemID, count, slot))
}

func NewUseItemMessage(playerID id.PlayerIdType, source MessageSource, itemID id.ItemIdType, slot int32) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgUseItem, NewUseItemRequest(itemID, slot))
}

func NewLearnSkillMessage(playerID id.PlayerIdType, source MessageSource, skillID int32) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgLearnSkill, NewSkillRequest(skillID))
}

func NewUseSkillMessage(playerID id.PlayerIdType, source MessageSource, skillID int32) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgUseSkill, NewSkillRequest(skillID))
}

func NewAddBuffMessage(playerID id.PlayerIdType, source MessageSource, buffID int32, duration time.Duration) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgAddBuff, NewBuffRequest(buffID, duration))
}

func NewAcceptQuestMessage(playerID id.PlayerIdType, source MessageSource, questID int64) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgAcceptQuest, NewQuestRequest(questID))
}

func NewEquipItemMessage(playerID id.PlayerIdType, source MessageSource, slot int32, equipSlot int32) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgEquipItem, NewEquipRequest(slot, equipSlot))
}

// ProtoToMessageType 将 Protobuf 协议 ID 映射为 Actor MessageType
func ProtoToMessageType(protoId int32) (MessageType, bool) {
	switch protoId {
	case 202:
		return MsgNetEnterGame, true
	case 204:
		return MsgNetLeaveGame, true
	case 1200:
		return MsgNetMapEnter, true
	case 1202:
		return MsgNetMapLeave, true
	case 1204:
		return MsgNetMapMove, true
	case 1206:
		return MsgNetMapAttack, true
	default:
		return 0, false
	}
}

// IsNetProto 判断协议 ID 是否为网络协议（需要通过 Actor 投递）
func IsNetProto(protoId int32) bool {
	_, ok := ProtoToMessageType(protoId)
	return ok
}
