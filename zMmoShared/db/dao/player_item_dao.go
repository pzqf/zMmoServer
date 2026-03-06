package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type PlayerItemDAO struct {
	connector connector.DBConnector
}

func NewPlayerItemDAO(dbConnector connector.DBConnector) *PlayerItemDAO {
	return &PlayerItemDAO{connector: dbConnector}
}

func (dao *PlayerItemDAO) GetItemsByPlayerID(playerID int64, callback func([]*models.PlayerItem, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerItem{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"player_id": playerID})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var items []*models.PlayerItem
		for cursor.Next(nil) {
			var item models.PlayerItem
			if err := cursor.Decode(&item); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			items = append(items, &item)
		}
		if callback != nil {
			callback(items, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.PlayerItem{}.TableName())
		dao.connector.Query(query, []interface{}{playerID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var items []*models.PlayerItem
			for rows.Next() {
				var item models.PlayerItem
				if err := rows.Scan(
					&item.ItemID, &item.PlayerID, &item.ItemConfigID, &item.Count,
					&item.Level, &item.Quality, &item.SlotIndex, &item.BindType,
					&item.ExpireTime, &item.Attrs, &item.CreatedAt, &item.UpdatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				items = append(items, &item)
			}
			if callback != nil {
				callback(items, nil)
			}
		})
	}
}

func (dao *PlayerItemDAO) CreateItem(item *models.PlayerItem, callback func(int64, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerItem{}.TableName())
		_, err := collection.InsertOne(nil, item)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(item.ItemID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (item_id, player_id, item_config_id, count, level, quality, slot_index, bind_type, expire_time, attrs, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", models.PlayerItem{}.TableName())
		args := []interface{}{
			item.ItemID, item.PlayerID, item.ItemConfigID, item.Count,
			item.Level, item.Quality, item.SlotIndex, item.BindType,
			item.ExpireTime, item.Attrs, item.CreatedAt, item.UpdatedAt,
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

func (dao *PlayerItemDAO) UpdateItem(item *models.PlayerItem, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerItem{}.TableName())
		update := bson.M{"$set": bson.M{
			"count": item.Count, "level": item.Level, "quality": item.Quality,
			"slot_index": item.SlotIndex, "bind_type": item.BindType,
			"expire_time": item.ExpireTime, "attrs": item.Attrs, "updated_at": item.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"item_id": item.ItemID}, update)
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
		query := fmt.Sprintf("UPDATE %s SET count = ?, level = ?, quality = ?, slot_index = ?, bind_type = ?, expire_time = ?, attrs = ?, updated_at = ? WHERE item_id = ?", models.PlayerItem{}.TableName())
		args := []interface{}{
			item.Count, item.Level, item.Quality, item.SlotIndex, item.BindType,
			item.ExpireTime, item.Attrs, item.UpdatedAt, item.ItemID,
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

func (dao *PlayerItemDAO) DeleteItem(itemID int64, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerItem{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"item_id": itemID})
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
		query := fmt.Sprintf("DELETE FROM %s WHERE item_id = ?", models.PlayerItem{}.TableName())
		dao.connector.Execute(query, []interface{}{itemID}, func(result sql.Result, err error) {
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
