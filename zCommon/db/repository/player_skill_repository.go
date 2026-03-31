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

func (r *PlayerSkillRepositoryImpl) GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerSkill, error)) {
	r.skillDAO.GetSkillsByPlayerID(playerID, callback)
}

func (r *PlayerSkillRepositoryImpl) CreateAsync(skill *models.PlayerSkill, callback func(int64, error)) {
	r.skillDAO.CreateSkill(skill, callback)
}

func (r *PlayerSkillRepositoryImpl) UpdateAsync(skill *models.PlayerSkill, callback func(bool, error)) {
	r.skillDAO.UpdateSkill(skill, callback)
}

func (r *PlayerSkillRepositoryImpl) DeleteAsync(id int64, callback func(bool, error)) {
	r.skillDAO.DeleteSkill(id, callback)
}

func (r *PlayerSkillRepositoryImpl) GetByPlayerID(playerID int64) ([]*models.PlayerSkill, error) {
	var result []*models.PlayerSkill
	var resultErr error
	ch := make(chan struct{})
	r.GetByPlayerIDAsync(playerID, func(skills []*models.PlayerSkill, err error) {
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
	r.CreateAsync(skill, func(id int64, err error) {
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
	r.UpdateAsync(skill, func(updated bool, err error) {
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
	r.DeleteAsync(id, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

