package repository

import (
	"github.com/pzqf/zMmoShared/db/dao"
	"github.com/pzqf/zMmoShared/db/models"
)

type MailLogRepositoryImpl struct {
	logDAO *dao.MailLogDAO
}

func NewMailLogRepository(logDAO *dao.MailLogDAO) *MailLogRepositoryImpl {
	return &MailLogRepositoryImpl{logDAO: logDAO}
}

func (r *MailLogRepositoryImpl) CreateAsync(mailLog *models.MailLog, callback func(int64, error)) {
	r.logDAO.CreateMailLog(mailLog, callback)
}

func (r *MailLogRepositoryImpl) GetByMailIDAsync(mailID int64, limit int, callback func([]*models.MailLog, error)) {
	r.logDAO.GetMailLogsByMailID(mailID, limit, callback)
}

func (r *MailLogRepositoryImpl) GetByPlayerIDAsync(playerID int64, limit int, callback func([]*models.MailLog, error)) {
	r.logDAO.GetMailLogsByPlayerID(playerID, limit, callback)
}

func (r *MailLogRepositoryImpl) Create(mailLog *models.MailLog) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(mailLog, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *MailLogRepositoryImpl) GetByMailID(mailID int64, limit int) ([]*models.MailLog, error) {
	var result []*models.MailLog
	var resultErr error
	ch := make(chan struct{})
	r.GetByMailIDAsync(mailID, limit, func(logs []*models.MailLog, err error) {
		result = logs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *MailLogRepositoryImpl) GetByPlayerID(playerID int64, limit int) ([]*models.MailLog, error) {
	var result []*models.MailLog
	var resultErr error
	ch := make(chan struct{})
	r.GetByPlayerIDAsync(playerID, limit, func(logs []*models.MailLog, err error) {
		result = logs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
