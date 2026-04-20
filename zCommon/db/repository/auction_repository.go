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
	return r.auctionDAO.GetAuctionByID(auctionID)
}

func (r *AuctionRepositoryImpl) GetBySellerID(sellerID int64) ([]*models.Auction, error) {
	return r.auctionDAO.GetAuctionsBySellerID(sellerID)
}

func (r *AuctionRepositoryImpl) Create(auction *models.Auction) (int64, error) {
	return r.auctionDAO.CreateAuction(auction)
}

func (r *AuctionRepositoryImpl) Update(auction *models.Auction) (bool, error) {
	return r.auctionDAO.UpdateAuction(auction)
}

func (r *AuctionRepositoryImpl) Delete(auctionID int64) (bool, error) {
	return r.auctionDAO.DeleteAuction(auctionID)
}
