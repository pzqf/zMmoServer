package repository

import (
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
)

type LoginLogRepositoryImpl struct {
	logDAO *dao.LoginLogDAO
}

func NewLoginLogRepository(logDAO *dao.LoginLogDAO) *LoginLogRepositoryImpl {
	return &LoginLogRepositoryImpl{logDAO: logDAO}
}

func (r *LoginLogRepositoryImpl) CreateAsync(loginLog *models.LoginLog, callback func(int64, error)) {
	r.logDAO.CreateLoginLog(loginLog, callback)
}

func (r *LoginLogRepositoryImpl) GetByPlayerIDAsync(playerID int64, limit int, callback func([]*models.LoginLog, error)) {
	r.logDAO.GetLoginLogsByPlayerID(playerID, limit, callback)
}

func (r *LoginLogRepositoryImpl) GetByOpTypeAsync(opType int32, limit int, callback func([]*models.LoginLog, error)) {
	r.logDAO.GetLoginLogsByOpType(opType, limit, callback)
}

func (r *LoginLogRepositoryImpl) Create(loginLog *models.LoginLog) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(loginLog, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *LoginLogRepositoryImpl) GetByPlayerID(playerID int64, limit int) ([]*models.LoginLog, error) {
	var result []*models.LoginLog
	var resultErr error
	ch := make(chan struct{})
	r.GetByPlayerIDAsync(playerID, limit, func(logs []*models.LoginLog, err error) {
		result = logs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *LoginLogRepositoryImpl) GetByOpType(opType int32, limit int) ([]*models.LoginLog, error) {
	var result []*models.LoginLog
	var resultErr error
	ch := make(chan struct{})
	r.GetByOpTypeAsync(opType, limit, func(logs []*models.LoginLog, err error) {
		result = logs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

