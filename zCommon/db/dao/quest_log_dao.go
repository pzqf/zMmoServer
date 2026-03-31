package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type QuestLogDAO struct {
	connector connector.DBConnector
}

func NewQuestLogDAO(dbConnector connector.DBConnector) *QuestLogDAO {
	return &QuestLogDAO{connector: dbConnector}
}

func (dao *QuestLogDAO) CreateQuestLog(questLog *models.QuestLog, callback func(int64, error)) {
	logID, err := id.GenerateLogID()
	if err != nil {
		if callback != nil {
			callback(0, err)
		}
		return
	}
	questLog.LogID = int64(logID)

	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.QuestLog{}.TableName())
		_, err := collection.InsertOne(nil, questLog)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(questLog.LogID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (log_id, player_id, quest_id, op_type, detail, created_at) VALUES (?, ?, ?, ?, ?, ?)", models.QuestLog{}.TableName())
		args := []interface{}{
			questLog.LogID, questLog.PlayerID, questLog.QuestID,
			questLog.OpType, questLog.Detail, questLog.CreatedAt,
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

func (dao *QuestLogDAO) GetQuestLogsByPlayerID(playerID int64, limit int, callback func([]*models.QuestLog, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.QuestLog{}.TableName())
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

		var logs []*models.QuestLog
		for cursor.Next(nil) {
			var log models.QuestLog
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
		query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ? ORDER BY created_at DESC", models.QuestLog{}.TableName())
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

			var logs []*models.QuestLog
			for rows.Next() {
				var log models.QuestLog
				if err := rows.Scan(
					&log.LogID, &log.PlayerID, &log.QuestID,
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

func (dao *QuestLogDAO) GetQuestLogsByQuestID(questID int32, limit int, callback func([]*models.QuestLog, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.QuestLog{}.TableName())
		filter := bson.M{"quest_id": questID}
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

		var logs []*models.QuestLog
		for cursor.Next(nil) {
			var log models.QuestLog
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
		query := fmt.Sprintf("SELECT * FROM %s WHERE quest_id = ? ORDER BY created_at DESC", models.QuestLog{}.TableName())
		if limit > 0 {
			query = fmt.Sprintf("%s LIMIT %d", query, limit)
		}
		dao.connector.Query(query, []interface{}{questID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var logs []*models.QuestLog
			for rows.Next() {
				var log models.QuestLog
				if err := rows.Scan(
					&log.LogID, &log.PlayerID, &log.QuestID,
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

