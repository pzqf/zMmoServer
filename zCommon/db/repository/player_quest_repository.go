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
	var result []*models.PlayerQuest
	var resultErr error
	ch := make(chan struct{})
	r.questDAO.GetQuestsByPlayerID(playerID, func(quests []*models.PlayerQuest, err error) {
		result = quests
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerQuestRepositoryImpl) Create(quest *models.PlayerQuest) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.questDAO.CreateQuest(quest, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerQuestRepositoryImpl) Update(quest *models.PlayerQuest) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.questDAO.UpdateQuest(quest, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerQuestRepositoryImpl) Delete(id int64) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.questDAO.DeleteQuest(id, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
