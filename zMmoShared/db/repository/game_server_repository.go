package repository

import (
	"fmt"
	"time"

	"github.com/pzqf/zMmoShared/db/dao"
	"github.com/pzqf/zMmoShared/db/models"
	"github.com/pzqf/zUtil/zCache"
)

// GameServerRepository 游戏服务器数据仓库接口
type GameServerRepository interface {
	GetByIDAsync(serverID int32, callback func(*models.GameServer, error))
	GetAllAsync(callback func([]*models.GameServer, error))
	GetByTypeAsync(serverType string, callback func([]*models.GameServer, error))
	GetByStatusAsync(status int32, callback func([]*models.GameServer, error))
	GetByGroupIDAsync(groupID int32, callback func([]*models.GameServer, error))
	GetByGroupIDAndTypeAsync(groupID int32, serverType string, callback func([]*models.GameServer, error))
	CreateAsync(gameServer *models.GameServer, callback func(int32, error))
	UpdateAsync(gameServer *models.GameServer, callback func(bool, error))
	UpdateOnlineCountAsync(serverID int32, onlineCount int32, callback func(bool, error))
	UpdateLastHeartbeatAsync(serverID int32, callback func(bool, error))
	DeleteAsync(id int32, callback func(bool, error))

	GetByID(serverID int32) (*models.GameServer, error)
	GetAll() ([]*models.GameServer, error)
	GetByType(serverType string) ([]*models.GameServer, error)
	GetByStatus(status int32) ([]*models.GameServer, error)
	GetByGroupID(groupID int32) ([]*models.GameServer, error)
	GetByGroupIDAndType(groupID int32, serverType string) ([]*models.GameServer, error)
	Create(gameServer *models.GameServer) (int32, error)
	Update(gameServer *models.GameServer) (bool, error)
	UpdateOnlineCount(serverID int32, onlineCount int32) (bool, error)
	UpdateLastHeartbeat(serverID int32) (bool, error)
	Delete(id int32) (bool, error)
}

// GameServerRepositoryImpl 游戏服务器数据仓库实现
type GameServerRepositoryImpl struct {
	gameServerDAO *dao.GameServerDAO
	cache         zCache.Cache
}

// NewGameServerRepository 创建游戏服务器数据仓库实例
func NewGameServerRepository(gameServerDAO *dao.GameServerDAO) *GameServerRepositoryImpl {
	return &GameServerRepositoryImpl{
		gameServerDAO: gameServerDAO,
		cache:         zCache.NewLRUCache(100, 5*time.Minute), // 100容量，5分钟过期
	}
}

// GetByIDAsync 根据ID异步获取游戏服务器
func (r *GameServerRepositoryImpl) GetByIDAsync(serverID int32, callback func(*models.GameServer, error)) {
	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("game_server:%d", serverID)
	if cached, err := r.cache.Get(cacheKey); err == nil {
		if gameServer, ok := cached.(*models.GameServer); ok {
			if callback != nil {
				callback(gameServer, nil)
			}
			return
		}
	}

	// 缓存未命中，从数据库获取
	r.gameServerDAO.GetByID(serverID, func(gs *models.GameServer, err error) {
		if err == nil && gs != nil {
			_ = r.cache.Set(cacheKey, gs, 5*time.Minute)
		}

		if callback != nil {
			callback(gs, err)
		}
	})
}

// GetAllAsync 异步获取所有游戏服务器
func (r *GameServerRepositoryImpl) GetAllAsync(callback func([]*models.GameServer, error)) {
	r.gameServerDAO.GetAll(func(gsList []*models.GameServer, err error) {
		if callback != nil {
			callback(gsList, err)
		}
	})
}

// GetByTypeAsync 根据类型异步获取游戏服务器
func (r *GameServerRepositoryImpl) GetByTypeAsync(serverType string, callback func([]*models.GameServer, error)) {
	r.gameServerDAO.GetByType(serverType, func(gsList []*models.GameServer, err error) {
		if callback != nil {
			callback(gsList, err)
		}
	})
}

// GetByStatusAsync 根据状态异步获取游戏服务器
func (r *GameServerRepositoryImpl) GetByStatusAsync(status int32, callback func([]*models.GameServer, error)) {
	r.gameServerDAO.GetByStatus(status, func(gsList []*models.GameServer, err error) {
		if callback != nil {
			callback(gsList, err)
		}
	})
}

// CreateAsync 异步创建游戏服务器
func (r *GameServerRepositoryImpl) CreateAsync(gameServer *models.GameServer, callback func(int32, error)) {
	r.gameServerDAO.Create(gameServer, func(serverID int32, err error) {
		if err == nil && serverID > 0 {
			cacheKey := fmt.Sprintf("game_server:%d", serverID)
			_ = r.cache.Set(cacheKey, gameServer, 5*time.Minute)
		}

		if callback != nil {
			callback(serverID, err)
		}
	})
}

// UpdateAsync 异步更新游戏服务器
func (r *GameServerRepositoryImpl) UpdateAsync(gameServer *models.GameServer, callback func(bool, error)) {
	r.gameServerDAO.Update(gameServer, func(updated bool, err error) {
		if err == nil && updated {
			cacheKey := fmt.Sprintf("game_server:%d", gameServer.ServerID)
			_ = r.cache.Set(cacheKey, gameServer, 5*time.Minute)
		}

		if callback != nil {
			callback(updated, err)
		}
	})
}

// UpdateOnlineCountAsync 异步更新游戏服务器在线人数
func (r *GameServerRepositoryImpl) UpdateOnlineCountAsync(serverID int32, onlineCount int32, callback func(bool, error)) {
	r.gameServerDAO.UpdateOnlineCount(serverID, onlineCount, func(updated bool, err error) {
		if err == nil && updated {
			cacheKey := fmt.Sprintf("game_server:%d", serverID)
			_ = r.cache.Delete(cacheKey)
		}

		if callback != nil {
			callback(updated, err)
		}
	})
}

// UpdateLastHeartbeatAsync 异步更新游戏服务器最后心跳时间
func (r *GameServerRepositoryImpl) UpdateLastHeartbeatAsync(serverID int32, callback func(bool, error)) {
	r.gameServerDAO.UpdateLastHeartbeat(serverID, func(updated bool, err error) {
		if err == nil && updated {
			cacheKey := fmt.Sprintf("game_server:%d", serverID)
			_ = r.cache.Delete(cacheKey)
		}

		if callback != nil {
			callback(updated, err)
		}
	})
}

// DeleteAsync 异步删除游戏服务器
func (r *GameServerRepositoryImpl) DeleteAsync(id int32, callback func(bool, error)) {
	r.gameServerDAO.Delete(id, func(deleted bool, err error) {
		if err == nil && deleted {
			cacheKey := fmt.Sprintf("game_server:%d", id)
			_ = r.cache.Delete(cacheKey)
		}

		if callback != nil {
			callback(deleted, err)
		}
	})
}

// GetByID 根据ID获取游戏服务器（同步兼容方法）
func (r *GameServerRepositoryImpl) GetByID(serverID int32) (*models.GameServer, error) {
	var result *models.GameServer
	var resultErr error
	ch := make(chan struct{})
	r.GetByIDAsync(serverID, func(gs *models.GameServer, err error) {
		result = gs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// GetAll 获取所有游戏服务器（同步兼容方法）
func (r *GameServerRepositoryImpl) GetAll() ([]*models.GameServer, error) {
	var result []*models.GameServer
	var resultErr error
	ch := make(chan struct{})
	r.GetAllAsync(func(gsList []*models.GameServer, err error) {
		result = gsList
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// GetByType 根据类型获取游戏服务器（同步兼容方法）
func (r *GameServerRepositoryImpl) GetByType(serverType string) ([]*models.GameServer, error) {
	var result []*models.GameServer
	var resultErr error
	ch := make(chan struct{})
	r.GetByTypeAsync(serverType, func(gsList []*models.GameServer, err error) {
		result = gsList
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// GetByStatus 根据状态获取游戏服务器（同步兼容方法）
func (r *GameServerRepositoryImpl) GetByStatus(status int32) ([]*models.GameServer, error) {
	var result []*models.GameServer
	var resultErr error
	ch := make(chan struct{})
	r.GetByStatusAsync(status, func(gsList []*models.GameServer, err error) {
		result = gsList
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// GetByGroupIDAsync 根据分组ID异步获取游戏服务器
func (r *GameServerRepositoryImpl) GetByGroupIDAsync(groupID int32, callback func([]*models.GameServer, error)) {
	r.gameServerDAO.GetByGroupID(groupID, func(gsList []*models.GameServer, err error) {
		if callback != nil {
			callback(gsList, err)
		}
	})
}

// GetByGroupIDAndTypeAsync 根据分组ID和类型异步获取游戏服务器
func (r *GameServerRepositoryImpl) GetByGroupIDAndTypeAsync(groupID int32, serverType string, callback func([]*models.GameServer, error)) {
	r.gameServerDAO.GetByGroupIDAndType(groupID, serverType, func(gsList []*models.GameServer, err error) {
		if callback != nil {
			callback(gsList, err)
		}
	})
}

// GetByGroupID 根据分组ID获取游戏服务器（同步兼容方法）
func (r *GameServerRepositoryImpl) GetByGroupID(groupID int32) ([]*models.GameServer, error) {
	var result []*models.GameServer
	var resultErr error
	ch := make(chan struct{})
	r.GetByGroupIDAsync(groupID, func(gsList []*models.GameServer, err error) {
		result = gsList
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// GetByGroupIDAndType 根据分组ID和类型获取游戏服务器（同步兼容方法）
func (r *GameServerRepositoryImpl) GetByGroupIDAndType(groupID int32, serverType string) ([]*models.GameServer, error) {
	var result []*models.GameServer
	var resultErr error
	ch := make(chan struct{})
	r.GetByGroupIDAndTypeAsync(groupID, serverType, func(gsList []*models.GameServer, err error) {
		result = gsList
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// Create 创建游戏服务器（同步兼容方法）
func (r *GameServerRepositoryImpl) Create(gameServer *models.GameServer) (int32, error) {
	var result int32
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(gameServer, func(serverID int32, err error) {
		result = serverID
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// Update 更新游戏服务器（同步兼容方法）
func (r *GameServerRepositoryImpl) Update(gameServer *models.GameServer) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateAsync(gameServer, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// UpdateOnlineCount 更新游戏服务器在线人数（同步兼容方法）
func (r *GameServerRepositoryImpl) UpdateOnlineCount(serverID int32, onlineCount int32) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateOnlineCountAsync(serverID, onlineCount, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// UpdateLastHeartbeat 更新游戏服务器最后心跳时间（同步兼容方法）
func (r *GameServerRepositoryImpl) UpdateLastHeartbeat(serverID int32) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateLastHeartbeatAsync(serverID, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// Delete 删除游戏服务器（同步兼容方法）
func (r *GameServerRepositoryImpl) Delete(id int32) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.DeleteAsync(id, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
