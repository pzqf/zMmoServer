package player

import (
	"os"
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
	warehouse   *game.Warehouse
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
		warehouse:    game.NewWarehouse(120),
		equipment:    game.NewEquipment(),
		skillMgr:     game.NewSkillManager(50),
		buffMgr:      game.NewBuffManager(),
		taskMgr:      game.NewTaskManager(20),
	}
	baseActor.SetSelf(p)

	// 业务层建设：测试种子物品（env ZMMO_TEST_ITEMS=1 门控，生产默认关）。放在 NewPlayer——所有
	// 玩家构造的必经点，供物品/仓库 E2E 有货可操作。两件不同 configID → 落入背包 slot 0 / slot 1。
	if os.Getenv("ZMMO_TEST_ITEMS") == "1" {
		_, _ = p.inventory.AddItem(game.NewItem(1001, 1001, "TestPotion", game.ItemTypeConsumable, 3))
		_, _ = p.inventory.AddItem(game.NewItem(1002, 1002, "TestMaterial", game.ItemTypeConsumable, 10))
	}

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
