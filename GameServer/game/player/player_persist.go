package player

import (
	"os"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/game"
	"github.com/pzqf/zEngine/zLog"
	playerservice "github.com/pzqf/zMmoServer/GameServer/services"
	"go.uber.org/zap"
)

// 背包 / 技能持久化（业务层建设 2026-07-25）：登录从 DB 载入到 actor、登出/存盘写回 DB。
// 采用「先清后插」使 DB 与内存一致。存盘为最小闭环——非事务、非崩溃安全（进程崩溃时未落盘的
// 背包/技能变更会丢），够验证"持久化接线"这条链路；崩溃安全需接一致性层，未做。

// loadPlayerAssets 登录时把 DB 的背包/技能载入 actor。须在 AddPlayer 前调用（actor 尚未处理消息，无并发）。
func loadPlayerAssets(ps *playerservice.PlayerService, p *Player, playerID id.PlayerIdType) {
	if items, err := ps.LoadPlayerItems(int64(playerID)); err != nil {
		zLog.Warn("Load player items failed", zap.Int64("player_id", int64(playerID)), zap.Error(err))
	} else {
		for _, r := range items {
			_, _ = p.inventory.AddItem(game.NewItem(int64(r.ConfigID), r.ConfigID, "", game.ItemTypeConsumable, r.Count))
		}
	}
	// 测试种子：仅当 DB 无持久化背包时（首次登录）才种，避免与持久化叠加累积。首次种下后即随登出落库，
	// 之后登录走 DB 载入分支不再重种。
	if p.inventory.Count() == 0 && os.Getenv("ZMMO_TEST_ITEMS") == "1" {
		_, _ = p.inventory.AddItem(game.NewItem(1001, 1001, "TestPotion", game.ItemTypeConsumable, 3))
		_, _ = p.inventory.AddItem(game.NewItem(1002, 1002, "TestMaterial", game.ItemTypeConsumable, 10))
	}

	if skills, err := ps.LoadPlayerSkills(int64(playerID)); err != nil {
		zLog.Warn("Load player skills failed", zap.Int64("player_id", int64(playerID)), zap.Error(err))
	} else {
		for _, r := range skills {
			_ = p.skillMgr.LearnSkill(&game.Skill{SkillID: r.SkillID, Level: r.Level, MaxLevel: 10})
		}
	}
}

// savePlayerAssets 登出/存盘时把 actor 的背包/技能写回 DB。
func savePlayerAssets(ps *playerservice.PlayerService, p *Player, playerID id.PlayerIdType) {
	items := p.inventory.GetAllItems()
	rows := make([]playerservice.PersistItem, 0, len(items))
	for slot, it := range items {
		rows = append(rows, playerservice.PersistItem{ConfigID: it.ConfigID, Count: it.Count, Slot: slot})
	}
	if err := ps.SavePlayerItems(int64(playerID), rows); err != nil {
		zLog.Warn("Save player items failed", zap.Int64("player_id", int64(playerID)), zap.Error(err))
	}

	skills := p.skillMgr.GetAllSkills()
	srows := make([]playerservice.PersistSkill, 0, len(skills))
	for _, sk := range skills {
		srows = append(srows, playerservice.PersistSkill{SkillID: sk.SkillID, Level: sk.Level})
	}
	if err := ps.SavePlayerSkills(int64(playerID), srows); err != nil {
		zLog.Warn("Save player skills failed", zap.Int64("player_id", int64(playerID)), zap.Error(err))
	}
}
