package message

import (
	"fmt"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"github.com/pzqf/zMmoServer/GameServer/session"
	playerservice "github.com/pzqf/zMmoServer/GameServer/services"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func getClientSessionID(session zNet.Session) zNet.SessionIdType {
	if tcpSess, ok := session.(*zNet.TcpServerSession); ok {
		if obj := tcpSess.GetObj(); obj != nil {
			if sid, ok := obj.(zNet.SessionIdType); ok {
				return sid
			}
		}
	}
	return session.GetSid()
}

type PlayerHandler struct {
	sessionManager *session.SessionManager
	playerManager  *player.PlayerManager
	playerService  *playerservice.PlayerService
	loginService   *player.LoginService
	serverID       int32
}

func NewPlayerHandler(sessionManager *session.SessionManager, playerManager *player.PlayerManager, playerService *playerservice.PlayerService, loginService *player.LoginService, serverID int32) *PlayerHandler {
	return &PlayerHandler{
		sessionManager: sessionManager,
		playerManager:  playerManager,
		playerService:  playerService,
		loginService:   loginService,
		serverID:       serverID,
	}
}

func (h *PlayerHandler) Handle(session zNet.Session, protoId int32, data []byte) error {
	switch protoId {
	case int32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME):
		return h.handlePlayerEnterGame(session, data)
	case int32(protocol.PlayerMsgId_MSG_PLAYER_CREATE):
		return h.handlePlayerCreate(session, data)
	case int32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME):
		return h.handlePlayerLeaveGame(session, data)
	default:
		zLog.Warn("Unknown player message", zap.Int32("proto_id", protoId))
		return nil
	}
}

func (h *PlayerHandler) sendToClient(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, protoId int32, data []byte) error {
	zLog.Info("sendToClient called",
		zap.Uint64("client_session_id", uint64(clientSessionID)),
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("proto_id", protoId),
		zap.Int("data_size", len(data)))

	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		SessionId: uint64(clientSessionID),
		PlayerId:  uint64(playerID),
		ServerId:  uint32(h.serverID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}

	crossMsg := &protocol.CrossServerMessage{
		TraceId:      uint64(time.Now().UnixNano()),
		FromServerId: uint32(h.serverID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeGateway),
		Message:      baseMsg,
	}

	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		zLog.Error("Failed to marshal cross server message", zap.Error(err))
		return err
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, h.serverID)
	wrappedData := crossserver.Wrap(meta, crossMsgData)

	zLog.Info("Sending to Gateway",
		zap.Int32("proto_id", protoId),
		zap.Int("data_size", len(wrappedData)),
		zap.Uint64("gw_session_id", uint64(gwSession.GetSid())))

	err = gwSession.Send(zNet.ProtoIdType(protoId), wrappedData)
	if err != nil {
		zLog.Error("Failed to send to Gateway", zap.Error(err))
		return err
	}

	zLog.Info("Sent to Gateway successfully", zap.Int32("proto_id", protoId))
	return nil
}

func (h *PlayerHandler) handlePlayerEnterGame(session zNet.Session, data []byte) error {
	var req protocol.PlayerLoginRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal player enter game request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	clientSessionID := getClientSessionID(session)
	sessionID := fmt.Sprintf("%d", clientSessionID)
	zLog.Info("Player enter game request", zap.Int64("player_id", int64(playerID)))

	if h.loginService != nil {
		if err := h.loginService.EnterGame(sessionID, playerID); err != nil {
			zLog.Error("Login service enter game failed", zap.Error(err))
			h.sendEnterGameResponse(session, clientSessionID, playerID, 1, err.Error(), nil)
			return nil
		}
	}

	h.sendEnterGameResponse(session, clientSessionID, playerID, 0, "", nil)

	return nil
}

func (h *PlayerHandler) handlePlayerCreate(netSession zNet.Session, data []byte) error {
	var req protocol.PlayerCreateRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal player create request", zap.Error(err))
		return err
	}

	zLog.Info("Player create request", zap.String("name", req.Name), zap.Int32("sex", req.Sex), zap.Int32("age", req.Age))

	clientSessionID := getClientSessionID(netSession)
	sessionID := fmt.Sprintf("%d", clientSessionID)

	sess, err := h.waitForSession(sessionID)
	if err != nil {
		h.sendPlayerCreateResponse(netSession, clientSessionID, 0, 1, err.Error(), nil)
		return nil
	}

	accountID := sess.AccountID

	var pid id.PlayerIdType

	if h.loginService != nil {
		pid, err = h.loginService.CreatePlayer(accountID, req.Name, req.Sex, req.Age)
	} else if h.playerService != nil {
		pid, err = h.playerService.CreatePlayer(accountID, req.Name, req.Sex, req.Age)
	} else {
		err = fmt.Errorf("service unavailable")
	}

	if err != nil {
		h.sendPlayerCreateResponse(netSession, clientSessionID, 0, 1, err.Error(), nil)
		return nil
	}

	playerInfo := &protocol.PlayerBasicInfo{
		PlayerId: int64(pid),
		Name:     req.Name,
		Level:    1,
		Gold:     1000,
	}

	h.sendPlayerCreateResponse(netSession, clientSessionID, pid, 0, "", playerInfo)
	return nil
}

func (h *PlayerHandler) handlePlayerLeaveGame(session zNet.Session, data []byte) error {
	var req protocol.PlayerLogoutRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		zLog.Error("Failed to unmarshal player leave game request", zap.Error(err))
		return err
	}

	playerID := id.PlayerIdType(req.PlayerId)
	clientSessionID := getClientSessionID(session)

	if playerID == 0 {
		sessionID := fmt.Sprintf("%d", clientSessionID)
		sess, exists := h.sessionManager.GetSession(sessionID)
		if exists && sess.PlayerID != 0 {
			playerID = id.PlayerIdType(sess.PlayerID)
		}
	}

	zLog.Info("Player leave game request", zap.Int64("player_id", int64(playerID)))

	if h.loginService != nil {
		if err := h.loginService.LeaveGame(playerID); err != nil {
			zLog.Error("Login service leave game failed", zap.Error(err))
		}
	}

	msg, callback := player.NewPlayerMessageWithCallback(
		playerID, player.SourceGateway, player.MsgNetLeaveGame,
		&player.NetLeaveGameRequest{PlayerID: playerID},
	)

	if err := h.playerManager.RouteMessage(playerID, msg); err != nil {
		zLog.Error("Failed to route leave game message", zap.Error(err))
		h.sendCommonResponse(session, clientSessionID, playerID, int32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME), 0)
		return nil
	}

	select {
	case resp := <-callback:
		if netResp, ok := resp.(*player.NetResponse); ok {
			return h.sendToClient(session, clientSessionID, playerID, int32(netResp.ProtoId), netResp.Data)
		}
		h.sendCommonResponse(session, clientSessionID, playerID, int32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME), 0)
	case <-time.After(5 * time.Second):
		zLog.Warn("Leave game timeout", zap.Int64("player_id", int64(playerID)))
		h.sendCommonResponse(session, clientSessionID, playerID, int32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME), 0)
	}

	return nil
}

func (h *PlayerHandler) sendEnterGameResponse(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, result int32, errMsg string, info *protocol.PlayerBasicInfo) {
	resp := &protocol.PlayerLoginResponse{
		Result:     result,
		ErrorMsg:   errMsg,
		PlayerInfo: info,
	}
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal enter game response", zap.Error(err))
		return
	}
	if err := h.sendToClient(gwSession, clientSessionID, playerID, int32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE), respData); err != nil {
		zLog.Error("Failed to send enter game response", zap.Error(err))
	}
}

func (h *PlayerHandler) sendCommonResponse(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, protoId int32, result int32) {
	resp := &protocol.CommonResponse{Result: result}
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal common response", zap.Error(err))
		return
	}
	if err := h.sendToClient(gwSession, clientSessionID, playerID, protoId, respData); err != nil {
		zLog.Error("Failed to send common response", zap.Error(err))
	}
}

func (h *PlayerHandler) sendPlayerCreateResponse(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, result int32, errMsg string, info *protocol.PlayerBasicInfo) {
	resp := &protocol.PlayerCreateResponse{
		Result:     result,
		ErrorMsg:   errMsg,
		PlayerInfo: info,
	}
	respData, err := proto.Marshal(resp)
	if err != nil {
		zLog.Error("Failed to marshal player create response", zap.Error(err))
		return
	}
	if err := h.sendToClient(gwSession, clientSessionID, playerID, int32(protocol.PlayerMsgId_MSG_PLAYER_CREATE_RESPONSE), respData); err != nil {
		zLog.Error("Failed to send player create response", zap.Error(err))
	}
}

func (h *PlayerHandler) waitForSession(sessionID string) (*session.Session, error) {
	for i := 0; i < 10; i++ {
		sess, exists := h.sessionManager.GetSession(sessionID)
		if exists && sess.AccountID != 0 {
			return sess, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil, fmt.Errorf("session not found or account not bound")
}
