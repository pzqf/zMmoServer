package repository

import (
	"fmt"
	"time"

	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
	"github.com/pzqf/zUtil/zCache"
)

type GameServerRepositoryImpl struct {
	gameServerDAO *dao.GameServerDAO
	cache         zCache.Cache
}

func NewGameServerRepository(gameServerDAO *dao.GameServerDAO) *GameServerRepositoryImpl {
	return &GameServerRepositoryImpl{
		gameServerDAO: gameServerDAO,
		cache:         zCache.NewLRUCache(100, 5*time.Minute),
	}
}

func (r *GameServerRepositoryImpl) GetByID(serverID int32) (*models.GameServer, error) {
	cacheKey := fmt.Sprintf("game_server:%d", serverID)
	if cached, err := r.cache.Get(cacheKey); err == nil {
		if gs, ok := cached.(*models.GameServer); ok {
			return gs, nil
		}
	}

	gs, err := r.gameServerDAO.GetByID(serverID)
	if err == nil && gs != nil {
		_ = r.cache.Set(cacheKey, gs, 5*time.Minute)
	}
	return gs, err
}

func (r *GameServerRepositoryImpl) GetAll() ([]*models.GameServer, error) {
	return r.gameServerDAO.GetAll()
}

func (r *GameServerRepositoryImpl) GetByType(serverType string) ([]*models.GameServer, error) {
	return r.gameServerDAO.GetByType(serverType)
}

func (r *GameServerRepositoryImpl) GetByStatus(status int32) ([]*models.GameServer, error) {
	return r.gameServerDAO.GetByStatus(status)
}

func (r *GameServerRepositoryImpl) GetByGroupID(groupID int32) ([]*models.GameServer, error) {
	return r.gameServerDAO.GetByGroupID(groupID)
}

func (r *GameServerRepositoryImpl) GetByGroupIDAndType(groupID int32, serverType string) ([]*models.GameServer, error) {
	return r.gameServerDAO.GetByGroupIDAndType(groupID, serverType)
}

func (r *GameServerRepositoryImpl) Create(gameServer *models.GameServer) (int32, error) {
	id, err := r.gameServerDAO.Create(gameServer)
	if err == nil && id > 0 {
		cacheKey := fmt.Sprintf("game_server:%d", id)
		_ = r.cache.Set(cacheKey, gameServer, 5*time.Minute)
	}
	return id, err
}

func (r *GameServerRepositoryImpl) Update(gameServer *models.GameServer) (bool, error) {
	updated, err := r.gameServerDAO.Update(gameServer)
	if err == nil && updated {
		cacheKey := fmt.Sprintf("game_server:%d", gameServer.ServerID)
		_ = r.cache.Set(cacheKey, gameServer, 5*time.Minute)
	}
	return updated, err
}

func (r *GameServerRepositoryImpl) UpdateOnlineCount(serverID int32, onlineCount int32) (bool, error) {
	updated, err := r.gameServerDAO.UpdateOnlineCount(serverID, onlineCount)
	if err == nil && updated {
		cacheKey := fmt.Sprintf("game_server:%d", serverID)
		_ = r.cache.Delete(cacheKey)
	}
	return updated, err
}

func (r *GameServerRepositoryImpl) UpdateLastHeartbeat(serverID int32) (bool, error) {
	updated, err := r.gameServerDAO.UpdateLastHeartbeat(serverID)
	if err == nil && updated {
		cacheKey := fmt.Sprintf("game_server:%d", serverID)
		_ = r.cache.Delete(cacheKey)
	}
	return updated, err
}

func (r *GameServerRepositoryImpl) Delete(id int32) (bool, error) {
	deleted, err := r.gameServerDAO.Delete(id)
	if err == nil && deleted {
		cacheKey := fmt.Sprintf("game_server:%d", id)
		_ = r.cache.Delete(cacheKey)
	}
	return deleted, err
}
