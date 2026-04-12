package message

import (
	"google.golang.org/protobuf/proto"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"github.com/pzqf/zMmoServer/GameServer/session"
	"go.uber.org/zap"
)

// PlayerHandler 玩家消息处理器
type PlayerHandler struct {
	sessionManager *session.SessionManager
	playerManager  *player.PlayerManager
}

// NewPlayerHandler 创建玩家消息处理器
func NewPlayerHandler(sessionManager *session.SessionManager, playerManager *player.PlayerManager) *PlayerHandler {
	return &PlayerHandler{
		sessionManager: sessionManager,
		playerManager:  playerManager,
	}
}

// Handle 处理玩家消息
func (h *PlayerHandler) Handle(session zNet.Session, protoId int32, data []byte) error {
	switch protoId {
	case int32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME):
		return h.handlePlayerEnterGame(session, data)
	case int32(protocol.PlayerMsgId_MSG_PLAYER_CREATE):
		return h.handlePlayerCreate(session, data)
	case int32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME):
		return h.handlePlayerLeaveGame(session, data)
	default:
		zLog.Info("Received unknown player message", zap.Int32("proto_id", protoId))
		return nil
	}
}

// handlePlayerEnterGame 处理玩家进入游戏
func (h *PlayerHandler) handlePlayerEnterGame(session zNet.Session, data []byte) error {
	var req protocol.PlayerLoginRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal player enter game request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	zLog.Info("Player enter game request", zap.Uint64("player_id", uint64(playerID)))

	// 获取玩家
	player, err := h.playerManager.GetPlayer(playerID)
	if err != nil {
		zLog.Error("Failed to get player", zap.Error(err))
		// 发送进入游戏失败的响应
		resp := &protocol.PlayerLoginResponse{
			Result:   1,
			ErrorMsg: err.Error(),
		}
		respData, err := proto.Marshal(resp)
		if err != nil {
			zLog.Error("Failed to marshal player enter game response", zap.Error(err))
			return err
		}
		err = session.Send(zNet.ProtoIdType(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE), respData)
		if err != nil {
			zLog.Error("Failed to send player enter game response", zap.Error(err))
			return err
		}
		return nil
	}

	zLog.Info("Player enter game successfully", zap.Uint64("player_id", uint64(player.GetPlayerID())))

	// 发送进入游戏成功的响应
	resp := &protocol.PlayerLoginResponse{
		Result: 0,
		PlayerInfo: &protocol.PlayerBasicInfo{
			PlayerId:      int64(player.GetPlayerID()),
			Name:          player.GetName(),
			Level:         1, // 默认等级
			Exp:           0,
			Gold:          0,
			VipLevel:      0,
			ServerId:      1, // 默认服务器ID
			CreateTime:    0,
			LastLoginTime: 0,
		},
	}

	// 序列化响应
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal player enter game response", zap.Error(err))
		return err
	}

	// 发送响应
	err = session.Send(zNet.ProtoIdType(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE), respData)
	if err != nil {
		zLog.Error("Failed to send player enter game response", zap.Error(err))
		return err
	}

	zLog.Info("Player enter game response sent", zap.Uint64("player_id", uint64(player.GetPlayerID())))
	return nil
}

// handlePlayerCreate 处理玩家创建角色
func (h *PlayerHandler) handlePlayerCreate(session zNet.Session, data []byte) error {
	var req protocol.PlayerCreateRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal player create request", zap.Error(err))
		return err
	}

	zLog.Info("Player create request", zap.String("name", req.Name), zap.Int32("sex", req.Sex), zap.Int32("age", req.Age))

	// 这里需要调用PlayerService创建角色
	// 暂时返回成功响应
	resp := &protocol.PlayerCreateResponse{
		Result: 0,
		PlayerInfo: &protocol.PlayerBasicInfo{
			PlayerId:      1, // 默认玩家ID
			Name:          req.Name,
			Level:         1, // 默认等级
			Exp:           0,
			Gold:          1000, // 初始金币
			VipLevel:      0,
			ServerId:      1, // 默认服务器ID
			CreateTime:    0,
			LastLoginTime: 0,
		},
	}

	// 序列化响应
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal player create response", zap.Error(err))
		return err
	}

	// 发送响应
	err = session.Send(zNet.ProtoIdType(protocol.PlayerMsgId_MSG_PLAYER_CREATE_RESPONSE), respData)
	if err != nil {
		zLog.Error("Failed to send player create response", zap.Error(err))
		return err
	}

	zLog.Info("Player create response sent", zap.String("name", req.Name))
	return nil
}

// handlePlayerLeaveGame 处理玩家离开游戏
func (h *PlayerHandler) handlePlayerLeaveGame(session zNet.Session, data []byte) error {
	// 处理玩家离开游戏逻辑
	zLog.Info("Player leave game request")

	// 发送离开游戏成功的响应
	resp := &protocol.CommonResponse{
		Result: 0,
	}

	// 序列化响应
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal player leave game response", zap.Error(err))
		return err
	}

	// 发送响应
	err = session.Send(zNet.ProtoIdType(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE), respData)
	if err != nil {
		zLog.Error("Failed to send player leave game response", zap.Error(err))
		return err
	}

	zLog.Info("Player leave game response sent")
	return nil
}
