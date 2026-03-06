package repository

import (
	"github.com/pzqf/zMmoShared/db/dao"
	"github.com/pzqf/zMmoShared/db/models"
)

type PlayerQuestRepositoryImpl struct {
	questDAO *dao.PlayerQuestDAO
}

func NewPlayerQuestRepository(questDAO *dao.PlayerQuestDAO) *PlayerQuestRepositoryImpl {
	return &PlayerQuestRepositoryImpl{questDAO: questDAO}
}

func (r *PlayerQuestRepositoryImpl) GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerQuest, error)) {
	r.questDAO.GetQuestsByPlayerID(playerID, callback)
}

func (r *PlayerQuestRepositoryImpl) CreateAsync(quest *models.PlayerQuest, callback func(int64, error)) {
	r.questDAO.CreateQuest(quest, callback)
}

func (r *PlayerQuestRepositoryImpl) UpdateAsync(quest *models.PlayerQuest, callback func(bool, error)) {
	r.questDAO.UpdateQuest(quest, callback)
}

func (r *PlayerQuestRepositoryImpl) DeleteAsync(id int64, callback func(bool, error)) {
	r.questDAO.DeleteQuest(id, callback)
}

func (r *PlayerQuestRepositoryImpl) GetByPlayerID(playerID int64) ([]*models.PlayerQuest, error) {
	var result []*models.PlayerQuest
	var resultErr error
	ch := make(chan struct{})
	r.GetByPlayerIDAsync(playerID, func(quests []*models.PlayerQuest, err error) {
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
	r.CreateAsync(quest, func(id int64, err error) {
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
	r.UpdateAsync(quest, func(updated bool, err error) {
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
	r.DeleteAsync(id, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
