package maps

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zCommon/consistency"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zEvent"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/connection"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoServer/GameServer/game/event"
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
	}
}

// SetConnectionManager 设置连接管理器
func (ms *MapService) SetConnectionManager(connManager *connection.ConnectionManager) {
	ms.connectionManager = connManager
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

	ms.subscribeAOIEvents()

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
	excelDir := "../resources/excel_tables"

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

// sendMapMessage 发送消息到MapServer
func (ms *MapService) sendMapMessage(mapID id.MapIdType, protoId int, data []byte, playerID id.PlayerIdType, meta crossserver.Meta) error {
	if ms.connectionManager == nil {
		return fmt.Errorf("connection manager not set")
	}

	baseMsg := &protocol.BaseMessage{
		MsgId:     uint32(protoId),
		PlayerId:  uint64(playerID),
		ServerId:  uint32(ms.config.Server.ServerID),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}

	crossMsg := &protocol.CrossServerMessage{
		TraceId:      meta.TraceID,
		FromServerId: uint32(ms.config.Server.ServerID),
		FromService:  uint32(crossserver.ServiceTypeGame),
		ToService:    uint32(crossserver.ServiceTypeMap),
		Message:      baseMsg,
	}

	crossMsgData, err := proto.Marshal(crossMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal cross server message: %w", err)
	}

	enveloped := crossserver.Wrap(meta, crossMsgData)
	msg := consistency.OutboxMessage{
		RequestID:   meta.RequestID,
		Topic:       fmt.Sprintf("map:%d:proto:%d", mapID, protoId),
		TargetMapID: int32(mapID),
		ProtoID:     int32(protoId),
		Payload:     enveloped,
	}
	ms.outbox.Add(msg)
	ms.outbox.MarkAttempt(meta.RequestID, nil)
	ms.publishOutboxStats()

	err = ms.sendFramedToMap(int(mapID), protoId, enveloped)
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
	// 创建地图进入请求
	mapEnterReq := &protocol.ClientMapEnterRequest{
		PlayerId: int64(playerID),
		MapId:    int32(mapID),
	}

	// 序列化具体消息
	reqData, err := proto.Marshal(mapEnterReq)
	if err != nil {
		return fmt.Errorf("failed to marshal map enter request: %w", err)
	}

	// 创建基础消息
	baseMsg := &protocol.BaseMessage{
		PlayerId:  uint64(playerID),
		MsgId:     uint32(protocol.MapMsgId_MSG_MAP_ENTER),
		ServerId:  uint32(ms.config.Server.ServerID),
		MapId:     uint32(mapID),
		Data:      reqData,
		Timestamp: uint64(time.Now().UnixNano()),
	}

	// 序列化基础消息
	data, err := proto.Marshal(baseMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal base message: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ms.config.Server.ServerID))
	err = ms.sendMapMessage(mapID, 300, data, playerID, meta)
	if err != nil {
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
	mapLeaveReq := &protocol.ClientMapLeaveRequest{
		PlayerId: int64(playerID),
		MapId:    int32(mapID),
	}

	reqData, err := proto.Marshal(mapLeaveReq)
	if err != nil {
		return fmt.Errorf("failed to marshal map leave request: %w", err)
	}

	baseMsg := &protocol.BaseMessage{
		PlayerId:  uint64(playerID),
		MsgId:     uint32(protocol.MapMsgId_MSG_MAP_LEAVE),
		ServerId:  uint32(ms.config.Server.ServerID),
		MapId:     uint32(mapID),
		Data:      reqData,
		Timestamp: uint64(time.Now().UnixNano()),
	}

	data, err := proto.Marshal(baseMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal base message: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ms.config.Server.ServerID))
	err = ms.sendMapMessage(mapID, 300, data, playerID, meta)
	if err != nil {
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
	// 创建地图移动请求
	mapMoveReq := &protocol.ClientMapMoveRequest{
		PlayerId: int64(playerID),
		MapId:    int32(mapID),
		Pos: &protocol.Position{
			X: pos.X,
			Y: pos.Y,
			Z: pos.Z,
		},
	}

	// 序列化具体消息
	reqData, err := proto.Marshal(mapMoveReq)
	if err != nil {
		return fmt.Errorf("failed to marshal map move request: %w", err)
	}

	// 创建基础消息
	baseMsg := &protocol.BaseMessage{
		PlayerId:  uint64(playerID),
		MsgId:     uint32(protocol.MapMsgId_MSG_MAP_MOVE),
		ServerId:  uint32(ms.config.Server.ServerID),
		MapId:     uint32(mapID),
		Data:      reqData,
		Timestamp: uint64(time.Now().UnixNano()),
	}

	// 序列化基础消息
	data, err := proto.Marshal(baseMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal base message: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ms.config.Server.ServerID))
	err = ms.sendMapMessage(mapID, 300, data, playerID, meta)
	if err != nil {
		return err
	}

	zLog.Debug("MapServer move request sent",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)))

	return nil
}

// HandlePlayerAttack 处理玩家攻击
func (ms *MapService) HandlePlayerAttack(playerID id.PlayerIdType, mapID id.MapIdType, targetID id.ObjectIdType) (int64, int64, error) {
	// 先在本地处理
	// 获取地图
	m, err := ms.GetMap(mapID)
	if err != nil {
		return 0, 0, err
	}

	// 获取玩家对象
	playerObj := m.GetObject(id.ObjectIdType(playerID))
	if playerObj == nil {
		return 0, 0, fmt.Errorf("player not found in map: %d", playerID)
	}

	// 获取目标对象
	targetObj := m.GetObject(targetID)
	if targetObj == nil {
		return 0, 0, fmt.Errorf("target not found in map: %d", targetID)
	}

	// 检查目标类型是否为怪物
	if targetObj.GetType() != common.GameObjectTypeMonster {
		return 0, 0, fmt.Errorf("target is not a monster: %d", targetID)
	}

	// 向MapServer发送攻击请求
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
	// 创建地图攻击请求
	mapAttackReq := &protocol.ClientMapAttackRequest{
		PlayerId: int64(playerID),
		MapId:    int32(mapID),
		TargetId: int64(targetID),
	}

	// 序列化具体消息
	reqData, err := proto.Marshal(mapAttackReq)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal map attack request: %w", err)
	}

	// 创建基础消息
	baseMsg := &protocol.BaseMessage{
		PlayerId:  uint64(playerID),
		MsgId:     uint32(protocol.MapMsgId_MSG_MAP_ATTACK),
		ServerId:  uint32(ms.config.Server.ServerID),
		MapId:     uint32(mapID),
		Data:      reqData,
		Timestamp: uint64(time.Now().UnixNano()),
	}

	// 序列化基础消息
	data, err := proto.Marshal(baseMsg)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal base message: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeGame, int32(ms.config.Server.ServerID))
	respCh := ms.registerPendingAttack(playerID, targetID, meta.RequestID)
	defer ms.removePendingAttack(playerID, targetID, meta.RequestID)

	err = ms.sendMapMessage(mapID, 300, data, playerID, meta)
	if err != nil {
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

// SendMapEnterResponse 发送进入地图响应
func (ms *MapService) SendMapEnterResponse(conn interface{}, playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error {
	// 创建地图进入响应
	resp := &protocol.ClientMapEnterResponse{
		Result: 0, // 成功
		MapId:  int32(mapID),
		Pos: &protocol.Position{
			X: pos.X,
			Y: pos.Y,
			Z: pos.Z,
		},
	}

	// 序列化响应
	respData, err := proto.Marshal(resp)
	if err != nil {
		return err
	}

	// 发送响应到客户端
	// 注意：这里需要根据实际的连接类型实现发送逻辑
	// 目前留作接口，后续根据具体连接类型实现
	_ = respData
	_ = conn
	return nil
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

// subscribeAOIEvents 订阅 AOI 视野事件，将事件转发给 PlayerManager 投递到玩家 Actor
func (ms *MapService) subscribeAOIEvents() {
	event.Subscribe(event.EventAOIEnterView, func(evt *zEvent.Event) {
		data, ok := evt.Data.(*event.AOIViewEventData)
		if !ok || ms.playerManager == nil {
			return
		}
		msg := player.NewPlayerMessage(
			data.WatcherID, player.SourceMapServer, player.MsgAOIEnterView,
			&player.AOIViewRequest{
				WatcherID: data.WatcherID,
				TargetID:  data.TargetID,
				MapID:     data.MapID,
				PosX:      data.PosX,
				PosY:      data.PosY,
			},
		)
		if err := ms.playerManager.RouteMessage(data.WatcherID, msg); err != nil {
			zLog.Debug("Failed to route AOI enter view message",
				zap.Int64("watcher_id", int64(data.WatcherID)),
				zap.Error(err))
		}
	})

	event.Subscribe(event.EventAOILeaveView, func(evt *zEvent.Event) {
		data, ok := evt.Data.(*event.AOIViewEventData)
		if !ok || ms.playerManager == nil {
			return
		}
		msg := player.NewPlayerMessage(
			data.WatcherID, player.SourceMapServer, player.MsgAOILeaveView,
			&player.AOIViewRequest{
				WatcherID: data.WatcherID,
				TargetID:  data.TargetID,
				MapID:     data.MapID,
				PosX:      data.PosX,
				PosY:      data.PosY,
			},
		)
		if err := ms.playerManager.RouteMessage(data.WatcherID, msg); err != nil {
			zLog.Debug("Failed to route AOI leave view message",
				zap.Int64("watcher_id", int64(data.WatcherID)),
				zap.Error(err))
		}
	})

	event.Subscribe(event.EventAOIMove, func(evt *zEvent.Event) {
		data, ok := evt.Data.(*event.AOIViewEventData)
		if !ok || ms.playerManager == nil {
			return
		}
		msg := player.NewPlayerMessage(
			data.WatcherID, player.SourceMapServer, player.MsgAOIMove,
			&player.AOIViewRequest{
				WatcherID: data.WatcherID,
				TargetID:  data.TargetID,
				MapID:     data.MapID,
				OldPosX:   data.OldPosX,
				OldPosY:   data.OldPosY,
				PosX:      data.PosX,
				PosY:      data.PosY,
			},
		)
		if err := ms.playerManager.RouteMessage(data.WatcherID, msg); err != nil {
			zLog.Debug("Failed to route AOI move message",
				zap.Int64("watcher_id", int64(data.WatcherID)),
				zap.Error(err))
		}
	})
}
