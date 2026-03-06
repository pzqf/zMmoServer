package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type PlayerBuffDAO struct {
	connector connector.DBConnector
}

func NewPlayerBuffDAO(dbConnector connector.DBConnector) *PlayerBuffDAO {
	return &PlayerBuffDAO{connector: dbConnector}
}

func (dao *PlayerBuffDAO) GetBuffsByPlayerID(playerID int64, callback func([]*models.PlayerBuff, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerBuff{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"player_id": playerID})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var buffs []*models.PlayerBuff
		for cursor.Next(nil) {
			var buff models.PlayerBuff
			if err := cursor.Decode(&buff); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			buffs = append(buffs, &buff)
		}
		if callback != nil {
			callback(buffs, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.PlayerBuff{}.TableName())
		dao.connector.Query(query, []interface{}{playerID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
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
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				buffs = append(buffs, &buff)
			}
			if callback != nil {
				callback(buffs, nil)
			}
		})
	}
}

func (dao *PlayerBuffDAO) CreateBuff(buff *models.PlayerBuff, callback func(int64, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerBuff{}.TableName())
		_, err := collection.InsertOne(nil, buff)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(buff.ID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (id, player_id, buff_id, stack_count, duration, end_time, caster_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", models.PlayerBuff{}.TableName())
		args := []interface{}{
			buff.ID, buff.PlayerID, buff.BuffID, buff.StackCount,
			buff.Duration, buff.EndTime, buff.CasterID,
			buff.CreatedAt, buff.UpdatedAt,
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

func (dao *PlayerBuffDAO) UpdateBuff(buff *models.PlayerBuff, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerBuff{}.TableName())
		update := bson.M{"$set": bson.M{
			"stack_count": buff.StackCount, "duration": buff.Duration,
			"end_time": buff.EndTime, "updated_at": buff.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"id": buff.ID}, update)
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
		query := fmt.Sprintf("UPDATE %s SET stack_count = ?, duration = ?, end_time = ?, updated_at = ? WHERE id = ?", models.PlayerBuff{}.TableName())
		args := []interface{}{buff.StackCount, buff.Duration, buff.EndTime, buff.UpdatedAt, buff.ID}
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

func (dao *PlayerBuffDAO) DeleteBuff(id int64, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerBuff{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"id": id})
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
		query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", models.PlayerBuff{}.TableName())
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
