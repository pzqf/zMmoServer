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
	return r.logDAO.CreateQuestLog(questLog)
}

func (r *QuestLogRepositoryImpl) GetByPlayerID(playerID int64, limit int) ([]*models.QuestLog, error) {
	return r.logDAO.GetQuestLogsByPlayerID(playerID, limit)
}

func (r *QuestLogRepositoryImpl) GetByQuestID(questID int32, limit int) ([]*models.QuestLog, error) {
	return r.logDAO.GetQuestLogsByQuestID(questID, limit)
}
