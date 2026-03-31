package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type PlayerSkillDAO struct {
	connector connector.DBConnector
}

func NewPlayerSkillDAO(dbConnector connector.DBConnector) *PlayerSkillDAO {
	return &PlayerSkillDAO{connector: dbConnector}
}

func (dao *PlayerSkillDAO) GetSkillsByPlayerID(playerID int64, callback func([]*models.PlayerSkill, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerSkill{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"player_id": playerID})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var skills []*models.PlayerSkill
		for cursor.Next(nil) {
			var skill models.PlayerSkill
			if err := cursor.Decode(&skill); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			skills = append(skills, &skill)
		}
		if callback != nil {
			callback(skills, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.PlayerSkill{}.TableName())
		dao.connector.Query(query, []interface{}{playerID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var skills []*models.PlayerSkill
			for rows.Next() {
				var skill models.PlayerSkill
				if err := rows.Scan(
					&skill.ID, &skill.PlayerID, &skill.SkillID, &skill.Level,
					&skill.Exp, &skill.HotKey, &skill.CreatedAt, &skill.UpdatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				skills = append(skills, &skill)
			}
			if callback != nil {
				callback(skills, nil)
			}
		})
	}
}

func (dao *PlayerSkillDAO) CreateSkill(skill *models.PlayerSkill, callback func(int64, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerSkill{}.TableName())
		_, err := collection.InsertOne(nil, skill)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(skill.ID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (id, player_id, skill_id, level, exp, hot_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", models.PlayerSkill{}.TableName())
		args := []interface{}{
			skill.ID, skill.PlayerID, skill.SkillID, skill.Level,
			skill.Exp, skill.HotKey, skill.CreatedAt, skill.UpdatedAt,
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

func (dao *PlayerSkillDAO) UpdateSkill(skill *models.PlayerSkill, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerSkill{}.TableName())
		update := bson.M{"$set": bson.M{
			"level": skill.Level, "exp": skill.Exp, "hot_key": skill.HotKey, "updated_at": skill.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"id": skill.ID}, update)
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
		query := fmt.Sprintf("UPDATE %s SET level = ?, exp = ?, hot_key = ?, updated_at = ? WHERE id = ?", models.PlayerSkill{}.TableName())
		args := []interface{}{skill.Level, skill.Exp, skill.HotKey, skill.UpdatedAt, skill.ID}
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

func (dao *PlayerSkillDAO) DeleteSkill(id int64, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerSkill{}.TableName())
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
		query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", models.PlayerSkill{}.TableName())
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

