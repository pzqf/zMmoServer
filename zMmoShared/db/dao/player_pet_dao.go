package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type PlayerPetDAO struct {
	connector connector.DBConnector
}

func NewPlayerPetDAO(dbConnector connector.DBConnector) *PlayerPetDAO {
	return &PlayerPetDAO{connector: dbConnector}
}

func (dao *PlayerPetDAO) GetPetsByPlayerID(playerID int64, callback func([]*models.PlayerPet, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerPet{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"player_id": playerID})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var pets []*models.PlayerPet
		for cursor.Next(nil) {
			var pet models.PlayerPet
			if err := cursor.Decode(&pet); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			pets = append(pets, &pet)
		}
		if callback != nil {
			callback(pets, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.PlayerPet{}.TableName())
		dao.connector.Query(query, []interface{}{playerID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var pets []*models.PlayerPet
			for rows.Next() {
				var pet models.PlayerPet
				if err := rows.Scan(
					&pet.PetID, &pet.PlayerID, &pet.PetConfigID, &pet.Name,
					&pet.Level, &pet.Exp, &pet.HP, &pet.MaxHP,
					&pet.Attack, &pet.Defense, &pet.Skills, &pet.IsActive,
					&pet.CreatedAt, &pet.UpdatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				pets = append(pets, &pet)
			}
			if callback != nil {
				callback(pets, nil)
			}
		})
	}
}

func (dao *PlayerPetDAO) CreatePet(pet *models.PlayerPet, callback func(int64, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerPet{}.TableName())
		_, err := collection.InsertOne(nil, pet)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(pet.PetID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (pet_id, player_id, pet_config_id, name, level, exp, hp, max_hp, attack, defense, skills, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", models.PlayerPet{}.TableName())
		args := []interface{}{
			pet.PetID, pet.PlayerID, pet.PetConfigID, pet.Name,
			pet.Level, pet.Exp, pet.HP, pet.MaxHP,
			pet.Attack, pet.Defense, pet.Skills, pet.IsActive,
			pet.CreatedAt, pet.UpdatedAt,
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

func (dao *PlayerPetDAO) UpdatePet(pet *models.PlayerPet, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerPet{}.TableName())
		update := bson.M{"$set": bson.M{
			"name": pet.Name, "level": pet.Level, "exp": pet.Exp,
			"hp": pet.HP, "max_hp": pet.MaxHP, "attack": pet.Attack,
			"defense": pet.Defense, "skills": pet.Skills, "is_active": pet.IsActive,
			"updated_at": pet.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"pet_id": pet.PetID}, update)
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
		query := fmt.Sprintf("UPDATE %s SET name = ?, level = ?, exp = ?, hp = ?, max_hp = ?, attack = ?, defense = ?, skills = ?, is_active = ?, updated_at = ? WHERE pet_id = ?", models.PlayerPet{}.TableName())
		args := []interface{}{
			pet.Name, pet.Level, pet.Exp, pet.HP, pet.MaxHP,
			pet.Attack, pet.Defense, pet.Skills, pet.IsActive,
			pet.UpdatedAt, pet.PetID,
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

func (dao *PlayerPetDAO) DeletePet(petID int64, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerPet{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"pet_id": petID})
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
		query := fmt.Sprintf("DELETE FROM %s WHERE pet_id = ?", models.PlayerPet{}.TableName())
		dao.connector.Execute(query, []interface{}{petID}, func(result sql.Result, err error) {
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
