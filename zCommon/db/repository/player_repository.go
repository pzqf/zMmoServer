package repository

import (
	"fmt"
	"time"

	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
	"github.com/pzqf/zUtil/zCache"
)

// PlayerRepositoryImpl 玩家仓库实现
// 提供玩家数据的缓存操作，包括数据库操作
type PlayerRepositoryImpl struct {
	playerDAO *dao.PlayerDAO // 玩家DAO
	cache     zCache.Cache   // LRU缓存
}

// NewPlayerRepository 创建玩家仓库
// 参数:
//   - playerDAO: 玩家DAO实现
//
// 返回: 玩家仓库实现
func NewPlayerRepository(playerDAO *dao.PlayerDAO) *PlayerRepositoryImpl {
	return &PlayerRepositoryImpl{
		playerDAO: playerDAO,
		cache:     zCache.NewLRUCache(1000, 5*time.Minute),
	}
}

// GetByIDAsync 异步获取玩家
// 先从缓存获取，如果缓存未命中则从数据库获取
// 参数:
//   - playerID: 玩家ID
//   - callback: 回调函数
func (r *PlayerRepositoryImpl) GetByIDAsync(playerID int64, callback func(*models.Player, error)) {
	cacheKey := fmt.Sprintf("player:%d", playerID)
	if cached, err := r.cache.Get(cacheKey); err == nil {
		if player, ok := cached.(*models.Player); ok {
			if callback != nil {
				callback(player, nil)
			}
			return
		}
	}

	r.playerDAO.GetPlayerByID(playerID, func(p *models.Player, err error) {
		if err == nil && p != nil {
			_ = r.cache.Set(cacheKey, p, 5*time.Minute)
		}

		if callback != nil {
			callback(p, err)
		}
	})
}

// GetByAccountIDAsync 异步获取账号下的玩家列表
// 参数:
//   - accountID: 账号ID
//   - callback: 回调函数
func (r *PlayerRepositoryImpl) GetByAccountIDAsync(accountID int64, callback func([]*models.Player, error)) {
	r.playerDAO.GetPlayersByAccountID(accountID, func(players []*models.Player, err error) {
		if callback != nil {
			callback(players, err)
		}
	})
}

// CreateAsync 异步创建玩家
// 创建成功后更新缓存
// 参数:
//   - player: 玩家对象
//   - callback: 回调函数，返回创建的玩家ID
func (r *PlayerRepositoryImpl) CreateAsync(player *models.Player, callback func(int64, error)) {
	r.playerDAO.CreatePlayer(player, func(id int64, err error) {
		if err == nil && id > 0 {
			cacheKey := fmt.Sprintf("player:%d", id)
			_ = r.cache.Set(cacheKey, player, 5*time.Minute)
		}

		if callback != nil {
			callback(id, err)
		}
	})
}

// UpdateAsync 异步更新玩家
// 更新成功后更新缓存
// 参数:
//   - player: 玩家对象
//   - callback: 回调函数，返回是否更新成功
func (r *PlayerRepositoryImpl) UpdateAsync(player *models.Player, callback func(bool, error)) {
	r.playerDAO.UpdatePlayer(player, func(updated bool, err error) {
		if err == nil && updated {
			cacheKey := fmt.Sprintf("player:%d", player.PlayerID)
			_ = r.cache.Set(cacheKey, player, 5*time.Minute)
		}

		if callback != nil {
			callback(updated, err)
		}
	})
}

// DeleteAsync 异步删除玩家
// 删除成功后清除缓存
// 参数:
//   - playerID: 玩家ID
//   - callback: 回调函数，返回是否删除成功
func (r *PlayerRepositoryImpl) DeleteAsync(playerID int64, callback func(bool, error)) {
	r.playerDAO.DeletePlayer(playerID, func(deleted bool, err error) {
		if err == nil && deleted {
			cacheKey := fmt.Sprintf("player:%d", playerID)
			_ = r.cache.Delete(cacheKey)
		}

		if callback != nil {
			callback(deleted, err)
		}
	})
}

// GetByID 同步获取玩家
// 参数:
//   - playerID: 玩家ID
//
// 返回: 玩家数据和错误
func (r *PlayerRepositoryImpl) GetByID(playerID int64) (*models.Player, error) {
	var result *models.Player
	var resultErr error
	ch := make(chan struct{})
	r.GetByIDAsync(playerID, func(p *models.Player, err error) {
		result = p
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// GetByAccountID 同步获取账号下的玩家列表
// 参数:
//   - accountID: 账号ID
//
// 返回: 玩家列表和错误
func (r *PlayerRepositoryImpl) GetByAccountID(accountID int64) ([]*models.Player, error) {
	var result []*models.Player
	var resultErr error
	ch := make(chan struct{})
	r.GetByAccountIDAsync(accountID, func(players []*models.Player, err error) {
		result = players
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// Create 同步创建玩家
// 参数:
//   - player: 玩家对象
//
// 返回: 创建的玩家ID和错误
func (r *PlayerRepositoryImpl) Create(player *models.Player) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(player, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// Update 同步更新玩家
// 参数:
//   - player: 玩家对象
//
// 返回: 是否更新成功和错误
func (r *PlayerRepositoryImpl) Update(player *models.Player) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateAsync(player, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

// Delete 同步删除玩家
// 参数:
//   - playerID: 玩家ID
//
// 返回: 是否删除成功和错误
func (r *PlayerRepositoryImpl) Delete(playerID int64) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.DeleteAsync(playerID, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

