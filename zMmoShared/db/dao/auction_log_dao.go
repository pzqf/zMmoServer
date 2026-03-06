package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zMmoShared/common/id"
	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AuctionLogDAO struct {
	connector connector.DBConnector
}

func NewAuctionLogDAO(dbConnector connector.DBConnector) *AuctionLogDAO {
	return &AuctionLogDAO{connector: dbConnector}
}

func (dao *AuctionLogDAO) CreateAuctionLog(auctionLog *models.AuctionLog, callback func(int64, error)) {
	logID, err := id.GenerateLogID()
	if err != nil {
		if callback != nil {
			callback(0, err)
		}
		return
	}
	auctionLog.LogID = int64(logID)

	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.AuctionLog{}.TableName())
		_, err := collection.InsertOne(nil, auctionLog)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(auctionLog.LogID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (log_id, auction_id, player_id, op_type, detail, created_at) VALUES (?, ?, ?, ?, ?, ?)", models.AuctionLog{}.TableName())
		args := []interface{}{
			auctionLog.LogID, auctionLog.AuctionID, auctionLog.PlayerID,
			auctionLog.OpType, auctionLog.Detail, auctionLog.CreatedAt,
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

func (dao *AuctionLogDAO) GetAuctionLogsByAuctionID(auctionID int64, limit int, callback func([]*models.AuctionLog, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.AuctionLog{}.TableName())
		filter := bson.M{"auction_id": auctionID}
		sort := bson.M{"created_at": -1}
		opts := &options.FindOptions{Sort: sort}
		if limit > 0 {
			opts.Limit = func(i int64) *int64 { return &i }(int64(limit))
		}

		cursor, err := collection.Find(nil, filter, opts)
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var logs []*models.AuctionLog
		for cursor.Next(nil) {
			var log models.AuctionLog
			if err := cursor.Decode(&log); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			logs = append(logs, &log)
		}
		if callback != nil {
			callback(logs, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE auction_id = ? ORDER BY created_at DESC", models.AuctionLog{}.TableName())
		if limit > 0 {
			query = fmt.Sprintf("%s LIMIT %d", query, limit)
		}
		dao.connector.Query(query, []interface{}{auctionID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var logs []*models.AuctionLog
			for rows.Next() {
				var log models.AuctionLog
				if err := rows.Scan(
					&log.LogID, &log.AuctionID, &log.PlayerID,
					&log.OpType, &log.Detail, &log.CreatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				logs = append(logs, &log)
			}
			if callback != nil {
				callback(logs, nil)
			}
		})
	}
}

func (dao *AuctionLogDAO) GetAuctionLogsByPlayerID(playerID int64, limit int, callback func([]*models.AuctionLog, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.AuctionLog{}.TableName())
		filter := bson.M{"player_id": playerID}
		sort := bson.M{"created_at": -1}
		opts := &options.FindOptions{Sort: sort}
		if limit > 0 {
			opts.Limit = func(i int64) *int64 { return &i }(int64(limit))
		}

		cursor, err := collection.Find(nil, filter, opts)
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var logs []*models.AuctionLog
		for cursor.Next(nil) {
			var log models.AuctionLog
			if err := cursor.Decode(&log); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			logs = append(logs, &log)
		}
		if callback != nil {
			callback(logs, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ? ORDER BY created_at DESC", models.AuctionLog{}.TableName())
		if limit > 0 {
			query = fmt.Sprintf("%s LIMIT %d", query, limit)
		}
		dao.connector.Query(query, []interface{}{playerID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var logs []*models.AuctionLog
			for rows.Next() {
				var log models.AuctionLog
				if err := rows.Scan(
					&log.LogID, &log.AuctionID, &log.PlayerID,
					&log.OpType, &log.Detail, &log.CreatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				logs = append(logs, &log)
			}
			if callback != nil {
				callback(logs, nil)
			}
		})
	}
}
