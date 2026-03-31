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

func (r *QuestLogRepositoryImpl) CreateAsync(questLog *models.QuestLog, callback func(int64, error)) {
	r.logDAO.CreateQuestLog(questLog, callback)
}

func (r *QuestLogRepositoryImpl) GetByPlayerIDAsync(playerID int64, limit int, callback func([]*models.QuestLog, error)) {
	r.logDAO.GetQuestLogsByPlayerID(playerID, limit, callback)
}

func (r *QuestLogRepositoryImpl) GetByQuestIDAsync(questID int32, limit int, callback func([]*models.QuestLog, error)) {
	r.logDAO.GetQuestLogsByQuestID(questID, limit, callback)
}

func (r *QuestLogRepositoryImpl) Create(questLog *models.QuestLog) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(questLog, func(id int64, err error) {
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
	r.GetByPlayerIDAsync(playerID, limit, func(logs []*models.QuestLog, err error) {
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
	r.GetByQuestIDAsync(questID, limit, func(logs []*models.QuestLog, err error) {
		result = logs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

