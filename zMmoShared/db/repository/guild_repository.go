package repository

import (
	"github.com/pzqf/zMmoShared/db/dao"
	"github.com/pzqf/zMmoShared/db/models"
)

type GuildRepositoryImpl struct {
	guildDAO *dao.GuildDAO
}

func NewGuildRepository(guildDAO *dao.GuildDAO) *GuildRepositoryImpl {
	return &GuildRepositoryImpl{guildDAO: guildDAO}
}

func (r *GuildRepositoryImpl) GetByIDAsync(guildID int64, callback func(*models.Guild, error)) {
	r.guildDAO.GetGuildByID(guildID, callback)
}

func (r *GuildRepositoryImpl) GetByNameAsync(name string, callback func(*models.Guild, error)) {
	r.guildDAO.GetGuildByName(name, callback)
}

func (r *GuildRepositoryImpl) CreateAsync(guild *models.Guild, callback func(int64, error)) {
	r.guildDAO.CreateGuild(guild, callback)
}

func (r *GuildRepositoryImpl) UpdateAsync(guild *models.Guild, callback func(bool, error)) {
	r.guildDAO.UpdateGuild(guild, callback)
}

func (r *GuildRepositoryImpl) DeleteAsync(guildID int64, callback func(bool, error)) {
	r.guildDAO.DeleteGuild(guildID, callback)
}

func (r *GuildRepositoryImpl) GetByID(guildID int64) (*models.Guild, error) {
	var result *models.Guild
	var resultErr error
	ch := make(chan struct{})
	r.GetByIDAsync(guildID, func(guild *models.Guild, err error) {
		result = guild
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *GuildRepositoryImpl) GetByName(name string) (*models.Guild, error) {
	var result *models.Guild
	var resultErr error
	ch := make(chan struct{})
	r.GetByNameAsync(name, func(guild *models.Guild, err error) {
		result = guild
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *GuildRepositoryImpl) Create(guild *models.Guild) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(guild, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *GuildRepositoryImpl) Update(guild *models.Guild) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateAsync(guild, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *GuildRepositoryImpl) Delete(guildID int64) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.DeleteAsync(guildID, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
