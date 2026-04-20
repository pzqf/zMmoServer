package sender

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zNet"
)

// MessageSender 消息发送器
type MessageSender struct {
	tcpClient *zNet.TcpClient
	token     string
	playerID  int64
}

// NewMessageSender 创建新的消息发送器
func NewMessageSender(tcpClient *zNet.TcpClient, token string) *MessageSender {
	return &MessageSender{
		tcpClient: tcpClient,
		token:     token,
	}
}

// SetTcpClient 设置TcpClient
func (s *MessageSender) SetTcpClient(tcpClient *zNet.TcpClient) {
	s.tcpClient = tcpClient
}

// SetToken 设置token
func (s *MessageSender) SetToken(token string) {
	s.token = token
}

// Send 发送消息
func (s *MessageSender) Send(msgID uint32, data []byte) error {
	if s.tcpClient == nil {
		return fmt.Errorf("client not connected")
	}
	return s.tcpClient.Send(zNet.ProtoIdType(msgID), data)
}

// SendHeartbeat 发送心跳消息
func (s *MessageSender) SendHeartbeat() error {
	// 心跳消息
	return s.Send(1, nil)
}

// SendTokenVerify 发送令牌验证消息
func (s *MessageSender) SendTokenVerify(token string) error {
	data := []byte(token)
	return s.Send(uint32(protocol.SystemMsgId_MSG_SYSTEM_TOKEN_VERIFY), data)
}

// SendPlayerLogin 发送玩家登录请求
func (s *MessageSender) SendPlayerLogin(playerID int64) error {
	// 玩家登录请求
	req := &protocol.PlayerLoginRequest{
		PlayerId: playerID,
		Token:    s.token,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME), data)
}

// SendPlayerCreate 发送角色创建请求
func (s *MessageSender) SendPlayerCreate(name string, sex, age int32) error {
	// 角色创建请求
	req := &protocol.PlayerCreateRequest{
		Name:       name,
		Sex:        sex,
		Age:        age,
		Profession: 1, // 默认职业
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.PlayerMsgId_MSG_PLAYER_CREATE), data)
}

func (s *MessageSender) SetPlayerID(playerID int64) {
	s.playerID = playerID
}

// SendPlayerLogout 发送玩家登出请求
func (s *MessageSender) SendPlayerLogout() error {
	req := &protocol.PlayerLogoutRequest{
		PlayerId: s.playerID,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME), data)
}

// SendMapEnter 发送进入地图请求
func (s *MessageSender) SendMapEnter(playerID int64, mapID int32) error {
	// 进入地图请求
	req := &protocol.ClientMapEnterRequest{
		PlayerId: playerID,
		MapId:    mapID,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.MapMsgId_MSG_MAP_ENTER), data)
}

// SendMapMove 发送移动请求
func (s *MessageSender) SendMapMove(playerID int64, mapID int32, x, y, z float32) error {
	// 移动请求
	req := &protocol.ClientMapMoveRequest{
		PlayerId: playerID,
		MapId:    mapID,
		Pos: &protocol.Position{
			X: x,
			Y: y,
			Z: z,
		},
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.MapMsgId_MSG_MAP_MOVE), data)
}

// SendMapAttack 发送攻击请求
func (s *MessageSender) SendMapAttack(playerID int64, mapID int32, targetID int64) error {
	// 攻击请求
	req := &protocol.ClientMapAttackRequest{
		PlayerId: playerID,
		MapId:    mapID,
		TargetId: targetID,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.MapMsgId_MSG_MAP_ATTACK), data)
}
