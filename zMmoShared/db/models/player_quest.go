package models

import (
	"time"
)

type PlayerQuest struct {
	ID           int64     `db:"id" bson:"id"`
	PlayerID     int64     `db:"player_id" bson:"player_id"`
	QuestID      int32     `db:"quest_id" bson:"quest_id"`
	Status       int32     `db:"status" bson:"status"`
	Progress     string    `db:"progress" bson:"progress"`
	AcceptTime   int64     `db:"accept_time" bson:"accept_time"`
	CompleteTime int64     `db:"complete_time" bson:"complete_time"`
	CreatedAt    time.Time `db:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" bson:"updated_at"`
}

func (PlayerQuest) TableName() string {
	return "`player_quests`"
}
