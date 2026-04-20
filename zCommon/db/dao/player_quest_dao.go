package dao

import (
	"fmt"

	"github.com/pzqf/zCommon/db/connector"
	"github.com/pzqf/zCommon/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type PlayerQuestDAO struct {
	connector connector.DBConnector
}

func NewPlayerQuestDAO(dbConnector connector.DBConnector) *PlayerQuestDAO {
	return &PlayerQuestDAO{connector: dbConnector}
}

func (dao *PlayerQuestDAO) GetQuestsByPlayerID(playerID int64) ([]*models.PlayerQuest, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerQuest{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"player_id": playerID})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(nil)

		var quests []*models.PlayerQuest
		for cursor.Next(nil) {
			var quest models.PlayerQuest
			if err := cursor.Decode(&quest); err != nil {
				return nil, err
			}
			quests = append(quests, &quest)
		}
		return quests, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.PlayerQuest{}.TableName())
	rows, err := dao.connector.QuerySync(query, playerID)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		quests = append(quests, &quest)
	}
	return quests, nil
}

func (dao *PlayerQuestDAO) CreateQuest(quest *models.PlayerQuest) (int64, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerQuest{}.TableName())
		_, err := collection.InsertOne(nil, quest)
		if err != nil {
			return 0, err
		}
		return quest.ID, nil
	}

	query := fmt.Sprintf("INSERT INTO %s (id, player_id, quest_id, status, progress, accept_time, complete_time, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", models.PlayerQuest{}.TableName())
	args := []interface{}{
		quest.ID, quest.PlayerID, quest.QuestID, quest.Status,
		quest.Progress, quest.AcceptTime, quest.CompleteTime,
		quest.CreatedAt, quest.UpdatedAt,
	}
	result, err := dao.connector.ExecSync(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (dao *PlayerQuestDAO) UpdateQuest(quest *models.PlayerQuest) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerQuest{}.TableName())
		update := bson.M{"$set": bson.M{
			"status": quest.Status, "progress": quest.Progress,
			"accept_time": quest.AcceptTime, "complete_time": quest.CompleteTime,
			"updated_at": quest.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"id": quest.ID}, update)
		if err != nil {
			return false, err
		}
		return result.ModifiedCount > 0, nil
	}

	query := fmt.Sprintf("UPDATE %s SET status = ?, progress = ?, accept_time = ?, complete_time = ?, updated_at = ? WHERE id = ?", models.PlayerQuest{}.TableName())
	args := []interface{}{
		quest.Status, quest.Progress, quest.AcceptTime,
		quest.CompleteTime, quest.UpdatedAt, quest.ID,
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

func (dao *PlayerQuestDAO) DeleteQuest(id int64) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerQuest{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"id": id})
		if err != nil {
			return false, err
		}
		return result.DeletedCount > 0, nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", models.PlayerQuest{}.TableName())
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
