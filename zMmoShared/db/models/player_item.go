package models

import (
	"time"
)

type PlayerItem struct {
	ItemID       int64     `db:"item_id" bson:"item_id"`
	PlayerID     int64     `db:"player_id" bson:"player_id"`
	ItemConfigID int32     `db:"item_config_id" bson:"item_config_id"`
	Count        int32     `db:"count" bson:"count"`
	Level        int32     `db:"level" bson:"level"`
	Quality      int32     `db:"quality" bson:"quality"`
	SlotIndex    int32     `db:"slot_index" bson:"slot_index"`
	BindType     int32     `db:"bind_type" bson:"bind_type"`
	ExpireTime   int64     `db:"expire_time" bson:"expire_time"`
	Attrs        string    `db:"attrs" bson:"attrs"`
	CreatedAt    time.Time `db:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" bson:"updated_at"`
}

func (PlayerItem) TableName() string {
	return "`player_items`"
}
