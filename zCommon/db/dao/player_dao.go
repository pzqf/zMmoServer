package dao

import (
	"fmt"
	"time"

	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type PlayerDAO struct {
	connector connector.DBConnector
}

func NewPlayerDAO(dbConnector connector.DBConnector) *PlayerDAO {
	return &PlayerDAO{
		connector: dbConnector,
	}
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

	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", models.Player{}.TableName())
	rows, err := dao.connector.QuerySync(query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var player models.Player
	if rows.Next() {
		if err := rows.Scan(&player.PlayerID, &player.AccountID, &player.PlayerName, &player.Sex, &player.Age, &player.Level, &player.Experience, &player.CreatedAt); err != nil {
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

	query := fmt.Sprintf("SELECT * FROM %s WHERE name = ?", models.Player{}.TableName())
	rows, err := dao.connector.QuerySync(query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var player models.Player
	if rows.Next() {
		if err := rows.Scan(&player.PlayerID, &player.AccountID, &player.PlayerName, &player.Sex, &player.Age, &player.Level, &player.Experience, &player.CreatedAt); err != nil {
			return nil, err
		}
		return &player, nil
	}
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

	query := fmt.Sprintf("SELECT * FROM %s WHERE account_id = ?", models.Player{}.TableName())
	rows, err := dao.connector.QuerySync(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []*models.Player
	for rows.Next() {
		var player models.Player
		if err := rows.Scan(&player.PlayerID, &player.AccountID, &player.PlayerName, &player.Sex, &player.Age, &player.Level, &player.Experience, &player.CreatedAt); err != nil {
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

	query := fmt.Sprintf("INSERT INTO %s (id, account_id, name, gender, age, level, exp, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", models.Player{}.TableName())
	result, err := dao.connector.ExecSync(query,
		player.PlayerID, player.AccountID, player.PlayerName, player.Sex,
		player.Age, player.Level, player.Experience, player.CreatedAt)
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

	query := fmt.Sprintf("UPDATE %s SET name = ?, gender = ?, age = ?, level = ?, exp = ?, updated_at = ? WHERE id = ?", models.Player{}.TableName())
	result, err := dao.connector.ExecSync(query,
		player.PlayerName, player.Sex, player.Age, player.Level,
		player.Experience, player.UpdatedAt, player.PlayerID)
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
