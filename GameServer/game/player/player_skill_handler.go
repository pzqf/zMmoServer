package player

import (
	"github.com/pzqf/zCommon/game"
	"github.com/pzqf/zCommon/protocol"
	"google.golang.org/protobuf/proto"
)

// 技能 handler（业务层建设 2026-07-25）。均在 Player actor goroutine 上运行，操作 Player 自己的
// skillMgr（学习/升级/释放）。释放为本地校验（已学 + 冷却/消耗，见 game.SkillManager.UseSkill）；
// 对目标造成实际战斗伤害需经 MapServer 战斗权威，属后续更深集成，此处 target_id 已透传备用。

func (p *Player) replySkill(msg *PlayerMessage, protoId protocol.SkillMsgId, m proto.Message) {
	if msg.Callback == nil {
		return
	}
	data, err := proto.Marshal(m)
	if err != nil {
		p.sendErrorResponse(msg, err.Error())
		return
	}
	msg.Callback <- &NetResponse{ProtoId: int32(protoId), Data: data}
}

func (p *Player) handleNetSkillList(msg *PlayerMessage) {
	all := p.skillMgr.GetAllSkills()
	skills := make([]*protocol.SkillInfo, 0, len(all))
	for _, s := range all {
		skills = append(skills, &protocol.SkillInfo{SkillId: s.SkillID, Level: s.Level, MaxLevel: s.MaxLevel})
	}
	p.replySkill(msg, protocol.SkillMsgId_MSG_SKILL_LIST_RESPONSE, &protocol.ClientSkillListResponse{
		Result: 0, Skills: skills,
	})
}

func (p *Player) handleNetSkillLearn(msg *PlayerMessage) {
	req, ok := msg.Data.(*protocol.ClientSkillLearnRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}
	resp := &protocol.ClientSkillLearnResponse{SkillId: req.SkillId}
	if err := p.skillMgr.LearnSkill(game.NewSkill(req.SkillId, "", game.SkillTypeActive)); err != nil {
		resp.Result = 1
		resp.Error = err.Error()
	}
	p.replySkill(msg, protocol.SkillMsgId_MSG_SKILL_LEARN_RESPONSE, resp)
}

func (p *Player) handleNetSkillUpgrade(msg *PlayerMessage) {
	req, ok := msg.Data.(*protocol.ClientSkillUpgradeRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}
	resp := &protocol.ClientSkillUpgradeResponse{SkillId: req.SkillId}
	if err := p.skillMgr.UpgradeSkill(req.SkillId); err != nil {
		resp.Result = 1
		resp.Error = err.Error()
	} else if s, ok := p.skillMgr.GetSkill(req.SkillId); ok {
		resp.Level = s.Level
	}
	p.replySkill(msg, protocol.SkillMsgId_MSG_SKILL_UPGRADE_RESPONSE, resp)
}

func (p *Player) handleNetSkillCast(msg *PlayerMessage) {
	req, ok := msg.Data.(*protocol.ClientSkillCastRequest)
	if !ok {
		p.sendErrorResponse(msg, "invalid request data")
		return
	}
	resp := &protocol.ClientSkillCastResponse{SkillId: req.SkillId}
	if err := p.skillMgr.UseSkill(req.SkillId); err != nil {
		resp.Result = 1
		resp.Error = err.Error()
	}
	p.replySkill(msg, protocol.SkillMsgId_MSG_SKILL_CAST_RESPONSE, resp)
}
