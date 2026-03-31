package models

// Monster 怪物配置结构
type Monster struct {
	MonsterID    int32   `json:"monster_id"`     // 怪物ID
	Name         string  `json:"name"`           // 怪物名称
	Level        int32   `json:"level"`          // 怪物等级
	HP           int32   `json:"hp"`             // 生命值
	MP           int32   `json:"mp"`             // 魔法值
	Attack       int32   `json:"attack"`         // 攻击力
	Defense      int32   `json:"defense"`        // 防御力
	Speed        int32   `json:"speed"`          // 移动速度
	Exp          int32   `json:"exp"`            // 击败获得经验
	DropItemRate float32 `json:"drop_item_rate"` // 物品掉落率
	DropItems    string  `json:"drop_items"`     // 掉落物品列表（JSON格式）
}
