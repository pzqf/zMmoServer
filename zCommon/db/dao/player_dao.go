package dao

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/models"
	"github.com/pzqf/zEngine/zLog"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

type PlayerDAO struct {
	connector connector.DBConnector
}

func NewPlayerDAO(dbConnector connector.DBConnector) *PlayerDAO {
	return &PlayerDAO{
		connector: dbConnector,
	}
}

const playerColumns = "id, account_id, name, gender, age, level, exp, gold, diamond, vip_level, last_login_at, last_logout_at, created_at, updated_at"

func scanPlayer(rows *sql.Rows, player *models.Player) error {
	var lastLoginAt, lastLogoutAt, createdAt, updatedAt sql.NullTime
	err := rows.Scan(
		&player.PlayerID, &player.AccountID, &player.PlayerName,
		&player.Sex, &player.Age, &player.Level, &player.Experience,
		&player.Gold, &player.Diamond, &player.VipLevel,
		&lastLoginAt, &lastLogoutAt,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return err
	}
	if lastLoginAt.Valid {
		player.LastLoginAt = lastLoginAt.Time
	}
	if lastLogoutAt.Valid {
		player.LastLogoutAt = lastLogoutAt.Time
	}
	if createdAt.Valid {
		player.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		player.UpdatedAt = updatedAt.Time
	}
	return nil
}

func (dao *PlayerDAO) GetPlayerByID(playerID int64) (*models.Player, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		var player models.Player
		err := collection.FindOne(nil, bson.M{"player_id": playerID}).Decode(&player)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				return nil, nil
			}
			return nil, err
		}
		return &player, nil
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", playerColumns, models.Player{}.TableName())
	rows, err := dao.connector.QuerySync(query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var player models.Player
	if rows.Next() {
		if err := scanPlayer(rows, &player); err != nil {
			return nil, err
		}
		return &player, nil
	}
	return nil, nil
}

func (dao *PlayerDAO) GetPlayerByName(name string) (*models.Player, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		var player models.Player
		err := collection.FindOne(nil, bson.M{"player_name": name}).Decode(&player)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				return nil, nil
			}
			return nil, err
		}
		return &player, nil
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE name = ?", playerColumns, models.Player{}.TableName())
	zLog.Info("GetPlayerByName query", zap.String("query", query), zap.String("name", name))
	rows, err := dao.connector.QuerySync(query, name)
	if err != nil {
		zLog.Error("GetPlayerByName query failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var player models.Player
	if rows.Next() {
		if err := scanPlayer(rows, &player); err != nil {
			zLog.Error("GetPlayerByName scan failed", zap.Error(err))
			return nil, err
		}
		zLog.Info("GetPlayerByName found player", zap.Int64("player_id", player.PlayerID), zap.String("name", player.PlayerName))
		return &player, nil
	}
	zLog.Info("GetPlayerByName no rows found", zap.String("name", name))
	return nil, nil
}

func (dao *PlayerDAO) GetPlayersByAccountID(accountID int64) ([]*models.Player, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"account_id": accountID})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(nil)

		var players []*models.Player
		for cursor.Next(nil) {
			var player models.Player
			if err := cursor.Decode(&player); err != nil {
				return nil, err
			}
			players = append(players, &player)
		}
		return players, nil
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE account_id = ?", playerColumns, models.Player{}.TableName())
	rows, err := dao.connector.QuerySync(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []*models.Player
	for rows.Next() {
		var player models.Player
		if err := scanPlayer(rows, &player); err != nil {
			return nil, err
		}
		players = append(players, &player)
	}
	return players, nil
}

func (dao *PlayerDAO) CreatePlayer(player *models.Player) (int64, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		_, err := collection.InsertOne(nil, player)
		if err != nil {
			return 0, err
		}
		return player.PlayerID, nil
	}

	query := fmt.Sprintf("INSERT INTO %s (id, account_id, name, gender, age, level, exp, gold, diamond, vip_level, last_login_at, last_logout_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", models.Player{}.TableName())
	result, err := dao.connector.ExecSync(query,
		player.PlayerID, player.AccountID, player.PlayerName, player.Sex,
		player.Age, player.Level, player.Experience,
		player.Gold, player.Diamond, player.VipLevel,
		player.LastLoginAt, player.LastLogoutAt,
		player.CreatedAt, player.UpdatedAt)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (dao *PlayerDAO) UpdatePlayer(player *models.Player) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		update := bson.M{
			"$set": bson.M{
				"player_name": player.PlayerName,
				"sex":         player.Sex,
				"age":         player.Age,
				"level":       player.Level,
				"updated_at":  player.UpdatedAt,
			},
		}
		result, err := collection.UpdateOne(nil, bson.M{"player_id": player.PlayerID}, update)
		if err != nil {
			return false, err
		}
		return result.ModifiedCount > 0, nil
	}

	query := fmt.Sprintf("UPDATE %s SET name = ?, gender = ?, age = ?, level = ?, exp = ?, gold = ?, diamond = ?, vip_level = ?, updated_at = ? WHERE id = ?", models.Player{}.TableName())
	result, err := dao.connector.ExecSync(query,
		player.PlayerName, player.Sex, player.Age, player.Level,
		player.Experience, player.Gold, player.Diamond, player.VipLevel,
		player.UpdatedAt, player.PlayerID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected > 0, err
}

func (dao *PlayerDAO) DeletePlayer(playerID int64) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"player_id": playerID})
		if err != nil {
			return false, err
		}
		return result.DeletedCount > 0, nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", models.Player{}.TableName())
	result, err := dao.connector.ExecSync(query, playerID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected > 0, err
}

func (dao *PlayerDAO) UpdatePlayerLastLogin(playerID int64, lastLoginAt time.Time) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		collection.UpdateOne(nil, bson.M{"player_id": playerID}, bson.M{
			"$set": bson.M{"last_login_at": lastLoginAt, "updated_at": lastLoginAt},
		})
	} else {
		query := fmt.Sprintf("UPDATE %s SET last_login_at = ?, updated_at = ? WHERE id = ?", models.Player{}.TableName())
		dao.connector.ExecSync(query, lastLoginAt, lastLoginAt, playerID)
	}
}

func (dao *PlayerDAO) UpdatePlayerLastLogout(playerID int64, lastLogoutAt time.Time) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		collection.UpdateOne(nil, bson.M{"player_id": playerID}, bson.M{
			"$set": bson.M{"last_logout_at": lastLogoutAt, "updated_at": lastLogoutAt},
		})
	} else {
		query := fmt.Sprintf("UPDATE %s SET last_logout_at = ?, updated_at = ? WHERE id = ?", models.Player{}.TableName())
		dao.connector.ExecSync(query, lastLogoutAt, lastLogoutAt, playerID)
	}
}
