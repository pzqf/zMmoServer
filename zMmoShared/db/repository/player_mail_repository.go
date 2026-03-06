package repository

import (
	"github.com/pzqf/zMmoShared/db/dao"
	"github.com/pzqf/zMmoShared/db/models"
)

type PlayerMailRepositoryImpl struct {
	mailDAO *dao.PlayerMailDAO
}

func NewPlayerMailRepository(mailDAO *dao.PlayerMailDAO) *PlayerMailRepositoryImpl {
	return &PlayerMailRepositoryImpl{mailDAO: mailDAO}
}

func (r *PlayerMailRepositoryImpl) GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerMail, error)) {
	r.mailDAO.GetMailsByPlayerID(playerID, callback)
}

func (r *PlayerMailRepositoryImpl) CreateAsync(mail *models.PlayerMail, callback func(int64, error)) {
	r.mailDAO.CreateMail(mail, callback)
}

func (r *PlayerMailRepositoryImpl) UpdateAsync(mail *models.PlayerMail, callback func(bool, error)) {
	r.mailDAO.UpdateMail(mail, callback)
}

func (r *PlayerMailRepositoryImpl) DeleteAsync(mailID int64, callback func(bool, error)) {
	r.mailDAO.DeleteMail(mailID, callback)
}

func (r *PlayerMailRepositoryImpl) GetByPlayerID(playerID int64) ([]*models.PlayerMail, error) {
	var result []*models.PlayerMail
	var resultErr error
	ch := make(chan struct{})
	r.GetByPlayerIDAsync(playerID, func(mails []*models.PlayerMail, err error) {
		result = mails
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerMailRepositoryImpl) Create(mail *models.PlayerMail) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(mail, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerMailRepositoryImpl) Update(mail *models.PlayerMail) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateAsync(mail, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerMailRepositoryImpl) Delete(mailID int64) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.DeleteAsync(mailID, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
