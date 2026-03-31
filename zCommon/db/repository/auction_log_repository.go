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

func (r *AuctionLogRepositoryImpl) CreateAsync(auctionLog *models.AuctionLog, callback func(int64, error)) {
	r.logDAO.CreateAuctionLog(auctionLog, callback)
}

func (r *AuctionLogRepositoryImpl) GetByAuctionIDAsync(auctionID int64, limit int, callback func([]*models.AuctionLog, error)) {
	r.logDAO.GetAuctionLogsByAuctionID(auctionID, limit, callback)
}

func (r *AuctionLogRepositoryImpl) GetByPlayerIDAsync(playerID int64, limit int, callback func([]*models.AuctionLog, error)) {
	r.logDAO.GetAuctionLogsByPlayerID(playerID, limit, callback)
}

func (r *AuctionLogRepositoryImpl) Create(auctionLog *models.AuctionLog) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(auctionLog, func(id int64, err error) {
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
	r.GetByAuctionIDAsync(auctionID, limit, func(logs []*models.AuctionLog, err error) {
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
	r.GetByPlayerIDAsync(playerID, limit, func(logs []*models.AuctionLog, err error) {
		result = logs
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

