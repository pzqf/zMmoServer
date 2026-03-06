package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type PlayerQuestDAO struct {
	connector connector.DBConnector
}

func NewPlayerQuestDAO(dbConnector connector.DBConnector) *PlayerQuestDAO {
	return &PlayerQuestDAO{connector: dbConnector}
}

func (dao *PlayerQuestDAO) GetQuestsByPlayerID(playerID int64, callback func([]*models.PlayerQuest, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerQuest{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"player_id": playerID})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var quests []*models.PlayerQuest
		for cursor.Next(nil) {
			var quest models.PlayerQuest
			if err := cursor.Decode(&quest); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			quests = append(quests, &quest)
		}
		if callback != nil {
			callback(quests, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.PlayerQuest{}.TableName())
		dao.connector.Query(query, []interface{}{playerID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var quests []*models.PlayerQuest
			for rows.Next() {
				var quest models.PlayerQuest
				if err := rows.Scan(
					&quest.ID, &quest.PlayerID, &quest.QuestID, &quest.Status,
					&quest.Progress, &quest.AcceptTime, &quest.CompleteTime,
					&quest.CreatedAt, &quest.UpdatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				quests = append(quests, &quest)
			}
			if callback != nil {
				callback(quests, nil)
			}
		})
	}
}

func (dao *PlayerQuestDAO) CreateQuest(quest *models.PlayerQuest, callback func(int64, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerQuest{}.TableName())
		_, err := collection.InsertOne(nil, quest)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(quest.ID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (id, player_id, quest_id, status, progress, accept_time, complete_time, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", models.PlayerQuest{}.TableName())
		args := []interface{}{
			quest.ID, quest.PlayerID, quest.QuestID, quest.Status,
			quest.Progress, quest.AcceptTime, quest.CompleteTime,
			quest.CreatedAt, quest.UpdatedAt,
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

func (dao *PlayerQuestDAO) UpdateQuest(quest *models.PlayerQuest, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerQuest{}.TableName())
		update := bson.M{"$set": bson.M{
			"status": quest.Status, "progress": quest.Progress,
			"accept_time": quest.AcceptTime, "complete_time": quest.CompleteTime,
			"updated_at": quest.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"id": quest.ID}, update)
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
		query := fmt.Sprintf("UPDATE %s SET status = ?, progress = ?, accept_time = ?, complete_time = ?, updated_at = ? WHERE id = ?", models.PlayerQuest{}.TableName())
		args := []interface{}{
			quest.Status, quest.Progress, quest.AcceptTime,
			quest.CompleteTime, quest.UpdatedAt, quest.ID,
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

func (dao *PlayerQuestDAO) DeleteQuest(id int64, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerQuest{}.TableName())
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
		query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", models.PlayerQuest{}.TableName())
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
