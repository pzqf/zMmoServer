package models

// ItemBase 物品配置结构
// 定义物品的基础属性，从Excel配置表加载
type ItemBase struct {
	ItemID      int32  `json:"item_id"`     // 物品ID
	Name        string `json:"name"`        // 物品名称
	Type        int32  `json:"type"`        // 物品类型（武器/防具/消耗品等）
	SubType     int32  `json:"sub_type"`    // 物品子类型
	Level       int32  `json:"level"`       // 物品等级
	Quality     int32  `json:"quality"`     // 品质等级（1-白色, 2-绿色, 3-蓝色, 4-紫色, 5-橙色）
	Price       int32  `json:"price"`       // 购买价格
	SellPrice   int32  `json:"sell_price"`  // 出售价格
	StackLimit  int32  `json:"stack_limit"` // 最大堆叠数量
	Description string `json:"description"` // 物品描述
	Effects     string `json:"effects"`     // JSON格式的效果描述
}
