package models

import (
	"time"
)

type Auction struct {
	AuctionID    int64     `db:"auction_id" bson:"auction_id"`
	SellerID     int64     `db:"seller_id" bson:"seller_id"`
	SellerName   string    `db:"seller_name" bson:"seller_name"`
	ItemConfigID int32     `db:"item_config_id" bson:"item_config_id"`
	ItemCount    int32     `db:"item_count" bson:"item_count"`
	ItemLevel    int32     `db:"item_level" bson:"item_level"`
	ItemQuality  int32     `db:"item_quality" bson:"item_quality"`
	PriceType    int32     `db:"price_type" bson:"price_type"`
	Price        int64     `db:"price" bson:"price"`
	BuyerID      int64     `db:"buyer_id" bson:"buyer_id"`
	Status       int32     `db:"status" bson:"status"`
	EndTime      int64     `db:"end_time" bson:"end_time"`
	CreatedAt    time.Time `db:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" bson:"updated_at"`
}

func (Auction) TableName() string {
	return "`auctions`"
}
