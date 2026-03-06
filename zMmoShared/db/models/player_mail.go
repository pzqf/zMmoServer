package models

import (
	"time"
)

type PlayerMail struct {
	MailID       int64     `db:"mail_id" bson:"mail_id"`
	PlayerID     int64     `db:"player_id" bson:"player_id"`
	SenderID     int64     `db:"sender_id" bson:"sender_id"`
	SenderName   string    `db:"sender_name" bson:"sender_name"`
	MailType     int32     `db:"mail_type" bson:"mail_type"`
	Title        string    `db:"title" bson:"title"`
	Content      string    `db:"content" bson:"content"`
	IsRead       int32     `db:"is_read" bson:"is_read"`
	IsReceived   int32     `db:"is_received" bson:"is_received"`
	Attachment   string    `db:"attachment" bson:"attachment"`
	ExpireTime   int64     `db:"expire_time" bson:"expire_time"`
	CreatedAt    time.Time `db:"created_at" bson:"created_at"`
}

func (PlayerMail) TableName() string {
	return "`player_mails`"
}
