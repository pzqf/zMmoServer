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

type MapServerInfo struct {
	ServerID   uint32
	Address    string
	MapIDs     []id.MapIdType
	LastUpdate time.Time
	Status     string
}

type MapServerManager struct {
	config           *config.Config
	serviceDiscovery *discovery.ServiceDiscovery
	mapServerInfo    *zMap.TypedMap[uint32, *MapServerInfo]
	mapToServer      *zMap.TypedMap[id.MapIdType, uint32]
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewMapServerManager(cfg *config.Config, sd *discovery.ServiceDiscovery) *MapServerManager {
	return &MapServerManager{
		config:           cfg,
		serviceDiscovery: sd,
		mapServerInfo:    zMap.NewTypedMap[uint32, *MapServerInfo](),
		mapToServer:      zMap.NewTypedMap[id.MapIdType, uint32](),
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
	zLog.Info("MapServerManager stopped")
}

func (msm *MapServerManager) discoveryLoop() {
	msm.doDiscovery()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-msm.ctx.Done():
			return
		case <-ticker.C:
			msm.doDiscovery()
		}
	}
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

		mapIDs := msm.getDefaultMapIDs()

		info := &MapServerInfo{
			ServerID:   serverIDInt,
			Address:    server.Address,
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
