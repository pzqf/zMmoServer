package repository

import (
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
)

type AuctionLogRepositoryImpl struct {
	logDAO *dao.AuctionLogDAO
}

func NewAuctionLogRepository(logDAO *dao.AuctionLogDAO) *AuctionLogRepositoryImpl {
	return &AuctionLogRepositoryImpl{logDAO: logDAO}
}

func (r *AuctionLogRepositoryImpl) Create(auctionLog *models.AuctionLog) (int64, error) {
	return r.logDAO.CreateAuctionLog(auctionLog)
}

func (r *AuctionLogRepositoryImpl) GetByAuctionID(auctionID int64, limit int) ([]*models.AuctionLog, error) {
	return r.logDAO.GetAuctionLogsByAuctionID(auctionID, limit)
}

func (r *AuctionLogRepositoryImpl) GetByPlayerID(playerID int64, limit int) ([]*models.AuctionLog, error) {
	return r.logDAO.GetAuctionLogsByPlayerID(playerID, limit)
}
