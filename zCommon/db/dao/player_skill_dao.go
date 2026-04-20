package dao

import (
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

func (dao *PlayerSkillDAO) GetSkillsByPlayerID(playerID int64) ([]*models.PlayerSkill, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerSkill{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"player_id": playerID})
		if err != nil {
			return nil, err
		}
		defer cursor.Close(nil)

		var skills []*models.PlayerSkill
		for cursor.Next(nil) {
			var skill models.PlayerSkill
			if err := cursor.Decode(&skill); err != nil {
				return nil, err
			}
			skills = append(skills, &skill)
		}
		return skills, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.PlayerSkill{}.TableName())
	rows, err := dao.connector.QuerySync(query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []*models.PlayerSkill
	for rows.Next() {
		var skill models.PlayerSkill
		if err := rows.Scan(
			&skill.ID, &skill.PlayerID, &skill.SkillID, &skill.Level,
			&skill.Exp, &skill.HotKey, &skill.CreatedAt, &skill.UpdatedAt,
		); err != nil {
			return nil, err
		}
		skills = append(skills, &skill)
	}
	return skills, nil
}

func (dao *PlayerSkillDAO) CreateSkill(skill *models.PlayerSkill) (int64, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerSkill{}.TableName())
		_, err := collection.InsertOne(nil, skill)
		if err != nil {
			return 0, err
		}
		return skill.ID, nil
	}

	query := fmt.Sprintf("INSERT INTO %s (id, player_id, skill_id, level, exp, hot_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", models.PlayerSkill{}.TableName())
	args := []interface{}{
		skill.ID, skill.PlayerID, skill.SkillID, skill.Level,
		skill.Exp, skill.HotKey, skill.CreatedAt, skill.UpdatedAt,
	}
	result, err := dao.connector.ExecSync(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (dao *PlayerSkillDAO) UpdateSkill(skill *models.PlayerSkill) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerSkill{}.TableName())
		update := bson.M{"$set": bson.M{
			"level": skill.Level, "exp": skill.Exp, "hot_key": skill.HotKey, "updated_at": skill.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"id": skill.ID}, update)
		if err != nil {
			return false, err
		}
		return result.ModifiedCount > 0, nil
	}

	query := fmt.Sprintf("UPDATE %s SET level = ?, exp = ?, hot_key = ?, updated_at = ? WHERE id = ?", models.PlayerSkill{}.TableName())
	args := []interface{}{skill.Level, skill.Exp, skill.HotKey, skill.UpdatedAt, skill.ID}
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

func (dao *PlayerSkillDAO) DeleteSkill(id int64) (bool, error) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerSkill{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"id": id})
		if err != nil {
			return false, err
		}
		return result.DeletedCount > 0, nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", models.PlayerSkill{}.TableName())
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
