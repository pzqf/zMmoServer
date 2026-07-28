package crossserver

import (
	"context"
	"errors"
	"testing"
	"time"

	consistency "github.com/pzqf/zEngine/zConsistency"
)

// 跨服迁移**控制面**双端集成测试：两个 MigrationManager（源服 / 目标服）经真实的
// CrossTransport + codec 收发，跑完 request → prepare ack → data → commit ack → complete 全程。
//
// 这条链此前从没跑通（msgID 没上线、响应无回投通道，见 docs/协议契约.md 变更记录 0.0.2）。
// 只有底层字节投递被换成内存连接（loopbackNet，见 transport_test.go），线格式/路由/关联全是生产代码。

type migNode struct {
	*loopbackNode
	mgr *MigrationManager
	ser *refSerializer
	cb  *refCallback
}

func newMigNode(net *loopbackNet, serverID int32, service uint8) *migNode {
	node := net.addNode(serverID, service)
	ser := newRefSerializer()
	cb := &refCallback{}
	mgr := NewMigrationManager(DefaultMigrationConfig(), ser, cb, node.transport,
		nil, nil, consistency.NewMemoryInbox())
	return &migNode{loopbackNode: node, mgr: mgr, ser: ser, cb: cb}
}

// newMigPair 建一对已互联的源/目标节点（模拟两个 realm 的 GameServer）。
func newMigPair(t *testing.T) (*loopbackNet, *migNode, *migNode) {
	t.Helper()
	net := newLoopbackNet()
	source := newMigNode(net, 1101, ServiceTypeGame)
	target := newMigNode(net, 2202, ServiceTypeGame)
	net.link(source.loopbackNode, target.loopbackNode)
	return net, source, target
}

// TestMigrationControlPlane_FullHandshake 一次成功迁移：玩家真的从源服落到目标服，两侧状态都收敛。
func TestMigrationControlPlane_FullHandshake(t *testing.T) {
	net, source, target := newMigPair(t)

	const playerID = int64(2001)
	source.ser.store[playerID] = &refPlayerState{PlayerID: playerID, Name: "Dora", Level: 55, Gold: 123456}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record, err := source.mgr.ExecuteMigration(ctx, playerID, 9001, "Dora",
		target.serverID, target.service, 5001, MigrationTypeGameToGame, "cross realm dungeon")
	if err != nil {
		t.Fatalf("ExecuteMigration: %v", err)
	}
	net.drain()

	if state, _ := source.mgr.State(record.ID); state != MigrationStateCompleted {
		t.Fatalf("源服记录应为 completed, got %s", state)
	}

	// 玩家数据真的到了目标服，且是源服的真实数值（非默认零值）。
	got, ok := target.ser.store[playerID]
	if !ok {
		t.Fatalf("目标服没拿到玩家 %d —— 控制面握手没跑通", playerID)
	}
	if got.Name != "Dora" || got.Level != 55 || got.Gold != 123456 {
		t.Fatalf("玩家数据迁移不忠实: %+v", got)
	}

	// 两侧回调各就各位：目标收数据 + 收完成通知；源服提交 + 完成。
	if target.cb.dataReceived != 1 {
		t.Fatalf("目标 OnMigrationDataReceived 应=1, got %d", target.cb.dataReceived)
	}
	if target.cb.complete != 1 {
		t.Fatalf("目标应收到完成通知, complete=%d", target.cb.complete)
	}
	if source.cb.commit != 1 || source.cb.complete != 1 {
		t.Fatalf("源服应提交并完成, commit=%d complete=%d", source.cb.commit, source.cb.complete)
	}
	if source.cb.rollback != 0 || target.cb.rollback != 0 {
		t.Fatalf("成功路径不应有回滚, source=%d target=%d", source.cb.rollback, target.cb.rollback)
	}

	// 玩家索引已释放（同一玩家可再次迁移）；无残留活跃迁移。
	if _, exists := source.mgr.GetMigrationByPlayer(playerID); exists {
		t.Fatalf("完成后应释放玩家索引，否则该玩家再也迁不动")
	}
	if source.mgr.ActiveCount() != 0 {
		t.Fatalf("不应残留活跃迁移: %d", source.mgr.ActiveCount())
	}
	if errs := net.errors(); len(errs) != 0 {
		t.Fatalf("投递不应报错: %v", errs)
	}
}

// TestMigrationControlPlane_TargetRejects 目标服拒绝接纳（满员/维护）→ 源服立刻失败，
// **不得**投数据、不得删玩家：玩家原地留在源服。
func TestMigrationControlPlane_TargetRejects(t *testing.T) {
	net, source, target := newMigPair(t)
	target.cb.reject = "target realm is full"

	const playerID = int64(2002)
	source.ser.store[playerID] = &refPlayerState{PlayerID: playerID, Name: "Eli", Level: 3, Gold: 7}

	record, err := source.mgr.ExecuteMigration(context.Background(), playerID, 9002, "Eli",
		target.serverID, target.service, 5001, MigrationTypeGameToGame, "manual")
	if !errors.Is(err, ErrMigrationRejected) {
		t.Fatalf("应返回 ErrMigrationRejected, got %v", err)
	}
	if record != nil {
		t.Fatalf("被拒时不应返回记录")
	}
	net.drain()

	if len(target.ser.store) != 0 {
		t.Fatalf("被拒后不应有数据落到目标服: %+v", target.ser.store)
	}
	if target.cb.dataReceived != 0 {
		t.Fatalf("被拒后不应进入数据阶段, dataReceived=%d", target.cb.dataReceived)
	}
	if _, exists := source.mgr.GetMigrationByPlayer(playerID); exists {
		t.Fatalf("失败后应释放玩家索引，否则玩家被永久卡住（迁不走也迁不回）")
	}
	if source.ser.store[playerID] == nil {
		t.Fatalf("被拒后玩家必须仍在源服")
	}
}

// TestMigrationControlPlane_TargetDataFailure 目标落地失败 → 源服拿到失败原因并判定迁移失败，
// 玩家留在源服（这正是"目标确认前不得删玩家"的保护）。
func TestMigrationControlPlane_TargetDataFailure(t *testing.T) {
	net, source, target := newMigPair(t)
	target.cb.dataErr = errors.New("target storage unavailable")

	const playerID = int64(2003)
	source.ser.store[playerID] = &refPlayerState{PlayerID: playerID, Name: "Fay", Level: 20, Gold: 42}

	_, err := source.mgr.ExecuteMigration(context.Background(), playerID, 9003, "Fay",
		target.serverID, target.service, 5001, MigrationTypeGameToGame, "manual")
	if !errors.Is(err, ErrMigrationDataInvalid) {
		t.Fatalf("目标落地失败应返回 ErrMigrationDataInvalid, got %v", err)
	}
	net.drain()

	if len(target.ser.store) != 0 {
		t.Fatalf("落地失败不应写入目标 store: %+v", target.ser.store)
	}
	if source.cb.commit != 0 {
		t.Fatalf("目标没落地，源服绝不能提交, commit=%d", source.cb.commit)
	}
	if state, _ := source.mgr.State(1); state != MigrationStateFailed {
		t.Fatalf("源服记录应为 failed, got %s", state)
	}
	if source.ser.store[playerID] == nil {
		t.Fatalf("落地失败后玩家必须仍在源服")
	}
}

// TestMigrationControlPlane_SourceCommitFailureRollsBackTarget 数据已落到目标、但源服提交失败
// → 必须把目标那份撤销掉，否则同一个玩家在两边各有一份（复制刷号）。
func TestMigrationControlPlane_SourceCommitFailureRollsBackTarget(t *testing.T) {
	net, source, target := newMigPair(t)
	source.cb.commitErr = errors.New("source transaction failed")

	const playerID = int64(2004)
	source.ser.store[playerID] = &refPlayerState{PlayerID: playerID, Name: "Gus", Level: 31, Gold: 900}

	_, err := source.mgr.ExecuteMigration(context.Background(), playerID, 9004, "Gus",
		target.serverID, target.service, 5001, MigrationTypeGameToGame, "manual")
	if err == nil {
		t.Fatalf("源服提交失败应返回错误")
	}
	net.drain()

	// 目标侧收到撤销通知并跑了业务回滚回调（真正的撤销由业务回调负责，参考实现只计数）。
	if target.cb.rollback != 1 {
		t.Fatalf("目标应收到回滚通知并回调 1 次, got %d", target.cb.rollback)
	}
	migrationID := uint64(1)
	if state, ok := target.mgr.State(migrationID); !ok || state != MigrationStateRolledBack {
		t.Fatalf("目标记录应为 rolled_back, got %s (exists=%v)", state, ok)
	}
	if state, _ := source.mgr.State(migrationID); state != MigrationStateRolledBack {
		t.Fatalf("源服记录应为 rolled_back, got %s", state)
	}
	if _, exists := source.mgr.GetMigrationByPlayer(playerID); exists {
		t.Fatalf("回滚后应释放玩家索引")
	}
}
