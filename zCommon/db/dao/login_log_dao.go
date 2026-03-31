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

type LoginLogDAO struct {
	connector connector.DBConnector
}

func NewLoginLogDAO(dbConnector connector.DBConnector) *LoginLogDAO {
	return &LoginLogDAO{
		connector: dbConnector,
	}
}

func (dao *LoginLogDAO) GetLoginLogByPlayerID(playerID int64, callback func(*models.LoginLog, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.LoginLog{}.TableName())
		var loginLog models.LoginLog

		result := collection.FindOne(nil, bson.M{"player_id": playerID})
		err := result.Decode(&loginLog)

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
			callback(&loginLog, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.LoginLog{}.TableName())

		dao.connector.Query(query, []interface{}{playerID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var loginLog models.LoginLog
			if rows.Next() {
				if err := rows.Scan(
					&loginLog.LogID,
					&loginLog.PlayerID,
					&loginLog.PlayerName,
					&loginLog.OpType,
					&loginLog.IP,
					&loginLog.Device,
					&loginLog.CreatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}

				if callback != nil {
					callback(&loginLog, nil)
				}
			} else {
				if callback != nil {
					callback(nil, nil)
				}
			}
		})
	}
}

func (dao *LoginLogDAO) CreateLoginLog(loginLog *models.LoginLog, callback func(int64, error)) {
	logID, err := id.GenerateLogID()
	if err != nil {
		if callback != nil {
			callback(0, err)
		}
		return
	}
	loginLog.LogID = int64(logID)

	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.LoginLog{}.TableName())

		_, err := collection.InsertOne(nil, loginLog)

		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}

		if callback != nil {
			callback(loginLog.LogID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (log_id, player_id, player_name, op_type, ip, device, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)", models.LoginLog{}.TableName())

		args := []interface{}{
			loginLog.LogID,
			loginLog.PlayerID,
			loginLog.PlayerName,
			loginLog.OpType,
			loginLog.IP,
			loginLog.Device,
			loginLog.CreatedAt,
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

func (dao *LoginLogDAO) GetLoginLogsByPlayerID(playerID int64, limit int, callback func([]*models.LoginLog, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.LoginLog{}.TableName())

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

		var loginLogs []*models.LoginLog
		for cursor.Next(nil) {
			var loginLog models.LoginLog
			if err := cursor.Decode(&loginLog); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			loginLogs = append(loginLogs, &loginLog)
		}

		if callback != nil {
			callback(loginLogs, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ? ORDER BY created_at DESC", models.LoginLog{}.TableName())
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

			var loginLogs []*models.LoginLog
			for rows.Next() {
				var loginLog models.LoginLog
				if err := rows.Scan(
					&loginLog.LogID,
					&loginLog.PlayerID,
					&loginLog.PlayerName,
					&loginLog.OpType,
					&loginLog.IP,
					&loginLog.Device,
					&loginLog.CreatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}

				loginLogs = append(loginLogs, &loginLog)
			}

			if callback != nil {
				callback(loginLogs, nil)
			}
		})
	}
}

func (dao *LoginLogDAO) GetLoginLogsByOpType(opType int32, limit int, callback func([]*models.LoginLog, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.LoginLog{}.TableName())

		filter := bson.M{"op_type": opType}
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

		var loginLogs []*models.LoginLog
		for cursor.Next(nil) {
			var loginLog models.LoginLog
			if err := cursor.Decode(&loginLog); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			loginLogs = append(loginLogs, &loginLog)
		}

		if callback != nil {
			callback(loginLogs, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE op_type = ? ORDER BY created_at DESC", models.LoginLog{}.TableName())
		if limit > 0 {
			query = fmt.Sprintf("%s LIMIT %d", query, limit)
		}

		dao.connector.Query(query, []interface{}{opType}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var loginLogs []*models.LoginLog
			for rows.Next() {
				var loginLog models.LoginLog
				if err := rows.Scan(
					&loginLog.LogID,
					&loginLog.PlayerID,
					&loginLog.PlayerName,
					&loginLog.OpType,
					&loginLog.IP,
					&loginLog.Device,
					&loginLog.CreatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}

				loginLogs = append(loginLogs, &loginLog)
			}

			if callback != nil {
				callback(loginLogs, nil)
			}
		})
	}
}

