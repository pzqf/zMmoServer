package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type GameServerDAO struct {
	connector connector.DBConnector
}

func NewGameServerDAO(dbConnector connector.DBConnector) *GameServerDAO {
	return &GameServerDAO{
		connector: dbConnector,
	}
}

func (dao *GameServerDAO) GetByID(serverID int32, callback func(*models.GameServer, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		var gameServer models.GameServer

		result := collection.FindOne(nil, bson.M{"server_id": serverID})
		err := result.Decode(&gameServer)

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
			callback(&gameServer, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE server_id = ?", models.GameServer{}.TableName())

		dao.connector.Query(query, []interface{}{serverID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var gameServer models.GameServer
			if rows.Next() {
				if err := rows.Scan(
					&gameServer.ServerID,
					&gameServer.ServerName,
					&gameServer.ServerType,
					&gameServer.GroupID,
					&gameServer.Address,
					&gameServer.Port,
					&gameServer.Status,
					&gameServer.OnlineCount,
					&gameServer.MaxOnlineCount,
					&gameServer.Region,
					&gameServer.Version,
					&gameServer.LastHeartbeat,
					&gameServer.CreatedAt,
					&gameServer.UpdatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}

				if callback != nil {
					callback(&gameServer, nil)
				}
			} else {
				if callback != nil {
					callback(nil, nil)
				}
			}
		})
	}
}

func (dao *GameServerDAO) GetAll(callback func([]*models.GameServer, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())

		cursor, err := collection.Find(nil, bson.M{})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var gameServers []*models.GameServer
		if err := cursor.All(nil, &gameServers); err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}

		if callback != nil {
			callback(gameServers, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s ORDER BY server_id", models.GameServer{}.TableName())

		dao.connector.Query(query, []interface{}{}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var gameServers []*models.GameServer
			for rows.Next() {
				var gameServer models.GameServer
				if err := rows.Scan(
					&gameServer.ServerID,
					&gameServer.ServerName,
					&gameServer.ServerType,
					&gameServer.GroupID,
					&gameServer.Address,
					&gameServer.Port,
					&gameServer.Status,
					&gameServer.OnlineCount,
					&gameServer.MaxOnlineCount,
					&gameServer.Region,
					&gameServer.Version,
					&gameServer.LastHeartbeat,
					&gameServer.CreatedAt,
					&gameServer.UpdatedAt,
				); err != nil {
					continue
				}
				gameServers = append(gameServers, &gameServer)
			}

			if callback != nil {
				callback(gameServers, nil)
			}
		})
	}
}

func (dao *GameServerDAO) GetByType(serverType string, callback func([]*models.GameServer, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())

		cursor, err := collection.Find(nil, bson.M{"server_type": serverType})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var gameServers []*models.GameServer
		if err := cursor.All(nil, &gameServers); err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}

		if callback != nil {
			callback(gameServers, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE server_type = ? ORDER BY server_id", models.GameServer{}.TableName())

		dao.connector.Query(query, []interface{}{serverType}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var gameServers []*models.GameServer
			for rows.Next() {
				var gameServer models.GameServer
				if err := rows.Scan(
					&gameServer.ServerID,
					&gameServer.ServerName,
					&gameServer.ServerType,
					&gameServer.GroupID,
					&gameServer.Address,
					&gameServer.Port,
					&gameServer.Status,
					&gameServer.OnlineCount,
					&gameServer.MaxOnlineCount,
					&gameServer.Region,
					&gameServer.Version,
					&gameServer.LastHeartbeat,
					&gameServer.CreatedAt,
					&gameServer.UpdatedAt,
				); err != nil {
					continue
				}
				gameServers = append(gameServers, &gameServer)
			}

			if callback != nil {
				callback(gameServers, nil)
			}
		})
	}
}

func (dao *GameServerDAO) GetByStatus(status int32, callback func([]*models.GameServer, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())

		cursor, err := collection.Find(nil, bson.M{"status": status})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var gameServers []*models.GameServer
		if err := cursor.All(nil, &gameServers); err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}

		if callback != nil {
			callback(gameServers, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE status = ? ORDER BY server_id", models.GameServer{}.TableName())

		dao.connector.Query(query, []interface{}{status}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var gameServers []*models.GameServer
			for rows.Next() {
				var gameServer models.GameServer
				if err := rows.Scan(
					&gameServer.ServerID,
					&gameServer.ServerName,
					&gameServer.ServerType,
					&gameServer.GroupID,
					&gameServer.Address,
					&gameServer.Port,
					&gameServer.Status,
					&gameServer.OnlineCount,
					&gameServer.MaxOnlineCount,
					&gameServer.Region,
					&gameServer.Version,
					&gameServer.LastHeartbeat,
					&gameServer.CreatedAt,
					&gameServer.UpdatedAt,
				); err != nil {
					continue
				}
				gameServers = append(gameServers, &gameServer)
			}

			if callback != nil {
				callback(gameServers, nil)
			}
		})
	}
}

func (dao *GameServerDAO) Create(gameServer *models.GameServer, callback func(int32, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())

		_, err := collection.InsertOne(nil, gameServer)

		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}

		if callback != nil {
			callback(gameServer.ServerID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (server_id, server_name, server_type, group_id, address, port, status, online_count, max_online_count, region, version, last_heartbeat, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", models.GameServer{}.TableName())

		args := []interface{}{
			gameServer.ServerID,
			gameServer.ServerName,
			gameServer.ServerType,
			gameServer.GroupID,
			gameServer.Address,
			gameServer.Port,
			gameServer.Status,
			gameServer.OnlineCount,
			gameServer.MaxOnlineCount,
			gameServer.Region,
			gameServer.Version,
			gameServer.LastHeartbeat,
			gameServer.CreatedAt,
			gameServer.UpdatedAt,
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
				callback(int32(id), err)
			}
		})
	}
}

func (dao *GameServerDAO) Update(gameServer *models.GameServer, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())

		update := bson.M{
			"$set": bson.M{
				"server_name":      gameServer.ServerName,
				"server_type":      gameServer.ServerType,
				"group_id":         gameServer.GroupID,
				"address":          gameServer.Address,
				"port":             gameServer.Port,
				"status":           gameServer.Status,
				"online_count":     gameServer.OnlineCount,
				"max_online_count": gameServer.MaxOnlineCount,
				"region":           gameServer.Region,
				"version":          gameServer.Version,
				"last_heartbeat":   gameServer.LastHeartbeat,
				"updated_at":       gameServer.UpdatedAt,
			},
		}

		result, err := collection.UpdateOne(nil, bson.M{"server_id": gameServer.ServerID}, update)

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
		query := fmt.Sprintf("UPDATE %s SET server_name = ?, server_type = ?, group_id = ?, address = ?, port = ?, status = ?, online_count = ?, max_online_count = ?, region = ?, version = ?, last_heartbeat = ?, updated_at = ? WHERE server_id = ?", models.GameServer{}.TableName())

		args := []interface{}{
			gameServer.ServerName,
			gameServer.ServerType,
			gameServer.GroupID,
			gameServer.Address,
			gameServer.Port,
			gameServer.Status,
			gameServer.OnlineCount,
			gameServer.MaxOnlineCount,
			gameServer.Region,
			gameServer.Version,
			gameServer.LastHeartbeat,
			gameServer.UpdatedAt,
			gameServer.ServerID,
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

func (dao *GameServerDAO) UpdateOnlineCount(serverID int32, onlineCount int32, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())

		update := bson.M{
			"$set": bson.M{
				"online_count": onlineCount,
			},
		}

		result, err := collection.UpdateOne(nil, bson.M{"server_id": serverID}, update)

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
		query := fmt.Sprintf("UPDATE %s SET online_count = ? WHERE server_id = ?", models.GameServer{}.TableName())

		dao.connector.Execute(query, []interface{}{onlineCount, serverID}, func(result sql.Result, err error) {
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

func (dao *GameServerDAO) UpdateLastHeartbeat(serverID int32, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())

		update := bson.M{
			"$set": bson.M{
				"last_heartbeat": bson.M{"$currentDate": true},
			},
		}

		result, err := collection.UpdateOne(nil, bson.M{"server_id": serverID}, update)

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
		query := fmt.Sprintf("UPDATE %s SET last_heartbeat = NOW() WHERE server_id = ?", models.GameServer{}.TableName())

		dao.connector.Execute(query, []interface{}{serverID}, func(result sql.Result, err error) {
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

func (dao *GameServerDAO) Delete(id int32, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())

		result, err := collection.DeleteOne(nil, bson.M{"server_id": id})

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
		query := fmt.Sprintf("DELETE FROM %s WHERE server_id = ?", models.GameServer{}.TableName())

		dao.connector.Execute(query, []interface{}{id}, func(result sql.Result, err error) {
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

func (dao *GameServerDAO) GetByGroupID(groupID int32, callback func([]*models.GameServer, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())

		cursor, err := collection.Find(nil, bson.M{"group_id": groupID})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var gameServers []*models.GameServer
		if err := cursor.All(nil, &gameServers); err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}

		if callback != nil {
			callback(gameServers, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE group_id = ? ORDER BY server_id", models.GameServer{}.TableName())

		dao.connector.Query(query, []interface{}{groupID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var gameServers []*models.GameServer
			for rows.Next() {
				var gameServer models.GameServer
				if err := rows.Scan(
					&gameServer.ServerID,
					&gameServer.ServerName,
					&gameServer.ServerType,
					&gameServer.GroupID,
					&gameServer.Address,
					&gameServer.Port,
					&gameServer.Status,
					&gameServer.OnlineCount,
					&gameServer.MaxOnlineCount,
					&gameServer.Region,
					&gameServer.Version,
					&gameServer.LastHeartbeat,
					&gameServer.CreatedAt,
					&gameServer.UpdatedAt,
				); err != nil {
					continue
				}
				gameServers = append(gameServers, &gameServer)
			}

			if callback != nil {
				callback(gameServers, nil)
			}
		})
	}
}

func (dao *GameServerDAO) GetByGroupIDAndType(groupID int32, serverType string, callback func([]*models.GameServer, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())

		cursor, err := collection.Find(nil, bson.M{"group_id": groupID, "server_type": serverType})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var gameServers []*models.GameServer
		if err := cursor.All(nil, &gameServers); err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}

		if callback != nil {
			callback(gameServers, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE group_id = ? AND server_type = ? ORDER BY server_id", models.GameServer{}.TableName())

		dao.connector.Query(query, []interface{}{groupID, serverType}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var gameServers []*models.GameServer
			for rows.Next() {
				var gameServer models.GameServer
				if err := rows.Scan(
					&gameServer.ServerID,
					&gameServer.ServerName,
					&gameServer.ServerType,
					&gameServer.GroupID,
					&gameServer.Address,
					&gameServer.Port,
					&gameServer.Status,
					&gameServer.OnlineCount,
					&gameServer.MaxOnlineCount,
					&gameServer.Region,
					&gameServer.Version,
					&gameServer.LastHeartbeat,
					&gameServer.CreatedAt,
					&gameServer.UpdatedAt,
				); err != nil {
					continue
				}
				gameServers = append(gameServers, &gameServer)
			}

			if callback != nil {
				callback(gameServers, nil)
			}
		})
	}
}

func (dao *GameServerDAO) GetTableName() string {
	return "game_servers"
}

