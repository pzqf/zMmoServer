package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type GuildMemberDAO struct {
	connector connector.DBConnector
}

func NewGuildMemberDAO(dbConnector connector.DBConnector) *GuildMemberDAO {
	return &GuildMemberDAO{connector: dbConnector}
}

func (dao *GuildMemberDAO) GetMembersByGuildID(guildID int64, callback func([]*models.GuildMember, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GuildMember{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"guild_id": guildID})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var members []*models.GuildMember
		for cursor.Next(nil) {
			var member models.GuildMember
			if err := cursor.Decode(&member); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			members = append(members, &member)
		}
		if callback != nil {
			callback(members, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE guild_id = ?", models.GuildMember{}.TableName())
		dao.connector.Query(query, []interface{}{guildID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var members []*models.GuildMember
			for rows.Next() {
				var member models.GuildMember
				if err := rows.Scan(
					&member.ID, &member.GuildID, &member.PlayerID, &member.Position,
					&member.Contribution, &member.TotalContribution,
					&member.JoinTime, &member.LastActive, &member.CreatedAt, &member.UpdatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				members = append(members, &member)
			}
			if callback != nil {
				callback(members, nil)
			}
		})
	}
}

func (dao *GuildMemberDAO) CreateMember(member *models.GuildMember, callback func(int64, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GuildMember{}.TableName())
		_, err := collection.InsertOne(nil, member)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(member.ID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (id, guild_id, player_id, position, contribution, total_contribution, join_time, last_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", models.GuildMember{}.TableName())
		args := []interface{}{
			member.ID, member.GuildID, member.PlayerID, member.Position,
			member.Contribution, member.TotalContribution,
			member.JoinTime, member.LastActive, member.CreatedAt, member.UpdatedAt,
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

func (dao *GuildMemberDAO) UpdateMember(member *models.GuildMember, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GuildMember{}.TableName())
		update := bson.M{"$set": bson.M{
			"position": member.Position, "contribution": member.Contribution,
			"total_contribution": member.TotalContribution, "last_active": member.LastActive,
			"updated_at": member.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"id": member.ID}, update)
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
		query := fmt.Sprintf("UPDATE %s SET position = ?, contribution = ?, total_contribution = ?, last_active = ?, updated_at = ? WHERE id = ?", models.GuildMember{}.TableName())
		args := []interface{}{
			member.Position, member.Contribution, member.TotalContribution,
			member.LastActive, member.UpdatedAt, member.ID,
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

func (dao *GuildMemberDAO) DeleteMember(id int64, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.GuildMember{}.TableName())
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
		query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", models.GuildMember{}.TableName())
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
