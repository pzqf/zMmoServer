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

func (r *PlayerItemRepositoryImpl) GetByPlayerID(playerID int64) ([]*models.PlayerItem, error) {
	return r.itemDAO.GetItemsByPlayerID(playerID)
}

func (r *PlayerItemRepositoryImpl) Create(item *models.PlayerItem) (int64, error) {
	return r.itemDAO.CreateItem(item)
}

func (r *PlayerItemRepositoryImpl) Update(item *models.PlayerItem) (bool, error) {
	return r.itemDAO.UpdateItem(item)
}

func (r *PlayerItemRepositoryImpl) Delete(itemID int64) (bool, error) {
	return r.itemDAO.DeleteItem(itemID)
}
