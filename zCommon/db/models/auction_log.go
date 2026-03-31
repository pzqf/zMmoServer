package models

import (
	"time"
)

type AuctionLog struct {
	LogID      int64     `db:"log_id" bson:"log_id"`
	AuctionID  int64     `db:"auction_id" bson:"auction_id"`
	PlayerID   int64     `db:"player_id" bson:"player_id"`
	OpType     int32     `db:"op_type" bson:"op_type"`
	Detail     string    `db:"detail" bson:"detail"`
	CreatedAt  time.Time `db:"created_at" bson:"created_at"`
}

func (AuctionLog) TableName() string {
	return "`auction_logs`"
}
