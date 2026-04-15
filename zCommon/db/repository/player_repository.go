package repository

import (
	"fmt"
	"time"

	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
	"github.com/pzqf/zUtil/zCache"
)

type PlayerRepositoryImpl struct {
	playerDAO *dao.PlayerDAO
	cache     zCache.Cache
}

func NewPlayerRepository(playerDAO *dao.PlayerDAO) *PlayerRepositoryImpl {
	return &PlayerRepositoryImpl{
		playerDAO: playerDAO,
		cache:     zCache.NewLRUCache(1000, 5*time.Minute),
	}
}

func (r *PlayerRepositoryImpl) GetByID(playerID int64) (*models.Player, error) {
	cacheKey := fmt.Sprintf("player:%d", playerID)
	if cached, err := r.cache.Get(cacheKey); err == nil {
		if player, ok := cached.(*models.Player); ok {
			return player, nil
		}
	}

	player, err := r.playerDAO.GetPlayerByID(playerID)
	if err == nil && player != nil {
		_ = r.cache.Set(cacheKey, player, 5*time.Minute)
	}
	return player, err
}

func (r *PlayerRepositoryImpl) GetByAccountID(accountID int64) ([]*models.Player, error) {
	return r.playerDAO.GetPlayersByAccountID(accountID)
}

func (r *PlayerRepositoryImpl) Create(player *models.Player) (int64, error) {
	id, err := r.playerDAO.CreatePlayer(player)
	if err == nil && id > 0 {
		cacheKey := fmt.Sprintf("player:%d", id)
		_ = r.cache.Set(cacheKey, player, 5*time.Minute)
	}
	return id, err
}

func (r *PlayerRepositoryImpl) Update(player *models.Player) (bool, error) {
	updated, err := r.playerDAO.UpdatePlayer(player)
	if err == nil && updated {
		cacheKey := fmt.Sprintf("player:%d", player.PlayerID)
		_ = r.cache.Set(cacheKey, player, 5*time.Minute)
	}
	return updated, err
}

func (r *PlayerRepositoryImpl) Delete(playerID int64) (bool, error) {
	deleted, err := r.playerDAO.DeletePlayer(playerID)
	if err == nil && deleted {
		cacheKey := fmt.Sprintf("player:%d", playerID)
		_ = r.cache.Delete(cacheKey)
	}
	return deleted, err
}
