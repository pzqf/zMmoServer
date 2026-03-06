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

type MailLogDAO struct {
	connector connector.DBConnector
}

func NewMailLogDAO(dbConnector connector.DBConnector) *MailLogDAO {
	return &MailLogDAO{connector: dbConnector}
}

func (dao *MailLogDAO) CreateMailLog(mailLog *models.MailLog, callback func(int64, error)) {
	logID, err := id.GenerateLogID()
	if err != nil {
		if callback != nil {
			callback(0, err)
		}
		return
	}
	mailLog.LogID = int64(logID)

	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.MailLog{}.TableName())
		_, err := collection.InsertOne(nil, mailLog)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(mailLog.LogID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (log_id, mail_id, sender_id, receiver_id, op_type, detail, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)", models.MailLog{}.TableName())
		args := []interface{}{
			mailLog.LogID, mailLog.MailID, mailLog.SenderID, mailLog.ReceiverID,
			mailLog.OpType, mailLog.Detail, mailLog.CreatedAt,
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

func (dao *MailLogDAO) GetMailLogsByMailID(mailID int64, limit int, callback func([]*models.MailLog, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.MailLog{}.TableName())
		filter := bson.M{"mail_id": mailID}
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

		var logs []*models.MailLog
		for cursor.Next(nil) {
			var log models.MailLog
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
		query := fmt.Sprintf("SELECT * FROM %s WHERE mail_id = ? ORDER BY created_at DESC", models.MailLog{}.TableName())
		if limit > 0 {
			query = fmt.Sprintf("%s LIMIT %d", query, limit)
		}
		dao.connector.Query(query, []interface{}{mailID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var logs []*models.MailLog
			for rows.Next() {
				var log models.MailLog
				if err := rows.Scan(
					&log.LogID, &log.MailID, &log.SenderID, &log.ReceiverID,
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

func (dao *MailLogDAO) GetMailLogsByPlayerID(playerID int64, limit int, callback func([]*models.MailLog, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.MailLog{}.TableName())
		filter := bson.M{"$or": []bson.M{{"sender_id": playerID}, {"receiver_id": playerID}}}
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

		var logs []*models.MailLog
		for cursor.Next(nil) {
			var log models.MailLog
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
		query := fmt.Sprintf("SELECT * FROM %s WHERE sender_id = ? OR receiver_id = ? ORDER BY created_at DESC", models.MailLog{}.TableName())
		if limit > 0 {
			query = fmt.Sprintf("%s LIMIT %d", query, limit)
		}
		dao.connector.Query(query, []interface{}{playerID, playerID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var logs []*models.MailLog
			for rows.Next() {
				var log models.MailLog
				if err := rows.Scan(
					&log.LogID, &log.MailID, &log.SenderID, &log.ReceiverID,
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
