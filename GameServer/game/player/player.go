package player

import (
	"sync"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/game"
	"github.com/pzqf/zEngine/zActor"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoServer/GameServer/game/object"
	"go.uber.org/zap"
)

type MapOperator interface {
	EnterMap(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error
	LeaveMap(playerID id.PlayerIdType, mapID id.MapIdType) error
	Move(playerID id.PlayerIdType, mapID id.MapIdType, pos common.Vector3) error
	Attack(playerID id.PlayerIdType, mapID id.MapIdType, targetID id.ObjectIdType) (int64, int64, error)
}

type Player struct {
	*zActor.BaseActor
	*object.LivingObject
	mu          sync.RWMutex
	accountID   id.AccountIdType
	attrs       *game.PlayerAttributes
	inventory   *game.Inventory
	equipment   *game.Equipment
	skillMgr    *game.SkillManager
	buffMgr     *game.BuffManager
	taskMgr     *game.TaskManager
	mapOp       MapOperator
	currentMap  id.MapIdType
	sessionID   interface{}
	clientSender common.ClientSender
}

func NewPlayer(playerID id.PlayerIdType, accountID id.AccountIdType, name string) *Player {
	baseActor := zActor.NewBaseActor(int64(playerID), 100)
	p := &Player{
		BaseActor:    baseActor,
		LivingObject: object.NewLivingObject(id.ObjectIdType(playerID), name, common.GameObjectTypePlayer),
		accountID:    accountID,
		attrs:        game.NewPlayerAttributes(),
		inventory:    game.NewInventory(60),
		equipment:    game.NewEquipment(),
		skillMgr:     game.NewSkillManager(50),
		buffMgr:      game.NewBuffManager(),
		taskMgr:      game.NewTaskManager(20),
	}
	baseActor.SetSelf(p)
	return p
}

func (p *Player) SetMapOperator(op MapOperator) {
	p.mapOp = op
}

func (p *Player) SetSessionInfo(sessionID interface{}, sender common.ClientSender) {
	p.sessionID = sessionID
	p.clientSender = sender
}

func (p *Player) Start() error {
	if err := p.BaseActor.Start(); err != nil {
		return err
	}
	zLog.Info("Player Actor started", zap.Int64("player_id", int64(p.GetPlayerID())))
	return nil
}

func (p *Player) Stop() error {
	if err := p.BaseActor.Stop(); err != nil {
		return err
	}
	zLog.Info("Player Actor stopped", zap.Int64("player_id", int64(p.GetPlayerID())))
	return nil
}

func (p *Player) ProcessMessage(msg zActor.ActorMessage) {
	switch typedMsg := msg.(type) {
	case *PlayerMessage:
		p.handleMessage(typedMsg)
	default:
		zLog.Warn("Unknown message type",
			zap.Int64("player_id", int64(p.GetPlayerID())),
			zap.Any("message", msg))
	}
}

func (p *Player) Update(deltaTime float64) {
	p.buffMgr.Update(deltaTime)
}

func (p *Player) GetAttrs() *game.PlayerAttributes {
	return p.attrs
}

func (p *Player) GetInventory() *game.Inventory {
	return p.inventory
}

func (p *Player) GetEquipment() *game.Equipment {
	return p.equipment
}

func (p *Player) GetSkillManager() *game.SkillManager {
	return p.skillMgr
}

func (p *Player) GetBuffManager() *game.BuffManager {
	return p.buffMgr
}

func (p *Player) GetTaskManager() *game.TaskManager {
	return p.taskMgr
}
