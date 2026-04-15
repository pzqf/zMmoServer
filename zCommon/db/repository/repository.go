package repository

import (
	"github.com/pzqf/zCommon/db/models"
)

type AccountRepository interface {
	GetByID(accountID int64) (*models.Account, error)
	GetByName(accountName string) (*models.Account, error)
	Create(account *models.Account) (int64, error)
	Update(account *models.Account) (bool, error)
	Delete(accountID int64) (bool, error)
	UpdateLastLoginAt(accountID int64, lastLoginAt string) (bool, error)
}

type PlayerRepository interface {
	GetByID(playerID int64) (*models.Player, error)
	GetByAccountID(accountID int64) ([]*models.Player, error)
	Create(player *models.Player) (int64, error)
	Update(player *models.Player) (bool, error)
	Delete(playerID int64) (bool, error)
}

type PlayerItemRepository interface {
	GetByPlayerID(playerID int64) ([]*models.PlayerItem, error)
	Create(item *models.PlayerItem) (int64, error)
	Update(item *models.PlayerItem) (bool, error)
	Delete(itemID int64) (bool, error)
}

type PlayerSkillRepository interface {
	GetByPlayerID(playerID int64) ([]*models.PlayerSkill, error)
	Create(skill *models.PlayerSkill) (int64, error)
	Update(skill *models.PlayerSkill) (bool, error)
	Delete(id int64) (bool, error)
}

type PlayerQuestRepository interface {
	GetByPlayerID(playerID int64) ([]*models.PlayerQuest, error)
	Create(quest *models.PlayerQuest) (int64, error)
	Update(quest *models.PlayerQuest) (bool, error)
	Delete(id int64) (bool, error)
}

type PlayerBuffRepository interface {
	GetByPlayerID(playerID int64) ([]*models.PlayerBuff, error)
	Create(buff *models.PlayerBuff) (int64, error)
	Update(buff *models.PlayerBuff) (bool, error)
	Delete(id int64) (bool, error)
}

type AuctionRepository interface {
	GetByID(auctionID int64) (*models.Auction, error)
	GetBySellerID(sellerID int64) ([]*models.Auction, error)
	Create(auction *models.Auction) (int64, error)
	Update(auction *models.Auction) (bool, error)
	Delete(auctionID int64) (bool, error)
}

type LoginLogRepository interface {
	Create(loginLog *models.LoginLog) (int64, error)
	GetByPlayerID(playerID int64, limit int) ([]*models.LoginLog, error)
	GetByOpType(opType int32, limit int) ([]*models.LoginLog, error)
}

type QuestLogRepository interface {
	Create(questLog *models.QuestLog) (int64, error)
	GetByPlayerID(playerID int64, limit int) ([]*models.QuestLog, error)
	GetByQuestID(questID int32, limit int) ([]*models.QuestLog, error)
}

type AuctionLogRepository interface {
	Create(auctionLog *models.AuctionLog) (int64, error)
	GetByAuctionID(auctionID int64, limit int) ([]*models.AuctionLog, error)
	GetByPlayerID(playerID int64, limit int) ([]*models.AuctionLog, error)
}

type GameServerRepository interface {
	GetByID(serverID int32) (*models.GameServer, error)
	GetAll() ([]*models.GameServer, error)
	GetByType(serverType string) ([]*models.GameServer, error)
	GetByStatus(status int32) ([]*models.GameServer, error)
	GetByGroupID(groupID int32) ([]*models.GameServer, error)
	GetByGroupIDAndType(groupID int32, serverType string) ([]*models.GameServer, error)
	Create(gameServer *models.GameServer) (int32, error)
	Update(gameServer *models.GameServer) (bool, error)
	UpdateOnlineCount(serverID int32, onlineCount int32) (bool, error)
	UpdateLastHeartbeat(serverID int32) (bool, error)
	Delete(id int32) (bool, error)
}
