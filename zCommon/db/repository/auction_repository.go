package repository

import (
	"github.com/pzqf/zCommon/db/dao"
	"github.com/pzqf/zCommon/db/models"
)

type AuctionRepositoryImpl struct {
	auctionDAO *dao.AuctionDAO
}

func NewAuctionRepository(auctionDAO *dao.AuctionDAO) *AuctionRepositoryImpl {
	return &AuctionRepositoryImpl{auctionDAO: auctionDAO}
}

func (r *AuctionRepositoryImpl) GetByIDAsync(auctionID int64, callback func(*models.Auction, error)) {
	r.auctionDAO.GetAuctionByID(auctionID, callback)
}

func (r *AuctionRepositoryImpl) GetBySellerIDAsync(sellerID int64, callback func([]*models.Auction, error)) {
	r.auctionDAO.GetAuctionsBySellerID(sellerID, callback)
}

func (r *AuctionRepositoryImpl) CreateAsync(auction *models.Auction, callback func(int64, error)) {
	r.auctionDAO.CreateAuction(auction, callback)
}

func (r *AuctionRepositoryImpl) UpdateAsync(auction *models.Auction, callback func(bool, error)) {
	r.auctionDAO.UpdateAuction(auction, callback)
}

func (r *AuctionRepositoryImpl) DeleteAsync(auctionID int64, callback func(bool, error)) {
	r.auctionDAO.DeleteAuction(auctionID, callback)
}

func (r *AuctionRepositoryImpl) GetByID(auctionID int64) (*models.Auction, error) {
	var result *models.Auction
	var resultErr error
	ch := make(chan struct{})
	r.GetByIDAsync(auctionID, func(auction *models.Auction, err error) {
		result = auction
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *AuctionRepositoryImpl) GetBySellerID(sellerID int64) ([]*models.Auction, error) {
	var result []*models.Auction
	var resultErr error
	ch := make(chan struct{})
	r.GetBySellerIDAsync(sellerID, func(auctions []*models.Auction, err error) {
		result = auctions
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *AuctionRepositoryImpl) Create(auction *models.Auction) (int64, error) {
	var result int64
	var resultErr error
	ch := make(chan struct{})
	r.CreateAsync(auction, func(id int64, err error) {
		result = id
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *AuctionRepositoryImpl) Update(auction *models.Auction) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.UpdateAsync(auction, func(updated bool, err error) {
		result = updated
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

func (r *AuctionRepositoryImpl) Delete(auctionID int64) (bool, error) {
	var result bool
	var resultErr error
	ch := make(chan struct{})
	r.DeleteAsync(auctionID, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}

