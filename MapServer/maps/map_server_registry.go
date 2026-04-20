package maps

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/crossserver"
	"github.com/pzqf/zMmoServer/MapServer/maps/dungeon"
	"go.uber.org/zap"
)

const (
	MsgMapServerRegister   uint32 = 50001
	MsgMapServerHeartbeat  uint32 = 50002
	MsgMapServerUnregister uint32 = 50003
	MsgMapServerStatus     uint32 = 50004
)

type MapServerRegistration struct {
	ServerID      int32   `json:"server_id"`
	Address       string  `json:"address"`
	MaxMaps       int32   `json:"max_maps"`
	CurrentMaps   int32   `json:"current_maps"`
	MaxPlayers    int32   `json:"max_players"`
	CurrentPlayers int32  `json:"current_players"`
	ServerGroupID int32   `json:"server_group_id"`
	SupportedMaps []int32 `json:"supported_maps"`
}

type MapServerRegistry struct {
	mu           sync.RWMutex
	localServer  *MapServerRegistration
	remoteServer *crossserver.ServerRouter
	crossRouter  *crossserver.CrossRouter
	dungeonHandler *dungeon.DungeonHandler
	mapManager   *MapManager
}

func NewMapServerRegistry(serverID int32, address string, maxMaps, maxPlayers int32, serverGroupID int32, mapManager *MapManager) *MapServerRegistry {
	reg := &MapServerRegistration{
		ServerID:      serverID,
		Address:       address,
		MaxMaps:       maxMaps,
		MaxPlayers:    maxPlayers,
		ServerGroupID: serverGroupID,
		SupportedMaps: make([]int32, 0),
	}

	return &MapServerRegistry{
		localServer: reg,
		mapManager:  mapManager,
	}
}

func (msr *MapServerRegistry) SetCrossRouter(router *crossserver.CrossRouter) {
	msr.crossRouter = router
	if router != nil {
		msr.registerHandlers()
	}
}

func (msr *MapServerRegistry) SetServerRouter(sr *crossserver.ServerRouter) {
	msr.remoteServer = sr
}

func (msr *MapServerRegistry) SetDungeonHandler(handler *dungeon.DungeonHandler) {
	msr.dungeonHandler = handler
}

func (msr *MapServerRegistry) registerHandlers() {
	if msr.crossRouter == nil {
		return
	}

	msr.crossRouter.RegisterHandler(crossserver.ServiceTypeGame, MsgMapServerRegister, msr.handleRegister)
	msr.crossRouter.RegisterHandler(crossserver.ServiceTypeGame, MsgMapServerHeartbeat, msr.handleHeartbeat)
	msr.crossRouter.RegisterHandler(crossserver.ServiceTypeGame, MsgMapServerUnregister, msr.handleUnregister)
	msr.crossRouter.RegisterHandler(crossserver.ServiceTypeGame, MsgMapServerStatus, msr.handleStatus)

	msr.crossRouter.RegisterHandler(crossserver.ServiceTypeGame, dungeon.MsgDungeonCreateRequest, msr.handleDungeonMessage)
	msr.crossRouter.RegisterHandler(crossserver.ServiceTypeGame, dungeon.MsgDungeonEnterRequest, msr.handleDungeonMessage)
	msr.crossRouter.RegisterHandler(crossserver.ServiceTypeGame, dungeon.MsgDungeonLeaveRequest, msr.handleDungeonMessage)
	msr.crossRouter.RegisterHandler(crossserver.ServiceTypeGame, dungeon.MsgDungeonStartRequest, msr.handleDungeonMessage)
	msr.crossRouter.RegisterHandler(crossserver.ServiceTypeGame, dungeon.MsgDungeonDestroyRequest, msr.handleDungeonMessage)
	msr.crossRouter.RegisterHandler(crossserver.ServiceTypeGame, dungeon.MsgDungeonQueryRequest, msr.handleDungeonMessage)
	msr.crossRouter.RegisterHandler(crossserver.ServiceTypeGame, dungeon.MsgDungeonMonsterKilled, msr.handleDungeonMessage)

	zLog.Info("MapServer cross-router handlers registered")
}

func (msr *MapServerRegistry) handleRegister(meta crossserver.Meta, payload []byte) ([]byte, error) {
	var reg MapServerRegistration
	if err := json.Unmarshal(payload, &reg); err != nil {
		return nil, fmt.Errorf("unmarshal registration: %w", err)
	}

	if msr.remoteServer != nil {
		msr.remoteServer.RegisterConnection(&crossserver.ServerConnection{
			ServerID:    fmt.Sprintf("%d", reg.ServerID),
			ServiceType: crossserver.ServiceTypeMap,
			Address:     reg.Address,
			Connected:   true,
		})
	}

	zLog.Info("MapServer registration received",
		zap.Int32("server_id", reg.ServerID),
		zap.String("address", reg.Address))

	resp := MapServerRegistration{
		ServerID:    msr.localServer.ServerID,
		Address:     msr.localServer.Address,
		CurrentMaps: msr.getCurrentMapCount(),
	}
	respData, _ := json.Marshal(resp)
	return respData, nil
}

func (msr *MapServerRegistry) handleHeartbeat(meta crossserver.Meta, payload []byte) ([]byte, error) {
	status := msr.GetStatus()
	data, _ := json.Marshal(status)
	return data, nil
}

func (msr *MapServerRegistry) handleUnregister(meta crossserver.Meta, payload []byte) ([]byte, error) {
	var reg MapServerRegistration
	if err := json.Unmarshal(payload, &reg); err != nil {
		return nil, fmt.Errorf("unmarshal unregister: %w", err)
	}

	if msr.remoteServer != nil {
		msr.remoteServer.UnregisterConnection(fmt.Sprintf("%d", reg.ServerID))
	}

	zLog.Info("MapServer unregistered",
		zap.Int32("server_id", reg.ServerID))

	return nil, nil
}

func (msr *MapServerRegistry) handleStatus(meta crossserver.Meta, payload []byte) ([]byte, error) {
	status := msr.GetStatus()
	data, _ := json.Marshal(status)
	return data, nil
}

func (msr *MapServerRegistry) handleDungeonMessage(meta crossserver.Meta, payload []byte) ([]byte, error) {
	if msr.dungeonHandler == nil {
		return nil, fmt.Errorf("dungeon handler not set")
	}

	msgID := uint32(0)
	var msg struct {
		MsgID uint32 `json:"msg_id"`
	}
	if err := json.Unmarshal(payload, &msg); err == nil {
		msgID = msg.MsgID
	}

	return msr.dungeonHandler.HandleMessage(msgID, payload)
}

func (msr *MapServerRegistry) RegisterToGlobalServer() error {
	if msr.remoteServer == nil {
		return fmt.Errorf("server router not set")
	}

	msr.localServer.CurrentMaps = msr.getCurrentMapCount()
	msr.localServer.CurrentPlayers = msr.getCurrentPlayerCount()

	data, err := json.Marshal(msr.localServer)
	if err != nil {
		return fmt.Errorf("marshal registration: %w", err)
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeMap, msr.localServer.ServerID)
	envelope := crossserver.Wrap(meta, data)

	gameServers := msr.remoteServer.GetConnectionsByService(crossserver.ServiceTypeGlobal)
	if len(gameServers) == 0 {
		return fmt.Errorf("no global server available")
	}

	for _, conn := range gameServers {
		if conn.Connected {
			if err := conn.SendFunc(envelope); err != nil {
				zLog.Warn("Failed to register to global server",
					zap.String("server_id", conn.ServerID),
					zap.String("error", err.Error()))
				continue
			}
			zLog.Info("Registered to global server",
				zap.String("server_id", conn.ServerID))
		}
	}

	return nil
}

func (msr *MapServerRegistry) RegisterMapToRouter(mapID id.MapIdType) {
	if msr.remoteServer != nil {
		msr.remoteServer.RegisterMapServer(int32(mapID), fmt.Sprintf("%d", msr.localServer.ServerID))
	}
}

func (msr *MapServerRegistry) GetStatus() *MapServerRegistration {
	msr.mu.RLock()
	defer msr.mu.RUnlock()

	status := *msr.localServer
	status.CurrentMaps = msr.getCurrentMapCount()
	status.CurrentPlayers = msr.getCurrentPlayerCount()

	return &status
}

func (msr *MapServerRegistry) getCurrentMapCount() int32 {
	if msr.mapManager == nil {
		return 0
	}
	return int32(msr.mapManager.GetMapCount())
}

func (msr *MapServerRegistry) getCurrentPlayerCount() int32 {
	if msr.mapManager == nil {
		return 0
	}

	total := int32(0)
	for _, m := range msr.mapManager.GetAllMaps() {
		total += int32(m.GetPlayerCount())
	}
	return total
}

func (msr *MapServerRegistry) SendDungeonMessageToGameServer(targetServerID int32, msgID uint32, data []byte) error {
	if msr.remoteServer == nil {
		return fmt.Errorf("server router not set")
	}

	meta := crossserver.NewRequestMeta(crossserver.ServiceTypeMap, msr.localServer.ServerID)
	envelope := crossserver.Wrap(meta, data)

	return msr.remoteServer.SendToServer(fmt.Sprintf("%d", targetServerID), envelope)
}
