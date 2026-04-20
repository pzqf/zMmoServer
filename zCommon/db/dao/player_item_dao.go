package dao

import (
	"fmt"

	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type PlayerItemDAO struct {
	connector connector.DBConnector
}

func NewPlayerItemDAO(dbConnector connector.DBConnector) *PlayerItemDAO {
	return &PlayerItemDAO{connector: dbConnector}
}

func (dao *PlayerItemDAO) GetItemsByPlayerID(playerID int64) ([]*models.PlayerItem, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerItem{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"player_id": playerID})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(nil)

		var items []*models.PlayerItem
		for cursor.Next(nil) {
			var item models.PlayerItem
			if err := cursor.Decode(&item); err != nil {
				return nil, err
			}
			items = append(items, &item)
		}
		return items, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.PlayerItem{}.TableName())
	rows, err := dao.connector.QuerySync(query, playerID)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		items = append(items, &item)
	}
	return items, nil
}

func (dao *PlayerItemDAO) CreateItem(item *models.PlayerItem) (int64, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerItem{}.TableName())
		_, err := collection.InsertOne(nil, item)
		if err != nil {
			return 0, err
		}
		return item.ItemID, nil
	}

	query := fmt.Sprintf("INSERT INTO %s (item_id, player_id, item_config_id, count, level, quality, slot_index, bind_type, expire_time, attrs, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", models.PlayerItem{}.TableName())
	args := []interface{}{
		item.ItemID, item.PlayerID, item.ItemConfigID, item.Count,
		item.Level, item.Quality, item.SlotIndex, item.BindType,
		item.ExpireTime, item.Attrs, item.CreatedAt, item.UpdatedAt,
	}
	result, err := dao.connector.ExecSync(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (dao *PlayerItemDAO) UpdateItem(item *models.PlayerItem) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerItem{}.TableName())
		update := bson.M{"$set": bson.M{
			"count": item.Count, "level": item.Level, "quality": item.Quality,
			"slot_index": item.SlotIndex, "bind_type": item.BindType,
			"expire_time": item.ExpireTime, "attrs": item.Attrs, "updated_at": item.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"item_id": item.ItemID}, update)
		if err != nil {
			return false, err
		}
		return result.ModifiedCount > 0, nil
	}

	query := fmt.Sprintf("UPDATE %s SET count = ?, level = ?, quality = ?, slot_index = ?, bind_type = ?, expire_time = ?, attrs = ?, updated_at = ? WHERE item_id = ?", models.PlayerItem{}.TableName())
	args := []interface{}{
		item.Count, item.Level, item.Quality, item.SlotIndex, item.BindType,
		item.ExpireTime, item.Attrs, item.UpdatedAt, item.ItemID,
	}
	result, err := dao.connector.ExecSync(query, args...)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (dao *PlayerItemDAO) DeleteItem(itemID int64) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerItem{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"item_id": itemID})
		if err != nil {
			return false, err
		}
		return result.DeletedCount > 0, nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE item_id = ?", models.PlayerItem{}.TableName())
	result, err := dao.connector.ExecSync(query, itemID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}
