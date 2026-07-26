package maps

import (
	"fmt"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
	"go.uber.org/zap"
)

// —— 无缝地图交接（Seamless World，realm 建议①）——
//
// 魔兽/传奇里，同一块连续大陆被切成相邻的分区；玩家走过区界时，服务器把他**透明交接**给拥有下一分区的
// map，实时态（位置/血量/等级）随身带过去、客户端不断线、无 loading。这是"单服却世界很大"的关键。
//
// ★关键约束（用户定）：只有**设计成无缝**的图对才走无缝流程。**若目标不是源的无缝邻居，就不走无缝交接，
// 回落普通 enter/leave**（跨大陆、进副本、进城 等有 loading 的切换）。无缝性是地图（对）的属性，靠
// RegisterSeamlessLink 登记；未登记链路的跨图一律走普通流程。
//
// 落点：无缝邻居图通常同在一台 MapServer（同一连续世界），故交接是 MapServer 内部操作——把玩家对象从
// A 图移到 B 图、带上实时态、更新 playerMap（②-b 的实际所在图记录），不经 GameServer 路由变更、不重登。

// RegisterSeamlessLink 登记 a、b 为相邻的无缝分区（双向）。只有登记过的图对才可无缝交接。
func (mm *MapManager) RegisterSeamlessLink(a, b id.MapIdType) {
	mm.seamlessMu.Lock()
	defer mm.seamlessMu.Unlock()
	if mm.seamlessLinks[a] == nil {
		mm.seamlessLinks[a] = make(map[id.MapIdType]bool)
	}
	if mm.seamlessLinks[b] == nil {
		mm.seamlessLinks[b] = make(map[id.MapIdType]bool)
	}
	mm.seamlessLinks[a][b] = true
	mm.seamlessLinks[b][a] = true
}

// IsSeamlessNeighbor from、to 是否为登记的相邻无缝分区。
func (mm *MapManager) IsSeamlessNeighbor(from, to id.MapIdType) bool {
	mm.seamlessMu.Lock()
	defer mm.seamlessMu.Unlock()
	return mm.seamlessLinks[from][to]
}

// HandleSeamlessHandoff 处理跨区界的无缝交接。
//
//   - 若 to 不是 from 的无缝邻居（或该图非无缝设计）→ 返回 (false, nil)：**不走无缝流程**，调用方应改走
//     普通 enter/leave（有 loading 的切换）。这正是"非无缝地图不走无缝流程"的约束落点。
//   - 若是无缝邻居 → 内部交接：快照实时态(血量/等级/位置) → 从 from 摘除 → 加入 to 的新坐标 → 恢复实时态 →
//     更新 playerMap。返回 (true, nil)。全程 MapServer 内部、不重登、位置连续。
func (mm *MapManager) HandleSeamlessHandoff(playerID int64, fromMapID, toMapID int64, x, y, z float32) (bool, error) {
	if !mm.IsSeamlessNeighbor(id.MapIdType(fromMapID), id.MapIdType(toMapID)) {
		return false, nil // 非无缝邻居：回落普通流程
	}
	from := mm.GetMap(id.MapIdType(fromMapID))
	to := mm.GetMap(id.MapIdType(toMapID))
	if from == nil || to == nil {
		return false, fmt.Errorf("seamless handoff: map not found (from=%d to=%d)", fromMapID, toMapID)
	}

	// 快照实时态（无缝迁移不重置：血量不满血、等级保留、位置由新坐标承接）。
	var hp, level int32 = -1, -1
	if obj := from.GetObject(id.ObjectIdType(playerID)); obj != nil {
		if p, ok := obj.(*object.Player); ok {
			hp, level = p.GetHealth(), p.GetLevel()
		}
	}

	from.RemovePlayer(id.PlayerIdType(playerID))
	if err := to.AddPlayer(id.PlayerIdType(playerID), id.ObjectIdType(playerID), x, y, z); err != nil {
		return false, fmt.Errorf("seamless handoff: add to target: %w", err)
	}

	// 恢复实时态到新图对象（位置已由 AddPlayer 承接）。
	if obj := to.GetObject(id.ObjectIdType(playerID)); obj != nil {
		if p, ok := obj.(*object.Player); ok {
			if level > 0 {
				p.SetLevel(level)
			}
			if hp >= 0 {
				p.SetHealth(hp)
			}
			p.SetPosition(common.Vector3{X: x, Y: y, Z: z})
		}
	}

	mm.setPlayerMap(playerID, id.MapIdType(toMapID))
	zLog.Info("Seamless handoff",
		zap.Int64("player_id", playerID),
		zap.Int64("from_map", fromMapID), zap.Int64("to_map", toMapID),
		zap.Int32("hp", hp), zap.Int32("level", level))
	return true, nil
}
