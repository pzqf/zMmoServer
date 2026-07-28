package config

import "testing"

func TestParseSeamlessLinks(t *testing.T) {
	mc := &MapsConfig{SeamlessLinks: "2001-2002, 2003-2004 ,,bad,5-0,0-5"}
	pairs := mc.ParseSeamlessLinks()
	// 合法项：2001-2002 / 2003-2004；非法项（空/bad/含0）跳过。
	if len(pairs) != 2 {
		t.Fatalf("want 2 valid pairs, got %d: %v", len(pairs), pairs)
	}
	if pairs[0] != [2]int32{2001, 2002} || pairs[1] != [2]int32{2003, 2004} {
		t.Fatalf("unexpected pairs: %v", pairs)
	}

	if got := (&MapsConfig{SeamlessLinks: ""}).ParseSeamlessLinks(); len(got) != 0 {
		t.Fatalf("empty should yield 0 pairs, got %v", got)
	}
}

func TestIsLayerableConfigured(t *testing.T) {
	mc := &MapsConfig{LayerableMaps: []int{1001, 1003}}
	if !mc.IsLayerableConfigured(1001) || !mc.IsLayerableConfigured(1003) {
		t.Fatalf("1001/1003 should be layerable")
	}
	if mc.IsLayerableConfigured(1002) {
		t.Fatalf("1002 should not be layerable")
	}
}

func TestValidate_LayerCaps(t *testing.T) {
	base := func() *Config {
		return &Config{
			Server: ServerConfig{ServerID: 101, ListenAddr: ":0"}, // group 1, index 1（合法 realm server id）
			Maps:   MapsConfig{Mode: MapModeSingleServer, MapIDs: []int{1001}},
		}
	}

	// 声明可分线但未给正上限 → 拒绝。
	c := base()
	c.Maps.LayerableMaps = []int{1001}
	if err := c.Validate(); err == nil {
		t.Fatalf("layerable without caps should fail validation")
	}

	// soft > hard → 拒绝。
	c = base()
	c.Maps.LayerableMaps = []int{1001}
	c.Maps.LayerSoftCap, c.Maps.LayerHardCap = 5, 3
	if err := c.Validate(); err == nil {
		t.Fatalf("soft>hard should fail validation")
	}

	// 合法。
	c = base()
	c.Maps.LayerableMaps = []int{1001}
	c.Maps.LayerSoftCap, c.Maps.LayerHardCap = 40, 50
	if err := c.Validate(); err != nil {
		t.Fatalf("valid layer caps should pass, got %v", err)
	}

	// 未声明可分线时不校验 caps。
	c = base()
	if err := c.Validate(); err != nil {
		t.Fatalf("no layerable maps should pass, got %v", err)
	}
}
