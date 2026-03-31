package models

import (
	"time"
)

type LoginLog struct {
	LogID      int64     `db:"log_id" bson:"log_id"`
	PlayerID   int64     `db:"player_id" bson:"player_id"`
	PlayerName string    `db:"player_name" bson:"player_name"`
	OpType     int32     `db:"op_type" bson:"op_type"`
	IP         string    `db:"ip" bson:"ip"`
	Device     string    `db:"device" bson:"device"`
	CreatedAt  time.Time `db:"created_at" bson:"created_at"`
}

func (LoginLog) TableName() string {
	return "`login_logs`"
}
