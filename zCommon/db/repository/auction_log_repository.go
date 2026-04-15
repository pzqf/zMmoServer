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
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.logDAO.CreateAuctionLog(auctionLog, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *AuctionLogRepositoryImpl) GetByAuctionID(auctionID int64, limit int) ([]*models.AuctionLog, error) {
	var result []*models.AuctionLog
	var resultErr error
	ch := make(chan struct{})
	r.logDAO.GetAuctionLogsByAuctionID(auctionID, limit, func(logs []*models.AuctionLog, err error) {
		result = logs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *AuctionLogRepositoryImpl) GetByPlayerID(playerID int64, limit int) ([]*models.AuctionLog, error) {
	var result []*models.AuctionLog
	var resultErr error
	ch := make(chan struct{})
	r.logDAO.GetAuctionLogsByPlayerID(playerID, limit, func(logs []*models.AuctionLog, err error) {
		result = logs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
