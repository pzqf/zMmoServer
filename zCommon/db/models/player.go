package models

import (
	"time"
)

type Player struct {
	PlayerID     int64     `db:"player_id" bson:"player_id"`
	PlayerName   string    `db:"player_name" bson:"player_name"`
	AccountID    int64     `db:"account_id" bson:"account_id"`
	Sex          int       `db:"sex" bson:"sex"`
	Age          int       `db:"age" bson:"age"`
	Level        int       `db:"level" bson:"level"`
	Experience   int64     `db:"experience" bson:"experience"`
	Gold         int64     `db:"gold" bson:"gold"`
	Diamond      int64     `db:"diamond" bson:"diamond"`
	VipLevel     int       `db:"vip_level" bson:"vip_level"`
	LastLoginAt  time.Time `db:"last_login_at" bson:"last_login_at"`
	LastLogoutAt time.Time `db:"last_logout_at" bson:"last_logout_at"`
	CreatedAt    time.Time `db:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" bson:"updated_at"`
}

func (Player) TableName() string {
	return "players"
}
