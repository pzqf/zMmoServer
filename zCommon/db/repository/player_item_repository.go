package repository

import (
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
)

type PlayerItemRepositoryImpl struct {
	itemDAO *dao.PlayerItemDAO
}

func NewPlayerItemRepository(itemDAO *dao.PlayerItemDAO) *PlayerItemRepositoryImpl {
	return &PlayerItemRepositoryImpl{itemDAO: itemDAO}
}

func (r *PlayerItemRepositoryImpl) GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerItem, error)) {
	r.itemDAO.GetItemsByPlayerID(playerID, callback)
}

func (r *PlayerItemRepositoryImpl) CreateAsync(item *models.PlayerItem, callback func(int64, error)) {
	r.itemDAO.CreateItem(item, callback)
}

func (r *PlayerItemRepositoryImpl) UpdateAsync(item *models.PlayerItem, callback func(bool, error)) {
	r.itemDAO.UpdateItem(item, callback)
}

func (r *PlayerItemRepositoryImpl) DeleteAsync(itemID int64, callback func(bool, error)) {
	r.itemDAO.DeleteItem(itemID, callback)
}

func (r *PlayerItemRepositoryImpl) GetByPlayerID(playerID int64) ([]*models.PlayerItem, error) {
	var result []*models.PlayerItem
	var resultErr error
	ch := make(chan struct{})
	r.GetByPlayerIDAsync(playerID, func(items []*models.PlayerItem, err error) {
		result = items
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerItemRepositoryImpl) Create(item *models.PlayerItem) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(item, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerItemRepositoryImpl) Update(item *models.PlayerItem) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateAsync(item, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerItemRepositoryImpl) Delete(itemID int64) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.DeleteAsync(itemID, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

