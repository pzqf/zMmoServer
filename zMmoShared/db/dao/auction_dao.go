package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type AuctionDAO struct {
	connector connector.DBConnector
}

func NewAuctionDAO(dbConnector connector.DBConnector) *AuctionDAO {
	return &AuctionDAO{connector: dbConnector}
}

func (dao *AuctionDAO) GetAuctionByID(auctionID int64, callback func(*models.Auction, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Auction{}.TableName())
		var auction models.Auction
		result := collection.FindOne(nil, bson.M{"auction_id": auctionID})
		err := result.Decode(&auction)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				if callback != nil {
					callback(nil, nil)
				}
				return
			}
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		if callback != nil {
			callback(&auction, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE auction_id = ?", models.Auction{}.TableName())
		dao.connector.Query(query, []interface{}{auctionID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
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
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				if callback != nil {
					callback(&auction, nil)
				}
			} else {
				if callback != nil {
					callback(nil, nil)
				}
			}
		})
	}
}

func (dao *AuctionDAO) GetAuctionsBySellerID(sellerID int64, callback func([]*models.Auction, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Auction{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"seller_id": sellerID})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var auctions []*models.Auction
		for cursor.Next(nil) {
			var auction models.Auction
			if err := cursor.Decode(&auction); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			auctions = append(auctions, &auction)
		}
		if callback != nil {
			callback(auctions, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE seller_id = ?", models.Auction{}.TableName())
		dao.connector.Query(query, []interface{}{sellerID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
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
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				auctions = append(auctions, &auction)
			}
			if callback != nil {
				callback(auctions, nil)
			}
		})
	}
}

func (dao *AuctionDAO) CreateAuction(auction *models.Auction, callback func(int64, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Auction{}.TableName())
		_, err := collection.InsertOne(nil, auction)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(auction.AuctionID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (auction_id, seller_id, seller_name, item_config_id, item_count, item_level, item_quality, price_type, price, buyer_id, status, end_time, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", models.Auction{}.TableName())
		args := []interface{}{
			auction.AuctionID, auction.SellerID, auction.SellerName,
			auction.ItemConfigID, auction.ItemCount, auction.ItemLevel,
			auction.ItemQuality, auction.PriceType, auction.Price,
			auction.BuyerID, auction.Status, auction.EndTime,
			auction.CreatedAt, auction.UpdatedAt,
		}
		dao.connector.Execute(query, args, func(result sql.Result, err error) {
			if err != nil {
				if callback != nil {
					callback(0, err)
				}
				return
			}
			id, err := result.LastInsertId()
			if callback != nil {
				callback(id, err)
			}
		})
	}
}

func (dao *AuctionDAO) UpdateAuction(auction *models.Auction, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Auction{}.TableName())
		update := bson.M{"$set": bson.M{
			"status": auction.Status, "buyer_id": auction.BuyerID, "updated_at": auction.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"auction_id": auction.AuctionID}, update)
		if err != nil {
			if callback != nil {
				callback(false, err)
			}
			return
		}
		if callback != nil {
			callback(result.ModifiedCount > 0, nil)
		}
	} else {
		query := fmt.Sprintf("UPDATE %s SET status = ?, buyer_id = ?, updated_at = ? WHERE auction_id = ?", models.Auction{}.TableName())
		args := []interface{}{auction.Status, auction.BuyerID, auction.UpdatedAt, auction.AuctionID}
		dao.connector.Execute(query, args, func(result sql.Result, err error) {
			if err != nil {
				if callback != nil {
					callback(false, err)
				}
				return
			}
			rowsAffected, err := result.RowsAffected()
			if callback != nil {
				callback(rowsAffected > 0, err)
			}
		})
	}
}

func (dao *AuctionDAO) DeleteAuction(auctionID int64, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Auction{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"auction_id": auctionID})
		if err != nil {
			if callback != nil {
				callback(false, err)
			}
			return
		}
		if callback != nil {
			callback(result.DeletedCount > 0, nil)
		}
	} else {
		query := fmt.Sprintf("DELETE FROM %s WHERE auction_id = ?", models.Auction{}.TableName())
		dao.connector.Execute(query, []interface{}{auctionID}, func(result sql.Result, err error) {
			if err != nil {
				if callback != nil {
					callback(false, err)
				}
				return
			}
			rowsAffected, err := result.RowsAffected()
			if callback != nil {
				callback(rowsAffected > 0, err)
			}
		})
	}
}
