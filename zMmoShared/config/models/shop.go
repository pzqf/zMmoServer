package models

// Shop 商店配置结构
type Shop struct {
	ShopID       int32 `json:"shop_id"`       // 商店ID
	ItemID       int32 `json:"item_id"`       // 商品ID
	Price        int32 `json:"price"`         // 商品价格
	CurrencyType int32 `json:"currency_type"` // 货币类型（1:金币, 2:钻石）
	Stock        int32 `json:"stock"`         // 库存数量
	LimitPerDay  int32 `json:"limit_per_day"` // 每日购买限制
	MinLevel     int32 `json:"min_level"`     // 购买最低等级
	ShopType     int32 `json:"shop_type"`     // 商店类型（1:普通, 2:稀有, 3:活动）
}
