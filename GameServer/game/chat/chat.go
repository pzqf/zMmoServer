package chat

import (
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
)

// ChatChannel 聊天频道类型
type ChatChannel int32

const (
	ChatChannelWorld   ChatChannel = 1 // 世界频道
	ChatChannelGuild   ChatChannel = 2 // 公会频道
	ChatChannelTeam    ChatChannel = 3 // 队伍频道
	ChatChannelPrivate ChatChannel = 4 // 私聊频道
	ChatChannelMap     ChatChannel = 5 // 地图频道
	ChatChannelNearby  ChatChannel = 6 // 附近频道
	ChatChannelSystem  ChatChannel = 7 // 系统频道
	ChatChannelGM      ChatChannel = 8 // GM频道
)

// ChatMessageType 消息类型
type ChatMessageType int32

const (
	ChatMessageTypeText  ChatMessageType = 1 // 文本消息
	ChatMessageTypeVoice ChatMessageType = 2 // 语音消息
	ChatMessageTypeImage ChatMessageType = 3 // 图片消息
	ChatMessageTypeEmoji ChatMessageType = 4 // 表情消息
)

// ChatMessage 聊天消息
type ChatMessage struct {
	mu         sync.RWMutex
	messageID  int64
	channel    ChatChannel
	msgType    ChatMessageType
	senderID   id.PlayerIdType
	senderName string
	targetID   id.PlayerIdType
	content    string
	extraData  string
	sendTime   int64
	mapID      id.MapIdType
	posX       float32
	posY       float32
	posZ       float32
}

// NewChatMessage 创建聊天消息
func NewChatMessage(channel ChatChannel, senderID id.PlayerIdType, senderName string, content string) *ChatMessage {
	return &ChatMessage{
		channel:    channel,
		msgType:    ChatMessageTypeText,
		senderID:   senderID,
		senderName: senderName,
		content:    content,
		sendTime:   time.Now().UnixMilli(),
	}
}

// NewPrivateMessage 创建私聊消息
func NewPrivateMessage(senderID id.PlayerIdType, senderName string, targetID id.PlayerIdType, content string) *ChatMessage {
	msg := NewChatMessage(ChatChannelPrivate, senderID, senderName, content)
	msg.targetID = targetID
	return msg
}

// NewMapMessage 创建地图频道消息
func NewMapMessage(senderID id.PlayerIdType, senderName string, mapID id.MapIdType, posX, posY, posZ float32, content string) *ChatMessage {
	msg := NewChatMessage(ChatChannelMap, senderID, senderName, content)
	msg.mapID = mapID
	msg.posX = posX
	msg.posY = posY
	msg.posZ = posZ
	return msg
}

// NewNearbyMessage 创建附近频道消息
func NewNearbyMessage(senderID id.PlayerIdType, senderName string, posX, posY, posZ float32, content string) *ChatMessage {
	msg := NewChatMessage(ChatChannelNearby, senderID, senderName, content)
	msg.posX = posX
	msg.posY = posY
	msg.posZ = posZ
	return msg
}

// NewSystemMessage 创建系统消息
func NewSystemMessage(content string) *ChatMessage {
	return &ChatMessage{
		channel:    ChatChannelSystem,
		msgType:    ChatMessageTypeText,
		senderName: "系统",
		content:    content,
		sendTime:   time.Now().UnixMilli(),
	}
}

// GetMessageID 获取消息ID
func (m *ChatMessage) GetMessageID() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.messageID
}

// SetMessageID 设置消息ID
func (m *ChatMessage) SetMessageID(messageID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageID = messageID
}

// GetChannel 获取频道
func (m *ChatMessage) GetChannel() ChatChannel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.channel
}

// GetMessageType 获取消息类型
func (m *ChatMessage) GetMessageType() ChatMessageType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.msgType
}

// SetMessageType 设置消息类型
func (m *ChatMessage) SetMessageType(msgType ChatMessageType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgType = msgType
}

// GetSenderID 获取发送者ID
func (m *ChatMessage) GetSenderID() id.PlayerIdType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.senderID
}

// GetSenderName 获取发送者名称
func (m *ChatMessage) GetSenderName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.senderName
}

// GetTargetID 获取目标ID
func (m *ChatMessage) GetTargetID() id.PlayerIdType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.targetID
}

// SetTargetID 设置目标ID
func (m *ChatMessage) SetTargetID(targetID id.PlayerIdType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.targetID = targetID
}

// GetContent 获取内容
func (m *ChatMessage) GetContent() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.content
}

// SetContent 设置内容
func (m *ChatMessage) SetContent(content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.content = content
}

// GetExtraData 获取额外数据
func (m *ChatMessage) GetExtraData() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.extraData
}

// SetExtraData 设置额外数据
func (m *ChatMessage) SetExtraData(extraData string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.extraData = extraData
}

// GetSendTime 获取发送时间
func (m *ChatMessage) GetSendTime() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sendTime
}

// GetMapID 获取地图ID
func (m *ChatMessage) GetMapID() id.MapIdType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mapID
}

// GetPosition 获取位置
func (m *ChatMessage) GetPosition() (float32, float32, float32) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.posX, m.posY, m.posZ
}

// IsSystemMessage 检查是否是系统消息
func (m *ChatMessage) IsSystemMessage() bool {
	return m.GetChannel() == ChatChannelSystem
}

// IsPrivateMessage 检查是否是私聊消息
func (m *ChatMessage) IsPrivateMessage() bool {
	return m.GetChannel() == ChatChannelPrivate
}

// IsMapMessage 检查是否是地图频道消息
func (m *ChatMessage) IsMapMessage() bool {
	return m.GetChannel() == ChatChannelMap
}

// IsNearbyMessage 检查是否是附近频道消息
func (m *ChatMessage) IsNearbyMessage() bool {
	return m.GetChannel() == ChatChannelNearby
}

// Clone 克隆消息
func (m *ChatMessage) Clone() *ChatMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &ChatMessage{
		messageID:  m.messageID,
		channel:    m.channel,
		msgType:    m.msgType,
		senderID:   m.senderID,
		senderName: m.senderName,
		targetID:   m.targetID,
		content:    m.content,
		extraData:  m.extraData,
		sendTime:   m.sendTime,
		mapID:      m.mapID,
		posX:       m.posX,
		posY:       m.posY,
		posZ:       m.posZ,
	}
}
