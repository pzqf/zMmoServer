package maps

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pzqf/zEngine/zNet"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/tables"
	consistency "github.com/pzqf/zEngine/zConsistency"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoServer/GameServer/game/object"
	"github.com/pzqf/zMmoServer/GameServer/game/player"
	"github.com/pzqf/zMmoServer/GameServer/net/protolayer"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// MapService 地图服务
// 职责：管理本地地图实例、处理玩家地图操作（进入/离开/移动/攻击）、与 MapServer 跨服通信
// 路由依赖：通过 MapServerManager 获取 mapID → MapServer 的路由信息
type MapService struct {
	config            *config.Config
	protocol          protolayer.Protocol
	maps              *zMap.TypedMap[id.MapIdType, *Map]
	connectionManager *connection.ConnectionManager
	mapServerManager  *MapServerManager
	playerMapManager  *player.PlayerMapManager
	playerManager     *player.PlayerManager
	pendingAttacks    *zMap.TypedMap[string, chan mapAttackResult]
	pendingByReq      *zMap.TypedMap[uint64, chan mapAttackResult]
	outbox            consistency.OutboxStore
	inbox             consistency.InboxStore
	retryCtx          context.Context
	retryCancel       context.CancelFunc
	onOutboxChanged   func(OutboxStats)
	// crossBindings 参加跨服活动的玩家 → 承载其实例的 MapServer serverID（跨服物理路由 ②）。
	// 有绑定的玩家，其全部地图消息按 serverID **定点**发到那台服务器（可能在别的 realm），
	// 不再按 mapID 走本 realm 路由——跨服实例的 mapID 是各 MapServer 实例池的派生号，会撞号。
	crossBindings *zMap.TypedMap[int64, uint32]
	// crossReqRouter 等"建跨服实例"应答（621）的关联器。
	crossReqRouter *zNet.RequestRouter
	// crossHTTP 调 GlobalServer 分配接口的 HTTP 客户端（短超时）。
	crossHTTP *http.Client
	// crossAllocCache 跨服活动分配的本地短 TTL 缓存：GlobalServer 侧是粘性分配，
	// 同一活动涌入的玩家没必要每人打一次 HTTP 去拿同一个答案。
	crossAllocCache *zMap.TypedMap[int64, *cachedCrossAllocation]
}

const maxOutboxRetry = 5

type OutboxStats struct {
	Pending int
	Dead    int
}

type mapAttackResult struct {
	damage   int64
	targetHP int64
	success  bool
	errorMsg string
}

// NewMapService 创建地图服务
func NewMapService(cfg *config.Config, protocol protolayer.Protocol) *MapService {
	return &MapService{
		config:         cfg,
		protocol:       protocol,
		maps:           zMap.NewTypedMap[id.MapIdType, *Map](),
		pendingAttacks: zMap.NewTypedMap[string, chan mapAttackResult](),
		pendingByReq:   zMap.NewTypedMap[uint64, chan mapAttackResult](),
		outbox:         consistency.NewMemoryOutbox(),
		inbox:          consistency.NewMemoryInbox(),
		crossBindings:   zMap.NewTypedMap[int64, uint32](),
		crossReqRouter:  zNet.NewRequestRouter(crossEnsureTimeout),
		crossHTTP:       &http.Client{Timeout: crossAllocateTimeout},
		crossAllocCache: zMap.NewTypedMap[int64, *cachedCrossAllocation](),
	}
}

// SetConnectionManager 设置连接管理器
func (ms *MapService) SetConnectionManager(connManager *connection.ConnectionManager) {
	ms.connectionManager = connManager
}

// SetConsistencyStores 用持久化（或其他）Outbox/Inbox 替换默认的内存实现（Phase 3.4）。
// 须在 Start 前调用。传 nil 的参数保留原有实现。
func (ms *MapService) SetConsistencyStores(outbox consistency.OutboxStore, inbox consistency.InboxStore) {
	if outbox != nil {
		ms.outbox = outbox
	}
	if inbox != nil {
		ms.inbox = inbox
	}
}

// SetMapServerManager 设置地图服务器管理器
func (ms *MapService) SetMapServerManager(mapServerManager *MapServerManager) {
	ms.mapServerManager = mapServerManager
}

// SetPlayerMapManager 设置玩家地图管理器
func (ms *MapService) SetPlayerMapManager(playerMapManager *player.PlayerMapManager) {
	ms.playerMapManager = playerMapManager
}

// SetPlayerManager 设置玩家管理器（用于 AOI 视野消息投递）
func (ms *MapService) SetPlayerManager(pm *player.PlayerManager) {
	ms.playerManager = pm
}

// SetOnOutboxStatsChanged 设置Outbox状态变更回调（用于实时监控更新）
func (ms *MapService) SetOnOutboxStatsChanged(cb func(OutboxStats)) {
	ms.onOutboxChanged = cb
}

// GetOutboxPendingMessages 返回待重试消息快照（用于监控/排障）
func (ms *MapService) GetOutboxPendingMessages(limit int) []consistency.OutboxMessage {
	return ms.outbox.ListPending(limit)
}

// GetOutboxDeadLetters 返回死信消息快照（用于监控/排障）
func (ms *MapService) GetOutboxDeadLetters(limit int) []consistency.OutboxMessage {
	return ms.outbox.ListDeadLetters(limit)
}

// GetOutboxStats 返回当前Outbox统计（用于监控/日志）
func (ms *MapService) GetOutboxStats() OutboxStats {
	return OutboxStats{
		Pending: ms.outbox.CountPending(),
		Dead:    ms.outbox.CountDeadLetters(),
	}
}

// PurgeOutboxDeadLetters 清理超过指定时长的死信
func (ms *MapService) PurgeOutboxDeadLetters(olderThan time.Duration) int {
	removed := ms.outbox.PurgeDeadLetters(olderThan)
	ms.publishOutboxStats()
	return removed
}

// Start 启动地图服务
func (ms *MapService) Start(ctx context.Context) error {
	zLog.Info("Starting MapService...")

	if err := ms.loadMaps(); err != nil {
		return err
	}
	ms.retryCtx, ms.retryCancel = context.WithCancel(ctx)
	go ms.outboxRetryLoop()
	// 跨服资源维护：清关联器超时挂起项 + 回收无人使用的跨域连接（见 cross_service.go）。
	go ms.StartCrossMaintainLoop(ms.retryCtx)

	// 客户端 AOI 视野改由 MapServer 分层 AOI 单一权威驱动（见 HandleAOINotify）；
	// GameServer 自建的本地 AOI 订阅（原 subscribeAOIEvents）已移除，避免冗余 + 跨 layer 串视野。

	zLog.Info("MapService started successfully")
	return nil
}

// Stop 停止地图服务
func (ms *MapService) Stop(ctx context.Context) error {
	zLog.Info("Stopping MapService...")
	if ms.retryCancel != nil {
		ms.retryCancel()
	}

	// 清理地图
	ms.maps.Clear()

	zLog.Info("MapService stopped")
	return nil
}

// loadMaps 加载地图
func (ms *MapService) loadMaps() error {
	// 从Excel配置表加载地图数据
	mapTableLoader := tables.NewMapTableLoader()
	// 去硬编码：经健壮解析器定位配置表目录（此前硬编码 "../resources/excel_tables"，
	// 从 repo 根跑时指向仓库外→退默认地图）。见 zCommon/config/tables/resource_dir.go。
	excelDir := tables.ExcelTablesDir()

	err := mapTableLoader.Load(excelDir)
	if err != nil {
		zLog.Warn("Failed to load map tables, using default maps", zap.Error(err))
		ms.loadDefaultMaps()
		return nil
	}

	// 加载所有地图
	maps := mapTableLoader.GetAllMaps()
	if len(maps) == 0 {
		zLog.Warn("No maps found in config, using default maps")
		ms.loadDefaultMaps()
		return nil
	}

	for mapID, mapConfig := range maps {
		newMap := NewMap(id.MapIdType(mapID), mapID, mapConfig.Name, float32(mapConfig.Width), float32(mapConfig.Height))
		ms.maps.Store(id.MapIdType(mapID), newMap)
		zLog.Info("Map loaded from config", zap.Int32("map_id", mapID), zap.String("name", mapConfig.Name))
	}

	return nil
}

// loadDefaultMaps 加载默认地图
func (ms *MapService) loadDefaultMaps() {
	// 加载新手村地图
	mapID := id.MapIdType(1001)
	mapName := "新手村"
	width, height := float32(500), float32(500)

	ms.maps.Store(mapID, NewMap(mapID, 1001, mapName, width, height))

	// 加载主城地图
	mapID2 := id.MapIdType(1002)
	mapName2 := "主城"
	width2, height2 := float32(800), float32(800)

	ms.maps.Store(mapID2, NewMap(mapID2, 1002, mapName2, width2, height2))

	// 加载野外地图
	mapID3 := id.MapIdType(2001)
	mapName3 := "野外"
	width3, height3 := float32(1000), float32(1000)

	ms.maps.Store(mapID3, NewMap(mapID3, 2001, mapName3, width3, height3))

	zLog.Info("Default maps loaded")
}

// GetMap 获取地图
func (ms *MapService) GetMap(mapID id.MapIdType) (*Map, error) {
	m, exists := ms.maps.Load(mapID)
	if !exists {
		return nil, fmt.Errorf("map not found: %d", mapID)
	}
	return m, nil
}

// GetDefaultMapID 返回可用的默认地图ID
func (ms *MapService) GetDefaultMapID() id.MapIdType {
	var defaultMapID id.MapIdType
	ms.maps.Range(func(mapID id.MapIdType, m *Map) bool {
		defaultMapID = mapID
		return false
	})

	if defaultMapID == 0 {
		return id.MapIdType(1001)
	}
	return defaultMapID
}

// HandlePlayerEnterMap 处理玩家进入地图
func (ms *MapService) HandlePlayerEnterMap(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	m, err := ms.GetMap(mapID)
	if err != nil {
		return err
	}

	playerObj := object.NewGameObjectWithType(id.ObjectIdType(playerID), "player", common.GameObjectTypePlayer)
	playerObj.SetPosition(pos)

	m.AddPlayer(playerID, playerObj)

	var mapServerID uint32
	if ms.mapServerManager != nil {
		mapServerID, _ = ms.mapServerManager.GetMapServerID(mapID)
	}

	if ms.playerMapManager != nil {
		ms.playerMapManager.SetPlayerMap(playerID, mapID, mapServerID)
	}

	err = ms.sendMapEnterRequest(playerID, mapID, pos)
	if err != nil {
		zLog.Warn("Failed to send map enter request to MapServer", zap.Error(err))
	}

	zLog.Info("Player entered map",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)),
		zap.Uint32("map_server_id", mapServerID),
		zap.Float32("x", pos.X),
		zap.Float32("y", pos.Y),
		zap.Float32("z", pos.Z))

	return nil
}

// sendMapMessage 发送消息到 MapServer。
// protoId 为 InternalMsgId（400/402/404/406），innerData 为具体业务消息（MapEnterRequest 等）
// 的 protobuf 字节。统一经 crossserver 共用 codec 打包（Envelope + protobuf CrossServerMessage），
// 详见 docs/协议契约.md，禁止在此手写编码。
func (ms *MapService) sendMapMessage(mapID id.MapIdType, protoId int, innerData []byte, playerID id.PlayerIdType, meta crossserver.Meta) error {
	if ms.connectionManager == nil {
		return fmt.Errorf("connection manager not set")
	}

	base := crossserver.BuildBaseMessage(uint32(protoId), uint64(playerID),
		uint32(ms.config.Server.ServerID), uint32(mapID), innerData)

	enveloped, err := crossserver.PackMessage(meta, crossserver.ServiceTypeGame, crossserver.ServiceTypeMap,
		uint32(ms.config.Server.ServerID), 0, base)
	if err != nil {
		return fmt.Errorf("failed to pack cross server message: %w", err)
	}

	// 跨服活动中的玩家：定点发到承载其实例的那台 MapServer（可能在别的 realm），不按 mapID 路由。
	crossServerID, isCross := ms.crossBindings.Load(int64(playerID))

	msg := consistency.OutboxMessage{
		RequestID:   meta.RequestID,
		Topic:       fmt.Sprintf("map:%d:proto:%d", mapID, protoId),
		TargetMapID: int32(mapID),
		ProtoID:     int32(protoId),
		Payload:     enveloped,
	}
	if isCross {
		// 记下定点目标：重投时必须仍发到那台服务器，按 mapID 重投会串到本 realm 的错服务器上。
		msg.TargetServerID = crossserver.ServerKey(int32(crossServerID))
	}
	ms.outbox.Add(msg)
	ms.outbox.MarkAttempt(meta.RequestID, nil)
	ms.publishOutboxStats()

	if isCross {
		err = ms.connectionManager.SendToCrossRealmMapServer(crossServerID, protoId, enveloped)
	} else {
		err = ms.sendFramedToMap(int(mapID), protoId, enveloped)
	}
	if err != nil {
		ms.outbox.MarkAttempt(meta.RequestID, err)
		zLog.Warn("Cross-server send failed",
			zap.Uint64("trace_id", meta.TraceID),
			zap.Uint64("request_id", meta.RequestID),
			zap.Int("proto_id", protoId),
			zap.Int32("map_id", int32(mapID)),
			zap.Error(err))
		return err
	}
	ms.outbox.MarkSent(meta.RequestID)
	ms.publishOutboxStats()
	zLog.Debug("Cross-server send succeeded",
		zap.Uint64("trace_id", meta.TraceID),
		zap.Uint64("request_id", meta.RequestID),
		zap.Int("proto_id", protoId),
		zap.Int32("map_id", int32(mapID)))
	return nil
}

func (ms *MapService) sendFramedToMap(mapID int, protoId int, enveloped []byte) error {
	err := ms.connectionManager.SendToMap(int(mapID), protoId, enveloped)
	if err != nil {
		return fmt.Errorf("failed to send map message: %w", err)
	}
	return nil
}

func (ms *MapService) outboxRetryLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	inboxCleanupCounter := 0
	for {
		select {
		case <-ms.retryCtx.Done():
			return
		case <-ticker.C:
			pending := ms.outbox.ListPending(50)
			if dead := ms.outbox.ListDeadLetters(1); len(dead) > 0 {
				zLog.Warn("Outbox dead-letter exists", zap.Int("sample_count", len(dead)))
			}
			for _, msg := range pending {
				if msg.Attempts >= maxOutboxRetry {
					ms.outbox.MarkDeadLetter(msg.RequestID, "max retry attempts exceeded")
					zLog.Error("Outbox message moved to dead-letter",
						zap.Uint64("request_id", msg.RequestID),
						zap.Int("attempts", msg.Attempts),
						zap.String("topic", msg.Topic))
					ms.publishOutboxStats()
					continue
				}

				// 跨服定点消息：按 TargetServerID 重投到那台 MapServer，绝不能回落到 mapID 路由
				// （跨服实例 mapID 会与本 realm 的实例撞号，重投会打到错的服务器上）。
				if msg.TargetServerID != "" {
					crossServerID, perr := strconv.ParseUint(msg.TargetServerID, 10, 32)
					if perr != nil || msg.ProtoID == 0 {
						ms.outbox.MarkDeadLetter(msg.RequestID, "invalid cross-realm target metadata")
						ms.publishOutboxStats()
						continue
					}
					ms.outbox.MarkAttempt(msg.RequestID, nil)
					if err := ms.connectionManager.SendToCrossRealmMapServer(uint32(crossServerID), int(msg.ProtoID), msg.Payload); err != nil {
						ms.outbox.MarkAttempt(msg.RequestID, err)
						continue
					}
					ms.outbox.MarkSent(msg.RequestID)
					ms.publishOutboxStats()
					continue
				}

				targetMapID, protoID := msg.TargetMapID, msg.ProtoID
				if targetMapID == 0 || protoID == 0 {
					parts := strings.Split(msg.Topic, ":")
					if len(parts) >= 4 {
						if mapID, err := strconv.Atoi(parts[1]); err == nil {
							targetMapID = int32(mapID)
						}
						if pid, err := strconv.Atoi(parts[3]); err == nil {
							protoID = int32(pid)
						}
					}
				}
				if targetMapID == 0 || protoID == 0 {
					ms.outbox.MarkDeadLetter(msg.RequestID, "invalid target metadata")
					ms.publishOutboxStats()
					continue
				}
				ms.outbox.MarkAttempt(msg.RequestID, nil)
				if err := ms.sendFramedToMap(int(targetMapID), int(protoID), msg.Payload); err != nil {
					ms.outbox.MarkAttempt(msg.RequestID, err)
					continue
				}
				ms.outbox.MarkSent(msg.RequestID)
				ms.publishOutboxStats()
			}

			// INF-11: 周期清理请求去重表，防止 inbox 随每次跨服攻击响应无界增长（GameServer 侧，
			// 对应 MapServer 的 inboxCleanupLoop）。每 ~60 拍（约 1 分钟）清一次 5 分钟前的记录。
			inboxCleanupCounter++
			if inboxCleanupCounter >= 60 {
				inboxCleanupCounter = 0
				if ms.inbox != nil {
					ms.inbox.Cleanup(5 * time.Minute)
				}
			}
		}
	}
}

func (ms *MapService) publishOutboxStats() {
	if ms.onOutboxChanged == nil {
		return
	}
	ms.onOutboxChanged(ms.GetOutboxStats())
}

// sendMapEnterRequest 发送进入地图请求到MapServer
func (ms *MapService) sendMapEnterRequest(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	// 内层业务消息：带坐标的 MapEnterRequest（旧代码误用无坐标的 ClientMapEnterRequest，丢 X/Y/Z）
	req := &protocol.MapEnterRequest{
		PlayerId: int64(playerID),
		MapId:    int64(mapID),
		X:        pos.X,
		Y:        pos.Y,
		Z:        pos.Z,
	}
	// 进场属性快照（铁律1）：从持久 actor 取只读快照下发，MapServer 据此初始化 object.Player 战斗数值，
	// 而非恒为默认 level 1。只读快照、不是第二权威——MapServer 战斗改的是瞬时态，持久变更走 AttrGrant 回流。
	if ms.playerManager != nil {
		if p, perr := ms.playerManager.GetPlayer(playerID); perr == nil && p != nil {
			if a := p.GetAttrs(); a != nil {
				req.Level = a.GetLevel()
				req.MaxHp = int32(a.GetMaxHP())
				req.Hp = int32(a.GetHP())
				req.Attack = a.GetStrength() // 简化派生（装备系统启用后再叠加真实攻防）
				req.Defense = a.GetStamina()
			}
		}
	}
	reqData, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal map enter request: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ms.config.Server.ServerID))
	if err := ms.sendMapMessage(mapID, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_REQUEST), reqData, playerID, meta); err != nil {
		return err
	}

	// 注意：由于我们使用异步通信，这里不再等待响应
	// 响应会通过ConnectionManager的接收处理

	zLog.Info("MapServer enter map request sent",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)))

	return nil
}

// HandlePlayerLeaveMap 处理玩家离开地图
func (ms *MapService) HandlePlayerLeaveMap(playerID id.PlayerIdType, mapID id.MapIdType) error {
	// 跨服实例先分流：本 GameServer 根本没有这张图的本地 Map 对象（实例在别的 MapServer 上），
	// 走下面的本地查图路径会在 GetMap 处直接报错返回 —— 于是既不发离开请求（玩家对象永远留在
	// 跨服实例里），也不解绑（该玩家回本服后，消息还会被定点发去外域那台服务器）。
	// 客户端的 MSG_MAP_LEAVE 与登出清理都走本函数，所以这条分流是必需的。
	if _, bound := ms.crossBindings.Load(int64(playerID)); bound {
		return ms.LeaveCrossServerMap(playerID, mapID)
	}

	m, err := ms.GetMap(mapID)
	if err != nil {
		return err
	}

	m.RemovePlayer(playerID)

	if ms.playerMapManager != nil {
		ms.playerMapManager.RemovePlayerMap(playerID)
	}

	if err := ms.sendMapLeaveRequest(playerID, mapID); err != nil {
		zLog.Warn("Failed to send map leave request to MapServer", zap.Error(err))
	}

	zLog.Info("Player left map",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)))

	return nil
}

// sendMapLeaveRequest 发送离开地图请求到MapServer
func (ms *MapService) sendMapLeaveRequest(playerID id.PlayerIdType, mapID id.MapIdType) error {
	req := &protocol.MapLeaveRequest{
		PlayerId: int64(playerID),
		MapId:    int64(mapID),
	}
	reqData, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal map leave request: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ms.config.Server.ServerID))
	if err := ms.sendMapMessage(mapID, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_LEAVE_REQUEST), reqData, playerID, meta); err != nil {
		return err
	}

	zLog.Info("MapServer leave map request sent",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)))

	return nil
}

// HandlePlayerMove 处理玩家移动
func (ms *MapService) HandlePlayerMove(playerID id.PlayerIdType, mapID id.MapIdType, targetPos common.Vector3) error {
	// 先在本地处理
	// 获取地图
	m, err := ms.GetMap(mapID)
	if err != nil {
		return err
	}

	// 获取玩家对象
	playerObj := m.GetObject(id.ObjectIdType(playerID))
	if playerObj == nil {
		return fmt.Errorf("player not found in map: %d", playerID)
	}

	// 移动玩家
	err = m.MoveObject(playerObj, targetPos)
	if err != nil {
		return err
	}

	// 向MapServer发送移动请求
	err = ms.sendMapMoveRequest(playerID, id.ObjectIdType(playerID), mapID, targetPos)
	if err != nil {
		zLog.Warn("Failed to send map move request to MapServer", zap.Error(err))
		// 继续执行，MapServer通信失败不影响本地处理
	}

	zLog.Debug("Player moved",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)),
		zap.Float32("x", targetPos.X),
		zap.Float32("y", targetPos.Y),
		zap.Float32("z", targetPos.Z))

	return nil
}

// sendMapMoveRequest 发送移动请求到MapServer
func (ms *MapService) sendMapMoveRequest(playerID id.PlayerIdType, objectID id.ObjectIdType, mapID id.MapIdType, pos common.Vector3) error {
	// 内层业务消息：带坐标的 MapMoveRequest
	req := &protocol.MapMoveRequest{
		PlayerId: int64(playerID),
		ObjectId: int64(objectID),
		MapId:    int64(mapID),
		X:        pos.X,
		Y:        pos.Y,
		Z:        pos.Z,
	}
	reqData, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal map move request: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ms.config.Server.ServerID))
	if err := ms.sendMapMessage(mapID, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_MOVE_REQUEST), reqData, playerID, meta); err != nil {
		return err
	}

	zLog.Debug("MapServer move request sent",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)))

	return nil
}

// HandlePlayerAttack 处理玩家攻击
func (ms *MapService) HandlePlayerAttack(playerID id.PlayerIdType, mapID id.MapIdType, targetID id.ObjectIdType) (int64, int64, error) {
	// 路由校验：玩家须在本 GameServer 的该地图
	m, err := ms.GetMap(mapID)
	if err != nil {
		return 0, 0, err
	}
	playerObj := m.GetObject(id.ObjectIdType(playerID))
	if playerObj == nil {
		return 0, 0, fmt.Errorf("player not found in map: %d", playerID)
	}

	// 目标（怪物）的存在性与合法性由 MapServer 权威校验——GameServer 不镜像怪物，
	// 不在此越权校验目标（架构：MapServer 是地图对象/战斗权威）。直接转发。
	damage, targetHP, err := ms.sendMapAttackRequest(playerID, id.ObjectIdType(playerID), mapID, targetID)
	if err != nil {
		zLog.Warn("Failed to send map attack request to MapServer", zap.Error(err))
		return 0, 0, err
	}

	zLog.Info("Player attacked monster",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("target_id", int64(targetID)),
		zap.Int32("map_id", int32(mapID)),
		zap.Int64("damage", damage),
		zap.Int64("target_hp", targetHP))

	// 这里可以添加战斗逻辑

	return damage, targetHP, nil
}

func (ms *MapService) attackResultKey(playerID id.PlayerIdType, targetID id.ObjectIdType) string {
	return fmt.Sprintf("%d:%d", playerID, targetID)
}

func (ms *MapService) registerPendingAttack(playerID id.PlayerIdType, targetID id.ObjectIdType, requestID uint64) chan mapAttackResult {
	key := ms.attackResultKey(playerID, targetID)
	ch := make(chan mapAttackResult, 1)
	ms.pendingAttacks.Store(key, ch)
	if requestID != 0 {
		ms.pendingByReq.Store(requestID, ch)
	}
	return ch
}

func (ms *MapService) removePendingAttack(playerID id.PlayerIdType, targetID id.ObjectIdType, requestID uint64) {
	key := ms.attackResultKey(playerID, targetID)
	ms.pendingAttacks.Delete(key)
	if requestID != 0 {
		ms.pendingByReq.Delete(requestID)
	}
}

// HandleMapAttackResponse 处理 MapServer 返回的攻击结果
func (ms *MapService) HandleMapAttackResponse(requestID uint64, playerID id.PlayerIdType, targetID id.ObjectIdType, damage int64, targetHP int64, success bool, errorMsg string) {
	if !ms.inbox.TryAccept(requestID) {
		zLog.Warn("Duplicate map attack response ignored", zap.Uint64("request_id", requestID))
		return
	}

	key := ms.attackResultKey(playerID, targetID)

	ch, exists := ms.pendingByReq.Load(requestID)
	if !exists {
		ch, exists = ms.pendingAttacks.Load(key)
	}
	if !exists {
		return
	}

	select {
	case ch <- mapAttackResult{
		damage:   damage,
		targetHP: targetHP,
		success:  success,
		errorMsg: errorMsg,
	}:
	default:
	}
}

// sendMapAttackRequest 发送攻击请求到MapServer
func (ms *MapService) sendMapAttackRequest(playerID id.PlayerIdType, objectID id.ObjectIdType, mapID id.MapIdType, targetID id.ObjectIdType) (int64, int64, error) {
	// 内层业务消息：MapAttackRequest（带 object_id/target_id）
	req := &protocol.MapAttackRequest{
		PlayerId: int64(playerID),
		ObjectId: int64(objectID),
		MapId:    int64(mapID),
		TargetId: int64(targetID),
	}
	reqData, err := proto.Marshal(req)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal map attack request: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ms.config.Server.ServerID))
	respCh := ms.registerPendingAttack(playerID, targetID, meta.RequestID)
	defer ms.removePendingAttack(playerID, targetID, meta.RequestID)

	if err := ms.sendMapMessage(mapID, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_ATTACK_REQUEST), reqData, playerID, meta); err != nil {
		return 0, 0, err
	}

	zLog.Debug("MapServer attack request sent",
		zap.Int64("player_id", int64(playerID)),
		zap.Int64("target_id", int64(targetID)),
		zap.Int32("map_id", int32(mapID)))

	select {
	case result := <-respCh:
		if !result.success {
			return 0, 0, fmt.Errorf("map attack failed: %s", result.errorMsg)
		}
		return result.damage, result.targetHP, nil
	case <-time.After(1500 * time.Millisecond):
		return 0, 0, fmt.Errorf("map attack response timeout")
	}
}

// EnterMap 实现 player.MapOperator 接口
func (ms *MapService) EnterMap(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	return ms.HandlePlayerEnterMap(playerID, mapID, pos)
}

// LeaveMap 实现 player.MapOperator 接口
func (ms *MapService) LeaveMap(playerID id.PlayerIdType, mapID id.MapIdType) error {
	return ms.HandlePlayerLeaveMap(playerID, mapID)
}

// Move 实现 player.MapOperator 接口
func (ms *MapService) Move(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	return ms.HandlePlayerMove(playerID, mapID, pos)
}

// Attack 实现 player.MapOperator 接口
func (ms *MapService) Attack(playerID id.PlayerIdType, mapID id.MapIdType, targetID id.ObjectIdType) (int64, int64, error) {
	return ms.HandlePlayerAttack(playerID, mapID, targetID)
}

// Pickup 实现 player.MapOperator 接口：把就近拾取请求转发给 MapServer（fire，不阻塞）。
// 物品权威在地图侧——是否拾到由 MapServer 判定，拾到后经 ItemGrant(506) 异步回推到背包，不在此等待。
func (ms *MapService) Pickup(playerID id.PlayerIdType, mapID id.MapIdType) error {
	req := &protocol.ClientItemPickupRequest{PlayerId: int64(playerID), MapId: int32(mapID)}
	reqData, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal pickup request: %w", err)
	}
	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ms.config.Server.ServerID))
	return ms.sendMapMessage(mapID, int(protocol.InternalMsgId_MSG_INTERNAL_MAP_PICKUP_REQUEST), reqData, playerID, meta)
}

// HandleItemGrant 处理 MapServer 拾取授权回推(crossserver 506)：把物品发进该玩家背包并推客户端。
// 经 PlayerManager 路由到玩家 Actor（背包权威在玩家侧）。requestID 幂等去重（对齐 attack 407），
// 防跨服重复投递导致同一次拾取重复发物品。
func (ms *MapService) HandleItemGrant(requestID uint64, playerID int64, itemID, count int32) {
	if ms.playerManager == nil {
		return
	}
	if ms.inbox != nil && requestID != 0 && !ms.inbox.TryAccept(requestID) {
		zLog.Warn("Duplicate item grant ignored", zap.Uint64("request_id", requestID), zap.Int64("player_id", playerID))
		return
	}
	pid := id.PlayerIdType(playerID)
	req := &player.ItemGrantRequest{ItemID: itemID, Count: count}
	_ = ms.playerManager.RouteMessage(pid, player.NewPlayerMessage(pid, player.SourceMapServer, player.MsgItemGrant, req))
}

// HandleAttrGrant 处理 MapServer 战斗/结算属性回写（crossserver 508，realm ④）：把经验/金币等持久变更加到
// 持久化 actor（持久权威在玩家侧）。requestID 幂等去重（对齐 506/407），防同一次结算重复投递被累加两次。
func (ms *MapService) HandleAttrGrant(requestID uint64, playerID int64, changes []*protocol.AttrChange) {
	if ms.playerManager == nil || len(changes) == 0 {
		return
	}
	if ms.inbox != nil && requestID != 0 && !ms.inbox.TryAccept(requestID) {
		zLog.Warn("Duplicate attr grant ignored", zap.Uint64("request_id", requestID), zap.Int64("player_id", playerID))
		return
	}
	pid := id.PlayerIdType(playerID)
	items := make([]player.AttrChange, 0, len(changes))
	for _, c := range changes {
		items = append(items, player.AttrChange{Kind: c.Kind, ID: c.Id, Delta: c.Delta})
	}
	req := &player.AttrGrantRequest{Changes: items}
	_ = ms.playerManager.RouteMessage(pid, player.NewPlayerMessage(pid, player.SourceMapServer, player.MsgAttrGrant, req))
}

// BattleReward 战斗奖励
type BattleReward struct {
	PlayerID  id.PlayerIdType
	Exp       int64
	Gold      int64
	ItemIDs   []int32
	KillCount int32
}

// HandleBattleReward 处理 MapServer 回传的战斗奖励
func (ms *MapService) HandleBattleReward(reward *BattleReward) error {
	if reward == nil {
		return nil
	}

	zLog.Info("Handling battle reward",
		zap.Int64("player_id", int64(reward.PlayerID)),
		zap.Int64("exp", reward.Exp),
		zap.Int64("gold", reward.Gold),
		zap.Int("item_count", len(reward.ItemIDs)),
		zap.Int32("kill_count", reward.KillCount))

	if ms.playerMapManager != nil {
		ms.playerMapManager.UpdatePlayerBattleStats(reward.PlayerID, reward.KillCount, reward.Exp)
	}

	return nil
}

// HandleMonsterDeath 处理 MapServer 回传的怪物死亡事件
func (ms *MapService) HandleMonsterDeath(playerID id.PlayerIdType, mapID id.MapIdType, monsterID id.ObjectIdType, expReward int64, itemDrops []int32) error {
	zLog.Info("Monster death notification from MapServer",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)),
		zap.Int64("monster_id", int64(monsterID)),
		zap.Int64("exp_reward", expReward),
		zap.Int("item_drops", len(itemDrops)))

	m, err := ms.GetMap(mapID)
	if err != nil {
		return err
	}

	m.RemoveObject(monsterID)

	reward := &BattleReward{
		PlayerID:  playerID,
		Exp:       expReward,
		ItemIDs:   itemDrops,
		KillCount: 1,
	}

	return ms.HandleBattleReward(reward)
}

// HandleAOINotify 处理 MapServer 回程的 AOI 视野事件（Phase 2.3）：把事件路由到 watcher
// 玩家 Actor，复用既有推送链（player_aoi_handler → EntityXxxViewNotify → pushToClient）。
// 与 subscribeAOIEvents 的本地路径等价，区别是事件源为跨服 MapServer（权威）。
func (ms *MapService) HandleAOINotify(watcherID, targetID int64, mapID int32, eventType uint32, x, y, z float32) {
	if ms.playerManager == nil {
		return
	}
	zLog.Debug("HandleAOINotify routing to player",
		zap.Int64("watcher", watcherID), zap.Int64("target", targetID), zap.Uint32("evt", eventType))
	req := &player.AOIViewRequest{
		WatcherID: id.PlayerIdType(watcherID),
		TargetID:  targetID,
		MapID:     id.MapIdType(mapID),
		PosX:      x,
		PosY:      y,
		PosZ:      z,
	}
	watcher := id.PlayerIdType(watcherID)
	switch eventType {
	case crossserver.MsgInternalAOIEnter:
		_ = ms.playerManager.RouteMessage(watcher, player.NewPlayerMessage(watcher, player.SourceMapServer, player.MsgAOIEnterView, req))
	case crossserver.MsgInternalAOILeave:
		_ = ms.playerManager.RouteMessage(watcher, player.NewPlayerMessage(watcher, player.SourceMapServer, player.MsgAOILeaveView, req))
	case crossserver.MsgInternalAOIMove:
		_ = ms.playerManager.RouteMessage(watcher, player.NewPlayerMessage(watcher, player.SourceMapServer, player.MsgAOIMove, req))
	case crossserver.MsgInternalAOIAttr:
		// x=当前血量, y=最大血量（复用 AOIViewRequest.PosX/PosY 承载）。
		_ = ms.playerManager.RouteMessage(watcher, player.NewPlayerMessage(watcher, player.SourceMapServer, player.MsgAOIAttr, req))
	case crossserver.MsgInternalAOIDeath:
		// x=killer 对象 ID（复用 AOIViewRequest.PosX 承载）。
		_ = ms.playerManager.RouteMessage(watcher, player.NewPlayerMessage(watcher, player.SourceMapServer, player.MsgAOIDeath, req))
	case crossserver.MsgInternalAOIBuff:
		// x=buffID, y=added(1/0), z=remaining_ms（复用 AOIViewRequest.PosX/PosY/PosZ 承载）。
		_ = ms.playerManager.RouteMessage(watcher, player.NewPlayerMessage(watcher, player.SourceMapServer, player.MsgAOIBuff, req))
	}
}

// 注：GameServer 自建的本地 AOI 订阅（subscribeAOIEvents）已移除。客户端视野事件现由 MapServer
// 分层 AOI 单一权威经 HandleAOINotify 推送（见上）。移除原因：本地 AOI 按逻辑图ID键值，无法感知
// ②-b 分线，会跨 layer 串视野（已实测复现）；且与 MapServer AOI 功能等价、纯冗余。
