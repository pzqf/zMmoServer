package maps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// 跨服物理路由（realm §5.1）——把本 realm 的玩家送进**可能属于别的 realm** 的一张跨服实例地图。
//
// 整条链（各环节的职责边界是这套设计的关键）：
//
//	 1) 问 GlobalServer："这场活动放在哪台 MapServer 上？"        ← 全区唯一决策方，保证各 realm 同一答案
//	 2) 连上那台 MapServer（按 serverID 定点，可能跨 realm）      ← ② 跨 realm 发现/连接
//	 3) 请它建/取该活动的实例地图，拿到物理 mapID                  ← ④ 按 activityID 幂等，多 realm 收敛同一张
//	 4) 绑定 玩家→该 MapServer，此后其地图消息全部定点发过去        ← 不能按 mapID 路由（实例 mapID 跨服会撞号）
//	 5) 发常规 MapEnterRequest 进图                              ← 复用既有进图链（含进场属性快照）
//
// 玩家在跨服实例里的战斗/结算持久变更，仍由 MapServer 按玩家归属经 AttrGrant 回流到**其家服**落库
//（④ 回流本就是多源的：MapServer 记的是"这个玩家来自哪台 GameServer"）。
//
// 注意：这些调用发生在**玩家自己的 actor goroutine** 上（与 Attack 等待响应同一模式），
// 故全链超时必须显著小于 MapHandler 的 5s 回调超时，否则玩家会先收到 timeout 再收到结果。

const (
	// crossAllocateTimeout 问 GlobalServer 要分配的超时。
	crossAllocateTimeout = 2 * time.Second
	// crossEnsureTimeout 等目标 MapServer 建实例应答的超时。
	crossEnsureTimeout = 2 * time.Second
)

// CrossEnterResult 跨服进图结果（供上层回给客户端/排查）。
type CrossEnterResult struct {
	ActivityID  int64
	MapServerID uint32
	MapID       id.MapIdType
	Address     string
}

const (
	// crossMaintainTick 跨服资源维护周期。
	crossMaintainTick = 30 * time.Second
	// crossConnIdleGrace 跨域连接被判为空闲前的最短存活时间。必须显著大于"连上→建实例→绑定"
	// 那几百毫秒，否则会把正在进图的玩家的连接收掉。
	crossConnIdleGrace = 5 * time.Minute
)

// StartCrossMaintainLoop 周期维护跨服资源：清关联器里超时未回的挂起项 + 回收没人再用的跨域连接。
// 由 MapService.Start 起；ctx 结束即退出。
func (ms *MapService) StartCrossMaintainLoop(ctx context.Context) {
	ticker := time.NewTicker(crossMaintainTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ms.crossReqRouter.Cleanup()
			ms.reapIdleCrossConnections()
		}
	}
}

// reapIdleCrossConnections 断开"已无任何玩家绑定"的跨域连接（活动结束/人走光后不该留着长连）。
func (ms *MapService) reapIdleCrossConnections() {
	if ms.connectionManager == nil {
		return
	}

	inUse := make(map[uint32]bool)
	ms.crossBindings.Range(func(_ int64, serverID uint32) bool {
		inUse[serverID] = true
		return true
	})

	for _, serverID := range ms.connectionManager.IdleCrossRealmServerIDs(inUse, crossConnIdleGrace) {
		zLog.Info("Reaping idle cross-realm connection", zap.Uint32("map_server_id", serverID))
		ms.connectionManager.DisconnectCrossRealmMapServer(serverID)
	}
}

// EnterCrossServerMap 把玩家送进一场跨服活动的实例地图。失败时不留下任何绑定（玩家仍在原 realm）。
func (ms *MapService) EnterCrossServerMap(playerID id.PlayerIdType, activityID int64, mapConfigID int32, pos common.Vector3) (*CrossEnterResult, error) {
	if activityID <= 0 {
		return nil, fmt.Errorf("invalid activity id %d", activityID)
	}
	if ms.connectionManager == nil {
		return nil, fmt.Errorf("connection manager not set")
	}

	// 1) 全区唯一决策方给出落点。
	alloc, err := ms.allocateCrossActivity(activityID, mapConfigID)
	if err != nil {
		return nil, err
	}

	// 2) 定点连上那台 MapServer（本 realm 的服务器也可能被选中，一样按 serverID 走定点连接：
	//    跨服实例的 mapID 不能进 mapID→连接 的路由表）。
	if err := ms.connectionManager.ConnectToCrossRealmMapServer(alloc.ServerID, alloc.Address); err != nil {
		return nil, fmt.Errorf("connect cross map server %d(%s): %w", alloc.ServerID, alloc.Address, err)
	}

	// 3) 建/取实例（按 activityID 幂等，多 realm 收敛到同一张）。
	mapID, err := ms.ensureCrossInstance(alloc.ServerID, activityID, mapConfigID)
	if err != nil {
		return nil, err
	}

	// 4) 绑定后其地图消息才会定点发往该服务器——必须在发进图请求之前绑。
	ms.crossBindings.Store(int64(playerID), alloc.ServerID)

	// 5) 复用常规进图链（带进场属性快照）。
	if err := ms.sendMapEnterRequest(playerID, mapID, pos); err != nil {
		ms.crossBindings.Delete(int64(playerID)) // 没进去就不能留绑定，否则该玩家后续消息全被发去外域
		return nil, fmt.Errorf("send cross map enter: %w", err)
	}

	if ms.playerMapManager != nil {
		ms.playerMapManager.SetPlayerMap(playerID, mapID, alloc.ServerID)
	}

	zLog.Info("Player entered cross-server instance",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("activity_id", activityID),
		zap.Uint32("map_server_id", alloc.ServerID),
		zap.String("group_id", alloc.GroupID),
		zap.Int32("map_id", int32(mapID)))

	return &CrossEnterResult{
		ActivityID:  activityID,
		MapServerID: alloc.ServerID,
		MapID:       mapID,
		Address:     alloc.Address,
	}, nil
}

// LeaveCrossServerMap 离开跨服实例：先按绑定把离开请求发到承载服务器，再解绑回本 realm。
// 顺序不能反——先解绑会让离开请求错发到本 realm，外域那边的玩家对象永远留在实例里。
func (ms *MapService) LeaveCrossServerMap(playerID id.PlayerIdType, mapID id.MapIdType) error {
	if _, bound := ms.crossBindings.Load(int64(playerID)); !bound {
		return fmt.Errorf("player %d is not in a cross-server instance", playerID)
	}

	err := ms.sendMapLeaveRequest(playerID, mapID)
	ms.crossBindings.Delete(int64(playerID))
	if ms.playerMapManager != nil {
		ms.playerMapManager.RemovePlayerMap(playerID)
	}

	zLog.Info("Player left cross-server instance",
		zap.Int64("player_id", int64(playerID)), zap.Int32("map_id", int32(mapID)))
	return err
}

// IsInCrossServerInstance 玩家当前是否在跨服实例里（供上层判断走哪条离开路径）。
func (ms *MapService) IsInCrossServerInstance(playerID id.PlayerIdType) (uint32, bool) {
	return ms.crossBindings.Load(int64(playerID))
}

// EnterCrossMap 实现 player.MapOperator 接口（玩家 actor 调用）。
func (ms *MapService) EnterCrossMap(playerID id.PlayerIdType, activityID int64, mapConfigID int32, pos common.Vector3) (uint32, id.MapIdType, error) {
	result, err := ms.EnterCrossServerMap(playerID, activityID, mapConfigID, pos)
	if err != nil {
		return 0, 0, err
	}
	return result.MapServerID, result.MapID, nil
}

// crossAllocation GlobalServer 分配结果。
type crossAllocation struct {
	ServerID uint32
	GroupID  string
	Address  string
}

// allocateCrossActivity 问 GlobalServer 这场活动落在哪台 MapServer 上。
// 走 HTTP：GlobalServer 是全区全服的 HTTP 进程（账号/服务器列表），无 TCP 栈；分配是低频控制面调用。
func (ms *MapService) allocateCrossActivity(activityID int64, mapConfigID int32) (*crossAllocation, error) {
	addr := ""
	if ms.config != nil {
		addr = ms.config.GlobalServer.GlobalServerAddr
	}
	if addr == "" {
		return nil, fmt.Errorf("GlobalServerAddr not configured, cannot allocate cross-server activity")
	}

	body, err := json.Marshal(&protocol.CrossAllocateRequest{
		ActivityId:  activityID,
		MapConfigId: mapConfigID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal allocate request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), crossAllocateTimeout)
	defer cancel()

	url := fmt.Sprintf("http://%s/api/v1/cross/allocate", addr)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build allocate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ms.crossHTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call cross allocate %s: %w", url, err)
	}
	defer resp.Body.Close()

	var out protocol.CrossAllocateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode allocate response (http %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK || out.Result != 0 {
		return nil, fmt.Errorf("cross allocate rejected (http %d): %s", resp.StatusCode, out.ErrorMsg)
	}
	if out.MapServerId <= 0 || out.Address == "" {
		return nil, fmt.Errorf("cross allocate returned empty target")
	}

	return &crossAllocation{
		ServerID: uint32(out.MapServerId),
		GroupID:  out.GroupId,
		Address:  out.Address,
	}, nil
}

// ensureCrossInstance 请目标 MapServer 建/取该活动的实例地图，返回其物理 mapID。
// 走 620/621 请求-响应：应答经 ConnectionManager → HandleCrossInstanceEnsureResponse 按 requestID 关联唤醒。
func (ms *MapService) ensureCrossInstance(mapServerID uint32, activityID int64, mapConfigID int32) (id.MapIdType, error) {
	reqData, err := proto.Marshal(&protocol.CrossInstanceEnsureRequest{
		ActivityId:  activityID,
		MapConfigId: mapConfigID,
		Name:        fmt.Sprintf("CrossActivity_%d", activityID),
	})
	if err != nil {
		return 0, fmt.Errorf("marshal ensure request: %w", err)
	}

	// 混入本机 serverID 使 requestID 全集群唯一（见 crossserver.ComposeRequestID）。
	reqID := crossserver.ComposeRequestID(int32(ms.config.Server.ServerID), ms.crossReqRouter.NextRequestID())
	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ms.config.Server.ServerID))
	meta.RequestID = reqID
	meta.TraceID = reqID

	protoID := int(protocol.InternalMsgId_MSG_INTERNAL_CROSS_INSTANCE_ENSURE)
	base := crossserver.BuildBaseMessage(uint32(protoID), 0, uint32(ms.config.Server.ServerID), 0, reqData)
	enveloped, err := crossserver.PackMessage(meta, crossserver.ServiceTypeGame, crossserver.ServiceTypeMap,
		uint32(ms.config.Server.ServerID), mapServerID, base)
	if err != nil {
		return 0, fmt.Errorf("pack ensure request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), crossEnsureTimeout)
	defer cancel()

	respData, err := ms.crossReqRouter.SendRequest(ctx, reqID, func() error {
		return ms.connectionManager.SendToCrossRealmMapServer(mapServerID, protoID, enveloped)
	})
	if err != nil {
		return 0, fmt.Errorf("ensure cross instance on server %d: %w", mapServerID, err)
	}

	var resp protocol.CrossInstanceEnsureResponse
	if err := proto.Unmarshal(respData, &resp); err != nil {
		return 0, fmt.Errorf("unmarshal ensure response: %w", err)
	}
	if !resp.Success {
		return 0, fmt.Errorf("target refused to create cross instance: %s", resp.ErrorMsg)
	}
	if resp.MapId <= 0 {
		return 0, fmt.Errorf("target returned invalid cross instance map id %d", resp.MapId)
	}
	return id.MapIdType(resp.MapId), nil
}

// HandleCrossInstanceEnsureResponse 实现 connection.MapResponseHandler：把 621 的内层字节按 requestID
// 回填给等待中的 ensureCrossInstance。
func (ms *MapService) HandleCrossInstanceEnsureResponse(requestID uint64, payload []byte) {
	if !ms.crossReqRouter.CompleteRequest(requestID, payload, nil) {
		zLog.Warn("Cross instance ensure response for unknown/expired request",
			zap.Uint64("request_id", requestID))
	}
}
