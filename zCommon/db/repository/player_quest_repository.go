package repository

import (
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
)

type PlayerQuestRepositoryImpl struct {
	questDAO *dao.PlayerQuestDAO
}

func NewPlayerQuestRepository(questDAO *dao.PlayerQuestDAO) *PlayerQuestRepositoryImpl {
	return &PlayerQuestRepositoryImpl{questDAO: questDAO}
}

func (r *PlayerQuestRepositoryImpl) GetByPlayerID(playerID int64) ([]*models.PlayerQuest, error) {
	return r.questDAO.GetQuestsByPlayerID(playerID)
}

func (r *PlayerQuestRepositoryImpl) Create(quest *models.PlayerQuest) (int64, error) {
	return r.questDAO.CreateQuest(quest)
}

func (r *PlayerQuestRepositoryImpl) Update(quest *models.PlayerQuest) (bool, error) {
	return r.questDAO.UpdateQuest(quest)
}

func (r *PlayerQuestRepositoryImpl) Delete(id int64) (bool, error) {
	return r.questDAO.DeleteQuest(id)
}
