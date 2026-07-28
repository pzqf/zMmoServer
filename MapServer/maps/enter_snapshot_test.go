package maps

import (
	"testing"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
)

// TestEnterSnapshot_AppliesLevelAndHealth 验证进场属性快照接线：GameServer 下发的 level/hp
// 经 HandlePlayerEnterMap 真正落到 object.Player，而非恒为 object 默认 level 1。
func TestEnterSnapshot_AppliesLevelAndHealth(t *testing.T) {
	mm := NewMapManager()
	const mapID = int64(2001)
	if m := mm.CreateMap(id.MapIdType(mapID), 1, "Field", 500, 500); m == nil {
		t.Fatalf("CreateMap 返回 nil")
	}

	const wantLevel = int32(7)
	const wantHP = int32(230)
	if err := mm.HandlePlayerEnterMap(1, mapID, 100, 0, 100, wantLevel, wantHP); err != nil {
		t.Fatalf("enter with snapshot: %v", err)
	}

	m := mm.resolveMap(1, mapID)
	if m == nil {
		t.Fatalf("resolveMap 为 nil")
	}

	// 快照经地图 actor(Do)异步设置，用一个同步 Do 排到其后读回，确保已生效。
	var gotLevel, gotHP int32
	m.Do(func() {
		obj := m.GetObject(id.ObjectIdType(1))
		if obj == nil {
			return
		}
		if p, ok := obj.(*object.Player); ok {
			gotLevel = p.GetLevel()
			gotHP = p.GetHealth()
		}
	})

	if gotLevel != wantLevel {
		t.Fatalf("level 未按快照应用: want %d, got %d（恒 1 即回归 bug）", wantLevel, gotLevel)
	}
	if gotHP != wantHP {
		t.Fatalf("hp 未按快照应用: want %d, got %d", wantHP, gotHP)
	}
}

// TestEnterSnapshot_ZeroKeepsDefault 验证 0 值快照不覆盖默认（沿用 object 默认 level）。
func TestEnterSnapshot_ZeroKeepsDefault(t *testing.T) {
	mm := NewMapManager()
	const mapID = int64(2002)
	mm.CreateMap(id.MapIdType(mapID), 1, "Field", 500, 500)

	if err := mm.HandlePlayerEnterMap(2, mapID, 100, 0, 100, 0, 0); err != nil {
		t.Fatalf("enter without snapshot: %v", err)
	}

	m := mm.resolveMap(2, mapID)
	if m == nil {
		t.Fatalf("resolveMap 为 nil")
	}
	var gotLevel int32
	m.Do(func() {
		if p, ok := m.GetObject(id.ObjectIdType(2)).(*object.Player); ok {
			gotLevel = p.GetLevel()
		}
	})
	if gotLevel <= 0 {
		t.Fatalf("0 值快照不应把 level 归零/无效, got %d", gotLevel)
	}
}
