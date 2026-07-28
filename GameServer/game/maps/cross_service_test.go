package maps

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"google.golang.org/protobuf/proto"
)

// 跨服物理路由 GameServer 侧的集成测试：真 zNet 连接 + 真 crossserver codec + 真 HTTP 分配接口，
// 只把"对端"换成受控的假 MapServer / 假 GlobalServer。
//
// 要钉死的核心不变量：
//  1. 整条链跑得通：分配 → 连外域 → 建实例(620/621) → 进图(400) 全部落到**被分配的那台**服务器；
//  2. **不串域**：跨服玩家的地图消息绝不能按 mapID 走本 realm 路由——跨服实例 mapID 是各
//     MapServer 实例池的派生号，本 realm 很可能有同号的图，串过去就是打到错的服务器上；
//  3. 任何一步失败都不留绑定，否则该玩家后续所有消息会被永久发往外域。

// fakeMapServer 受控的假 MapServer：应答 620，并记录收到的所有跨服消息。
type fakeMapServer struct {
	t    *testing.T
	addr string
	srv  *zNet.TcpServer

	mu             sync.Mutex
	ensureRequests []*protocol.CrossInstanceEnsureRequest
	enterRequests  []*protocol.MapEnterRequest
	moveRequests   []*protocol.MapMoveRequest
	leaveRequests  []*protocol.MapLeaveRequest

	// ensureMapID 应答里返回的实例 mapID；ensureFail=true 则返回失败。
	ensureMapID int32
	ensureFail  bool
	serverID    uint32
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick free port: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func newFakeMapServer(t *testing.T, serverID uint32, ensureMapID int32) *fakeMapServer {
	t.Helper()
	f := &fakeMapServer{
		t:           t,
		addr:        freePort(t),
		ensureMapID: ensureMapID,
		serverID:    serverID,
	}

	f.srv = zNet.NewTcpServer(&zNet.TcpConfig{
		ListenAddress:     f.addr,
		MaxClientCount:    16,
		ChanSize:          128,
		HeartbeatDuration: 60,
		MaxPacketDataSize: 1024 * 1024,
		DisableEncryption: true,
	})
	f.srv.RegisterDispatcher(f.dispatch)
	if err := f.srv.Start(); err != nil {
		t.Fatalf("start fake map server: %v", err)
	}
	t.Cleanup(func() { f.srv.Close() })
	return f
}

func (f *fakeMapServer) dispatch(session zNet.Session, packet *zNet.NetPacket) error {
	meta, _, base, err := crossserver.UnpackMessage(packet.Data)
	if err != nil || base == nil {
		f.t.Errorf("fake map server got undecodable packet: %v", err)
		return nil
	}

	switch base.MsgId {
	case uint32(protocol.InternalMsgId_MSG_INTERNAL_CROSS_INSTANCE_ENSURE):
		var req protocol.CrossInstanceEnsureRequest
		if err := proto.Unmarshal(base.Data, &req); err != nil {
			f.t.Errorf("unmarshal ensure request: %v", err)
			return nil
		}
		f.mu.Lock()
		f.ensureRequests = append(f.ensureRequests, &req)
		fail, mapID := f.ensureFail, f.ensureMapID
		f.mu.Unlock()

		resp := &protocol.CrossInstanceEnsureResponse{
			Success:    !fail,
			ActivityId: req.ActivityId,
			MapId:      mapID,
		}
		if fail {
			resp.ErrorMsg = "instance rejected by test"
			resp.MapId = 0
		}
		f.reply(session, meta, base, int(protocol.InternalMsgId_MSG_INTERNAL_CROSS_INSTANCE_ENSURE_RESPONSE), resp)

	case uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_REQUEST):
		var req protocol.MapEnterRequest
		if err := proto.Unmarshal(base.Data, &req); err != nil {
			f.t.Errorf("unmarshal enter request: %v", err)
			return nil
		}
		f.mu.Lock()
		f.enterRequests = append(f.enterRequests, &req)
		f.mu.Unlock()

	case uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_MOVE_REQUEST):
		var req protocol.MapMoveRequest
		if err := proto.Unmarshal(base.Data, &req); err == nil {
			f.mu.Lock()
			f.moveRequests = append(f.moveRequests, &req)
			f.mu.Unlock()
		}

	case uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_LEAVE_REQUEST):
		var req protocol.MapLeaveRequest
		if err := proto.Unmarshal(base.Data, &req); err == nil {
			f.mu.Lock()
			f.leaveRequests = append(f.leaveRequests, &req)
			f.mu.Unlock()
		}
	}
	return nil
}

func (f *fakeMapServer) reply(session zNet.Session, reqMeta crossserver.Meta, reqBase *protocol.BaseMessage, msgID int, msg proto.Message) {
	data, err := proto.Marshal(msg)
	if err != nil {
		f.t.Errorf("marshal reply: %v", err)
		return
	}
	respBase := crossserver.BuildBaseMessage(uint32(msgID), reqBase.PlayerId, f.serverID, reqBase.MapId, data)
	respMeta := crossserver.NewResponseMetaFromRequest(reqMeta, crossserver.ServiceTypeMap, int32(f.serverID))
	enveloped, err := crossserver.PackMessage(respMeta, crossserver.ServiceTypeMap, crossserver.ServiceTypeGame,
		f.serverID, reqBase.ServerId, respBase)
	if err != nil {
		f.t.Errorf("pack reply: %v", err)
		return
	}
	if err := session.Send(zNet.ProtoIdType(msgID), enveloped); err != nil {
		f.t.Errorf("send reply: %v", err)
	}
}

func (f *fakeMapServer) ensureCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.ensureRequests)
}

func (f *fakeMapServer) enters() []*protocol.MapEnterRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*protocol.MapEnterRequest(nil), f.enterRequests...)
}

func (f *fakeMapServer) moves() []*protocol.MapMoveRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*protocol.MapMoveRequest(nil), f.moveRequests...)
}

func (f *fakeMapServer) leaves() []*protocol.MapLeaveRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*protocol.MapLeaveRequest(nil), f.leaveRequests...)
}

// waitFor 轮询等待条件成立（网络投递是异步的）。
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("等待超时: %s", what)
}

// countingGlobalServer 假的跨服分配接口，并记录被调用次数（用于验证本地缓存真的削掉了重复 HTTP）。
type countingGlobalServer struct {
	*httptest.Server
	mu       sync.Mutex
	calls    int
	serverID uint32
	address  string
}

func (g *countingGlobalServer) callCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls
}

// retarget 让后续分配指向新的落点（模拟承载服务器切换/故障转移）。
func (g *countingGlobalServer) retarget(serverID uint32, address string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.serverID, g.address = serverID, address
}

func newCountingGlobalServer(t *testing.T, serverID uint32, groupID, address string, fail bool) *countingGlobalServer {
	t.Helper()
	g := &countingGlobalServer{serverID: serverID, address: address}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/cross/allocate", func(w http.ResponseWriter, r *http.Request) {
		var req protocol.CrossAllocateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		g.mu.Lock()
		g.calls++
		curServerID, curAddress := g.serverID, g.address
		g.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if fail {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(&protocol.CrossAllocateResponse{
				Result: 1, ErrorMsg: "no healthy map server", ActivityId: req.ActivityId,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(&protocol.CrossAllocateResponse{
			Result:      0,
			ActivityId:  req.ActivityId,
			MapServerId: int32(curServerID),
			GroupId:     groupID,
			Address:     curAddress,
		})
	})

	g.Server = httptest.NewServer(mux)
	t.Cleanup(g.Close)
	return g
}

// fakeGlobalServer 假的跨服分配接口。
func fakeGlobalServer(t *testing.T, serverID uint32, groupID, address string, fail bool) *httptest.Server {
	t.Helper()
	return newCountingGlobalServer(t, serverID, groupID, address, fail).Server
}

// newTestMapService 造一个可用于跨服链路的 MapService（不跑 Start，避免依赖 Excel 配置表）。
func newTestMapService(t *testing.T, ownServerID int, globalAddr string) (*MapService, *connection.ConnectionManager) {
	t.Helper()
	cfg := &config.Config{}
	cfg.Server.ServerID = ownServerID
	cfg.GlobalServer.GlobalServerAddr = globalAddr

	cm := connection.NewConnectionManager(cfg)
	ms := NewMapService(cfg, nil)
	ms.SetConnectionManager(cm)
	cm.SetMapResponseHandler(ms)
	return ms, cm
}

// hostPort 去掉 httptest.Server URL 的 scheme，得到 GlobalServerAddr 形态的 host:port。
func hostPort(url string) string { return strings.TrimPrefix(url, "http://") }

// TestCrossEnter_FullOrchestration 分配 → 连外域 → 建实例 → 进图，全链落到被分配的那台服务器。
func TestCrossEnter_FullOrchestration(t *testing.T) {
	const (
		crossServerID = uint32(201)
		instanceMapID = int32(100001)
		activityID    = int64(70001)
		mapConfigID   = int32(5001)
		playerID      = 60001
	)

	fakeMap := newFakeMapServer(t, crossServerID, instanceMapID)
	global := fakeGlobalServer(t, crossServerID, "2", fakeMap.addr, false)

	ms, _ := newTestMapService(t, 101, hostPort(global.URL))

	result, err := ms.EnterCrossServerMap(playerID, activityID, mapConfigID, common.Vector3{X: 10, Y: 20, Z: 0})
	if err != nil {
		t.Fatalf("EnterCrossServerMap: %v", err)
	}

	if result.MapServerID != crossServerID {
		t.Fatalf("承载服务器应为分配结果 %d, got %d", crossServerID, result.MapServerID)
	}
	if int32(result.MapID) != instanceMapID {
		t.Fatalf("实例 mapID 应来自目标 MapServer 的应答 %d, got %d", instanceMapID, result.MapID)
	}

	// 620 请求带对了活动与地图配置。
	waitFor(t, "ensure 请求送达", func() bool { return fakeMap.ensureCount() == 1 })
	if got := fakeMap.ensureRequests[0]; got.ActivityId != activityID || got.MapConfigId != mapConfigID {
		t.Fatalf("ensure 请求内容错: %+v", got)
	}

	// 400 进图请求落到了外域那台，且用的是实例 mapID。
	waitFor(t, "进图请求送达外域 MapServer", func() bool { return len(fakeMap.enters()) == 1 })
	enter := fakeMap.enters()[0]
	if enter.PlayerId != playerID {
		t.Fatalf("进图请求玩家错: %d", enter.PlayerId)
	}
	if int32(enter.MapId) != instanceMapID {
		t.Fatalf("进图应用实例 mapID %d, got %d", instanceMapID, enter.MapId)
	}

	// 绑定已建立：该玩家后续消息都会定点发往这台服务器。
	if sid, bound := ms.IsInCrossServerInstance(playerID); !bound || sid != crossServerID {
		t.Fatalf("应绑定到 %d, got sid=%d bound=%v", crossServerID, sid, bound)
	}
}

// TestCrossEnter_DoesNotLeakIntoLocalRealmRouting 最关键的一条：跨服玩家的后续消息必须定点发到
// 外域服务器，**即使本 realm 恰好有一张同号的地图**（实例 mapID 跨 MapServer 会撞号）。
func TestCrossEnter_DoesNotLeakIntoLocalRealmRouting(t *testing.T) {
	const (
		crossServerID = uint32(201)
		instanceMapID = int32(100001) // 与本 realm 的某张实例图同号——正是要防的撞号场景
		activityID    = int64(70002)
		playerID      = 60002
	)

	crossMap := newFakeMapServer(t, crossServerID, instanceMapID)
	localMap := newFakeMapServer(t, 102, instanceMapID)
	global := fakeGlobalServer(t, crossServerID, "2", crossMap.addr, false)

	ms, cm := newTestMapService(t, 101, hostPort(global.URL))

	// 本 realm 也注册了 mapID=100001 的连接（模拟本服的同号实例图）。
	if err := cm.ConnectToMapServer(localMap.addr, []int{int(instanceMapID)}); err != nil {
		t.Fatalf("connect local map server: %v", err)
	}

	if _, err := ms.EnterCrossServerMap(playerID, activityID, 5001, common.Vector3{X: 1, Y: 2, Z: 0}); err != nil {
		t.Fatalf("EnterCrossServerMap: %v", err)
	}
	waitFor(t, "进图落到外域", func() bool { return len(crossMap.enters()) == 1 })

	// 进图之后的普通地图消息（移动）同样必须定点发外域。
	if err := ms.sendMapMoveRequest(playerID, 60002, id.MapIdType(instanceMapID), common.Vector3{X: 5, Y: 6, Z: 0}); err != nil {
		t.Fatalf("sendMapMoveRequest: %v", err)
	}
	waitFor(t, "移动请求落到外域", func() bool { return len(crossMap.moves()) == 1 })

	// 本 realm 那台应当**一条都没收到**。
	time.Sleep(150 * time.Millisecond) // 给可能的误投一点到达时间
	if n := len(localMap.enters()); n != 0 {
		t.Fatalf("跨服玩家的进图请求串到了本 realm 的同号地图上（%d 条）", n)
	}
	if n := len(localMap.moves()); n != 0 {
		t.Fatalf("跨服玩家的移动请求串到了本 realm 的同号地图上（%d 条）", n)
	}
}

// TestCrossEnter_AllocateFailureLeavesNoBinding 分配失败 → 报错且不留绑定（玩家仍在本 realm）。
func TestCrossEnter_AllocateFailureLeavesNoBinding(t *testing.T) {
	global := fakeGlobalServer(t, 0, "", "", true)
	ms, _ := newTestMapService(t, 101, hostPort(global.URL))

	if _, err := ms.EnterCrossServerMap(60003, 70003, 5001, common.Vector3{}); err == nil {
		t.Fatalf("分配失败时应报错")
	}
	if _, bound := ms.IsInCrossServerInstance(60003); bound {
		t.Fatalf("失败不得留下绑定，否则该玩家后续消息会被永久发往外域")
	}
}

// TestCrossEnter_EnsureFailureLeavesNoBinding 目标拒绝建实例 → 报错且不留绑定。
func TestCrossEnter_EnsureFailureLeavesNoBinding(t *testing.T) {
	fakeMap := newFakeMapServer(t, 201, 0)
	fakeMap.ensureFail = true
	global := fakeGlobalServer(t, 201, "2", fakeMap.addr, false)

	ms, _ := newTestMapService(t, 101, hostPort(global.URL))

	if _, err := ms.EnterCrossServerMap(60004, 70004, 5001, common.Vector3{}); err == nil {
		t.Fatalf("目标拒绝建实例时应报错")
	}
	if _, bound := ms.IsInCrossServerInstance(60004); bound {
		t.Fatalf("失败不得留下绑定")
	}
	if n := len(fakeMap.enters()); n != 0 {
		t.Fatalf("没拿到实例就不该发进图请求, got %d", n)
	}
}

// TestLeaveCrossServerMap_SendsToCrossServerThenUnbinds 离开必须**先**按绑定把离开请求发到外域、
// 再解绑；顺序反了会把离开请求错发到本 realm，外域那边的玩家对象永远留在实例里。
func TestLeaveCrossServerMap_SendsToCrossServerThenUnbinds(t *testing.T) {
	const (
		crossServerID = uint32(201)
		instanceMapID = int32(100007)
		playerID      = 60005
	)

	fakeMap := newFakeMapServer(t, crossServerID, instanceMapID)
	global := fakeGlobalServer(t, crossServerID, "2", fakeMap.addr, false)
	ms, _ := newTestMapService(t, 101, hostPort(global.URL))

	result, err := ms.EnterCrossServerMap(playerID, 70005, 5001, common.Vector3{})
	if err != nil {
		t.Fatalf("EnterCrossServerMap: %v", err)
	}

	if err := ms.LeaveCrossServerMap(playerID, result.MapID); err != nil {
		t.Fatalf("LeaveCrossServerMap: %v", err)
	}
	waitFor(t, "离开请求送达外域", func() bool { return len(fakeMap.leaves()) == 1 })

	if got := fakeMap.leaves()[0]; got.PlayerId != playerID || int32(got.MapId) != instanceMapID {
		t.Fatalf("离开请求内容错: %+v", got)
	}
	if _, bound := ms.IsInCrossServerInstance(playerID); bound {
		t.Fatalf("离开后应解绑，玩家回到本 realm 路由")
	}

	// 未在跨服实例中的玩家调离开应明确报错，而不是静默发到错地方。
	if err := ms.LeaveCrossServerMap(99999, 1001); err == nil {
		t.Fatalf("未在跨服实例中的玩家应报错")
	}
}

// TestHandlePlayerLeaveMap_UnbindsCrossPlayer 客户端的 MSG_MAP_LEAVE 与登出清理走的都是
// HandlePlayerLeaveMap（不是 LeaveCrossServerMap）。它必须为跨服玩家分流，否则：
//   - 本 GameServer 没有跨服实例那张图的本地 Map 对象 → GetMap 直接报错返回 →
//     **离开请求根本没发出去**，玩家对象永远留在外域的实例里；
//   - **绑定没清** → 该玩家回本服后，进本地图的消息还会被定点发去外域那台服务器。
func TestHandlePlayerLeaveMap_UnbindsCrossPlayer(t *testing.T) {
	const (
		crossServerID = uint32(201)
		instanceMapID = int32(100011)
		playerID      = 60006
	)

	fakeMap := newFakeMapServer(t, crossServerID, instanceMapID)
	global := fakeGlobalServer(t, crossServerID, "2", fakeMap.addr, false)
	ms, _ := newTestMapService(t, 101, hostPort(global.URL))

	result, err := ms.EnterCrossServerMap(playerID, 70006, 5001, common.Vector3{})
	if err != nil {
		t.Fatalf("EnterCrossServerMap: %v", err)
	}

	// 走普通离开路径（= 客户端 MSG_MAP_LEAVE / 登出清理的那条）。
	if err := ms.HandlePlayerLeaveMap(playerID, result.MapID); err != nil {
		t.Fatalf("HandlePlayerLeaveMap 不应报错（跨服玩家应被分流处理）: %v", err)
	}

	waitFor(t, "离开请求送达外域", func() bool { return len(fakeMap.leaves()) == 1 })
	if got := fakeMap.leaves()[0]; int32(got.MapId) != instanceMapID {
		t.Fatalf("离开请求应发往跨服实例 %d, got %d", instanceMapID, got.MapId)
	}
	if _, bound := ms.IsInCrossServerInstance(playerID); bound {
		t.Fatalf("普通离开路径也必须解绑，否则该玩家回本服后消息仍被发去外域")
	}
}

// TestCrossAllocation_CachedAcrossPlayers 同一活动涌入多个玩家时，只应打**一次** GlobalServer：
// 分配在 GlobalServer 侧本就是粘性的，每人一次 HTTP 拿同一个答案纯属浪费。
func TestCrossAllocation_CachedAcrossPlayers(t *testing.T) {
	const (
		crossServerID = uint32(201)
		instanceMapID = int32(100021)
		activityID    = int64(70011)
	)

	fakeMap := newFakeMapServer(t, crossServerID, instanceMapID)
	global := newCountingGlobalServer(t, crossServerID, "2", fakeMap.addr, false)
	ms, _ := newTestMapService(t, 101, hostPort(global.URL))

	for i := 0; i < 5; i++ {
		if _, err := ms.EnterCrossServerMap(id.PlayerIdType(60100+i), activityID, 5001, common.Vector3{}); err != nil {
			t.Fatalf("第 %d 个玩家进图失败: %v", i, err)
		}
	}

	if n := global.callCount(); n != 1 {
		t.Fatalf("5 个玩家进同一活动只应向 GlobalServer 要一次分配, got %d", n)
	}
	waitFor(t, "5 个玩家的进图请求都到了外域", func() bool { return len(fakeMap.enters()) == 5 })

	// 不同活动不共享缓存。
	if _, err := ms.EnterCrossServerMap(60200, activityID+1, 5001, common.Vector3{}); err != nil {
		t.Fatalf("另一活动进图失败: %v", err)
	}
	if n := global.callCount(); n != 2 {
		t.Fatalf("另一场活动应重新要分配, got %d", n)
	}
}

// TestCrossAllocation_StaleCacheFailsOver 缓存里的落点用不上了（承载服换了/挂了）→
// 必须作废缓存并用新鲜分配重试，而不是把玩家一路往死服务器上送。
func TestCrossAllocation_StaleCacheFailsOver(t *testing.T) {
	const (
		oldServerID = uint32(201)
		newServerID = uint32(301)
		oldMapID    = int32(100022)
		newMapID    = int32(100023)
		activityID  = int64(70012)
	)

	oldMap := newFakeMapServer(t, oldServerID, oldMapID)
	newMap := newFakeMapServer(t, newServerID, newMapID)
	global := newCountingGlobalServer(t, oldServerID, "2", oldMap.addr, false)
	ms, _ := newTestMapService(t, 101, hostPort(global.URL))

	// 先进一次，把落点写进缓存。
	first, err := ms.EnterCrossServerMap(60301, activityID, 5001, common.Vector3{})
	if err != nil {
		t.Fatalf("首次进图失败: %v", err)
	}
	if first.MapServerID != oldServerID {
		t.Fatalf("首次应落到 %d, got %d", oldServerID, first.MapServerID)
	}

	// 承载服务器挂掉，GlobalServer 改分配到新的一台。
	oldMap.srv.Close()
	global.retarget(newServerID, newMap.addr)

	second, err := ms.EnterCrossServerMap(60302, activityID, 5001, common.Vector3{})
	if err != nil {
		t.Fatalf("承载服切换后应能重试成功: %v", err)
	}
	if second.MapServerID != newServerID {
		t.Fatalf("应改落到新承载服 %d, got %d", newServerID, second.MapServerID)
	}
	waitFor(t, "进图请求落到新承载服", func() bool { return len(newMap.enters()) == 1 })
}

// TestReapIdleCrossConnections 没有玩家再绑定的跨域连接会被回收；但**刚建连的不能收**——
// 否则会把正在进图（连上→建实例→绑定，几百毫秒）的玩家的连接掐掉。
func TestReapIdleCrossConnections(t *testing.T) {
	const (
		crossServerID = uint32(201)
		instanceMapID = int32(100012)
		playerID      = 60007
	)

	fakeMap := newFakeMapServer(t, crossServerID, instanceMapID)
	global := fakeGlobalServer(t, crossServerID, "2", fakeMap.addr, false)
	ms, cm := newTestMapService(t, 101, hostPort(global.URL))

	result, err := ms.EnterCrossServerMap(playerID, 70007, 5001, common.Vector3{})
	if err != nil {
		t.Fatalf("EnterCrossServerMap: %v", err)
	}

	// 玩家还在里面 → 不能回收。
	ms.reapIdleCrossConnections()
	if !cm.IsCrossRealmConnected(crossServerID) {
		t.Fatalf("有玩家绑定时不得回收连接")
	}

	// 玩家离开后，连接虽已无人使用，但建连不足 grace 时间，仍不回收（防掐掉在途进图）。
	if err := ms.HandlePlayerLeaveMap(playerID, result.MapID); err != nil {
		t.Fatalf("leave: %v", err)
	}
	ms.reapIdleCrossConnections()
	if !cm.IsCrossRealmConnected(crossServerID) {
		t.Fatalf("刚建连的空闲连接不应被立刻回收（会掐掉正在进图的玩家）")
	}

	// 越过 grace 后才回收。
	idle := cm.IdleCrossRealmServerIDs(map[uint32]bool{}, 0)
	if len(idle) != 1 || idle[0] != crossServerID {
		t.Fatalf("grace=0 时该连接应被判为空闲, got %v", idle)
	}
	cm.DisconnectCrossRealmMapServer(crossServerID)
	if cm.IsCrossRealmConnected(crossServerID) {
		t.Fatalf("回收后不应仍标记为已连接")
	}
}
