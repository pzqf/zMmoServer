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

func (r *LoginLogRepositoryImpl) Create(loginLog *models.LoginLog) (int64, error) {
	return r.logDAO.CreateLoginLog(loginLog)
}

func (r *LoginLogRepositoryImpl) GetByPlayerID(playerID int64, limit int) ([]*models.LoginLog, error) {
	return r.logDAO.GetLoginLogsByPlayerID(playerID, limit)
}

func (r *LoginLogRepositoryImpl) GetByOpType(opType int32, limit int) ([]*models.LoginLog, error) {
	return r.logDAO.GetLoginLogsByOpType(opType, limit)
}
