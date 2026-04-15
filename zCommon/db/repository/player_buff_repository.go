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
	var result []*models.PlayerBuff
	var resultErr error
	ch := make(chan struct{})
	r.buffDAO.GetBuffsByPlayerID(playerID, func(buffs []*models.PlayerBuff, err error) {
		result = buffs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerBuffRepositoryImpl) Create(buff *models.PlayerBuff) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.buffDAO.CreateBuff(buff, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerBuffRepositoryImpl) Update(buff *models.PlayerBuff) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.buffDAO.UpdateBuff(buff, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerBuffRepositoryImpl) Delete(id int64) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.buffDAO.DeleteBuff(id, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
