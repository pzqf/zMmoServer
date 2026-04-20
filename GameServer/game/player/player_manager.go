package player

import (
	"errors"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/lifecycle"
	"github.com/pzqf/zEngine/zActor"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

type PlayerManager struct {
	players          *zMap.TypedMap[id.PlayerIdType, *Player]
	playersByAccount *zMap.TypedMap[id.AccountIdType, *Player]
	lifecycleMgr     *lifecycle.Manager
	mapOp            MapOperator
	clientSender     common.ClientSender
	loginService     *LoginService
}

func NewPlayerManager() *PlayerManager {
	pm := &PlayerManager{
		players:          zMap.NewTypedMap[id.PlayerIdType, *Player](),
		playersByAccount: zMap.NewTypedMap[id.AccountIdType, *Player](),
		lifecycleMgr:     lifecycle.NewManager(),
	}

	pm.lifecycleMgr.RegisterType("player", func(id int64) lifecycle.Object {
		return &lifecycle.BaseLifecycleObject{ObjectID: id, ObjectType: "player"}
	}, &lifecycle.LifecycleHooks{
		OnCreate: func(obj lifecycle.Object) error {
			zLog.Info("Lifecycle: player creating", zap.Int64("player_id", obj.GetObjectID()))
			return nil
		},
		OnActivate: func(obj lifecycle.Object) error {
			zLog.Info("Lifecycle: player activated", zap.Int64("player_id", obj.GetObjectID()))
			return nil
		},
		OnSuspend: func(obj lifecycle.Object) error {
			zLog.Info("Lifecycle: player suspending", zap.Int64("player_id", obj.GetObjectID()))
			return nil
		},
		OnResume: func(obj lifecycle.Object) error {
			zLog.Info("Lifecycle: player resuming", zap.Int64("player_id", obj.GetObjectID()))
			return nil
		},
		OnDestroy: func(obj lifecycle.Object) error {
			zLog.Info("Lifecycle: player destroying", zap.Int64("player_id", obj.GetObjectID()))
			return nil
		},
	})

	return pm
}

func (pm *PlayerManager) SetMapOperator(op MapOperator) {
	pm.mapOp = op
}

func (pm *PlayerManager) SetClientSender(sender common.ClientSender) {
	pm.clientSender = sender
}

func (pm *PlayerManager) SetLoginService(ls *LoginService) {
	pm.loginService = ls
}

func (pm *PlayerManager) GetLoginService() *LoginService {
	return pm.loginService
}

func (pm *PlayerManager) CreatePlayer(playerID id.PlayerIdType, accountID id.AccountIdType, name string) (*Player, error) {
	if _, exists := pm.players.Load(playerID); exists {
		return nil, ErrPlayerAlreadyExists
	}

	if _, err := pm.lifecycleMgr.Create("player", int64(playerID)); err != nil {
		return nil, err
	}

	player := NewPlayer(playerID, accountID, name)
	if pm.mapOp != nil {
		player.SetMapOperator(pm.mapOp)
	}

	if err := player.Start(); err != nil {
		pm.lifecycleMgr.Destroy(int64(playerID))
		return nil, err
	}

	pm.players.Store(playerID, player)
	pm.playersByAccount.Store(accountID, player)

	return player, nil
}

func (pm *PlayerManager) AddPlayer(player *Player) error {
	if player == nil {
		return ErrPlayerNil
	}

	playerID := player.GetPlayerID()
	if playerID == 0 {
		return ErrPlayerIDZero
	}

	if _, exists := pm.players.Load(playerID); exists {
		return ErrPlayerAlreadyExists
	}

	if pm.mapOp != nil {
		player.SetMapOperator(pm.mapOp)
	}

	if err := player.Start(); err != nil {
		return err
	}

	pm.players.Store(playerID, player)
	pm.playersByAccount.Store(player.GetAccountID(), player)

	return nil
}

func (pm *PlayerManager) GetPlayer(playerID id.PlayerIdType) (*Player, error) {
	player, ok := pm.players.Load(playerID)
	if !ok {
		return nil, ErrPlayerNotFound
	}
	return player, nil
}

func (pm *PlayerManager) GetPlayerByAccount(accountID id.AccountIdType) (*Player, error) {
	player, exists := pm.playersByAccount.Load(accountID)
	if !exists {
		return nil, ErrPlayerNotFound
	}
	return player, nil
}

func (pm *PlayerManager) RemovePlayer(playerID id.PlayerIdType) error {
	player, ok := pm.players.Load(playerID)
	if !ok {
		return ErrPlayerNotFound
	}

	player.Stop()

	pm.players.Delete(playerID)
	pm.playersByAccount.Delete(player.GetAccountID())

	if err := pm.lifecycleMgr.Destroy(int64(playerID)); err != nil {
		zLog.Warn("Failed to destroy lifecycle object",
			zap.Int64("player_id", int64(playerID)),
			zap.Error(err))
	}

	return nil
}

func (pm *PlayerManager) SuspendPlayer(playerID id.PlayerIdType) error {
	if _, ok := pm.players.Load(playerID); !ok {
		return ErrPlayerNotFound
	}
	return pm.lifecycleMgr.Suspend(int64(playerID))
}

func (pm *PlayerManager) ResumePlayer(playerID id.PlayerIdType) error {
	if _, ok := pm.players.Load(playerID); !ok {
		return ErrPlayerNotFound
	}
	return pm.lifecycleMgr.Resume(int64(playerID))
}

func (pm *PlayerManager) GetPlayerState(playerID id.PlayerIdType) (lifecycle.ObjectState, error) {
	obj, ok := pm.lifecycleMgr.Get(int64(playerID))
	if !ok {
		return lifecycle.ObjectStateNone, ErrPlayerNotFound
	}
	return obj.GetState(), nil
}

func (pm *PlayerManager) GetAllPlayers() []*Player {
	players := make([]*Player, 0)
	pm.players.Range(func(pid id.PlayerIdType, player *Player) bool {
		players = append(players, player)
		return true
	})
	return players
}

func (pm *PlayerManager) GetPlayerCount() int64 {
	return pm.players.Len()
}

func (pm *PlayerManager) GetActivePlayerCount() int64 {
	return int64(pm.lifecycleMgr.CountByState(lifecycle.ObjectStateActive))
}

func (pm *PlayerManager) RouteMessage(playerID id.PlayerIdType, msg *PlayerMessage) error {
	player, err := pm.GetPlayer(playerID)
	if err != nil {
		return err
	}

	msg.ActorID = int64(playerID)
	player.SendMessage(msg)
	return nil
}

func (pm *PlayerManager) BroadcastMessage(msg *PlayerMessage) {
	pm.players.Range(func(pid id.PlayerIdType, player *Player) bool {
		msgCopy := &PlayerMessage{
			BaseActorMessage: zActor.BaseActorMessage{ActorID: int64(player.GetPlayerID())},
			Source:           msg.Source,
			Type:             msg.Type,
			Data:             msg.Data,
		}
		player.SendMessage(msgCopy)
		return true
	})
}

func (pm *PlayerManager) BroadcastMessageToPlayers(playerIDs []id.PlayerIdType, msg *PlayerMessage) {
	for _, playerID := range playerIDs {
		pm.RouteMessage(playerID, msg)
	}
}

func (pm *PlayerManager) HasPlayer(playerID id.PlayerIdType) bool {
	_, ok := pm.players.Load(playerID)
	return ok
}

func (pm *PlayerManager) HasPlayerByAccount(accountID id.AccountIdType) bool {
	_, exists := pm.playersByAccount.Load(accountID)
	return exists
}

func (pm *PlayerManager) Range(f func(playerID id.PlayerIdType, player *Player) bool) {
	pm.players.Range(f)
}

func (pm *PlayerManager) ClearAll() {
	pm.players.Range(func(pid id.PlayerIdType, player *Player) bool {
		player.Stop()
		return true
	})

	pm.players.Clear()
	pm.playersByAccount.Clear()
	pm.lifecycleMgr.DestroyAll()
}

func (pm *PlayerManager) SerializePlayer(playerID id.PlayerIdType) ([]byte, error) {
	return pm.lifecycleMgr.Serialize(int64(playerID))
}

func (pm *PlayerManager) DeserializePlayer(data []byte) (lifecycle.Object, error) {
	return pm.lifecycleMgr.Deserialize("player", data)
}

var (
	ErrPlayerAlreadyExists = errors.New("player already exists")
	ErrPlayerNotFound      = errors.New("player not found")
	ErrPlayerNil           = errors.New("player can't be nil")
	ErrPlayerIDZero        = errors.New("player id can't be 0")
)
