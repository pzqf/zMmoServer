package repository

import (
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
)

type PlayerBuffRepositoryImpl struct {
	buffDAO *dao.PlayerBuffDAO
}

func NewPlayerBuffRepository(buffDAO *dao.PlayerBuffDAO) *PlayerBuffRepositoryImpl {
	return &PlayerBuffRepositoryImpl{buffDAO: buffDAO}
}

func (r *PlayerBuffRepositoryImpl) GetByPlayerID(playerID int64) ([]*models.PlayerBuff, error) {
	return r.buffDAO.GetBuffsByPlayerID(playerID)
}

func (r *PlayerBuffRepositoryImpl) Create(buff *models.PlayerBuff) (int64, error) {
	return r.buffDAO.CreateBuff(buff)
}

func (r *PlayerBuffRepositoryImpl) Update(buff *models.PlayerBuff) (bool, error) {
	return r.buffDAO.UpdateBuff(buff)
}

func (r *PlayerBuffRepositoryImpl) Delete(id int64) (bool, error) {
	return r.buffDAO.DeleteBuff(id)
}
