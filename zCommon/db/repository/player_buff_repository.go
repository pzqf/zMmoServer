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

func (r *PlayerBuffRepositoryImpl) GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerBuff, error)) {
	r.buffDAO.GetBuffsByPlayerID(playerID, callback)
}

func (r *PlayerBuffRepositoryImpl) CreateAsync(buff *models.PlayerBuff, callback func(int64, error)) {
	r.buffDAO.CreateBuff(buff, callback)
}

func (r *PlayerBuffRepositoryImpl) UpdateAsync(buff *models.PlayerBuff, callback func(bool, error)) {
	r.buffDAO.UpdateBuff(buff, callback)
}

func (r *PlayerBuffRepositoryImpl) DeleteAsync(id int64, callback func(bool, error)) {
	r.buffDAO.DeleteBuff(id, callback)
}

func (r *PlayerBuffRepositoryImpl) GetByPlayerID(playerID int64) ([]*models.PlayerBuff, error) {
	var result []*models.PlayerBuff
	var resultErr error
	ch := make(chan struct{})
	r.GetByPlayerIDAsync(playerID, func(buffs []*models.PlayerBuff, err error) {
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
	r.CreateAsync(buff, func(id int64, err error) {
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
	r.UpdateAsync(buff, func(updated bool, err error) {
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
	r.DeleteAsync(id, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

