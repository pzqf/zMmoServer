package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	sharedDB "github.com/pzqf/zCommon/db"
	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GlobalServer/config"
	"go.uber.org/zap"
)

type DBService struct {
	mu        sync.RWMutex
	config    *sharedDB.DBConfig
	connector connector.DBConnector
	isRunning bool
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewDBService(cfg *config.Config) *DBService {
	ctx, cancel := context.WithCancel(context.Background())
	return &DBService{
		config: &cfg.Database,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *DBService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("db service already running")
	}

	dbMgr := sharedDB.GetMgr()
	if dbMgr != nil {
		if conn := dbMgr.GetConnector("global"); conn != nil {
			s.connector = conn
			s.isRunning = true

			zLog.Info("Database service started (reusing DBManager connector)",
				zap.String("dbname", s.config.DBName))

			go s.monitorLoop()
			return nil
		}
	}

	zLog.Warn("DBManager connector not available, creating standalone connection")
	conn := connector.NewDBConnector("global", s.config.Driver, s.config.MaxPoolSize)

	dbConfig := connector.DBConfig{
		Host:           s.config.Host,
		Port:           s.config.Port,
		User:           s.config.User,
		Password:       s.config.Password,
		DBName:         s.config.DBName,
		Driver:         s.config.Driver,
		MaxOpen:        s.config.MaxPoolSize,
		MaxIdle:        s.config.MinPoolSize,
		ConnectTimeout: s.config.ConnectTimeout,
	}

	if err := conn.Init(dbConfig); err != nil {
		return fmt.Errorf("failed to init db connector: %v", err)
	}

	if err := conn.Start(); err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	s.connector = conn
	s.isRunning = true

	zLog.Info("Database service started",
		zap.String("driver", s.config.Driver),
		zap.String("host", s.config.Host),
		zap.Int("port", s.config.Port),
		zap.String("dbname", s.config.DBName))

	go s.monitorLoop()
	return nil
}

// Stop 停止数据库服务
func (s *DBService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	s.isRunning = false
	s.cancel()

	// 关闭数据库连�?
	if s.connector != nil {
		s.connector.Close()
		zLog.Info("Database connection closed")
	}

	zLog.Info("Database service stopped")
}

// GetConnector 获取数据库连接器
func (s *DBService) GetConnector() connector.DBConnector {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connector
}

// GetGameServers 获取所有游戏服务器
func (s *DBService) GetGameServers() ([]*models.GameServer, error) {
	s.mu.RLock()
	conn := s.connector
	s.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("database connector not initialized")
	}

	gameServerDAO := dao.NewGameServerDAO(conn)
	return gameServerDAO.GetAll()
}

// monitorLoop 监控数据库连接池
func (s *DBService) monitorLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkConnection()
		}
	}
}

// checkConnection 检查数据库连接
func (s *DBService) checkConnection() {
	s.mu.RLock()
	connector := s.connector
	s.mu.RUnlock()

	if connector != nil {
		// 这里可以添加连接检查逻辑
		zLog.Debug("Database connection check success")
	}
}
