package dao

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type GameServerDAO struct {
	connector connector.DBConnector
}

func NewGameServerDAO(dbConnector connector.DBConnector) *GameServerDAO {
	return &GameServerDAO{connector: dbConnector}
}

func scanGameServer(rows *sql.Rows) (*models.GameServer, error) {
	var gs models.GameServer
	err := rows.Scan(
		&gs.ServerID, &gs.ServerName, &gs.ServerType, &gs.GroupID,
		&gs.Address, &gs.Port, &gs.Status, &gs.OnlineCount,
		&gs.MaxOnlineCount, &gs.Region, &gs.Version,
		&gs.LastHeartbeat, &gs.CreatedAt, &gs.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &gs, nil
}

func scanGameServers(rows *sql.Rows) ([]*models.GameServer, error) {
	var list []*models.GameServer
	for rows.Next() {
		gs, err := scanGameServer(rows)
		if err != nil {
			continue
		}
		list = append(list, gs)
	}
	return list, nil
}

func (dao *GameServerDAO) GetByID(serverID int32) (*models.GameServer, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		var gs models.GameServer
		err := collection.FindOne(nil, bson.M{"server_id": serverID}).Decode(&gs)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				return nil, nil
			}
			return nil, err
		}
		return &gs, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE server_id = ?", models.GameServer{}.TableName())
	rows, err := dao.connector.QuerySync(query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return scanGameServer(rows)
	}
	return nil, nil
}

func (dao *GameServerDAO) GetAll() ([]*models.GameServer, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		cursor, err := collection.Find(nil, bson.M{})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(nil)
		var list []*models.GameServer
		if err := cursor.All(nil, &list); err != nil {
			return nil, err
		}
		return list, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s ORDER BY server_id", models.GameServer{}.TableName())
	rows, err := dao.connector.QuerySync(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGameServers(rows)
}

func (dao *GameServerDAO) GetByType(serverType string) ([]*models.GameServer, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"server_type": serverType})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(nil)
		var list []*models.GameServer
		if err := cursor.All(nil, &list); err != nil {
			return nil, err
		}
		return list, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE server_type = ? ORDER BY server_id", models.GameServer{}.TableName())
	rows, err := dao.connector.QuerySync(query, serverType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGameServers(rows)
}

func (dao *GameServerDAO) GetByStatus(status int32) ([]*models.GameServer, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"status": status})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(nil)
		var list []*models.GameServer
		if err := cursor.All(nil, &list); err != nil {
			return nil, err
		}
		return list, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE status = ? ORDER BY server_id", models.GameServer{}.TableName())
	rows, err := dao.connector.QuerySync(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGameServers(rows)
}

func (dao *GameServerDAO) GetByGroupID(groupID int32) ([]*models.GameServer, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"group_id": groupID})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(nil)
		var list []*models.GameServer
		if err := cursor.All(nil, &list); err != nil {
			return nil, err
		}
		return list, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE group_id = ? ORDER BY server_id", models.GameServer{}.TableName())
	rows, err := dao.connector.QuerySync(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGameServers(rows)
}

func (dao *GameServerDAO) GetByGroupIDAndType(groupID int32, serverType string) ([]*models.GameServer, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"group_id": groupID, "server_type": serverType})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(nil)
		var list []*models.GameServer
		if err := cursor.All(nil, &list); err != nil {
			return nil, err
		}
		return list, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE group_id = ? AND server_type = ? ORDER BY server_id", models.GameServer{}.TableName())
	rows, err := dao.connector.QuerySync(query, groupID, serverType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGameServers(rows)
}

func (dao *GameServerDAO) Create(gameServer *models.GameServer) (int32, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		_, err := collection.InsertOne(nil, gameServer)
		if err != nil {
			return 0, err
		}
		return gameServer.ServerID, nil
	}

	query := fmt.Sprintf("INSERT INTO %s (server_id, server_name, server_type, group_id, address, port, status, online_count, max_online_count, region, version, last_heartbeat, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", models.GameServer{}.TableName())
	result, err := dao.connector.ExecSync(query,
		gameServer.ServerID, gameServer.ServerName, gameServer.ServerType, gameServer.GroupID,
		gameServer.Address, gameServer.Port, gameServer.Status, gameServer.OnlineCount,
		gameServer.MaxOnlineCount, gameServer.Region, gameServer.Version,
		gameServer.LastHeartbeat, gameServer.CreatedAt, gameServer.UpdatedAt,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return int32(id), err
}

func (dao *GameServerDAO) Update(gameServer *models.GameServer) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		result, err := collection.UpdateOne(nil, bson.M{"server_id": gameServer.ServerID}, bson.M{
			"$set": bson.M{
				"server_name": gameServer.ServerName, "server_type": gameServer.ServerType,
				"group_id": gameServer.GroupID, "address": gameServer.Address,
				"port": gameServer.Port, "status": gameServer.Status,
				"online_count": gameServer.OnlineCount, "max_online_count": gameServer.MaxOnlineCount,
				"region": gameServer.Region, "version": gameServer.Version,
				"last_heartbeat": gameServer.LastHeartbeat, "updated_at": gameServer.UpdatedAt,
			},
		})
		if err != nil {
			return false, err
		}
		return result.ModifiedCount > 0, nil
	}

	query := fmt.Sprintf("UPDATE %s SET server_name = ?, server_type = ?, group_id = ?, address = ?, port = ?, status = ?, online_count = ?, max_online_count = ?, region = ?, version = ?, last_heartbeat = ?, updated_at = ? WHERE server_id = ?", models.GameServer{}.TableName())
	result, err := dao.connector.ExecSync(query,
		gameServer.ServerName, gameServer.ServerType, gameServer.GroupID,
		gameServer.Address, gameServer.Port, gameServer.Status, gameServer.OnlineCount,
		gameServer.MaxOnlineCount, gameServer.Region, gameServer.Version,
		gameServer.LastHeartbeat, gameServer.UpdatedAt, gameServer.ServerID,
	)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected > 0, err
}

func (dao *GameServerDAO) UpdateOnlineCount(serverID int32, onlineCount int32) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		result, err := collection.UpdateOne(nil, bson.M{"server_id": serverID}, bson.M{
			"$set": bson.M{"online_count": onlineCount},
		})
		if err != nil {
			return false, err
		}
		return result.ModifiedCount > 0, nil
	}

	query := fmt.Sprintf("UPDATE %s SET online_count = ? WHERE server_id = ?", models.GameServer{}.TableName())
	result, err := dao.connector.ExecSync(query, onlineCount, serverID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected > 0, err
}

func (dao *GameServerDAO) UpdateLastHeartbeat(serverID int32) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		result, err := collection.UpdateOne(nil, bson.M{"server_id": serverID}, bson.M{
			"$set": bson.M{"last_heartbeat": time.Now()},
		})
		if err != nil {
			return false, err
		}
		return result.ModifiedCount > 0, nil
	}

	query := fmt.Sprintf("UPDATE %s SET last_heartbeat = NOW() WHERE server_id = ?", models.GameServer{}.TableName())
	result, err := dao.connector.ExecSync(query, serverID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected > 0, err
}

func (dao *GameServerDAO) Delete(id int32) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GameServer{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"server_id": id})
		if err != nil {
			return false, err
		}
		return result.DeletedCount > 0, nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE server_id = ?", models.GameServer{}.TableName())
	result, err := dao.connector.ExecSync(query, id)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected > 0, err
}
