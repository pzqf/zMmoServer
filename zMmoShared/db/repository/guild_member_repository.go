package repository

import (
	"github.com/pzqf/zMmoShared/db/dao"
	"github.com/pzqf/zMmoShared/db/models"
)

type GuildMemberRepositoryImpl struct {
	memberDAO *dao.GuildMemberDAO
}

func NewGuildMemberRepository(memberDAO *dao.GuildMemberDAO) *GuildMemberRepositoryImpl {
	return &GuildMemberRepositoryImpl{memberDAO: memberDAO}
}

func (r *GuildMemberRepositoryImpl) GetByGuildIDAsync(guildID int64, callback func([]*models.GuildMember, error)) {
	r.memberDAO.GetMembersByGuildID(guildID, callback)
}

func (r *GuildMemberRepositoryImpl) CreateAsync(member *models.GuildMember, callback func(int64, error)) {
	r.memberDAO.CreateMember(member, callback)
}

func (r *GuildMemberRepositoryImpl) UpdateAsync(member *models.GuildMember, callback func(bool, error)) {
	r.memberDAO.UpdateMember(member, callback)
}

func (r *GuildMemberRepositoryImpl) DeleteAsync(id int64, callback func(bool, error)) {
	r.memberDAO.DeleteMember(id, callback)
}

func (r *GuildMemberRepositoryImpl) GetByGuildID(guildID int64) ([]*models.GuildMember, error) {
	var result []*models.GuildMember
	var resultErr error
	ch := make(chan struct{})
	r.GetByGuildIDAsync(guildID, func(members []*models.GuildMember, err error) {
		result = members
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *GuildMemberRepositoryImpl) Create(member *models.GuildMember) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(member, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *GuildMemberRepositoryImpl) Update(member *models.GuildMember) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateAsync(member, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *GuildMemberRepositoryImpl) Delete(id int64) (bool, error) {
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
