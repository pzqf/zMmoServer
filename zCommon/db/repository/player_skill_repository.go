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
	return r.skillDAO.GetSkillsByPlayerID(playerID)
}

func (r *PlayerSkillRepositoryImpl) Create(skill *models.PlayerSkill) (int64, error) {
	return r.skillDAO.CreateSkill(skill)
}

func (r *PlayerSkillRepositoryImpl) Update(skill *models.PlayerSkill) (bool, error) {
	return r.skillDAO.UpdateSkill(skill)
}

func (r *PlayerSkillRepositoryImpl) Delete(id int64) (bool, error) {
	return r.skillDAO.DeleteSkill(id)
}
