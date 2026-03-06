package models

import (
	"time"
)

type Guild struct {
	GuildID    int64     `db:"guild_id" bson:"guild_id"`
	GuildName  string    `db:"guild_name" bson:"guild_name"`
	LeaderID   int64     `db:"leader_id" bson:"leader_id"`
	Level      int32     `db:"level" bson:"level"`
	Exp        int64     `db:"exp" bson:"exp"`
	MemberCount int32    `db:"member_count" bson:"member_count"`
	MaxMembers int32     `db:"max_members" bson:"max_members"`
	Notice     string    `db:"notice" bson:"notice"`
	Announcement string   `db:"announcement" bson:"announcement"`
	CreatedAt  time.Time `db:"created_at" bson:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" bson:"updated_at"`
}

func (Guild) TableName() string {
	return "`guilds`"
}
