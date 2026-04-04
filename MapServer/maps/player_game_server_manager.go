package maps

import (
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zUtil/zMap"
)

type PlayerGameServerInfo struct {
	PlayerID      id.PlayerIdType
	GameServerID  uint32
	GameServerAddr string
	MapID         id.MapIdType
}

type PlayerGameServerManager struct {
	playerGameServer *zMap.TypedMap[id.PlayerIdType, *PlayerGameServerInfo]
	gameServerAddr   *zMap.TypedMap[uint32, string]
}

func NewPlayerGameServerManager() *PlayerGameServerManager {
	return &PlayerGameServerManager{
		playerGameServer: zMap.NewTypedMap[id.PlayerIdType, *PlayerGameServerInfo](),
		gameServerAddr:   zMap.NewTypedMap[uint32, string](),
	}
}

func (pgsm *PlayerGameServerManager) SetPlayerGameServer(playerID id.PlayerIdType, gameServerID uint32, gameServerAddr string, mapID id.MapIdType) {
	info := &PlayerGameServerInfo{
		PlayerID:       playerID,
		GameServerID:   gameServerID,
		GameServerAddr: gameServerAddr,
		MapID:          mapID,
	}
	pgsm.playerGameServer.Store(playerID, info)
	pgsm.gameServerAddr.Store(gameServerID, gameServerAddr)
}

func (pgsm *PlayerGameServerManager) GetPlayerGameServerInfo(playerID id.PlayerIdType) (*PlayerGameServerInfo, bool) {
	return pgsm.playerGameServer.Load(playerID)
}

func (pgsm *PlayerGameServerManager) RemovePlayerGameServer(playerID id.PlayerIdType) {
	pgsm.playerGameServer.Delete(playerID)
}

func (pgsm *PlayerGameServerManager) GetGameServerID(playerID id.PlayerIdType) (uint32, bool) {
	info, exists := pgsm.playerGameServer.Load(playerID)
	if !exists {
		return 0, false
	}
	return info.GameServerID, true
}

func (pgsm *PlayerGameServerManager) GetGameServerAddr(gameServerID uint32) (string, bool) {
	return pgsm.gameServerAddr.Load(gameServerID)
}

func (pgsm *PlayerGameServerManager) GetPlayerCount() int {
	return int(pgsm.playerGameServer.Len())
}

func (pgsm *PlayerGameServerManager) Clear() {
	pgsm.playerGameServer.Clear()
	pgsm.gameServerAddr.Clear()
}
