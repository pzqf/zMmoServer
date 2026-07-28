package crossserver

import (
	"encoding/json"
	"hash/crc32"
	"testing"

	consistency "github.com/pzqf/zEngine/zConsistency"
)

// 本测试覆盖跨服迁移的**数据面**（player 字节 序列化→投递→反序列化，带校验和 + inbox 幂等），
// 这是 MigrationManager 中已成型、可单进程验证的部分。
//
// 明确不覆盖（是真缺口、非本测试范围）：迁移的**控制面握手**（request/prepare/commit 往返）——
// 现设计里 handler 返回响应字节、但传输是单向 SendToServer，响应无回投通道；且 msgID 未随
// Envelope 上线（envelope.go 无 msgID 字段）。这两处是"跨服物理路由"从零工程的传输层设计洞，
// 需专门重构 + 多进程 E2E，不在此单进程数据面测试内假装打通。

// refPlayerState 一个代表性的玩家持久态（真实现里由 GameServer 玩家 actor 提供）。
type refPlayerState struct {
	PlayerID int64  `json:"player_id"`
	Name     string `json:"name"`
	Level    int32  `json:"level"`
	Gold     int64  `json:"gold"`
}

// refSerializer 参考 PlayerDataSerializer 实现：读/写自己持有的 store。
// 源服的 serializer 从源 store 取（SerializePlayer）；目标服的 serializer 落到目标 store（DeserializePlayer）。
type refSerializer struct {
	store       map[int64]*refPlayerState
	deserialize int // DeserializePlayer 被真正应用的次数（验证幂等：重投不应重复落地）
}

func newRefSerializer() *refSerializer {
	return &refSerializer{store: make(map[int64]*refPlayerState)}
}

func (s *refSerializer) SerializePlayer(playerID int64) ([]byte, []byte, error) {
	p, ok := s.store[playerID]
	if !ok {
		return nil, nil, ErrMigrationNotFound
	}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, nil, err
	}
	return data, nil, nil
}

func (s *refSerializer) DeserializePlayer(playerData []byte, mapData []byte) error {
	var p refPlayerState
	if err := json.Unmarshal(playerData, &p); err != nil {
		return err
	}
	s.store[p.PlayerID] = &p
	s.deserialize++
	return nil
}

// refCallback 参考 MigrationCallback 实现：只记录被调用次数。
type refCallback struct {
	dataReceived int
	commit       int
	rollback     int
}

func (c *refCallback) OnMigrationPrepare(*MigrationRecord) (bool, string) { return true, "" }
func (c *refCallback) OnMigrationDataReceived(*MigrationRecord) error     { c.dataReceived++; return nil }
func (c *refCallback) OnMigrationCommit(*MigrationRecord) error           { c.commit++; return nil }
func (c *refCallback) OnMigrationRollback(*MigrationRecord) error         { c.rollback++; return nil }
func (c *refCallback) OnMigrationComplete(*MigrationRecord) error         { return nil }

// buildDataPayload 复刻 sendMigrationData 的数据组装（序列化 + 校验和），产出目标侧 handleMigrationData 的入参。
func buildDataPayload(t *testing.T, src *refSerializer, migrationID uint64, playerID int64) []byte {
	t.Helper()
	playerData, mapData, err := src.SerializePlayer(playerID)
	if err != nil {
		t.Fatalf("serialize source player: %v", err)
	}
	payload := MigrationDataPayload{
		MigrationID: migrationID,
		PlayerID:    playerID,
		PlayerData:  playerData,
		MapData:     mapData,
		Checksum:    crc32.ChecksumIEEE(playerData),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal data payload: %v", err)
	}
	return b
}

func newTargetMM(inbox consistency.InboxStore, ser PlayerDataSerializer, cb MigrationCallback) *MigrationManager {
	// router=nil → registerHandlers 早返回（我们直接调 handleMigrationData 模拟传输已投递）；
	// serverRouter/txManager/outbox 数据面收侧用不到，传 nil。
	return NewMigrationManager(DefaultMigrationConfig(), ser, cb, nil, nil, nil, nil, inbox)
}

// TestMigrationDataPlane_DeliverAndApply 源序列化 → 目标收下并反序列化落地；校验和一致 → Success。
func TestMigrationDataPlane_DeliverAndApply(t *testing.T) {
	source := newRefSerializer()
	source.store[1001] = &refPlayerState{PlayerID: 1001, Name: "Alice", Level: 42, Gold: 99999}

	target := newRefSerializer()
	cb := &refCallback{}
	mm := newTargetMM(consistency.NewMemoryInbox(), target, cb)

	const migrationID = uint64(7001)
	payload := buildDataPayload(t, source, migrationID, 1001)

	meta := NewRequestMeta(ServiceTypeGame, 101)
	respBytes, err := mm.handleMigrationData(meta, payload)
	if err != nil {
		t.Fatalf("handleMigrationData: %v", err)
	}
	var resp MigrationCommitPayload
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal resp: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected Success, got failure: %s", resp.Reason)
	}

	// 目标 store 应已落地源玩家的真实数据（非默认零值）。
	got, ok := target.store[1001]
	if !ok {
		t.Fatalf("target store missing player 1001 after migration")
	}
	if got.Name != "Alice" || got.Level != 42 || got.Gold != 99999 {
		t.Fatalf("player data not faithfully migrated: %+v", got)
	}
	if cb.dataReceived != 1 {
		t.Fatalf("OnMigrationDataReceived should fire once, got %d", cb.dataReceived)
	}
	if target.deserialize != 1 {
		t.Fatalf("deserialize should apply once, got %d", target.deserialize)
	}
}

// TestMigrationDataPlane_Idempotent 同一 migrationID 重投 → inbox 去重，返回成功但不重复落地（INF-2 前半）。
func TestMigrationDataPlane_Idempotent(t *testing.T) {
	source := newRefSerializer()
	source.store[1002] = &refPlayerState{PlayerID: 1002, Name: "Bob", Level: 10, Gold: 5}
	target := newRefSerializer()
	cb := &refCallback{}
	mm := newTargetMM(consistency.NewMemoryInbox(), target, cb)

	const migrationID = uint64(7002)
	payload := buildDataPayload(t, source, migrationID, 1002)
	meta := NewRequestMeta(ServiceTypeGame, 101)

	for i := 0; i < 3; i++ {
		respBytes, err := mm.handleMigrationData(meta, payload)
		if err != nil {
			t.Fatalf("deliver #%d: %v", i, err)
		}
		var resp MigrationCommitPayload
		_ = json.Unmarshal(respBytes, &resp)
		if !resp.Success {
			t.Fatalf("deliver #%d expected Success, got %s", i, resp.Reason)
		}
	}
	// 3 次投递，只应真正落地 1 次。
	if target.deserialize != 1 {
		t.Fatalf("重投应被 inbox 去重，deserialize 应=1, got %d", target.deserialize)
	}
	if cb.dataReceived != 1 {
		t.Fatalf("OnMigrationDataReceived 应=1, got %d", cb.dataReceived)
	}
}

// TestMigrationDataPlane_ChecksumMismatchReleases 校验和不符 → Success=false 且 inbox 撤销占用，
// 允许后续（修正后的）重投重新处理（INF-2 后半：失败不得永久判重）。
func TestMigrationDataPlane_ChecksumMismatchReleases(t *testing.T) {
	source := newRefSerializer()
	source.store[1003] = &refPlayerState{PlayerID: 1003, Name: "Carol", Level: 7, Gold: 1}
	target := newRefSerializer()
	cb := &refCallback{}
	inbox := consistency.NewMemoryInbox()
	mm := newTargetMM(inbox, target, cb)

	const migrationID = uint64(7003)

	// 篡改校验和：解出 payload、改 Checksum、重新 marshal。
	good := buildDataPayload(t, source, migrationID, 1003)
	var dp MigrationDataPayload
	if err := json.Unmarshal(good, &dp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	dp.Checksum = dp.Checksum ^ 0xFFFFFFFF // 保证不等
	bad, _ := json.Marshal(dp)

	meta := NewRequestMeta(ServiceTypeGame, 101)
	respBytes, err := mm.handleMigrationData(meta, bad)
	if err != nil {
		t.Fatalf("handleMigrationData(bad): %v", err)
	}
	var resp MigrationCommitPayload
	_ = json.Unmarshal(respBytes, &resp)
	if resp.Success {
		t.Fatalf("校验和不符应返回失败")
	}
	if target.deserialize != 0 {
		t.Fatalf("失败不应落地, deserialize=%d", target.deserialize)
	}
	// 关键（INF-2）：失败后 inbox 未标记为已处理，允许修正后重投重新被接受。
	if inbox.IsProcessed(migrationID) {
		t.Fatalf("失败后不应把 migrationID 永久判为已处理（否则重投误返成功→玩家丢失）")
	}

	// 修正后重投（同 ID，正确校验和）应成功落地。
	respBytes2, err := mm.handleMigrationData(meta, good)
	if err != nil {
		t.Fatalf("redeliver good: %v", err)
	}
	_ = json.Unmarshal(respBytes2, &resp)
	if !resp.Success {
		t.Fatalf("修正后重投应成功: %s", resp.Reason)
	}
	if target.deserialize != 1 {
		t.Fatalf("修正后应落地 1 次, got %d", target.deserialize)
	}
}
