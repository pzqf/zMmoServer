package maps

import (
	"testing"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
)

// TestSeamless_NeighborHandoffCarriesState 无缝邻居间交接：玩家带实时态（血量/等级/新坐标）迁到相邻图，
// 不重登；playerMap 更新到新图。
func TestSeamless_NeighborHandoffCarriesState(t *testing.T) {
	mm := NewMapManager()
	a := mm.CreateMap(id.MapIdType(2001), 1, "Plains", 500, 500)
	b := mm.CreateMap(id.MapIdType(2002), 1, "Forest", 500, 500)
	if a == nil || b == nil {
		t.Fatalf("建图失败")
	}
	mm.RegisterSeamlessLink(id.MapIdType(2001), id.MapIdType(2002)) // 相邻无缝分区

	// 玩家在 A，受伤到 HP=37、升到 5 级。
	if err := a.AddPlayer(id.PlayerIdType(1), id.ObjectIdType(1), 490, 0, 100); err != nil {
		t.Fatalf("AddPlayer A: %v", err)
	}
	if p, ok := a.GetObject(id.ObjectIdType(1)).(*object.Player); ok {
		p.SetLevel(5)
		p.SetHealth(37)
	}

	// 走过区界 → 无缝交接到 B 的入口坐标 (10, 0, 100)。
	handed, err := mm.HandleSeamlessHandoff(1, 2001, 2002, 10, 0, 100)
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}
	if !handed {
		t.Fatalf("无缝邻居应走无缝交接(handed=true)")
	}

	if a.GetPlayerCount() != 0 {
		t.Fatalf("交接后源图应无此玩家")
	}
	if b.GetPlayerCount() != 1 {
		t.Fatalf("交接后目标图应有此玩家")
	}
	// playerMap 指向新图。
	if got := mm.resolveMap(1, 2001).GetID(); got != id.MapIdType(2002) {
		t.Fatalf("playerMap 应更新到目标图 2002, got %d", got)
	}
	// 实时态随身带过去（不满血、等级保留、新坐标）。
	p, ok := b.GetObject(id.ObjectIdType(1)).(*object.Player)
	if !ok {
		t.Fatalf("目标图应有玩家对象")
	}
	if p.GetHealth() != 37 {
		t.Fatalf("血量应保留 37, got %d", p.GetHealth())
	}
	if p.GetLevel() != 5 {
		t.Fatalf("等级应保留 5, got %d", p.GetLevel())
	}
	if pos := p.GetPosition(); pos.X != 10 {
		t.Fatalf("位置应承接新坐标 x=10, got %v", pos.X)
	}
}

// TestSeamless_NonNeighborFallsBackToNormal 非无缝邻居（如进副本/跨大陆）不走无缝流程——返回 false，
// 调用方应改走普通 enter/leave。玩家不被交接。
func TestSeamless_NonNeighborFallsBackToNormal(t *testing.T) {
	mm := NewMapManager()
	a := mm.CreateMap(id.MapIdType(3001), 1, "Town", 500, 500)
	c := mm.CreateMap(id.MapIdType(3002), 1, "Dungeon", 500, 500) // 未登记无缝链路
	if a == nil || c == nil {
		t.Fatalf("建图失败")
	}
	_ = a.AddPlayer(id.PlayerIdType(1), id.ObjectIdType(1), 100, 0, 100)

	handed, err := mm.HandleSeamlessHandoff(1, 3001, 3002, 10, 0, 100)
	if err != nil {
		t.Fatalf("handoff err: %v", err)
	}
	if handed {
		t.Fatalf("非无缝邻居不应走无缝交接(应 handed=false 回落普通流程)")
	}
	// 玩家未被交接，仍在源图。
	if a.GetPlayerCount() != 1 {
		t.Fatalf("非无缝交接不应移动玩家, 源图应仍有 1 人")
	}
	if c.GetPlayerCount() != 0 {
		t.Fatalf("目标图不应有玩家")
	}
}

// TestSeamless_LinkIsBidirectional 无缝链路双向。
func TestSeamless_LinkIsBidirectional(t *testing.T) {
	mm := NewMapManager()
	mm.RegisterSeamlessLink(id.MapIdType(1), id.MapIdType(2))
	if !mm.IsSeamlessNeighbor(1, 2) || !mm.IsSeamlessNeighbor(2, 1) {
		t.Fatalf("无缝链路应双向")
	}
	if mm.IsSeamlessNeighbor(1, 3) {
		t.Fatalf("未登记不应为无缝邻居")
	}
}
