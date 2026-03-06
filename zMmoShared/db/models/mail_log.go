package models

import (
	"time"
)

type MailLog struct {
	LogID      int64     `db:"log_id" bson:"log_id"`
	MailID     int64     `db:"mail_id" bson:"mail_id"`
	SenderID   int64     `db:"sender_id" bson:"sender_id"`
	ReceiverID int64     `db:"receiver_id" bson:"receiver_id"`
	OpType     int32     `db:"op_type" bson:"op_type"`
	Detail     string    `db:"detail" bson:"detail"`
	CreatedAt  time.Time `db:"created_at" bson:"created_at"`
}

func (MailLog) TableName() string {
	return "`mail_logs`"
}
