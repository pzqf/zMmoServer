package dao

import (
	"database/sql"
	"fmt"

	"github.com/pzqf/zMmoShared/db/connector"
	"github.com/pzqf/zMmoShared/db/models"
	"go.mongodb.org/mongo-driver/bson"
)

type PlayerMailDAO struct {
	connector connector.DBConnector
}

func NewPlayerMailDAO(dbConnector connector.DBConnector) *PlayerMailDAO {
	return &PlayerMailDAO{connector: dbConnector}
}

func (dao *PlayerMailDAO) GetMailsByPlayerID(playerID int64, callback func([]*models.PlayerMail, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerMail{}.TableName())
		cursor, err := collection.Find(nil, bson.M{"player_id": playerID})
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		defer cursor.Close(nil)

		var mails []*models.PlayerMail
		for cursor.Next(nil) {
			var mail models.PlayerMail
			if err := cursor.Decode(&mail); err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			mails = append(mails, &mail)
		}
		if callback != nil {
			callback(mails, nil)
		}
	} else {
		query := fmt.Sprintf("SELECT * FROM %s WHERE player_id = ?", models.PlayerMail{}.TableName())
		dao.connector.Query(query, []interface{}{playerID}, func(rows *sql.Rows, err error) {
			if err != nil {
				if callback != nil {
					callback(nil, err)
				}
				return
			}
			defer rows.Close()

			var mails []*models.PlayerMail
			for rows.Next() {
				var mail models.PlayerMail
				if err := rows.Scan(
					&mail.MailID, &mail.PlayerID, &mail.SenderID, &mail.SenderName,
					&mail.MailType, &mail.Title, &mail.Content, &mail.IsRead,
					&mail.IsReceived, &mail.Attachment, &mail.ExpireTime, &mail.CreatedAt,
				); err != nil {
					if callback != nil {
						callback(nil, err)
					}
					return
				}
				mails = append(mails, &mail)
			}
			if callback != nil {
				callback(mails, nil)
			}
		})
	}
}

func (dao *PlayerMailDAO) CreateMail(mail *models.PlayerMail, callback func(int64, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerMail{}.TableName())
		_, err := collection.InsertOne(nil, mail)
		if err != nil {
			if callback != nil {
				callback(0, err)
			}
			return
		}
		if callback != nil {
			callback(mail.MailID, nil)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (mail_id, player_id, sender_id, sender_name, mail_type, title, content, is_read, is_received, attachment, expire_time, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", models.PlayerMail{}.TableName())
		args := []interface{}{
			mail.MailID, mail.PlayerID, mail.SenderID, mail.SenderName,
			mail.MailType, mail.Title, mail.Content, mail.IsRead,
			mail.IsReceived, mail.Attachment, mail.ExpireTime, mail.CreatedAt,
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

func (dao *PlayerMailDAO) UpdateMail(mail *models.PlayerMail, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerMail{}.TableName())
		update := bson.M{"$set": bson.M{
			"is_read": mail.IsRead, "is_received": mail.IsReceived,
		}}
		result, err := collection.UpdateOne(nil, bson.M{"mail_id": mail.MailID}, update)
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
		query := fmt.Sprintf("UPDATE %s SET is_read = ?, is_received = ? WHERE mail_id = ?", models.PlayerMail{}.TableName())
		args := []interface{}{mail.IsRead, mail.IsReceived, mail.MailID}
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

func (dao *PlayerMailDAO) DeleteMail(mailID int64, callback func(bool, error)) {
	if dao.connector.GetDriver() == "mongo" {
		collection := dao.connector.GetMongoDB().Collection(models.PlayerMail{}.TableName())
		result, err := collection.DeleteOne(nil, bson.M{"mail_id": mailID})
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
		query := fmt.Sprintf("DELETE FROM %s WHERE mail_id = ?", models.PlayerMail{}.TableName())
		dao.connector.Execute(query, []interface{}{mailID}, func(result sql.Result, err error) {
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
