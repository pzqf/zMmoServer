package message

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"github.com/pzqf/zMmoServer/GameServer/game/trade"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// TradeHandler 网关侧玩家间交易处理器（业务层建设 2026-07-25）。
// 引入「两方有状态事务 + 跨玩家一致的原子金币交换」这条新架构路径：会话态由 TradeManager 权威持有，
// 双方确认后由本 handler 执行 trade.ExecuteSwap（原子 CAS + 回滚），状态经 UPDATE_NOTIFY 推给双方。
type TradeHandler struct {
	playerManager *player.PlayerManager
	tradeManager  *trade.TradeManager
	serverID      int32
}

func NewTradeHandler(pm *player.PlayerManager, serverID int32) *TradeHandler {
	h := &TradeHandler{playerManager: pm, tradeManager: trade.NewTradeManager(), serverID: serverID}
	// 玩家下线清理钩子（T3）：掉线/登出时取消其所在交易并通知对方，避免会话悬挂→重登被锁"already trading"。
	pm.RegisterOfflineHook(func(pid id.PlayerIdType) {
		if s, err := h.tradeManager.Cancel(pid); err == nil {
			h.broadcast(s, int32(protocol.TradeState_TRADE_CANCELLED), "对方掉线，交易取消")
		}
	})
	return h
}

func (h *TradeHandler) Handle(session zNet.Session, protoId int32, data []byte) error {
	switch protoId {
	case int32(protocol.TradeMsgId_MSG_TRADE_START):
		var req protocol.ClientTradeStartRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		s, err := h.tradeManager.Start(id.PlayerIdType(req.PlayerId), id.PlayerIdType(req.TargetId))
		if err != nil {
			zLog.Warn("Trade start failed", zap.Int64("from", req.PlayerId), zap.Int64("to", req.TargetId), zap.Error(err))
			return nil
		}
		zLog.Info("Trade started", zap.Int64("a", req.PlayerId), zap.Int64("b", req.TargetId))
		h.broadcast(s, int32(protocol.TradeState_TRADE_OPEN), "")

	case int32(protocol.TradeMsgId_MSG_TRADE_SET_GOLD):
		var req protocol.ClientTradeSetGoldRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		s, err := h.tradeManager.SetGold(id.PlayerIdType(req.PlayerId), req.Gold)
		if err != nil {
			zLog.Warn("Trade set gold failed", zap.Int64("player", req.PlayerId), zap.Error(err))
			return nil
		}
		h.broadcast(s, int32(protocol.TradeState_TRADE_OPEN), "")

	case int32(protocol.TradeMsgId_MSG_TRADE_CONFIRM):
		var req protocol.ClientTradeConfirmRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		s, both, err := h.tradeManager.Confirm(id.PlayerIdType(req.PlayerId))
		if err != nil {
			zLog.Warn("Trade confirm failed", zap.Int64("player", req.PlayerId), zap.Error(err))
			return nil
		}
		if !both {
			h.broadcast(s, int32(protocol.TradeState_TRADE_OPEN), "")
			return nil
		}
		// 双方确认 → 执行原子金币交换
		pA, errA := h.playerManager.GetPlayer(s.A)
		pB, errB := h.playerManager.GetPlayer(s.B)
		h.tradeManager.Finish(s.A, s.B)
		if errA != nil || errB != nil || pA == nil || pB == nil {
			h.broadcast(s, int32(protocol.TradeState_TRADE_CANCELLED), "对方不在线")
			return nil
		}
		ok, msg := trade.ExecuteSwap(pA, pB, s.GoldA, s.GoldB)
		if !ok {
			h.broadcast(s, int32(protocol.TradeState_TRADE_CANCELLED), msg)
			return nil
		}
		zLog.Info("Trade done", zap.Int64("a", int64(s.A)), zap.Int64("b", int64(s.B)),
			zap.Int64("goldA", s.GoldA), zap.Int64("goldB", s.GoldB))
		h.broadcast(s, int32(protocol.TradeState_TRADE_DONE), "")

	case int32(protocol.TradeMsgId_MSG_TRADE_CANCEL):
		var req protocol.ClientTradeCancelRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			return err
		}
		s, err := h.tradeManager.Cancel(id.PlayerIdType(req.PlayerId))
		if err != nil {
			return nil
		}
		h.broadcast(s, int32(protocol.TradeState_TRADE_CANCELLED), "已取消")
	}
	return nil
}

// broadcast 把交易状态推给双方（对称同一份）。
func (h *TradeHandler) broadcast(s *trade.Session, state int32, msg string) {
	notify := &protocol.ClientTradeUpdateNotify{
		State:      state,
		APlayerId:  int64(s.A),
		AGold:      s.GoldA,
		AConfirmed: s.ConfirmA,
		BPlayerId:  int64(s.B),
		BGold:      s.GoldB,
		BConfirmed: s.ConfirmB,
		Message:    msg,
	}
	out, err := proto.Marshal(notify)
	if err != nil {
		zLog.Error("Failed to marshal trade update", zap.Error(err))
		return
	}
	for _, pid := range []id.PlayerIdType{s.A, s.B} {
		h.playerManager.RouteMessage(pid,
			player.NewPlayerMessage(pid, player.SourceGateway, player.MsgTradeUpdate, &player.TradeUpdateData{Data: out}))
	}
}
