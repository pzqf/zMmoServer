package player

import (
	"encoding/json"
	"os"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/game"
	"github.com/pzqf/zEngine/zLog"
	playerservice "github.com/pzqf/zMmoServer/GameServer/services"
	"go.uber.org/zap"
)

// 背包 / 技能 / 仓库 / buff / 任务持久化（业务层建设 2026-07-25）：登录从 DB 载入到 actor、登出/存盘写回 DB。
// 采用「先清后插」使 DB 与内存一致。存盘为最小闭环——非事务、非崩溃安全（进程崩溃时未落盘的变更会丢），
// 够验证"持久化接线"这条链路；崩溃安全需接一致性层，未做。buff 按剩余时长续时、过期不还原；
// 任务还原原状态 + 条件进度（条件/奖励 JSON）。

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

	if wh, err := ps.LoadPlayerWarehouse(int64(playerID)); err != nil {
		zLog.Warn("Load player warehouse failed", zap.Int64("player_id", int64(playerID)), zap.Error(err))
	} else {
		for _, r := range wh {
			_, _ = p.warehouse.Store(game.NewItem(int64(r.ConfigID), r.ConfigID, "", game.ItemTypeConsumable, r.Count))
		}
	}

	// buff 载入（F-8）：按登出时的剩余时长还原；已过期的不还原（等价于登出即失效）。
	if buffs, err := ps.LoadPlayerBuffs(int64(playerID)); err != nil {
		zLog.Warn("Load player buffs failed", zap.Int64("player_id", int64(playerID)), zap.Error(err))
	} else {
		for _, b := range buffs {
			var buff *game.Buff
			if b.IsPermanent {
				buff = game.NewBuff(b.BuffID, b.Name, game.BuffType(b.Type), 0)
				buff.IsPermanent = true
			} else {
				if b.RemainingMS <= 0 {
					continue // 已过期
				}
				buff = game.NewBuff(b.BuffID, b.Name, game.BuffType(b.Type), time.Duration(b.RemainingMS)*time.Millisecond)
			}
			buff.Value = b.Value
			buff.Property = b.Property
			p.buffMgr.AddBuff(buff)
		}
	}

	// 任务载入（F-8）：还原任务原状态 + 条件进度（条件/奖励为 JSON）。
	if tasks, err := ps.LoadPlayerTasks(int64(playerID)); err != nil {
		zLog.Warn("Load player tasks failed", zap.Int64("player_id", int64(playerID)), zap.Error(err))
	} else {
		for _, t := range tasks {
			task := game.NewTask(t.TaskID, t.ConfigID, t.Name, game.TaskType(t.Type))
			task.Status = game.TaskStatus(t.Status)
			if t.ConditionsJSON != "" {
				_ = json.Unmarshal([]byte(t.ConditionsJSON), &task.Conditions)
			}
			if t.RewardsJSON != "" {
				_ = json.Unmarshal([]byte(t.RewardsJSON), &task.Rewards)
			}
			p.taskMgr.RestoreTask(task)
		}
	}

	// 测试种子：仅首次登录（DB 无持久化 buff/task）且开关打开时种，之后走 DB 载入不重种。
	if p.buffMgr.Count() == 0 && p.taskMgr.Count() == 0 && os.Getenv("ZMMO_TEST_BUFF") == "1" {
		perm := game.NewBuff(9001, "TestPermBuff", game.BuffTypePositive, 0)
		perm.IsPermanent = true
		perm.Value = 10
		perm.Property = "atk"
		p.buffMgr.AddBuff(perm)
		p.buffMgr.AddBuff(game.NewBuff(9002, "TestTimedBuff", game.BuffTypePositive, time.Hour))
		task := game.NewTask(8001, 8001, "TestTask", game.TaskType(1))
		task.Status = game.TaskStatusInProgress
		task.Conditions = []*game.TaskCondition{{Type: game.TaskCondKillMonster, TargetID: 1001, Current: 2, Required: 5}}
		p.taskMgr.RestoreTask(task)
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

	whItems := p.warehouse.GetAllItems()
	wrows := make([]playerservice.PersistItem, 0, len(whItems))
	for slot, it := range whItems {
		wrows = append(wrows, playerservice.PersistItem{ConfigID: it.ConfigID, Count: it.Count, Slot: slot})
	}
	if err := ps.SavePlayerWarehouse(int64(playerID), wrows); err != nil {
		zLog.Warn("Save player warehouse failed", zap.Int64("player_id", int64(playerID)), zap.Error(err))
	}

	// buff 存盘（F-8）：只存未过期的；非永久 buff 记登出时的剩余毫秒（还原时按剩余续时）。
	buffs := p.buffMgr.GetAllBuffs()
	brows := make([]playerservice.PersistBuff, 0, len(buffs))
	for _, b := range buffs {
		if b.IsExpired() {
			continue
		}
		remMS := int64(-1)
		if !b.IsPermanent {
			remMS = int64(b.GetRemaining() / time.Millisecond)
			if remMS <= 0 {
				continue
			}
		}
		brows = append(brows, playerservice.PersistBuff{
			BuffID: b.ID, Name: b.Name, Type: int32(b.Type), Value: b.Value,
			Property: b.Property, IsPermanent: b.IsPermanent, RemainingMS: remMS,
		})
	}
	if err := ps.SavePlayerBuffs(int64(playerID), brows); err != nil {
		zLog.Warn("Save player buffs failed", zap.Int64("player_id", int64(playerID)), zap.Error(err))
	}

	// 任务存盘（F-8）：条件（含进度）/奖励序列化为 JSON。
	tasks := p.taskMgr.GetAllTasks()
	trows := make([]playerservice.PersistTask, 0, len(tasks))
	for _, t := range tasks {
		condJSON, _ := json.Marshal(t.Conditions)
		rewJSON, _ := json.Marshal(t.Rewards)
		trows = append(trows, playerservice.PersistTask{
			TaskID: t.TaskID, ConfigID: t.ConfigID, Name: t.Name, Type: int32(t.Type),
			Status: int32(t.Status), ConditionsJSON: string(condJSON), RewardsJSON: string(rewJSON),
		})
	}
	if err := ps.SavePlayerTasks(int64(playerID), trows); err != nil {
		zLog.Warn("Save player tasks failed", zap.Int64("player_id", int64(playerID)), zap.Error(err))
	}
}
