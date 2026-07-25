package message

import (
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	playerservice "github.com/pzqf/zMmoServer/GameServer/services"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// MailHandler 网关侧邮件处理器（业务层建设 2026-07-25）。
// 测「离线持久异步投递 + 领取」这条新架构路径：发件直接落库（收件人在线与否均可），
// 收件人登录后拉取/领取，领取把金币加进其在线 actor（金币已随登出持久化）。
type MailHandler struct {
	playerManager *player.PlayerManager
	playerService *playerservice.PlayerService
	serverID      int32
}

func NewMailHandler(pm *player.PlayerManager, ps *playerservice.PlayerService, serverID int32) *MailHandler {
	return &MailHandler{playerManager: pm, playerService: ps, serverID: serverID}
}

func (h *MailHandler) sendToClient(gwSession zNet.Session, clientSessionID zNet.SessionIdType, playerID id.PlayerIdType, protoId int32, data []byte) error {
	baseMsg := &protocol.BaseMessage{
		MsgId: uint32(protoId), SessionId: uint64(clientSessionID), PlayerId: uint64(playerID),
		ServerId: uint32(h.serverID), Timestamp: uint64(time.Now().Unix()), Data: data,
	}
	crossMsg := &protocol.CrossServerMessage{
		TraceId: uint64(time.Now().UnixNano()), FromServerId: uint32(h.serverID),
		FromService: uint32(crossserver.ServiceTypeGame), ToService: uint32(crossserver.ServiceTypeGateway), Message: baseMsg,
	}
	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		return err
	}
	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, h.serverID)
	return gwSession.Send(zNet.ProtoIdType(protoId), crossserver.Wrap(meta, crossMsgData))
}

func (h *MailHandler) Handle(session zNet.Session, protoId int32, data []byte) error {
	clientSessionID := getClientSessionID(session)
	switch protoId {
	case int32(protocol.MailMsgId_MSG_MAIL_SEND):
		var req protocol.ClientMailSendRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		sender := "系统"
		if p, err := h.playerManager.GetPlayer(id.PlayerIdType(req.FromPlayerId)); err == nil && p != nil {
			sender = p.GetName()
		}
		if err := h.playerService.SendMail(req.ToPlayerId, sender, req.Title, req.Content, req.Gold); err != nil {
			zLog.Warn("Send mail failed", zap.Int64("to", req.ToPlayerId), zap.Error(err))
			return nil
		}
		zLog.Info("Mail sent", zap.Int64("from", req.FromPlayerId), zap.Int64("to", req.ToPlayerId), zap.Int64("gold", req.Gold))
		// 发件为 fire，无响应

	case int32(protocol.MailMsgId_MSG_MAIL_LIST):
		var req protocol.ClientMailListRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		mails, err := h.playerService.ListMail(req.PlayerId)
		resp := &protocol.ClientMailListResponse{}
		if err != nil {
			resp.Result = 1
		} else {
			for _, m := range mails {
				resp.Mails = append(resp.Mails, &protocol.MailInfo{
					MailId: m.MailID, Sender: m.Sender, Title: m.Title, Content: m.Content, Gold: m.Gold, IsClaimed: m.IsClaimed,
				})
			}
		}
		out, _ := proto.Marshal(resp)
		return h.sendToClient(session, clientSessionID, id.PlayerIdType(req.PlayerId), int32(protocol.MailMsgId_MSG_MAIL_LIST_RESPONSE), out)

	case int32(protocol.MailMsgId_MSG_MAIL_CLAIM):
		var req protocol.ClientMailClaimRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		gold, ok, err := h.playerService.ClaimMail(req.PlayerId, req.MailId)
		resp := &protocol.ClientMailClaimResponse{MailId: req.MailId}
		if err != nil || !ok {
			resp.Result = 1
			resp.Error = "邮件不存在/非本人/已领取"
		} else {
			// 金币加进领取者在线 actor（金币原子；随登出持久化）
			if gold > 0 {
				if p, e := h.playerManager.GetPlayer(id.PlayerIdType(req.PlayerId)); e == nil && p != nil {
					p.AddGold(gold)
				}
			}
			resp.Gold = gold
			zLog.Info("Mail claimed", zap.Int64("player", req.PlayerId), zap.Int64("mail", req.MailId), zap.Int64("gold", gold))
		}
		out, _ := proto.Marshal(resp)
		return h.sendToClient(session, clientSessionID, id.PlayerIdType(req.PlayerId), int32(protocol.MailMsgId_MSG_MAIL_CLAIM_RESPONSE), out)
	}
	return nil
}
