package di

import (
	"github.com/pzqf/zEngine/zInject"
	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/repository"
)

const (
	ConnectorGlobal = "connector:global"
	ConnectorGame   = "connector:game"
	ConnectorLog    = "connector:log"

	DAOAccount     = "dao:account"
	DAOPlayer      = "dao:player"
	DAOPlayerItem  = "dao:player_item"
	DAOPlayerSkill = "dao:player_skill"
	DAOPlayerQuest = "dao:player_quest"
	DAOPlayerBuff  = "dao:player_buff"
	DAOAuction     = "dao:auction"
	DAOLoginLog    = "dao:login_log"
	DAOQuestLog    = "dao:quest_log"
	DAOAuctionLog  = "dao:auction_log"
	DAOGameServer  = "dao:game_server"

	RepoAccount     = "repo:account"
	RepoPlayer      = "repo:player"
	RepoPlayerItem  = "repo:player_item"
	RepoPlayerSkill = "repo:player_skill"
	RepoPlayerQuest = "repo:player_quest"
	RepoPlayerBuff  = "repo:player_buff"
	RepoAuction     = "repo:auction"
	RepoLoginLog    = "repo:login_log"
	RepoQuestLog    = "repo:quest_log"
	RepoAuctionLog  = "repo:auction_log"
	RepoGameServer  = "repo:game_server"
)

func RegisterConnectors(container zInject.Container, connectors map[string]connector.DBConnector) {
	for name, conn := range connectors {
		container.RegisterSingleton("connector:"+name, conn)
	}
}

func RegisterDAOs(container zInject.Container) {
	if container.Has(ConnectorGlobal) {
		container.Register(DAOAccount, func() interface{} {
			conn, _ := container.Resolve(ConnectorGlobal)
			return dao.NewAccountDAO(conn.(connector.DBConnector))
		})

		container.Register(DAOGameServer, func() interface{} {
			conn, _ := container.Resolve(ConnectorGlobal)
			return dao.NewGameServerDAO(conn.(connector.DBConnector))
		})
	}

	if container.Has(ConnectorGame) {
		container.Register(DAOPlayer, func() interface{} {
			conn, _ := container.Resolve(ConnectorGame)
			return dao.NewPlayerDAO(conn.(connector.DBConnector))
		})

		container.Register(DAOPlayerItem, func() interface{} {
			conn, _ := container.Resolve(ConnectorGame)
			return dao.NewPlayerItemDAO(conn.(connector.DBConnector))
		})

		container.Register(DAOPlayerSkill, func() interface{} {
			conn, _ := container.Resolve(ConnectorGame)
			return dao.NewPlayerSkillDAO(conn.(connector.DBConnector))
		})

		container.Register(DAOPlayerQuest, func() interface{} {
			conn, _ := container.Resolve(ConnectorGame)
			return dao.NewPlayerQuestDAO(conn.(connector.DBConnector))
		})

		container.Register(DAOPlayerBuff, func() interface{} {
			conn, _ := container.Resolve(ConnectorGame)
			return dao.NewPlayerBuffDAO(conn.(connector.DBConnector))
		})

		container.Register(DAOAuction, func() interface{} {
			conn, _ := container.Resolve(ConnectorGame)
			return dao.NewAuctionDAO(conn.(connector.DBConnector))
		})
	}

	if container.Has(ConnectorLog) {
		container.Register(DAOLoginLog, func() interface{} {
			conn, _ := container.Resolve(ConnectorLog)
			return dao.NewLoginLogDAO(conn.(connector.DBConnector))
		})

		container.Register(DAOQuestLog, func() interface{} {
			conn, _ := container.Resolve(ConnectorLog)
			return dao.NewQuestLogDAO(conn.(connector.DBConnector))
		})

		container.Register(DAOAuctionLog, func() interface{} {
			conn, _ := container.Resolve(ConnectorLog)
			return dao.NewAuctionLogDAO(conn.(connector.DBConnector))
		})
	}
}

func RegisterRepositories(container zInject.Container) {
	container.Register(RepoAccount, func() interface{} {
		if !container.Has(DAOAccount) {
			return nil
		}
		d, _ := container.Resolve(DAOAccount)
		return repository.NewAccountRepository(d.(*dao.AccountDAO))
	})

	container.Register(RepoPlayer, func() interface{} {
		if !container.Has(DAOPlayer) {
			return nil
		}
		d, _ := container.Resolve(DAOPlayer)
		return repository.NewPlayerRepository(d.(*dao.PlayerDAO))
	})

	container.Register(RepoPlayerItem, func() interface{} {
		if !container.Has(DAOPlayerItem) {
			return nil
		}
		d, _ := container.Resolve(DAOPlayerItem)
		return repository.NewPlayerItemRepository(d.(*dao.PlayerItemDAO))
	})

	container.Register(RepoPlayerSkill, func() interface{} {
		if !container.Has(DAOPlayerSkill) {
			return nil
		}
		d, _ := container.Resolve(DAOPlayerSkill)
		return repository.NewPlayerSkillRepository(d.(*dao.PlayerSkillDAO))
	})

	container.Register(RepoPlayerQuest, func() interface{} {
		if !container.Has(DAOPlayerQuest) {
			return nil
		}
		d, _ := container.Resolve(DAOPlayerQuest)
		return repository.NewPlayerQuestRepository(d.(*dao.PlayerQuestDAO))
	})

	container.Register(RepoPlayerBuff, func() interface{} {
		if !container.Has(DAOPlayerBuff) {
			return nil
		}
		d, _ := container.Resolve(DAOPlayerBuff)
		return repository.NewPlayerBuffRepository(d.(*dao.PlayerBuffDAO))
	})

	container.Register(RepoAuction, func() interface{} {
		if !container.Has(DAOAuction) {
			return nil
		}
		d, _ := container.Resolve(DAOAuction)
		return repository.NewAuctionRepository(d.(*dao.AuctionDAO))
	})

	container.Register(RepoLoginLog, func() interface{} {
		if !container.Has(DAOLoginLog) {
			return nil
		}
		d, _ := container.Resolve(DAOLoginLog)
		return repository.NewLoginLogRepository(d.(*dao.LoginLogDAO))
	})

	container.Register(RepoQuestLog, func() interface{} {
		if !container.Has(DAOQuestLog) {
			return nil
		}
		d, _ := container.Resolve(DAOQuestLog)
		return repository.NewQuestLogRepository(d.(*dao.QuestLogDAO))
	})

	container.Register(RepoAuctionLog, func() interface{} {
		if !container.Has(DAOAuctionLog) {
			return nil
		}
		d, _ := container.Resolve(DAOAuctionLog)
		return repository.NewAuctionLogRepository(d.(*dao.AuctionLogDAO))
	})

	container.Register(RepoGameServer, func() interface{} {
		if !container.Has(DAOGameServer) {
			return nil
		}
		d, _ := container.Resolve(DAOGameServer)
		return repository.NewGameServerRepository(d.(*dao.GameServerDAO))
	})
}

func Resolve[T any](container zInject.Container, name string) (T, error) {
	var zero T
	instance, err := container.Resolve(name)
	if err != nil {
		return zero, err
	}
	result, ok := instance.(T)
	if !ok {
		return zero, nil
	}
	return result, nil
}

func ResolveDAO[T any](container zInject.Container, name string) T {
	result, _ := Resolve[T](container, name)
	return result
}

func ResolveRepo[T any](container zInject.Container, name string) T {
	result, _ := Resolve[T](container, name)
	return result
}

