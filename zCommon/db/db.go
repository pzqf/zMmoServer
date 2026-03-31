package db

import (
	"sync"

	"github.com/pzqf/zEngine/zInject"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/di"
	"github.com/pzqf/zCommon/db/models"
	"github.com/pzqf/zCommon/db/repository"
	"go.uber.org/zap"
)

type DBManager struct {
	container             zInject.Container
	connectors            map[string]connector.DBConnector
	repoTypes             []RepoType
	PlayerRepository      repository.PlayerRepository
	AccountRepository     repository.AccountRepository
	PlayerItemRepository  repository.PlayerItemRepository
	PlayerSkillRepository repository.PlayerSkillRepository
	PlayerQuestRepository repository.PlayerQuestRepository
	PlayerBuffRepository  repository.PlayerBuffRepository
	AuctionRepository     repository.AuctionRepository
	LoginLogRepository    repository.LoginLogRepository
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

// InitDBManagerWithConfig 使用配置初始化DB管理器（初始化所有Repository）
func InitDBManagerWithConfig(dbConfigs map[string]DBConfig) error {
	return InitDBManagerWithRepos(dbConfigs, RepoTypeAll)
}

// InitDBManagerWithRepos 使用配置初始化DB管理器（按需初始化Repository）
func InitDBManagerWithRepos(dbConfigs map[string]DBConfig, repoTypes []RepoType) error {
	zLog.Info("Initializing DB manager with config...")
	var err error
	dbOnce.Do(func() {
		zLog.Info("Creating DB manager instance...")
		dbManager = &DBManager{
			container:  zInject.NewContainer(),
			connectors: make(map[string]connector.DBConnector),
			repoTypes:  repoTypes,
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

	zLog.Info("Initializing repositories...", zap.Int("count", len(manager.repoTypes)))
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
	for _, repoType := range manager.repoTypes {
		switch repoType {
		case RepoTypeAccount:
			zLog.Info("Resolving AccountRepository...")
			manager.AccountRepository = di.ResolveRepo[repository.AccountRepository](manager.container, di.RepoAccount)
			zLog.Info("AccountRepository resolved")

		case RepoTypePlayer:
			zLog.Info("Resolving PlayerRepository...")
			manager.PlayerRepository = di.ResolveRepo[repository.PlayerRepository](manager.container, di.RepoPlayer)
			zLog.Info("PlayerRepository resolved")

		case RepoTypePlayerItem:
			zLog.Info("Resolving PlayerItemRepository...")
			manager.PlayerItemRepository = di.ResolveRepo[repository.PlayerItemRepository](manager.container, di.RepoPlayerItem)
			zLog.Info("PlayerItemRepository resolved")

		case RepoTypePlayerSkill:
			zLog.Info("Resolving PlayerSkillRepository...")
			manager.PlayerSkillRepository = di.ResolveRepo[repository.PlayerSkillRepository](manager.container, di.RepoPlayerSkill)
			zLog.Info("PlayerSkillRepository resolved")

		case RepoTypePlayerQuest:
			zLog.Info("Resolving PlayerQuestRepository...")
			manager.PlayerQuestRepository = di.ResolveRepo[repository.PlayerQuestRepository](manager.container, di.RepoPlayerQuest)
			zLog.Info("PlayerQuestRepository resolved")

		case RepoTypePlayerBuff:
			zLog.Info("Resolving PlayerBuffRepository...")
			manager.PlayerBuffRepository = di.ResolveRepo[repository.PlayerBuffRepository](manager.container, di.RepoPlayerBuff)
			zLog.Info("PlayerBuffRepository resolved")

		case RepoTypeAuction:
			zLog.Info("Resolving AuctionRepository...")
			manager.AuctionRepository = di.ResolveRepo[repository.AuctionRepository](manager.container, di.RepoAuction)
			zLog.Info("AuctionRepository resolved")

		case RepoTypeLoginLog:
			zLog.Info("Resolving LoginLogRepository...")
			manager.LoginLogRepository = di.ResolveRepo[repository.LoginLogRepository](manager.container, di.RepoLoginLog)
			zLog.Info("LoginLogRepository resolved")

		case RepoTypeQuestLog:
			zLog.Info("Resolving QuestLogRepository...")
			manager.QuestLogRepository = di.ResolveRepo[repository.QuestLogRepository](manager.container, di.RepoQuestLog)
			zLog.Info("QuestLogRepository resolved")

		case RepoTypeAuctionLog:
			zLog.Info("Resolving AuctionLogRepository...")
			manager.AuctionLogRepository = di.ResolveRepo[repository.AuctionLogRepository](manager.container, di.RepoAuctionLog)
			zLog.Info("AuctionLogRepository resolved")

		case RepoTypeGameServer:
			zLog.Info("Resolving GameServerRepository...")
			manager.GameServerRepository = di.ResolveRepo[repository.GameServerRepository](manager.container, di.RepoGameServer)
			zLog.Info("GameServerRepository resolved")
		}
	}

	zLog.Info("All repositories resolved successfully", zap.Int("count", len(manager.repoTypes)))
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
