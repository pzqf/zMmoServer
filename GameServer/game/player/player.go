package player

import (
	"sync"

	"github.com/pzqf/zEngine/zActor"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/game/common"
	"github.com/pzqf/zMmoServer/GameServer/game/object"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// Player 玩家对象，继承zActor.BaseActor
type Player struct {
	*zActor.BaseActor
	*object.LivingObject
	mu         sync.RWMutex
	accountID  id.AccountIdType
	gold       int64
	diamond    int64
	vipLevel   int32
	vipExp     int32
	guildID    id.GuildIdType
	teamID     id.TeamIdType
	friendList []id.PlayerIdType

	// 组件
	teamComp        *PlayerTeam        // 队伍组件
	petComp         *PlayerPet         // 宠物组件
	mountComp       *PlayerMount       // 坐骑组件
	achievementComp *PlayerAchievement // 成就组件
}

// NewPlayer 创建新的玩家对象
func NewPlayer(playerID id.PlayerIdType, accountID id.AccountIdType, name string) *Player {
	baseActor := zActor.NewBaseActor(int64(playerID), 100)
	p := &Player{
		BaseActor:    baseActor,
		LivingObject: object.NewLivingObject(id.ObjectIdType(playerID), name, common.GameObjectTypePlayer),
		accountID:    accountID,
		gold:         0,
		diamond:      0,
		vipLevel:     0,
		vipExp:       0,
		guildID:      0,
		teamID:       0,
		friendList:   make([]id.PlayerIdType, 0),
		// 初始化组件
		teamComp:        NewPlayerTeam(playerID),
		petComp:         NewPlayerPet(playerID),
		mountComp:       NewPlayerMount(playerID),
		achievementComp: NewPlayerAchievement(playerID),
	}
	return p
}

// GetTeamComponent 获取队伍组件
func (p *Player) GetTeamComponent() *PlayerTeam {
	return p.teamComp
}

// GetPetComponent 获取宠物组件
func (p *Player) GetPetComponent() *PlayerPet {
	return p.petComp
}

// GetMountComponent 获取坐骑组件
func (p *Player) GetMountComponent() *PlayerMount {
	return p.mountComp
}

// GetAchievementComponent 获取成就组件
func (p *Player) GetAchievementComponent() *PlayerAchievement {
	return p.achievementComp
}

// Start 启动玩家Actor
func (p *Player) Start() error {
	if err := p.BaseActor.Start(); err != nil {
		return err
	}
	zLog.Info("Player Actor started", zap.Int64("player_id", int64(p.GetPlayerID())))
	return nil
}

// Stop 停止玩家Actor
func (p *Player) Stop() error {
	if err := p.BaseActor.Stop(); err != nil {
		return err
	}
	zLog.Info("Player Actor stopped", zap.Int64("player_id", int64(p.GetPlayerID())))
	return nil
}

// ProcessMessage 实现zActor.Actor接口的消息处理方法
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

// update 玩家更新
func (p *Player) update() {
	// 更新组件（暂时注释，组件还未实现Update方法）
	// if p.teamComp != nil {
	// 	p.teamComp.Update()
	// }
	// if p.petComp != nil {
	// 	p.petComp.Update()
	// }
	// if p.mountComp != nil {
	// 	p.mountComp.Update()
	// }
	// if p.achievementComp != nil {
	// 	p.achievementComp.Update()
	// }
}
