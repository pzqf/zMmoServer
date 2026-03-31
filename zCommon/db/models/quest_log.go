package models

import (
	"time"
)

type QuestLog struct {
	LogID     int64     `db:"log_id" bson:"log_id"`
	PlayerID   int64     `db:"player_id" bson:"player_id"`
	QuestID    int32     `db:"quest_id" bson:"quest_id"`
	OpType     int32     `db:"op_type" bson:"op_type"`
	Detail     string    `db:"detail" bson:"detail"`
	CreatedAt  time.Time `db:"created_at" bson:"created_at"`
}

func (QuestLog) TableName() string {
	return "`quest_logs`"
}
