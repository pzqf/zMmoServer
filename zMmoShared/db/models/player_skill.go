package models

import (
	"time"
)

type PlayerSkill struct {
	ID         int64     `db:"id" bson:"id"`
	PlayerID   int64     `db:"player_id" bson:"player_id"`
	SkillID    int32     `db:"skill_id" bson:"skill_id"`
	Level      int32     `db:"level" bson:"level"`
	Exp        int64     `db:"exp" bson:"exp"`
	HotKey     int32     `db:"hot_key" bson:"hot_key"`
	CreatedAt  time.Time `db:"created_at" bson:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" bson:"updated_at"`
}

func (PlayerSkill) TableName() string {
	return "`player_skills`"
}
