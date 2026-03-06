package models

import (
	"time"
)

type PlayerBuff struct {
	ID         int64     `db:"id" bson:"id"`
	PlayerID   int64     `db:"player_id" bson:"player_id"`
	BuffID     int32     `db:"buff_id" bson:"buff_id"`
	StackCount int32     `db:"stack_count" bson:"stack_count"`
	Duration   int32     `db:"duration" bson:"duration"`
	EndTime    int64     `db:"end_time" bson:"end_time"`
	CasterID   int64     `db:"caster_id" bson:"caster_id"`
	CreatedAt  time.Time `db:"created_at" bson:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" bson:"updated_at"`
}

func (PlayerBuff) TableName() string {
	return "`player_buffs`"
}
