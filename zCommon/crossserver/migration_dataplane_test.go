package crossserver

import (
	"encoding/json"
	"hash/crc32"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/pzqf/zCommon/protocol"
	consistency "github.com/pzqf/zEngine/zConsistency"
)

// 本测试覆盖跨服迁移的**数据面**（player 字节 序列化→投递→反序列化，带校验和 + inbox 幂等），
// 直接调目标侧 handler，不经网络。
//
// **控制面握手**（request→prepare ack→data→commit ack→complete 往返，含真编解码与响应回投）
// 见 migration_controlplane_test.go —— 2026-07-28 修好传输层（msgID 上线 + 响应回投通道）后才跑通。
//
// 注：refSerializer 用 JSON 编玩家态，这是 **serializer 自己的选择**（player_data 对传输层是
// 不透明字节），不违反"跨服线格式全程 protobuf"的契约——线格式由 codec.go 统一负责。

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

// refCallback 参考 MigrationCallback 实现：记录被调用次数，可注入拒绝/失败以走异常分支。
type refCallback struct {
	dataReceived int
	commit       int
	rollback     int
	complete     int
	reject       string // 非空 = OnMigrationPrepare 拒绝，并以此为理由
	dataErr      error  // 非 nil = OnMigrationDataReceived 失败（目标落地失败）
	commitErr    error  // 非 nil = OnMigrationCommit 失败（源服提交失败 → 应回滚目标）
}

func (c *refCallback) OnMigrationPrepare(*MigrationRecord) (bool, string) {
	if c.reject != "" {
		return false, c.reject
	}
	return true, ""
}
func (c *refCallback) OnMigrationDataReceived(*MigrationRecord) error {
	c.dataReceived++
	return c.dataErr
}
func (c *refCallback) OnMigrationCommit(*MigrationRecord) error {
	c.commit++
	return c.commitErr
}
func (c *refCallback) OnMigrationRollback(*MigrationRecord) error { c.rollback++; return nil }
func (c *refCallback) OnMigrationComplete(*MigrationRecord) error { c.complete++; return nil }

// buildDataPayload 复刻 transferData 的数据组装（序列化 + 校验和），产出目标侧 handleMigrationData 的入参。
func buildDataPayload(t *testing.T, src *refSerializer, migrationID uint64, playerID int64) []byte {
	t.Helper()
	playerData, mapData, err := src.SerializePlayer(playerID)
	if err != nil {
		t.Fatalf("serialize source player: %v", err)
	}
	b, err := proto.Marshal(&protocol.MigrationDataTransfer{
		MigrationId: migrationID,
		PlayerId:    playerID,
		PlayerData:  playerData,
		MapData:     mapData,
		Checksum:    crc32.ChecksumIEEE(playerData),
	})
	if err != nil {
		t.Fatalf("marshal data payload: %v", err)
	}
	return b
}

func parseCommitAck(t *testing.T, respBytes []byte) *protocol.MigrationCommitAck {
	t.Helper()
	var ack protocol.MigrationCommitAck
	if err := proto.Unmarshal(respBytes, &ack); err != nil {
		t.Fatalf("unmarshal commit ack: %v", err)
	}
	return &ack
}

func newTargetMM(inbox consistency.InboxStore, ser PlayerDataSerializer, cb MigrationCallback) *MigrationManager {
	// transport=nil → registerHandlers 早返回（我们直接调 handleMigrationData 模拟传输已投递）；
	// txManager/outbox 数据面收侧用不到，传 nil。
	return NewMigrationManager(DefaultMigrationConfig(), ser, cb, nil, nil, nil, inbox)
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
	if ack := parseCommitAck(t, respBytes); !ack.Success {
		t.Fatalf("expected Success, got failure: %s", ack.Reason)
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
		if ack := parseCommitAck(t, respBytes); !ack.Success {
			t.Fatalf("deliver #%d expected Success, got %s", i, ack.Reason)
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

// TestMigrationDataPlane_InFlightDuplicateNotAckedAsSuccess 另一次投递**还在处理中**时收到重投，
// 绝不能回成功——源服会据此提交并删玩家，而在途那次可能随后失败 → 玩家丢失。应回失败让源服重试，
// 且**不得**撤销在途那次的 inbox 占用。
func TestMigrationDataPlane_InFlightDuplicateNotAckedAsSuccess(t *testing.T) {
	source := newRefSerializer()
	source.store[1004] = &refPlayerState{PlayerID: 1004, Name: "Hana", Level: 12, Gold: 88}
	target := newRefSerializer()
	cb := &refCallback{}
	inbox := consistency.NewMemoryInbox()
	mm := newTargetMM(inbox, target, cb)

	const migrationID = uint64(7004)
	// 模拟"第一次投递已占用 inbox、尚未落地完成"。
	if !inbox.TryAccept(migrationID) {
		t.Fatalf("首次占用应成功")
	}

	payload := buildDataPayload(t, source, migrationID, 1004)
	respBytes, err := mm.handleMigrationData(NewRequestMeta(ServiceTypeGame, 101), payload)
	if err != nil {
		t.Fatalf("handleMigrationData: %v", err)
	}
	if ack := parseCommitAck(t, respBytes); ack.Success {
		t.Fatalf("在途未落地时重投不能回成功（会致源服删玩家、玩家丢失）")
	}
	if target.deserialize != 0 {
		t.Fatalf("重投不应落地, deserialize=%d", target.deserialize)
	}
	if !inbox.IsProcessed(migrationID) {
		t.Fatalf("不得撤销在途那次的 inbox 占用")
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
	var dp protocol.MigrationDataTransfer
	if err := proto.Unmarshal(good, &dp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	dp.Checksum = dp.Checksum ^ 0xFFFFFFFF // 保证不等
	bad, _ := proto.Marshal(&dp)

	meta := NewRequestMeta(ServiceTypeGame, 101)
	respBytes, err := mm.handleMigrationData(meta, bad)
	if err != nil {
		t.Fatalf("handleMigrationData(bad): %v", err)
	}
	if ack := parseCommitAck(t, respBytes); ack.Success {
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
	if ack := parseCommitAck(t, respBytes2); !ack.Success {
		t.Fatalf("修正后重投应成功: %s", ack.Reason)
	}
	if target.deserialize != 1 {
		t.Fatalf("修正后应落地 1 次, got %d", target.deserialize)
	}
}
