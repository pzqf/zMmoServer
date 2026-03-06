package repository

import (
	"github.com/pzqf/zMmoShared/db/models"
)

// AccountRepository 账号数据仓库接口
type AccountRepository interface {
	// GetByIDAsync 根据ID异步获取账号
	GetByIDAsync(accountID int64, callback func(*models.Account, error))
	// GetByNameAsync 根据名称异步获取账号
	GetByNameAsync(accountName string, callback func(*models.Account, error))
	// CreateAsync 异步创建账号
	CreateAsync(account *models.Account, callback func(int64, error))
	// UpdateAsync 异步更新账号
	UpdateAsync(account *models.Account, callback func(bool, error))
	// DeleteAsync 异步删除账号
	DeleteAsync(accountID int64, callback func(bool, error))
	// UpdateLastLoginAtAsync 异步更新最后登录时间
	UpdateLastLoginAtAsync(accountID int64, lastLoginAt string, callback func(bool, error))

	// GetByID 根据ID获取账号
	GetByID(accountID int64) (*models.Account, error)
	// GetByName 根据名称获取账号
	GetByName(accountName string) (*models.Account, error)
	// Create 创建账号
	Create(account *models.Account) (int64, error)
	// Update 更新账号
	Update(account *models.Account) (bool, error)
	// Delete 删除账号
	Delete(accountID int64) (bool, error)
	// UpdateLastLoginAt 更新最后登录时间
	UpdateLastLoginAt(accountID int64, lastLoginAt string) (bool, error)
}

type PlayerRepository interface {
	GetByIDAsync(playerID int64, callback func(*models.Player, error))
	GetByAccountIDAsync(accountID int64, callback func([]*models.Player, error))
	CreateAsync(player *models.Player, callback func(int64, error))
	UpdateAsync(player *models.Player, callback func(bool, error))
	DeleteAsync(playerID int64, callback func(bool, error))

	GetByID(playerID int64) (*models.Player, error)
	GetByAccountID(accountID int64) ([]*models.Player, error)
	Create(player *models.Player) (int64, error)
	Update(player *models.Player) (bool, error)
	Delete(playerID int64) (bool, error)
}

type PlayerItemRepository interface {
	GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerItem, error))
	CreateAsync(item *models.PlayerItem, callback func(int64, error))
	UpdateAsync(item *models.PlayerItem, callback func(bool, error))
	DeleteAsync(itemID int64, callback func(bool, error))

	GetByPlayerID(playerID int64) ([]*models.PlayerItem, error)
	Create(item *models.PlayerItem) (int64, error)
	Update(item *models.PlayerItem) (bool, error)
	Delete(itemID int64) (bool, error)
}

type PlayerSkillRepository interface {
	GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerSkill, error))
	CreateAsync(skill *models.PlayerSkill, callback func(int64, error))
	UpdateAsync(skill *models.PlayerSkill, callback func(bool, error))
	DeleteAsync(id int64, callback func(bool, error))

	GetByPlayerID(playerID int64) ([]*models.PlayerSkill, error)
	Create(skill *models.PlayerSkill) (int64, error)
	Update(skill *models.PlayerSkill) (bool, error)
	Delete(id int64) (bool, error)
}

type PlayerMailRepository interface {
	GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerMail, error))
	CreateAsync(mail *models.PlayerMail, callback func(int64, error))
	UpdateAsync(mail *models.PlayerMail, callback func(bool, error))
	DeleteAsync(mailID int64, callback func(bool, error))

	GetByPlayerID(playerID int64) ([]*models.PlayerMail, error)
	Create(mail *models.PlayerMail) (int64, error)
	Update(mail *models.PlayerMail) (bool, error)
	Delete(mailID int64) (bool, error)
}

type PlayerQuestRepository interface {
	GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerQuest, error))
	CreateAsync(quest *models.PlayerQuest, callback func(int64, error))
	UpdateAsync(quest *models.PlayerQuest, callback func(bool, error))
	DeleteAsync(id int64, callback func(bool, error))

	GetByPlayerID(playerID int64) ([]*models.PlayerQuest, error)
	Create(quest *models.PlayerQuest) (int64, error)
	Update(quest *models.PlayerQuest) (bool, error)
	Delete(id int64) (bool, error)
}

type PlayerPetRepository interface {
	GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerPet, error))
	CreateAsync(pet *models.PlayerPet, callback func(int64, error))
	UpdateAsync(pet *models.PlayerPet, callback func(bool, error))
	DeleteAsync(petID int64, callback func(bool, error))

	GetByPlayerID(playerID int64) ([]*models.PlayerPet, error)
	Create(pet *models.PlayerPet) (int64, error)
	Update(pet *models.PlayerPet) (bool, error)
	Delete(petID int64) (bool, error)
}

type PlayerBuffRepository interface {
	GetByPlayerIDAsync(playerID int64, callback func([]*models.PlayerBuff, error))
	CreateAsync(buff *models.PlayerBuff, callback func(int64, error))
	UpdateAsync(buff *models.PlayerBuff, callback func(bool, error))
	DeleteAsync(id int64, callback func(bool, error))

	GetByPlayerID(playerID int64) ([]*models.PlayerBuff, error)
	Create(buff *models.PlayerBuff) (int64, error)
	Update(buff *models.PlayerBuff) (bool, error)
	Delete(id int64) (bool, error)
}

type GuildRepository interface {
	GetByIDAsync(guildID int64, callback func(*models.Guild, error))
	GetByNameAsync(name string, callback func(*models.Guild, error))
	CreateAsync(guild *models.Guild, callback func(int64, error))
	UpdateAsync(guild *models.Guild, callback func(bool, error))
	DeleteAsync(guildID int64, callback func(bool, error))

	GetByID(guildID int64) (*models.Guild, error)
	GetByName(name string) (*models.Guild, error)
	Create(guild *models.Guild) (int64, error)
	Update(guild *models.Guild) (bool, error)
	Delete(guildID int64) (bool, error)
}

type GuildMemberRepository interface {
	GetByGuildIDAsync(guildID int64, callback func([]*models.GuildMember, error))
	CreateAsync(member *models.GuildMember, callback func(int64, error))
	UpdateAsync(member *models.GuildMember, callback func(bool, error))
	DeleteAsync(id int64, callback func(bool, error))

	GetByGuildID(guildID int64) ([]*models.GuildMember, error)
	Create(member *models.GuildMember) (int64, error)
	Update(member *models.GuildMember) (bool, error)
	Delete(id int64) (bool, error)
}

type AuctionRepository interface {
	GetByIDAsync(auctionID int64, callback func(*models.Auction, error))
	GetBySellerIDAsync(sellerID int64, callback func([]*models.Auction, error))
	CreateAsync(auction *models.Auction, callback func(int64, error))
	UpdateAsync(auction *models.Auction, callback func(bool, error))
	DeleteAsync(auctionID int64, callback func(bool, error))

	GetByID(auctionID int64) (*models.Auction, error)
	GetBySellerID(sellerID int64) ([]*models.Auction, error)
	Create(auction *models.Auction) (int64, error)
	Update(auction *models.Auction) (bool, error)
	Delete(auctionID int64) (bool, error)
}

type LoginLogRepository interface {
	CreateAsync(loginLog *models.LoginLog, callback func(int64, error))
	GetByPlayerIDAsync(playerID int64, limit int, callback func([]*models.LoginLog, error))
	GetByOpTypeAsync(opType int32, limit int, callback func([]*models.LoginLog, error))

	Create(loginLog *models.LoginLog) (int64, error)
	GetByPlayerID(playerID int64, limit int) ([]*models.LoginLog, error)
	GetByOpType(opType int32, limit int) ([]*models.LoginLog, error)
}

type MailLogRepository interface {
	CreateAsync(mailLog *models.MailLog, callback func(int64, error))
	GetByMailIDAsync(mailID int64, limit int, callback func([]*models.MailLog, error))
	GetByPlayerIDAsync(playerID int64, limit int, callback func([]*models.MailLog, error))

	Create(mailLog *models.MailLog) (int64, error)
	GetByMailID(mailID int64, limit int) ([]*models.MailLog, error)
	GetByPlayerID(playerID int64, limit int) ([]*models.MailLog, error)
}

type QuestLogRepository interface {
	CreateAsync(questLog *models.QuestLog, callback func(int64, error))
	GetByPlayerIDAsync(playerID int64, limit int, callback func([]*models.QuestLog, error))
	GetByQuestIDAsync(questID int32, limit int, callback func([]*models.QuestLog, error))

	Create(questLog *models.QuestLog) (int64, error)
	GetByPlayerID(playerID int64, limit int) ([]*models.QuestLog, error)
	GetByQuestID(questID int32, limit int) ([]*models.QuestLog, error)
}

type AuctionLogRepository interface {
	CreateAsync(auctionLog *models.AuctionLog, callback func(int64, error))
	GetByAuctionIDAsync(auctionID int64, limit int, callback func([]*models.AuctionLog, error))
	GetByPlayerIDAsync(playerID int64, limit int, callback func([]*models.AuctionLog, error))

	Create(auctionLog *models.AuctionLog) (int64, error)
	GetByAuctionID(auctionID int64, limit int) ([]*models.AuctionLog, error)
	GetByPlayerID(playerID int64, limit int) ([]*models.AuctionLog, error)
}
