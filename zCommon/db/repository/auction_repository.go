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

func (r *AuctionRepositoryImpl) GetByID(auctionID int64) (*models.Auction, error) {
	var result *models.Auction
	var resultErr error
	ch := make(chan struct{})
	r.auctionDAO.GetAuctionByID(auctionID, func(auction *models.Auction, err error) {
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
	r.auctionDAO.GetAuctionsBySellerID(sellerID, func(auctions []*models.Auction, err error) {
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
	r.auctionDAO.CreateAuction(auction, func(id int64, err error) {
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
	r.auctionDAO.UpdateAuction(auction, func(updated bool, err error) {
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
	r.auctionDAO.DeleteAuction(auctionID, func(deleted bool, err error) {
		result = deleted
		resultErr = err
		close(ch)
	})
	<-ch
	return result, resultErr
}
