package repository

import (
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
)

type PlayerSkillRepositoryImpl struct {
	skillDAO *dao.PlayerSkillDAO
}

func NewPlayerSkillRepository(skillDAO *dao.PlayerSkillDAO) *PlayerSkillRepositoryImpl {
	return &PlayerSkillRepositoryImpl{skillDAO: skillDAO}
}

func (r *PlayerSkillRepositoryImpl) GetByPlayerID(playerID int64) ([]*models.PlayerSkill, error) {
	var result []*models.PlayerSkill
	var resultErr error
	ch := make(chan struct{})
	r.skillDAO.GetSkillsByPlayerID(playerID, func(skills []*models.PlayerSkill, err error) {
		result = skills
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerSkillRepositoryImpl) Create(skill *models.PlayerSkill) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.skillDAO.CreateSkill(skill, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerSkillRepositoryImpl) Update(skill *models.PlayerSkill) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.skillDAO.UpdateSkill(skill, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerSkillRepositoryImpl) Delete(id int64) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.skillDAO.DeleteSkill(id, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
