package player

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zUtil/zMap"
)

type MapInfo struct {
	MapID        id.MapIdType
	MapServerID  uint32
	MapServerAddr string
}

type PlayerMapManager struct {
	playerMapInfo    *zMap.TypedMap[id.PlayerIdType, MapInfo]
	mapServerInfo    *zMap.TypedMap[id.MapIdType, uint32]
	mapServerAddr    *zMap.TypedMap[uint32, string]
}

func NewPlayerMapManager() *PlayerMapManager {
	return &PlayerMapManager{
		playerMapInfo: zMap.NewTypedMap[id.PlayerIdType, MapInfo](),
		mapServerInfo: zMap.NewTypedMap[id.MapIdType, uint32](),
		mapServerAddr: zMap.NewTypedMap[uint32, string](),
	}
}

func (pmm *PlayerMapManager) SetPlayerMap(playerID id.PlayerIdType, mapID id.MapIdType, mapServerID uint32) {
	info := MapInfo{
		MapID:       mapID,
		MapServerID: mapServerID,
	}
	pmm.playerMapInfo.Store(playerID, info)
	pmm.mapServerInfo.Store(mapID, mapServerID)
}

func (pmm *PlayerMapManager) GetPlayerMapInfo(playerID id.PlayerIdType) (MapInfo, bool) {
	return pmm.playerMapInfo.Load(playerID)
}

func (pmm *PlayerMapManager) RemovePlayerMap(playerID id.PlayerIdType) {
	pmm.playerMapInfo.Delete(playerID)
}

func (pmm *PlayerMapManager) GetMapServerID(mapID id.MapIdType) (uint32, bool) {
	return pmm.mapServerInfo.Load(mapID)
}

func (pmm *PlayerMapManager) SetMapServerAddr(mapServerID uint32, addr string) {
	pmm.mapServerAddr.Store(mapServerID, addr)
}

func (pmm *PlayerMapManager) GetMapServerAddr(mapServerID uint32) (string, bool) {
	return pmm.mapServerAddr.Load(mapServerID)
}

func (pmm *PlayerMapManager) GetPlayerCount() int {
	return int(pmm.playerMapInfo.Len())
}

func (pmm *PlayerMapManager) Clear() {
	pmm.playerMapInfo.Clear()
	pmm.mapServerInfo.Clear()
	pmm.mapServerAddr.Clear()
}
