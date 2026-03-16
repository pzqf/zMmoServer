package player

import (
	"github.com/pzqf/zEngine/zActor"
	"github.com/pzqf/zMmoShared/common/id"
)

// ==================== 消息类型定义 ====================

// MessageType 消息类型
type MessageType uint32

const (
	// 资源相关 (10000-10999)
	MsgAddGold       MessageType = 10001 // 增加金币
	MsgDeductGold    MessageType = 10002 // 扣除金币
	MsgAddDiamond    MessageType = 10003 // 增加钻石
	MsgDeductDiamond MessageType = 10004 // 扣除钻石

	// 物品相关 (11000-11999)
	MsgAddItem    MessageType = 11001 // 增加物品
	MsgRemoveItem MessageType = 11002 // 移除物品
	MsgUseItem    MessageType = 11003 // 使用物品

	// 任务相关 (12000-12999)
	MsgAcceptQuest   MessageType = 12001 // 接受任务
	MsgCompleteQuest MessageType = 12002 // 完成任务

	// 社交相关 (13000-13999)
	MsgAddFriend    MessageType = 13001 // 添加好友
	MsgRemoveFriend MessageType = 13002 // 移除好友

	// 队伍相关 (14000-14999)
	MsgCreateTeam MessageType = 14001 // 创建队伍
	MsgJoinTeam   MessageType = 14002 // 加入队伍
	MsgLeaveTeam  MessageType = 14003 // 离开队伍
)

// ==================== 消息来源定义 ====================

// MessageSource 消息来源
type MessageSource int

const (
	SourceGateway   MessageSource = iota // 来自Gateway（客户端）
	SourceMapServer                      // 来自MapServer
	SourceAuction                        // 来自拍卖行
)

// ==================== 请求消息定义 ====================

// GoldRequest 金币请求
type GoldRequest struct {
	Amount int64 // 数量
}

// DiamondRequest 钻石请求
type DiamondRequest struct {
	Amount int64 // 数量
}

// AddItemRequest 增加物品请求
type AddItemRequest struct {
	ItemID    id.ItemIdType // 物品ID
	ItemCount int32         // 物品数量
}

// RemoveItemRequest 移除物品请求
type RemoveItemRequest struct {
	ItemID    id.ItemIdType // 物品ID
	ItemCount int32         // 物品数量
}

// UseItemRequest 使用物品请求
type UseItemRequest struct {
	ItemID id.ItemIdType // 物品ID
	Slot   int32         // 槽位
}

// AddFriendRequest 添加好友请求
type AddFriendRequest struct {
	FriendID id.PlayerIdType // 好友ID
}

// RemoveFriendRequest 移除好友请求
type RemoveFriendRequest struct {
	FriendID id.PlayerIdType // 好友ID
}

// CreateTeamRequest 创建队伍请求
type CreateTeamRequest struct {
	TeamName string // 队伍名称
}

// JoinTeamRequest 加入队伍请求
type JoinTeamRequest struct {
	TeamID id.TeamIdType // 队伍ID
}

// LeaveTeamRequest 离开队伍请求
type LeaveTeamRequest struct {
}

// ==================== 响应消息定义 ====================

// BaseResponse 基础响应
type BaseResponse struct {
	Success bool   // 是否成功
	Error   string // 错误信息（失败时）
}

// GoldResponse 金币操作响应
type GoldResponse struct {
	BaseResponse
	CurrentGold int64 // 当前金币
}

// DiamondResponse 钻石操作响应
type DiamondResponse struct {
	BaseResponse
	CurrentDiamond int64 // 当前钻石
}

// ItemResponse 物品操作响应
type ItemResponse struct {
	BaseResponse
	ItemID    id.ItemIdType // 物品ID
	ItemCount int32         // 剩余数量
}

// FriendResponse 好友操作响应
type FriendResponse struct {
	BaseResponse
	FriendID id.PlayerIdType // 好友ID
}

// TeamResponse 队伍操作响应
type TeamResponse struct {
	BaseResponse
	TeamID id.TeamIdType // 队伍ID
}

// ==================== PlayerMessage 定义 ====================

// PlayerMessage 玩家消息，实现zActor.ActorMessage接口
type PlayerMessage struct {
	zActor.BaseActorMessage
	Source   MessageSource
	Type     MessageType
	Data     interface{}
	Callback chan interface{} // 回调通道
}

// ==================== 便捷构造函数 ====================

// NewGoldRequest 创建金币请求
func NewGoldRequest(amount int64) *GoldRequest {
	return &GoldRequest{Amount: amount}
}

// NewDiamondRequest 创建钻石请求
func NewDiamondRequest(amount int64) *DiamondRequest {
	return &DiamondRequest{Amount: amount}
}

// NewAddItemRequest 创建增加物品请求
func NewAddItemRequest(itemID id.ItemIdType, count int32) *AddItemRequest {
	return &AddItemRequest{ItemID: itemID, ItemCount: count}
}

// NewRemoveItemRequest 创建移除物品请求
func NewRemoveItemRequest(itemID id.ItemIdType, count int32) *RemoveItemRequest {
	return &RemoveItemRequest{ItemID: itemID, ItemCount: count}
}

// NewUseItemRequest 创建使用物品请求
func NewUseItemRequest(itemID id.ItemIdType, slot int32) *UseItemRequest {
	return &UseItemRequest{ItemID: itemID, Slot: slot}
}

// NewAddFriendRequest 创建添加好友请求
func NewAddFriendRequest(friendID id.PlayerIdType) *AddFriendRequest {
	return &AddFriendRequest{FriendID: friendID}
}

// NewRemoveFriendRequest 创建移除好友请求
func NewRemoveFriendRequest(friendID id.PlayerIdType) *RemoveFriendRequest {
	return &RemoveFriendRequest{FriendID: friendID}
}

// NewPlayerMessage 创建玩家消息
func NewPlayerMessage(playerID id.PlayerIdType, source MessageSource, msgType MessageType, data interface{}) *PlayerMessage {
	return &PlayerMessage{
		BaseActorMessage: zActor.BaseActorMessage{ActorID: int64(playerID)},
		Source:           source,
		Type:             msgType,
		Data:             data,
	}
}

// NewPlayerMessageWithCallback 创建带回调的玩家消息
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

// ==================== 便捷消息创建函数 ====================

// NewAddGoldMessage 创建增加金币消息
func NewAddGoldMessage(playerID id.PlayerIdType, source MessageSource, amount int64) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgAddGold, NewGoldRequest(amount))
}

// NewDeductGoldMessage 创建扣除金币消息
func NewDeductGoldMessage(playerID id.PlayerIdType, source MessageSource, amount int64) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgDeductGold, NewGoldRequest(amount))
}

// NewAddDiamondMessage 创建增加钻石消息
func NewAddDiamondMessage(playerID id.PlayerIdType, source MessageSource, amount int64) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgAddDiamond, NewDiamondRequest(amount))
}

// NewDeductDiamondMessage 创建扣除钻石消息
func NewDeductDiamondMessage(playerID id.PlayerIdType, source MessageSource, amount int64) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgDeductDiamond, NewDiamondRequest(amount))
}

// NewAddItemMessage 创建增加物品消息
func NewAddItemMessage(playerID id.PlayerIdType, source MessageSource, itemID id.ItemIdType, count int32) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgAddItem, NewAddItemRequest(itemID, count))
}

// NewRemoveItemMessage 创建移除物品消息
func NewRemoveItemMessage(playerID id.PlayerIdType, source MessageSource, itemID id.ItemIdType, count int32) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgRemoveItem, NewRemoveItemRequest(itemID, count))
}

// NewUseItemMessage 创建使用物品消息
func NewUseItemMessage(playerID id.PlayerIdType, source MessageSource, itemID id.ItemIdType, slot int32) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgUseItem, NewUseItemRequest(itemID, slot))
}

// NewAddFriendMessage 创建添加好友消息
func NewAddFriendMessage(playerID id.PlayerIdType, source MessageSource, friendID id.PlayerIdType) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgAddFriend, NewAddFriendRequest(friendID))
}

// NewRemoveFriendMessage 创建移除好友消息
func NewRemoveFriendMessage(playerID id.PlayerIdType, source MessageSource, friendID id.PlayerIdType) *PlayerMessage {
	return NewPlayerMessage(playerID, source, MsgRemoveFriend, NewRemoveFriendRequest(friendID))
}
