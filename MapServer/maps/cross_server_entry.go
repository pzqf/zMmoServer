package maps

import (
	"context"
	"fmt"
	"sync"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/config/tables"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/connection"
	"github.com/pzqf/zMmoServer/MapServer/maps/dungeon"
	"go.uber.org/zap"
)

type CrossServerMapEntry struct {
	mu           sync.RWMutex
	mapManager   *MapManager
	connManager  *connection.ConnectionManager
	tableManager *tables.TableManager
	migrationMgr *crossserver.MigrationManager
	serverID     int32
}

func NewCrossServerMapEntry(mapManager *MapManager, connManager *connection.ConnectionManager, tableManager *tables.TableManager, serverID int32) *CrossServerMapEntry {
	return &CrossServerMapEntry{
		mapManager:   mapManager,
		connManager:  connManager,
		tableManager: tableManager,
		serverID:     serverID,
	}
}

func (cse *CrossServerMapEntry) SetMigrationManager(mm *crossserver.MigrationManager) {
	cse.migrationMgr = mm
}

func (cse *CrossServerMapEntry) EnterDungeonMap(playerID id.PlayerIdType, dungeonID id.DungeonIdType) (*Map, *dungeon.DungeonInstance, error) {
	dungeonMap, instance, err := cse.mapManager.CreateDungeonMap(dungeonID, []id.PlayerIdType{playerID}, cse.connManager)
	if err != nil {
		return nil, nil, fmt.Errorf("create dungeon map: %w", err)
	}

	zLog.Info("Player entering dungeon map",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("dungeon_id", int32(dungeonID)),
		zap.Int32("map_id", int32(dungeonMap.GetID())))

	return dungeonMap, instance, nil
}

func (cse *CrossServerMapEntry) EnterCrossServerMap(playerID id.PlayerIdType, mapConfigID int32, mode MapMode, serverGroupID int32) (*Map, error) {
	if cse.tableManager == nil {
		return nil, fmt.Errorf("table manager not set")
	}

	mapConfig, ok := cse.tableManager.GetMapLoader().GetMap(mapConfigID)
	if !ok {
		return nil, fmt.Errorf("map config not found: %d", mapConfigID)
	}

	crossMap := cse.mapManager.CreateCrossServerMap(
		mapConfigID,
		mapConfig.Name,
		float32(mapConfig.Width),
		float32(mapConfig.Height),
		mode,
		serverGroupID,
		cse.connManager,
	)

	if cse.migrationMgr != nil {
		_, err := cse.migrationMgr.RequestMigration(
			context.Background(),
			int64(playerID),
			0,
			"",
			cse.serverID,
			crossserver.ServiceTypeMap,
			mapConfigID,
			crossserver.MigrationTypeGameToMap,
			"enter_cross_server_map",
		)
		if err != nil {
			zLog.Warn("Migration request failed for cross-server map entry",
				zap.Int64("player_id", int64(playerID)),
				zap.String("error", err.Error()))
		}
	}

	zLog.Info("Player entering cross-server map",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_config_id", mapConfigID),
		zap.Int("mode", int(mode)),
		zap.Int32("server_group_id", serverGroupID))

	return crossMap, nil
}

func (cse *CrossServerMapEntry) LeaveCrossServerMap(playerID id.PlayerIdType, mapID id.MapIdType) error {
	m := cse.mapManager.GetMap(mapID)
	if m == nil {
		return fmt.Errorf("map not found: %d", mapID)
	}

	m.RemovePlayer(playerID)

	if m.IsDungeon() {
		instanceID := m.GetDungeonInstanceID()
		dlm := cse.mapManager.GetDungeonLifecycleManager()
		if dlm != nil {
			dlm.DestroyDungeon(instanceID)
		}
	}

	zLog.Info("Player left cross-server map",
		zap.Int64("player_id", int64(playerID)),
		zap.Int32("map_id", int32(mapID)))

	return nil
}
