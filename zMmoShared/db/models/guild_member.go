package models

import (
	"time"
)

type GuildMember struct {
	ID          int64     `db:"id" bson:"id"`
	GuildID     int64     `db:"guild_id" bson:"guild_id"`
	PlayerID    int64     `db:"player_id" bson:"player_id"`
	Position    int32     `db:"position" bson:"position"`
	Contribution int64    `db:"contribution" bson:"contribution"`
	TotalContribution int64 `db:"total_contribution" bson:"total_contribution"`
	JoinTime    int64     `db:"join_time" bson:"join_time"`
	LastActive  int64     `db:"last_active" bson:"last_active"`
	CreatedAt   time.Time `db:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" bson:"updated_at"`
}

func (GuildMember) TableName() string {
	return "`guild_members`"
}
