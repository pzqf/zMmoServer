package dao

import (
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
	return &LoginLogDAO{connector: dbConnector}
}

func (dao *LoginLogDAO) GetLoginLogByPlayerID(playerID int64) (*models.LoginLog, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.LoginLog{}.TableName())
		var loginLog models.LoginLog
		err := collection.FindOne(nil, bson.M{"player_id": playerID}).Decode(&loginLog)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				return nil, nil
			}
			return nil, err
		}
		return &loginLog, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.LoginLog{}.TableName())
	rows, err := dao.connector.QuerySync(query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var loginLog models.LoginLog
		if err := rows.Scan(
			&loginLog.LogID, &loginLog.PlayerID, &loginLog.PlayerName,
			&loginLog.OpType, &loginLog.IP, &loginLog.Device, &loginLog.CreatedAt,
		); err != nil {
			return nil, err
		}
		return &loginLog, nil
	}
	return nil, nil
}

func (dao *LoginLogDAO) CreateLoginLog(loginLog *models.LoginLog) (int64, error) {
	logID, err := id.GenerateLogID()
	if err != nil {
		return 0, err
	}
	loginLog.LogID = int64(logID)

	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.LoginLog{}.TableName())
		_, err := collection.InsertOne(nil, loginLog)
		if err != nil {
			return 0, err
		}
		return loginLog.LogID, nil
	}

	query := fmt.Sprintf("INSERT INTO %s (log_id, player_id, player_name, op_type, ip, device, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)", models.LoginLog{}.TableName())
	args := []interface{}{
		loginLog.LogID, loginLog.PlayerID, loginLog.PlayerName,
		loginLog.OpType, loginLog.IP, loginLog.Device, loginLog.CreatedAt,
	}
	result, err := dao.connector.ExecSync(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (dao *LoginLogDAO) GetLoginLogsByPlayerID(playerID int64, limit int) ([]*models.LoginLog, error) {
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
			return nil, err
		}
		defer cursor.Close(nil)

		var loginLogs []*models.LoginLog
		for cursor.Next(nil) {
			var loginLog models.LoginLog
			if err := cursor.Decode(&loginLog); err != nil {
				return nil, err
			}
			loginLogs = append(loginLogs, &loginLog)
		}
		return loginLogs, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ? ORDER BY created_at DESC", models.LoginLog{}.TableName())
	if limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}
	rows, err := dao.connector.QuerySync(query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var loginLogs []*models.LoginLog
	for rows.Next() {
		var loginLog models.LoginLog
		if err := rows.Scan(
			&loginLog.LogID, &loginLog.PlayerID, &loginLog.PlayerName,
			&loginLog.OpType, &loginLog.IP, &loginLog.Device, &loginLog.CreatedAt,
		); err != nil {
			return nil, err
		}
		loginLogs = append(loginLogs, &loginLog)
	}
	return loginLogs, nil
}

func (dao *LoginLogDAO) GetLoginLogsByOpType(opType int32, limit int) ([]*models.LoginLog, error) {
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
			return nil, err
		}
		defer cursor.Close(nil)

		var loginLogs []*models.LoginLog
		for cursor.Next(nil) {
			var loginLog models.LoginLog
			if err := cursor.Decode(&loginLog); err != nil {
				return nil, err
			}
			loginLogs = append(loginLogs, &loginLog)
		}
		return loginLogs, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE op_type = ? ORDER BY created_at DESC", models.LoginLog{}.TableName())
	if limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}
	rows, err := dao.connector.QuerySync(query, opType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var loginLogs []*models.LoginLog
	for rows.Next() {
		var loginLog models.LoginLog
		if err := rows.Scan(
			&loginLog.LogID, &loginLog.PlayerID, &loginLog.PlayerName,
			&loginLog.OpType, &loginLog.IP, &loginLog.Device, &loginLog.CreatedAt,
		); err != nil {
			return nil, err
		}
		loginLogs = append(loginLogs, &loginLog)
	}
	return loginLogs, nil
}
