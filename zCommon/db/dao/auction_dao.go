package dao

import (
	"fmt"

	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type AuctionDAO struct {
	connector connector.DBConnector
}

func NewAuctionDAO(dbConnector connector.DBConnector) *AuctionDAO {
	return &AuctionDAO{connector: dbConnector}
}

func (dao *AuctionDAO) GetAuctionByID(auctionID int64) (*models.Auction, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Auction{}.TableName())
		var auction models.Auction
		err := collection.FindOne(nil, bson.M{"auction_id": auctionID}).Decode(&auction)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				return nil, nil
			}
			return nil, err
		}
		return &auction, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE auction_id = ?", models.Auction{}.TableName())
	rows, err := dao.connector.QuerySync(query, auctionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var auction models.Auction
		if err := rows.Scan(
			&auction.AuctionID, &auction.SellerID, &auction.SellerName,
			&auction.ItemConfigID, &auction.ItemCount, &auction.ItemLevel,
			&auction.ItemQuality, &auction.PriceType, &auction.Price,
			&auction.BuyerID, &auction.Status, &auction.EndTime,
			&auction.CreatedAt, &auction.UpdatedAt,
		); err != nil {
			return nil, err
		}
		return &auction, nil
	}
	return nil, nil
}

func (dao *AuctionDAO) GetAuctionsBySellerID(sellerID int64) ([]*models.Auction, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Auction{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"seller_id": sellerID})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(nil)

		var auctions []*models.Auction
		for cursor.Next(nil) {
			var auction models.Auction
			if err := cursor.Decode(&auction); err != nil {
				return nil, err
			}
			auctions = append(auctions, &auction)
		}
		return auctions, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE seller_id = ?", models.Auction{}.TableName())
	rows, err := dao.connector.QuerySync(query, sellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var auctions []*models.Auction
	for rows.Next() {
		var auction models.Auction
		if err := rows.Scan(
			&auction.AuctionID, &auction.SellerID, &auction.SellerName,
			&auction.ItemConfigID, &auction.ItemCount, &auction.ItemLevel,
			&auction.ItemQuality, &auction.PriceType, &auction.Price,
			&auction.BuyerID, &auction.Status, &auction.EndTime,
			&auction.CreatedAt, &auction.UpdatedAt,
		); err != nil {
			return nil, err
		}
		auctions = append(auctions, &auction)
	}
	return auctions, nil
}

func (dao *AuctionDAO) CreateAuction(auction *models.Auction) (int64, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Auction{}.TableName())
		_, err := collection.InsertOne(nil, auction)
		if err != nil {
			return 0, err
		}
		return auction.AuctionID, nil
	}

	query := fmt.Sprintf("INSERT INTO %s (auction_id, seller_id, seller_name, item_config_id, item_count, item_level, item_quality, price_type, price, buyer_id, status, end_time, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", models.Auction{}.TableName())
	args := []interface{}{
		auction.AuctionID, auction.SellerID, auction.SellerName,
		auction.ItemConfigID, auction.ItemCount, auction.ItemLevel,
		auction.ItemQuality, auction.PriceType, auction.Price,
		auction.BuyerID, auction.Status, auction.EndTime,
		auction.CreatedAt, auction.UpdatedAt,
	}
	result, err := dao.connector.ExecSync(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (dao *AuctionDAO) UpdateAuction(auction *models.Auction) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Auction{}.TableName())
		update := bson.M{"$set": bson.M{
			"status": auction.Status, "buyer_id": auction.BuyerID, "updated_at": auction.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"auction_id": auction.AuctionID}, update)
		if err != nil {
			return false, err
		}
		return result.ModifiedCount > 0, nil
	}

	query := fmt.Sprintf("UPDATE %s SET status = ?, buyer_id = ?, updated_at = ? WHERE auction_id = ?", models.Auction{}.TableName())
	args := []interface{}{auction.Status, auction.BuyerID, auction.UpdatedAt, auction.AuctionID}
	result, err := dao.connector.ExecSync(query, args...)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (dao *AuctionDAO) DeleteAuction(auctionID int64) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Auction{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"auction_id": auctionID})
		if err != nil {
			return false, err
		}
		return result.DeletedCount > 0, nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE auction_id = ?", models.Auction{}.TableName())
	result, err := dao.connector.ExecSync(query, auctionID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}
