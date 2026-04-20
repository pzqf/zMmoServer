package dao

import (
	"fmt"

	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type PlayerBuffDAO struct {
	connector connector.DBConnector
}

func NewPlayerBuffDAO(dbConnector connector.DBConnector) *PlayerBuffDAO {
	return &PlayerBuffDAO{connector: dbConnector}
}

func (dao *PlayerBuffDAO) GetBuffsByPlayerID(playerID int64) ([]*models.PlayerBuff, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerBuff{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"player_id": playerID})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(nil)

		var buffs []*models.PlayerBuff
		for cursor.Next(nil) {
			var buff models.PlayerBuff
			if err := cursor.Decode(&buff); err != nil {
				return nil, err
			}
			buffs = append(buffs, &buff)
		}
		return buffs, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.PlayerBuff{}.TableName())
	rows, err := dao.connector.QuerySync(query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buffs []*models.PlayerBuff
	for rows.Next() {
		var buff models.PlayerBuff
		if err := rows.Scan(
			&buff.ID, &buff.PlayerID, &buff.BuffID, &buff.StackCount,
			&buff.Duration, &buff.EndTime, &buff.CasterID,
			&buff.CreatedAt, &buff.UpdatedAt,
		); err != nil {
			return nil, err
		}
		buffs = append(buffs, &buff)
	}
	return buffs, nil
}

func (dao *PlayerBuffDAO) CreateBuff(buff *models.PlayerBuff) (int64, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerBuff{}.TableName())
		_, err := collection.InsertOne(nil, buff)
		if err != nil {
			return 0, err
		}
		return buff.ID, nil
	}

	query := fmt.Sprintf("INSERT INTO %s (id, player_id, buff_id, stack_count, duration, end_time, caster_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", models.PlayerBuff{}.TableName())
	args := []interface{}{
		buff.ID, buff.PlayerID, buff.BuffID, buff.StackCount,
		buff.Duration, buff.EndTime, buff.CasterID,
		buff.CreatedAt, buff.UpdatedAt,
	}
	result, err := dao.connector.ExecSync(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (dao *PlayerBuffDAO) UpdateBuff(buff *models.PlayerBuff) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerBuff{}.TableName())
		update := bson.M{"$set": bson.M{
			"stack_count": buff.StackCount, "duration": buff.Duration,
			"end_time": buff.EndTime, "updated_at": buff.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"id": buff.ID}, update)
		if err != nil {
			return false, err
		}
		return result.ModifiedCount > 0, nil
	}

	query := fmt.Sprintf("UPDATE %s SET stack_count = ?, duration = ?, end_time = ?, updated_at = ? WHERE id = ?", models.PlayerBuff{}.TableName())
	args := []interface{}{buff.StackCount, buff.Duration, buff.EndTime, buff.UpdatedAt, buff.ID}
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

func (dao *PlayerBuffDAO) DeleteBuff(id int64) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerBuff{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"id": id})
		if err != nil {
			return false, err
		}
		return result.DeletedCount > 0, nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", models.PlayerBuff{}.TableName())
	result, err := dao.connector.ExecSync(query, id)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}
