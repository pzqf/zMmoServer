package chat

import (
	"errors"
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// ChatManager 聊天管理器
type ChatManager struct {
	mu               sync.RWMutex
	playerID         id.PlayerIdType
	playerName       string
	guildID          id.GuildIdType
	teamID           id.TeamIdType
	currentMapID     id.MapIdType
	currentPos       common.Vector3
	mutedChannels    map[ChatChannel]bool
	mutedPlayers     map[id.PlayerIdType]bool
	lastSpeakTime    map[ChatChannel]int64
	cooldownTime     map[ChatChannel]int64 // 各频道冷却时间（毫秒）
	maxMessageLength int32
	isGM             bool
}

// NewChatManager 创建聊天管理器
func NewChatManager(playerID id.PlayerIdType, playerName string) *ChatManager {
	return &ChatManager{
		playerID:         playerID,
		playerName:       playerName,
		mutedChannels:    make(map[ChatChannel]bool),
		mutedPlayers:     make(map[id.PlayerIdType]bool),
		lastSpeakTime:    make(map[ChatChannel]int64),
		cooldownTime: map[ChatChannel]int64{
			ChatChannelWorld:   5000,  // 世界频道5秒冷却
			ChatChannelGuild:   1000,  // 公会频道1秒冷却
			ChatChannelTeam:    1000,  // 队伍频道1秒冷却
			ChatChannelPrivate: 500,   // 私聊0.5秒冷却
			ChatChannelMap:     2000,  // 地图频道2秒冷却
			ChatChannelNearby:  1000,  // 附近频道1秒冷却
		},
		maxMessageLength: 200,
		isGM:             false,
	}
}

// GetPlayerID 获取玩家ID
func (cm *ChatManager) GetPlayerID() id.PlayerIdType {
	return cm.playerID
}

// GetPlayerName 获取玩家名称
func (cm *ChatManager) GetPlayerName() string {
	return cm.playerName
}

// SetGuildID 设置公会ID
func (cm *ChatManager) SetGuildID(guildID id.GuildIdType) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.guildID = guildID
}

// GetGuildID 获取公会ID
func (cm *ChatManager) GetGuildID() id.GuildIdType {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.guildID
}

// SetTeamID 设置队伍ID
func (cm *ChatManager) SetTeamID(teamID id.TeamIdType) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.teamID = teamID
}

// GetTeamID 获取队伍ID
func (cm *ChatManager) GetTeamID() id.TeamIdType {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.teamID
}

// SetCurrentMapID 设置当前地图ID
func (cm *ChatManager) SetCurrentMapID(mapID id.MapIdType) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.currentMapID = mapID
}

// GetCurrentMapID 获取当前地图ID
func (cm *ChatManager) GetCurrentMapID() id.MapIdType {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.currentMapID
}

// SetCurrentPosition 设置当前位置
func (cm *ChatManager) SetCurrentPosition(pos common.Vector3) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.currentPos = pos
}

// GetCurrentPosition 获取当前位置
func (cm *ChatManager) GetCurrentPosition() common.Vector3 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.currentPos
}

// SetGM 设置GM状态
func (cm *ChatManager) SetGM(isGM bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.isGM = isGM
}

// IsGM 检查是否是GM
func (cm *ChatManager) IsGM() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.isGM
}

// MuteChannel 屏蔽频道
func (cm *ChatManager) MuteChannel(channel ChatChannel) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.mutedChannels[channel] = true
}

// UnmuteChannel 取消屏蔽频道
func (cm *ChatManager) UnmuteChannel(channel ChatChannel) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.mutedChannels, channel)
}

// IsChannelMuted 检查频道是否被屏蔽
func (cm *ChatManager) IsChannelMuted(channel ChatChannel) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.mutedChannels[channel]
}

// MutePlayer 屏蔽玩家
func (cm *ChatManager) MutePlayer(playerID id.PlayerIdType) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.mutedPlayers[playerID] = true
}

// UnmutePlayer 取消屏蔽玩家
func (cm *ChatManager) UnmutePlayer(playerID id.PlayerIdType) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.mutedPlayers, playerID)
}

// IsPlayerMuted 检查玩家是否被屏蔽
func (cm *ChatManager) IsPlayerMuted(playerID id.PlayerIdType) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.mutedPlayers[playerID]
}

// CanSendMessage 检查是否可以发送消息
func (cm *ChatManager) CanSendMessage(channel ChatChannel, currentTime int64) (bool, string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// GM不受限制
	if cm.isGM {
		return true, ""
	}

	// 检查频道是否被屏蔽
	if cm.mutedChannels[channel] {
		return false, "channel is muted"
	}

	// 检查冷却时间
	if cooldown, exists := cm.cooldownTime[channel]; exists && cooldown > 0 {
		lastTime := cm.lastSpeakTime[channel]
		if currentTime-lastTime < cooldown {
			return false, "message too frequent"
		}
	}

	return true, ""
}

// SendWorldMessage 发送世界频道消息
func (cm *ChatManager) SendWorldMessage(content string, currentTime int64) (*ChatMessage, error) {
	if ok, reason := cm.CanSendMessage(ChatChannelWorld, currentTime); !ok {
		return nil, errors.New(reason)
	}

	if len(content) > int(cm.maxMessageLength) {
		return nil, errors.New("message too long")
	}

	msg := NewChatMessage(ChatChannelWorld, cm.playerID, cm.playerName, content)
	cm.updateLastSpeakTime(ChatChannelWorld, currentTime)

	cm.publishChatMessageEvent(msg)

	zLog.Debug("World message sent",
		zap.Int64("player_id", int64(cm.playerID)),
		zap.String("content", content))

	return msg, nil
}

// SendGuildMessage 发送公会频道消息
func (cm *ChatManager) SendGuildMessage(content string, currentTime int64) (*ChatMessage, error) {
	if cm.guildID == 0 {
		return nil, errors.New("not in guild")
	}

	if ok, reason := cm.CanSendMessage(ChatChannelGuild, currentTime); !ok {
		return nil, errors.New(reason)
	}

	if len(content) > int(cm.maxMessageLength) {
		return nil, errors.New("message too long")
	}

	msg := NewChatMessage(ChatChannelGuild, cm.playerID, cm.playerName, content)
	cm.updateLastSpeakTime(ChatChannelGuild, currentTime)

	cm.publishChatMessageEvent(msg)

	return msg, nil
}

// SendTeamMessage 发送队伍频道消息
func (cm *ChatManager) SendTeamMessage(content string, currentTime int64) (*ChatMessage, error) {
	if cm.teamID == 0 {
		return nil, errors.New("not in team")
	}

	if ok, reason := cm.CanSendMessage(ChatChannelTeam, currentTime); !ok {
		return nil, errors.New(reason)
	}

	if len(content) > int(cm.maxMessageLength) {
		return nil, errors.New("message too long")
	}

	msg := NewChatMessage(ChatChannelTeam, cm.playerID, cm.playerName, content)
	cm.updateLastSpeakTime(ChatChannelTeam, currentTime)

	cm.publishChatMessageEvent(msg)

	return msg, nil
}

// SendPrivateMessage 发送私聊消息
func (cm *ChatManager) SendPrivateMessage(targetID id.PlayerIdType, targetName string, content string, currentTime int64) (*ChatMessage, error) {
	if ok, reason := cm.CanSendMessage(ChatChannelPrivate, currentTime); !ok {
		return nil, errors.New(reason)
	}

	if len(content) > int(cm.maxMessageLength) {
		return nil, errors.New("message too long")
	}

	msg := NewPrivateMessage(cm.playerID, cm.playerName, targetID, content)
	cm.updateLastSpeakTime(ChatChannelPrivate, currentTime)

	cm.publishChatMessageEvent(msg)

	zLog.Debug("Private message sent",
		zap.Int64("player_id", int64(cm.playerID)),
		zap.Int64("target_id", int64(targetID)),
		zap.String("content", content))

	return msg, nil
}

// SendMapMessage 发送地图频道消息
func (cm *ChatManager) SendMapMessage(content string, currentTime int64) (*ChatMessage, error) {
	if cm.currentMapID == 0 {
		return nil, errors.New("not in map")
	}

	if ok, reason := cm.CanSendMessage(ChatChannelMap, currentTime); !ok {
		return nil, errors.New(reason)
	}

	if len(content) > int(cm.maxMessageLength) {
		return nil, errors.New("message too long")
	}

	msg := NewMapMessage(cm.playerID, cm.playerName, cm.currentMapID, cm.currentPos.X, cm.currentPos.Y, cm.currentPos.Z, content)
	cm.updateLastSpeakTime(ChatChannelMap, currentTime)

	cm.publishChatMessageEvent(msg)

	zLog.Debug("Map message sent",
		zap.Int64("player_id", int64(cm.playerID)),
		zap.Int32("map_id", int32(cm.currentMapID)),
		zap.String("content", content))

	return msg, nil
}

// SendNearbyMessage 发送附近频道消息
func (cm *ChatManager) SendNearbyMessage(content string, currentTime int64) (*ChatMessage, error) {
	if ok, reason := cm.CanSendMessage(ChatChannelNearby, currentTime); !ok {
		return nil, errors.New(reason)
	}

	if len(content) > int(cm.maxMessageLength) {
		return nil, errors.New("message too long")
	}

	msg := NewNearbyMessage(cm.playerID, cm.playerName, cm.currentPos.X, cm.currentPos.Y, cm.currentPos.Z, content)
	cm.updateLastSpeakTime(ChatChannelNearby, currentTime)

	cm.publishChatMessageEvent(msg)

	zLog.Debug("Nearby message sent",
		zap.Int64("player_id", int64(cm.playerID)),
		zap.String("content", content))

	return msg, nil
}

// SendSystemMessage 发送系统消息（GM专用）
func (cm *ChatManager) SendSystemMessage(content string) (*ChatMessage, error) {
	if !cm.isGM {
		return nil, errors.New("permission denied")
	}

	msg := NewSystemMessage(content)

	cm.publishChatMessageEvent(msg)

	zLog.Info("System message sent",
		zap.Int64("player_id", int64(cm.playerID)),
		zap.String("content", content))

	return msg, nil
}

// ReceiveMessage 接收消息
func (cm *ChatManager) ReceiveMessage(msg *ChatMessage) bool {
	if msg == nil {
		return false
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 检查频道是否被屏蔽
	if cm.mutedChannels[msg.GetChannel()] {
		return false
	}

	// 检查发送者是否被屏蔽
	if cm.mutedPlayers[msg.GetSenderID()] {
		return false
	}

	// 私聊消息检查
	if msg.IsPrivateMessage() && msg.GetTargetID() != cm.playerID {
		return false
	}

	// 地图频道消息检查
	if msg.IsMapMessage() && msg.GetMapID() != cm.currentMapID {
		return false
	}

	// 附近频道消息检查（距离判断）
	if msg.IsNearbyMessage() {
		msgX, msgY, msgZ := msg.GetPosition()
		distance := cm.currentPos.DistanceTo(common.Vector3{X: msgX, Y: msgY, Z: msgZ})
		if distance > 500 { // 附近频道范围500
			return false
		}
	}

	return true
}

// updateLastSpeakTime 更新最后发言时间
func (cm *ChatManager) updateLastSpeakTime(channel ChatChannel, currentTime int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.lastSpeakTime[channel] = currentTime
}

// publishChatMessageEvent 发布聊天消息事件
func (cm *ChatManager) publishChatMessageEvent(msg *ChatMessage) {
	event.Publish(event.NewEvent(event.EventChatMessage, cm, &event.ChatMessageEventData{
		PlayerID:   cm.playerID,
		PlayerName: cm.playerName,
		Channel:    int32(msg.GetChannel()),
		Content:    msg.GetContent(),
		TargetID:   msg.GetTargetID(),
		MapID:      msg.GetMapID(),
	}))
}
