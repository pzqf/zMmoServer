package repository

import (
	"github.com/pzqf/zMmoShared/db/dao"
	"github.com/pzqf/zMmoShared/db/models"
)

type PlayerPetRepositoryImpl struct {
	petDAO *dao.PlayerPetDAO
}

func NewPlayerPetRepository(petDAO *dao.PlayerPetDAO) *PlayerPetRepositoryImpl {
	return &PlayerPetRepositoryImpl{petDAO: petDAO}
}

func (r *PlayerPetRepositoryImpl) GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerPet, error)) {
	r.petDAO.GetPetsByPlayerID(playerID, callback)
}

func (r *PlayerPetRepositoryImpl) CreateAsync(pet *models.PlayerPet, callback func(int64, error)) {
	r.petDAO.CreatePet(pet, callback)
}

func (r *PlayerPetRepositoryImpl) UpdateAsync(pet *models.PlayerPet, callback func(bool, error)) {
	r.petDAO.UpdatePet(pet, callback)
}

func (r *PlayerPetRepositoryImpl) DeleteAsync(petID int64, callback func(bool, error)) {
	r.petDAO.DeletePet(petID, callback)
}

func (r *PlayerPetRepositoryImpl) GetByPlayerID(playerID int64) ([]*models.PlayerPet, error) {
	var result []*models.PlayerPet
	var resultErr error
	ch := make(chan struct{})
	r.GetByPlayerIDAsync(playerID, func(pets []*models.PlayerPet, err error) {
		result = pets
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerPetRepositoryImpl) Create(pet *models.PlayerPet) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(pet, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerPetRepositoryImpl) Update(pet *models.PlayerPet) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateAsync(pet, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *PlayerPetRepositoryImpl) Delete(petID int64) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.DeleteAsync(petID, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
