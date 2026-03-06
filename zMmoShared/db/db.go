package db

import (
	"sync"

	"github.com/pzqf/zEngine/zInject"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/di"
	"github.com/pzqf/zMmoShared/db/models"
	"github.com/pzqf/zMmoShared/db/repository"
	"go.uber.org/zap"
)

type DBManager struct {
	container             zInject.Container
	connectors            map[string]connector.DBConnector
	PlayerRepository      repository.PlayerRepository
	AccountRepository     repository.AccountRepository
	PlayerItemRepository  repository.PlayerItemRepository
	PlayerSkillRepository repository.PlayerSkillRepository
	PlayerMailRepository  repository.PlayerMailRepository
	PlayerQuestRepository repository.PlayerQuestRepository
	PlayerPetRepository   repository.PlayerPetRepository
	PlayerBuffRepository  repository.PlayerBuffRepository
	GuildRepository       repository.GuildRepository
	GuildMemberRepository repository.GuildMemberRepository
	AuctionRepository     repository.AuctionRepository
	LoginLogRepository    repository.LoginLogRepository
	MailLogRepository     repository.MailLogRepository
	QuestLogRepository    repository.QuestLogRepository
	AuctionLogRepository  repository.AuctionLogRepository
	GameServerRepository  repository.GameServerRepository
}

var (
	dbManager *DBManager
	dbOnce    sync.Once
)

func GetMgr() *DBManager {
	return dbManager
}

// InitDBManagerWithConfig 使用配置初始化DB管理器
func InitDBManagerWithConfig(dbConfigs map[string]DBConfig) error {
	zLog.Info("Initializing DB manager with config...")
	var err error
	dbOnce.Do(func() {
		zLog.Info("Creating DB manager instance...")
		dbManager = &DBManager{
			container:  zInject.NewContainer(),
			connectors: make(map[string]connector.DBConnector),
		}
		zLog.Info("Initializing DB manager...")
		err = dbManager.InitWithConfig(dbConfigs)
		if err != nil {
			zLog.Error("Failed to initialize DB manager", zap.Error(err))
		} else {
			zLog.Info("DB manager initialized successfully")
		}
	})
	zLog.Info("DB manager initialization completed")
	return err
}

// InitDBManager 初始化DB管理器（使用默认配置，已废弃）
func InitDBManager() error {
	zLog.Warn("InitDBManager is deprecated, use InitDBManagerWithConfig instead")
	return InitDBManagerWithConfig(nil)
}

func (manager *DBManager) InitWithConfig(dbConfigs map[string]DBConfig) error {
	if dbConfigs == nil || len(dbConfigs) == 0 {
		zLog.Warn("No DB configs provided, skipping DB initialization")
		return nil
	}

	zLog.Info("Processing DB configs...", zap.Int("count", len(dbConfigs)))

	for dbName, dbConfig := range dbConfigs {
		zLog.Info("Processing DB config", zap.String("dbName", dbName))

		conn := connector.NewDBConnector(dbName, dbConfig.Driver, 1000)
		zLog.Info("Initializing connector...", zap.String("dbName", dbName))
		if err := conn.Init(toConnectorConfig(dbConfig)); err != nil {
			zLog.Error("Failed to initialize connector", zap.String("dbName", dbName), zap.Error(err))
			return err
		}
		zLog.Info("Starting connector...", zap.String("dbName", dbName))
		if err := conn.Start(); err != nil {
			zLog.Error("Failed to start connector", zap.String("dbName", dbName), zap.Error(err))
			return err
		}
		manager.connectors[dbName] = conn
		zLog.Info("Connector started successfully", zap.String("dbName", dbName))
	}

	zLog.Info("Registering connectors to container...")
	di.RegisterConnectors(manager.container, manager.connectors)
	zLog.Info("Registering DAOs to container...")
	di.RegisterDAOs(manager.container)
	zLog.Info("Registering repositories to container...")
	di.RegisterRepositories(manager.container)

	zLog.Info("Initializing repositories...")
	manager.initRepositories()
	zLog.Info("Repositories initialized successfully")
	return nil
}

// toConnectorConfig 转换为connector.DBConfig
func toConnectorConfig(cfg DBConfig) connector.DBConfig {
	return connector.DBConfig{
		Host:           cfg.Host,
		Port:           cfg.Port,
		User:           cfg.User,
		Password:       cfg.Password,
		DBName:         cfg.DBName,
		Charset:        cfg.Charset,
		MaxIdle:        cfg.MaxIdle,
		MaxOpen:        cfg.MaxOpen,
		Driver:         cfg.Driver,
		URI:            cfg.URI,
		MaxPoolSize:    cfg.MaxPoolSize,
		MinPoolSize:    cfg.MinPoolSize,
		ConnectTimeout: cfg.ConnectTimeout,
	}
}

func (manager *DBManager) initRepositories() {
	zLog.Info("Resolving AccountRepository...")
	manager.AccountRepository = di.ResolveRepo[repository.AccountRepository](manager.container, di.RepoAccount)
	zLog.Info("AccountRepository resolved")

	zLog.Info("Resolving PlayerRepository...")
	manager.PlayerRepository = di.ResolveRepo[repository.PlayerRepository](manager.container, di.RepoPlayer)
	zLog.Info("PlayerRepository resolved")

	zLog.Info("Resolving PlayerItemRepository...")
	manager.PlayerItemRepository = di.ResolveRepo[repository.PlayerItemRepository](manager.container, di.RepoPlayerItem)
	zLog.Info("PlayerItemRepository resolved")

	zLog.Info("Resolving PlayerSkillRepository...")
	manager.PlayerSkillRepository = di.ResolveRepo[repository.PlayerSkillRepository](manager.container, di.RepoPlayerSkill)
	zLog.Info("PlayerSkillRepository resolved")

	zLog.Info("Resolving PlayerMailRepository...")
	manager.PlayerMailRepository = di.ResolveRepo[repository.PlayerMailRepository](manager.container, di.RepoPlayerMail)
	zLog.Info("PlayerMailRepository resolved")

	zLog.Info("Resolving PlayerQuestRepository...")
	manager.PlayerQuestRepository = di.ResolveRepo[repository.PlayerQuestRepository](manager.container, di.RepoPlayerQuest)
	zLog.Info("PlayerQuestRepository resolved")

	zLog.Info("Resolving PlayerPetRepository...")
	manager.PlayerPetRepository = di.ResolveRepo[repository.PlayerPetRepository](manager.container, di.RepoPlayerPet)
	zLog.Info("PlayerPetRepository resolved")

	zLog.Info("Resolving PlayerBuffRepository...")
	manager.PlayerBuffRepository = di.ResolveRepo[repository.PlayerBuffRepository](manager.container, di.RepoPlayerBuff)
	zLog.Info("PlayerBuffRepository resolved")

	zLog.Info("Resolving GuildRepository...")
	manager.GuildRepository = di.ResolveRepo[repository.GuildRepository](manager.container, di.RepoGuild)
	zLog.Info("GuildRepository resolved")

	zLog.Info("Resolving GuildMemberRepository...")
	manager.GuildMemberRepository = di.ResolveRepo[repository.GuildMemberRepository](manager.container, di.RepoGuildMember)
	zLog.Info("GuildMemberRepository resolved")

	zLog.Info("Resolving AuctionRepository...")
	manager.AuctionRepository = di.ResolveRepo[repository.AuctionRepository](manager.container, di.RepoAuction)
	zLog.Info("AuctionRepository resolved")

	zLog.Info("Resolving LoginLogRepository...")
	manager.LoginLogRepository = di.ResolveRepo[repository.LoginLogRepository](manager.container, di.RepoLoginLog)
	zLog.Info("LoginLogRepository resolved")

	zLog.Info("Resolving MailLogRepository...")
	manager.MailLogRepository = di.ResolveRepo[repository.MailLogRepository](manager.container, di.RepoMailLog)
	zLog.Info("MailLogRepository resolved")

	zLog.Info("Resolving QuestLogRepository...")
	manager.QuestLogRepository = di.ResolveRepo[repository.QuestLogRepository](manager.container, di.RepoQuestLog)
	zLog.Info("QuestLogRepository resolved")

	zLog.Info("Resolving AuctionLogRepository...")
	manager.AuctionLogRepository = di.ResolveRepo[repository.AuctionLogRepository](manager.container, di.RepoAuctionLog)
	zLog.Info("AuctionLogRepository resolved")

	zLog.Info("Resolving GameServerRepository...")
	manager.GameServerRepository = di.ResolveRepo[repository.GameServerRepository](manager.container, di.RepoGameServer)
	zLog.Info("GameServerRepository resolved")

	zLog.Info("All repositories resolved successfully")
}

func (manager *DBManager) Close() error {
	for _, conn := range manager.connectors {
		if err := conn.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (manager *DBManager) GetConnector(dbName string) connector.DBConnector {
	return manager.connectors[dbName]
}

func (manager *DBManager) GetAllConnectors() map[string]connector.DBConnector {
	return manager.connectors
}

func (manager *DBManager) GetContainer() zInject.Container {
	return manager.container
}

func ValidateModelTags() error {
	return models.ValidateModelTags()
}
