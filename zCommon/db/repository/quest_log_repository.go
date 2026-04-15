package repository

import (
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
)

type QuestLogRepositoryImpl struct {
	logDAO *dao.QuestLogDAO
}

func NewQuestLogRepository(logDAO *dao.QuestLogDAO) *QuestLogRepositoryImpl {
	return &QuestLogRepositoryImpl{logDAO: logDAO}
}

func (r *QuestLogRepositoryImpl) Create(questLog *models.QuestLog) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.logDAO.CreateQuestLog(questLog, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *QuestLogRepositoryImpl) GetByPlayerID(playerID int64, limit int) ([]*models.QuestLog, error) {
	var result []*models.QuestLog
	var resultErr error
	ch := make(chan struct{})
	r.logDAO.GetQuestLogsByPlayerID(playerID, limit, func(logs []*models.QuestLog, err error) {
		result = logs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *QuestLogRepositoryImpl) GetByQuestID(questID int32, limit int) ([]*models.QuestLog, error) {
	var result []*models.QuestLog
	var resultErr error
	ch := make(chan struct{})
	r.logDAO.GetQuestLogsByQuestID(questID, limit, func(logs []*models.QuestLog, err error) {
		result = logs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
