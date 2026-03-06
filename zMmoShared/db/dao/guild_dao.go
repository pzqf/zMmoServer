package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type GuildDAO struct {
	connector connector.DBConnector
}

func NewGuildDAO(dbConnector connector.DBConnector) *GuildDAO {
	return &GuildDAO{connector: dbConnector}
}

func (dao *GuildDAO) GetGuildByID(guildID int64, callback func(*models.Guild, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Guild{}.TableName())
		var guild models.Guild
		result := collection.FindOne(nil, bson.M{"guild_id": guildID})
		err := result.Decode(&guild)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				if callback != nil {
					callback(nil, nil)
				}
				return
			}
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		if callback != nil {
			callback(&guild, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE guild_id = ?", models.Guild{}.TableName())
		dao.connector.Query(query, []interface{}{guildID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			if rows.Next() {
				var guild models.Guild
				if err := rows.Scan(
					&guild.GuildID, &guild.GuildName, &guild.LeaderID, &guild.Level,
					&guild.Exp, &guild.MemberCount, &guild.MaxMembers,
					&guild.Notice, &guild.Announcement, &guild.CreatedAt, &guild.UpdatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				if callback != nil {
					callback(&guild, nil)
				}
			} else {
				if callback != nil {
					callback(nil, nil)
				}
			}
		})
	}
}

func (dao *GuildDAO) GetGuildByName(name string, callback func(*models.Guild, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Guild{}.TableName())
		var guild models.Guild
		result := collection.FindOne(nil, bson.M{"guild_name": name})
		err := result.Decode(&guild)
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				if callback != nil {
					callback(nil, nil)
				}
				return
			}
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		if callback != nil {
			callback(&guild, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE guild_name = ?", models.Guild{}.TableName())
		dao.connector.Query(query, []interface{}{name}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			if rows.Next() {
				var guild models.Guild
				if err := rows.Scan(
					&guild.GuildID, &guild.GuildName, &guild.LeaderID, &guild.Level,
					&guild.Exp, &guild.MemberCount, &guild.MaxMembers,
					&guild.Notice, &guild.Announcement, &guild.CreatedAt, &guild.UpdatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				if callback != nil {
					callback(&guild, nil)
				}
			} else {
				if callback != nil {
					callback(nil, nil)
				}
			}
		})
	}
}

func (dao *GuildDAO) CreateGuild(guild *models.Guild, callback func(int64, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Guild{}.TableName())
		_, err := collection.InsertOne(nil, guild)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(guild.GuildID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (guild_id, guild_name, leader_id, level, exp, member_count, max_members, notice, announcement, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", models.Guild{}.TableName())
		args := []interface{}{
			guild.GuildID, guild.GuildName, guild.LeaderID, guild.Level,
			guild.Exp, guild.MemberCount, guild.MaxMembers,
			guild.Notice, guild.Announcement, guild.CreatedAt, guild.UpdatedAt,
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

func (dao *GuildDAO) UpdateGuild(guild *models.Guild, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Guild{}.TableName())
		update := bson.M{"$set": bson.M{
			"level": guild.Level, "exp": guild.Exp, "member_count": guild.MemberCount,
			"notice": guild.Notice, "announcement": guild.Announcement, "updated_at": guild.UpdatedAt,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"guild_id": guild.GuildID}, update)
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
		query := fmt.Sprintf("UPDATE %s SET level = ?, exp = ?, member_count = ?, notice = ?, announcement = ?, updated_at = ? WHERE guild_id = ?", models.Guild{}.TableName())
		args := []interface{}{
			guild.Level, guild.Exp, guild.MemberCount,
			guild.Notice, guild.Announcement, guild.UpdatedAt, guild.GuildID,
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

func (dao *GuildDAO) DeleteGuild(guildID int64, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.Guild{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"guild_id": guildID})
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
		query := fmt.Sprintf("DELETE FROM %s WHERE guild_id = ?", models.Guild{}.TableName())
		dao.connector.Execute(query, []interface{}{guildID}, func(result sql.Result, err error) {
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
