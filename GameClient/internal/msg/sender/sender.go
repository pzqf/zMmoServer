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

// —— 物品 / 仓库（业务层建设 2026-07-25）——

func (s *MessageSender) SendItemList(playerID int64) error {
	data, err := proto.Marshal(&protocol.ClientItemListRequest{PlayerId: playerID})
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.ItemMsgId_MSG_ITEM_LIST), data)
}

func (s *MessageSender) SendItemUse(playerID int64, slot, count int32) error {
	data, err := proto.Marshal(&protocol.ClientItemUseRequest{PlayerId: playerID, Slot: slot, Count: count})
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.ItemMsgId_MSG_ITEM_USE), data)
}

func (s *MessageSender) SendItemPickup(playerID int64, mapID int32) error {
	data, err := proto.Marshal(&protocol.ClientItemPickupRequest{PlayerId: playerID, MapId: mapID})
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.ItemMsgId_MSG_ITEM_PICKUP), data)
}

func (s *MessageSender) SendItemMove(playerID int64, from, to int32) error {
	data, err := proto.Marshal(&protocol.ClientItemMoveRequest{PlayerId: playerID, FromSlot: from, ToSlot: to})
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.ItemMsgId_MSG_ITEM_MOVE), data)
}

func (s *MessageSender) SendWarehouseList(playerID int64) error {
	data, err := proto.Marshal(&protocol.ClientWarehouseListRequest{PlayerId: playerID})
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.ItemMsgId_MSG_WAREHOUSE_LIST), data)
}

func (s *MessageSender) SendWarehouseStore(playerID int64, bagSlot, count int32) error {
	data, err := proto.Marshal(&protocol.ClientWarehouseStoreRequest{PlayerId: playerID, BagSlot: bagSlot, Count: count})
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.ItemMsgId_MSG_WAREHOUSE_STORE), data)
}

func (s *MessageSender) SendWarehouseRetrieve(playerID int64, warehouseSlot, count int32) error {
	data, err := proto.Marshal(&protocol.ClientWarehouseRetrieveRequest{PlayerId: playerID, WarehouseSlot: warehouseSlot, Count: count})
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.ItemMsgId_MSG_WAREHOUSE_RETRIEVE), data)
}

// —— 技能（业务层建设 2026-07-25）——

func (s *MessageSender) SendSkillList(playerID int64) error {
	data, err := proto.Marshal(&protocol.ClientSkillListRequest{PlayerId: playerID})
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.SkillMsgId_MSG_SKILL_LIST), data)
}

func (s *MessageSender) SendSkillLearn(playerID int64, skillID int32) error {
	data, err := proto.Marshal(&protocol.ClientSkillLearnRequest{PlayerId: playerID, SkillId: skillID})
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.SkillMsgId_MSG_SKILL_LEARN), data)
}

func (s *MessageSender) SendSkillUpgrade(playerID int64, skillID int32) error {
	data, err := proto.Marshal(&protocol.ClientSkillUpgradeRequest{PlayerId: playerID, SkillId: skillID})
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.SkillMsgId_MSG_SKILL_UPGRADE), data)
}

func (s *MessageSender) SendSkillCast(playerID int64, skillID int32, targetID int64) error {
	data, err := proto.Marshal(&protocol.ClientSkillCastRequest{PlayerId: playerID, SkillId: skillID, TargetId: targetID})
	if err != nil {
		return err
	}
	return s.Send(uint32(protocol.SkillMsgId_MSG_SKILL_CAST), data)
}
