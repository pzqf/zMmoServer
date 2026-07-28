package maps

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

// MapServerInfo 地图服务器信息。Address 为可直接拨号的 host:port。
type MapServerInfo struct {
	ServerID   uint32
	Address    string
	GroupID    string // 所属 realm（区分本 realm / 跨 realm）
	MapIDs     []id.MapIdType
	LastUpdate time.Time
	Status     string
}

// MapServerManager 地图服务器管理器
// 职责：通过 etcd 服务发现管理 MapServer 实例信息，维护 mapID → serverID 的路由映射
// 注意：本管理器仅负责服务发现和路由，不涉及地图游戏逻辑（由 MapService 负责）
type MapServerManager struct {
	config           *config.Config
	serviceDiscovery *discovery.ServiceDiscovery
	mapServerInfo    *zMap.TypedMap[uint32, *MapServerInfo]
	mapToServer      *zMap.TypedMap[id.MapIdType, uint32]
	// crossRealmServers **其它 realm** 的 MapServer 目录（跨服物理路由用，realm §5.1 ②）。
	// 刻意与 mapServerInfo/mapToServer **分开存**：普通进图路由必须只看本 realm（铁律5 realm 隔离），
	// 跨 realm 的服务器只在玩家明确参加跨服活动时按 serverID 定点使用，不参与 mapID→server 路由
	//（实例 mapID 由各 MapServer 的实例池派生、跨服务器会撞号，按 mapID 路由必错）。
	crossRealmServers *zMap.TypedMap[uint32, *MapServerInfo]
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
}

func NewMapServerManager(cfg *config.Config, sd *discovery.ServiceDiscovery) *MapServerManager {
	return &MapServerManager{
		config:            cfg,
		serviceDiscovery:  sd,
		mapServerInfo:     zMap.NewTypedMap[uint32, *MapServerInfo](),
		mapToServer:       zMap.NewTypedMap[id.MapIdType, uint32](),
		crossRealmServers: zMap.NewTypedMap[uint32, *MapServerInfo](),
	}
}

func (msm *MapServerManager) Start(ctx context.Context) error {
	msm.ctx, msm.cancel = context.WithCancel(ctx)
	go msm.discoveryLoop()
	zLog.Info("MapServerManager started")
	return nil
}

func (msm *MapServerManager) Stop() {
	if msm.cancel != nil {
		msm.cancel()
	}
	msm.mapServerInfo.Clear()
	msm.mapToServer.Clear()
	msm.crossRealmServers.Clear()
	zLog.Info("MapServerManager stopped")
}

func (msm *MapServerManager) discoveryLoop() {
	msm.doDiscovery()
	msm.doCrossRealmDiscovery()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-msm.ctx.Done():
			return
		case <-ticker.C:
			msm.doDiscovery()
			msm.doCrossRealmDiscovery()
		}
	}
}

// doCrossRealmDiscovery 发现**其它 realm** 的 MapServer（跨服物理路由 ②）。
//
// 底层 Discover(serviceType, groupID="") 本就能拿全量，只是上层此前从没暴露过——GameServer 只
// Discover 本 group，于是"跨 realm 连不上"。这里补上入口，但结果单独存一张表：绝不并入
// mapToServer（那会让普通进图路由跨出本 realm，破坏 realm 隔离）。
func (msm *MapServerManager) doCrossRealmDiscovery() {
	serverID := id.MustParseServerIDInt(int32(msm.config.Server.ServerID))
	ownGroupID := id.GroupIDStringFromServerID(serverID)

	all, err := msm.serviceDiscovery.Discover("map", "") // "" = 不限 group
	if err != nil {
		zLog.Warn("Failed to discover cross-realm map servers", zap.Error(err))
		return
	}

	now := time.Now()
	active := make(map[uint32]bool)

	for _, server := range all {
		if server == nil || server.GroupID == ownGroupID {
			continue // 本 realm 的走 doDiscovery，不进跨域表
		}
		serverIDInt, err := msm.parseServerID(server.ID)
		if err != nil {
			zLog.Warn("Failed to parse cross-realm map server ID", zap.String("server_id", server.ID), zap.Error(err))
			continue
		}
		addr, ok := dialAddress(server.Address, server.Port)
		if !ok {
			zLog.Warn("Cross-realm map server has no usable address",
				zap.String("server_id", server.ID), zap.String("address", server.Address), zap.Int("port", server.Port))
			continue
		}

		active[serverIDInt] = true
		msm.crossRealmServers.Store(serverIDInt, &MapServerInfo{
			ServerID:   serverIDInt,
			Address:    addr,
			GroupID:    server.GroupID,
			LastUpdate: now,
			Status:     string(server.Status),
		})
	}

	msm.crossRealmServers.Range(func(sid uint32, info *MapServerInfo) bool {
		if !active[sid] {
			msm.crossRealmServers.Delete(sid)
			zLog.Info("Removed inactive cross-realm map server", zap.Uint32("server_id", sid))
		}
		return true
	})

	zLog.Debug("Cross-realm map servers discovered",
		zap.Int("count", int(msm.crossRealmServers.Len())), zap.String("own_group", ownGroupID))
}

// dialAddress 把服务发现里的 host/port 拼成可拨号地址（0.0.0.0 视为回环，与 base_server 一致）。
func dialAddress(host string, port int) (string, bool) {
	if port <= 0 {
		return "", false
	}
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", host, port), true
}

// GetCrossRealmMapServer 查一台跨 realm 的 MapServer（跨服活动分配结果的落地校验）。
func (msm *MapServerManager) GetCrossRealmMapServer(serverID uint32) (*MapServerInfo, bool) {
	return msm.crossRealmServers.Load(serverID)
}

// ListCrossRealmMapServers 列出已发现的跨 realm MapServer（运维/测试）。
func (msm *MapServerManager) ListCrossRealmMapServers() []*MapServerInfo {
	var out []*MapServerInfo
	msm.crossRealmServers.Range(func(_ uint32, info *MapServerInfo) bool {
		out = append(out, info)
		return true
	})
	return out
}

func (msm *MapServerManager) doDiscovery() {
	serverID := id.MustParseServerIDInt(int32(msm.config.Server.ServerID))
	groupID := id.GroupIDStringFromServerID(serverID)

	mapServers, err := msm.serviceDiscovery.Discover("map", groupID)
	if err != nil {
		zLog.Warn("Failed to discover map servers", zap.Error(err))
		return
	}

	zLog.Debug("Discovered map servers", zap.Int("count", len(mapServers)))

	now := time.Now()
	activeServerIDs := make(map[uint32]bool)

	for _, server := range mapServers {
		serverIDInt, err := msm.parseServerID(server.ID)
		if err != nil {
			zLog.Warn("Failed to parse map server ID", zap.String("server_id", server.ID), zap.Error(err))
			continue
		}

		activeServerIDs[serverIDInt] = true

		// 去硬编码：优先用 MapServer 在 etcd 注册的 MapIDs（它实际服务的地图），
		// 仅当注册为空时回退到默认列表，避免路由表落空。
		mapIDs := make([]id.MapIdType, 0, len(server.MapIDs))
		for _, mid := range server.MapIDs {
			mapIDs = append(mapIDs, id.MapIdType(mid))
		}
		if len(mapIDs) == 0 {
			mapIDs = msm.getDefaultMapIDs()
			zLog.Warn("MapServer registered no map IDs, falling back to defaults",
				zap.Uint32("server_id", serverIDInt))
		}

		addr, ok := dialAddress(server.Address, server.Port)
		if !ok {
			addr = server.Address // 端口缺失时退回原样，至少不丢信息
		}
		info := &MapServerInfo{
			ServerID:   serverIDInt,
			Address:    addr,
			GroupID:    server.GroupID,
			MapIDs:     mapIDs,
			LastUpdate: now,
			Status:     string(server.Status),
		}

		msm.mu.Lock()
		msm.mapServerInfo.Store(serverIDInt, info)
		for _, mapID := range mapIDs {
			msm.mapToServer.Store(mapID, serverIDInt)
		}
		msm.mu.Unlock()

		zLog.Debug("Updated map server info",
			zap.Uint32("server_id", serverIDInt),
			zap.String("address", server.Address),
			zap.Int("map_count", len(mapIDs)),
			zap.String("status", string(server.Status)))
	}

	msm.mu.Lock()
	msm.mapServerInfo.Range(func(serverID uint32, info *MapServerInfo) bool {
		if !activeServerIDs[serverID] {
			for _, mapID := range info.MapIDs {
				msm.mapToServer.Delete(mapID)
			}
			msm.mapServerInfo.Delete(serverID)
			zLog.Info("Removed inactive map server", zap.Uint32("server_id", serverID))
		}
		return true
	})
	msm.mu.Unlock()
}

func (msm *MapServerManager) parseServerID(serverIDStr string) (uint32, error) {
	parts := strings.Split(serverIDStr, "-")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid server ID format: %s", serverIDStr)
	}

	idStr := parts[len(parts)-1]
	idInt, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse server ID: %w", err)
	}

	return uint32(idInt), nil
}

func (msm *MapServerManager) getDefaultMapIDs() []id.MapIdType {
	return []id.MapIdType{1001, 1002, 2001, 2002, 3001, 3002, 4001, 4002, 5001}
}

func (msm *MapServerManager) GetMapServerID(mapID id.MapIdType) (uint32, bool) {
	msm.mu.RLock()
	defer msm.mu.RUnlock()
	return msm.mapToServer.Load(mapID)
}

func (msm *MapServerManager) GetMapServerInfo(serverID uint32) (*MapServerInfo, bool) {
	msm.mu.RLock()
	defer msm.mu.RUnlock()
	return msm.mapServerInfo.Load(serverID)
}

func (msm *MapServerManager) GetMapServerAddr(mapID id.MapIdType) (string, uint32, error) {
	msm.mu.RLock()
	defer msm.mu.RUnlock()

	serverID, exists := msm.mapToServer.Load(mapID)
	if !exists {
		return "", 0, fmt.Errorf("map server not found for map %d", mapID)
	}

	info, exists := msm.mapServerInfo.Load(serverID)
	if !exists {
		return "", 0, fmt.Errorf("map server info not found for server %d", serverID)
	}

	return info.Address, serverID, nil
}

func (msm *MapServerManager) GetAllMapServers() []*MapServerInfo {
	msm.mu.RLock()
	defer msm.mu.RUnlock()

	var servers []*MapServerInfo
	msm.mapServerInfo.Range(func(serverID uint32, info *MapServerInfo) bool {
		servers = append(servers, info)
		return true
	})
	return servers
}

func (msm *MapServerManager) GetMapCount() int {
	msm.mu.RLock()
	defer msm.mu.RUnlock()
	return int(msm.mapToServer.Len())
}

func (msm *MapServerManager) GetServerCount() int {
	msm.mu.RLock()
	defer msm.mu.RUnlock()
	return int(msm.mapServerInfo.Len())
}
