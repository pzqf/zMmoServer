package player

import (
	"sync"
	"sync/atomic"

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
	Pickup(playerID id.PlayerIdType, mapID id.MapIdType) error
	// AttachCrossMap 把玩家挂到**已解析好**的跨服活动实例上（realm §5.1）：绑路由 + 发进图请求。
	// 落点的解析（问 GlobalServer 分配 → 连外域 MapServer → 建实例，数秒量级）刻意留在 actor 之外，
	// 这里只做快的那一小段，避免玩家 actor 被网络往返占住。
	AttachCrossMap(playerID id.PlayerIdType, mapServerID uint32, mapID id.MapIdType, pos common.Vector3) error
}

type Player struct {
	*zActor.BaseActor
	*object.LivingObject
	mu          sync.RWMutex
	accountID   id.AccountIdType
	attrs       *game.PlayerAttributes
	inventory   *game.Inventory
	warehouse   *game.Warehouse
	equipment   *game.Equipment
	skillMgr    *game.SkillManager
	buffMgr     *game.BuffManager
	taskMgr     *game.TaskManager
	mapOp       MapOperator
	currentMap  id.MapIdType
	sessionID   interface{}
	clientSender common.ClientSender
	// assetsDirty 资产（背包/技能/仓库/buff/任务）有未落库的变更。
	// 见 player_assets_dirty.go：这些资产原本只在登出/关服才写库，崩溃会丢掉整场游戏的变更。
	assetsDirty atomic.Bool
}

func NewPlayer(playerID id.PlayerIdType, accountID id.AccountIdType, name string) *Player {
	baseActor := zActor.NewBaseActor(int64(playerID), 100)
	p := &Player{
		BaseActor:    baseActor,
		LivingObject: object.NewLivingObject(id.ObjectIdType(playerID), name, common.GameObjectTypePlayer),
		accountID:    accountID,
		attrs:        game.NewPlayerAttributes(),
		inventory:    game.NewInventory(60),
		warehouse:    game.NewWarehouse(120),
		equipment:    game.NewEquipment(),
		skillMgr:     game.NewSkillManager(50),
		buffMgr:      game.NewBuffManager(),
		taskMgr:      game.NewTaskManager(20),
	}
	baseActor.SetSelf(p)

	// 背包/技能不再在此种子——已移至 player_persist.loadPlayerAssets（登录时先从 DB 载入，仅当 DB
	// 无持久化背包时才按 ZMMO_TEST_ITEMS 种子），避免种子与持久化叠加累积。
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

func (p *Player) GetWarehouse() *game.Warehouse {
	return p.warehouse
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
