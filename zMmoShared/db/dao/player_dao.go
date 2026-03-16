package dao

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

// PlayerDAO 玩家数据访问对象
// 提供玩家数据的CRUD操作，支持MongoDB和MySQL双数据库
type PlayerDAO struct {
	connector connector.DBConnector // 数据库连接器
}

// NewPlayerDAO 创建玩家DAO
// 参数:
//   - dbConnector: 数据库连接器
//
// 返回: PlayerDAO实例
func NewPlayerDAO(dbConnector connector.DBConnector) *PlayerDAO {
	return &PlayerDAO{
		connector: dbConnector,
	}
}

// GetPlayerByID 根据ID获取玩家
// 参数:
//   - playerID: 玩家ID
//   - callback: 回调函数
func (dao *PlayerDAO) GetPlayerByID(playerID int64, callback func(*models.Player, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		var player models.Player

		result := collection.FindOne(nil, bson.M{"player_id": playerID})
		err := result.Decode(&player)

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
			callback(&player, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", models.Player{}.TableName())

		dao.connector.Query(query, []interface{}{playerID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var player models.Player
			if rows.Next() {
				if err := rows.Scan(
					&player.PlayerID,
					&player.AccountID,
					&player.PlayerName,
					&player.Sex,
					&player.Age,
					&player.Level,
					&player.Experience,
					&player.CreatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}

				if callback != nil {
					callback(&player, nil)
				}
			} else {
				if callback != nil {
					callback(nil, nil)
				}
			}
		})
	}
}

// CreatePlayer 创建玩家
// 参数:
//   - player: 玩家数据
//   - callback: 回调函数，返回创建的玩家ID
func (dao *PlayerDAO) CreatePlayer(player *models.Player, callback func(int64, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())

		_, err := collection.InsertOne(nil, player)

		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}

		if callback != nil {
			callback(player.PlayerID, nil)
		}
	} else {
		// 使用与数据库表结构匹配的字段名
		query := fmt.Sprintf("INSERT INTO %s (id, account_id, name, gender, age, level, exp, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", models.Player{}.TableName())

		args := []interface{}{
			player.PlayerID, // 使用player_id作为id
			player.AccountID,
			player.PlayerName, // 使用player_name作为name
			player.Sex,        // 使用sex作为gender
			player.Age,
			player.Level,
			player.Experience, // 使用experience作为exp
			player.CreatedAt,
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

// UpdatePlayer 更新玩家
// 参数:
//   - player: 玩家数据
//   - callback: 回调函数，返回是否更新成功
func (dao *PlayerDAO) UpdatePlayer(player *models.Player, callback func(bool, error)) {
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
			if callback != nil {
				callback(false, err)
			}
			return
		}

		if callback != nil {
			callback(result.ModifiedCount > 0, nil)
		}
	} else {
		query := fmt.Sprintf("UPDATE %s SET name = ?, gender = ?, age = ?, level = ?, exp = ?, updated_at = ? WHERE id = ?", models.Player{}.TableName())

		args := []interface{}{
			player.PlayerName,
			player.Sex,
			player.Age,
			player.Level,
			player.Experience,
			player.UpdatedAt,
			player.PlayerID,
		}

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

// DeletePlayer 删除玩家
// 参数:
//   - playerID: 玩家ID
//   - callback: 回调函数，返回是否删除成功
func (dao *PlayerDAO) DeletePlayer(playerID int64, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())

		result, err := collection.DeleteOne(nil, bson.M{"player_id": playerID})

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
		query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", models.Player{}.TableName())

		dao.connector.Execute(query, []interface{}{playerID}, func(result sql.Result, err error) {
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

// GetAllPlayers 获取所有玩家
// 参数:
//   - callback: 回调函数，返回玩家列表
func (dao *PlayerDAO) GetAllPlayers(callback func([]*models.Player, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())

		cursor, err := collection.Find(nil, bson.M{})

		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var players []*models.Player
		for cursor.Next(nil) {
			var player models.Player
			if err := cursor.Decode(&player); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			players = append(players, &player)
		}

		if callback != nil {
			callback(players, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s", models.Player{}.TableName())

		dao.connector.Query(query, nil, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var players []*models.Player
			for rows.Next() {
				var player models.Player
				if err := rows.Scan(
					&player.PlayerID,
					&player.AccountID,
					&player.PlayerName,
					&player.Sex,
					&player.Age,
					&player.Level,
					&player.Experience,
					&player.CreatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}

				players = append(players, &player)
			}

			if callback != nil {
				callback(players, nil)
			}
		})
	}
}

// GetPlayersByAccountID 根据账号ID获取玩家列表
// 参数:
//   - accountID: 账号ID
//   - callback: 回调函数，返回玩家列表
func (dao *PlayerDAO) GetPlayersByAccountID(accountID int64, callback func([]*models.Player, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())

		cursor, err := collection.Find(nil, bson.M{"account_id": accountID})

		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var players []*models.Player
		for cursor.Next(nil) {
			var player models.Player
			if err := cursor.Decode(&player); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			players = append(players, &player)
		}

		if callback != nil {
			callback(players, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE account_id = ?", models.Player{}.TableName())

		dao.connector.Query(query, []interface{}{accountID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}

			var players []*models.Player
			for rows.Next() {
				var player models.Player
				if err := rows.Scan(
					&player.PlayerID,
					&player.AccountID,
					&player.PlayerName,
					&player.Sex,
					&player.Age,
					&player.Level,
					&player.Experience,
					&player.CreatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}

				players = append(players, &player)
			}
			rows.Close()

			if callback != nil {
				callback(players, nil)
			}
		})
	}
}

// GetPlayerByName 根据名称获取玩家
// 参数:
//   - name: 玩家名称
//   - callback: 回调函数，返回玩家数据
func (dao *PlayerDAO) GetPlayerByName(name string, callback func(*models.Player, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		var player models.Player

		result := collection.FindOne(nil, bson.M{"player_name": name})
		err := result.Decode(&player)

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
			callback(&player, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE name = ?", models.Player{}.TableName())

		dao.connector.Query(query, []interface{}{name}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var player models.Player
			if rows.Next() {
				if err := rows.Scan(
					&player.PlayerID,
					&player.AccountID,
					&player.PlayerName,
					&player.Sex,
					&player.Age,
					&player.Level,
					&player.Experience,
					&player.CreatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}

				if callback != nil {
					callback(&player, nil)
				}
			} else {
				if callback != nil {
					callback(nil, nil)
				}
			}
		})
	}
}

// UpdatePlayerLastLogin 更新玩家最后登录时间
// 参数:
//   - playerID: 玩家ID
//   - lastLoginAt: 最后登录时间
func (dao *PlayerDAO) UpdatePlayerLastLogin(playerID int64, lastLoginAt time.Time) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		update := bson.M{
			"$set": bson.M{
				"last_login_at": lastLoginAt,
				"updated_at":    lastLoginAt,
			},
		}
		collection.UpdateOne(nil, bson.M{"player_id": playerID}, update)
	} else {
		query := fmt.Sprintf("UPDATE %s SET last_login_at = ?, updated_at = ? WHERE id = ?", models.Player{}.TableName())
		dao.connector.Execute(query, []interface{}{lastLoginAt, lastLoginAt, playerID}, nil)
	}
}

// UpdatePlayerLastLogout 更新玩家最后登出时间
// 参数:
//   - playerID: 玩家ID
//   - lastLogoutAt: 最后登出时间
func (dao *PlayerDAO) UpdatePlayerLastLogout(playerID int64, lastLogoutAt time.Time) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Player{}.TableName())
		update := bson.M{
			"$set": bson.M{
				"last_logout_at": lastLogoutAt,
				"updated_at":     lastLogoutAt,
			},
		}
		collection.UpdateOne(nil, bson.M{"player_id": playerID}, update)
	} else {
		query := fmt.Sprintf("UPDATE %s SET last_logout_at = ?, updated_at = ? WHERE id = ?", models.Player{}.TableName())
		dao.connector.Execute(query, []interface{}{lastLogoutAt, lastLogoutAt, playerID}, nil)
	}
}
