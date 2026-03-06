package models

// Guild 公会配置结构
type Guild struct {
	GuildLevel    int32   `json:"guild_level"`    // 公会等级
	RequiredExp   int64   `json:"required_exp"`   // 升级所需经验
	MaxMembers    int32   `json:"max_members"`    // 最大成员数
	BuildingSlots int32   `json:"building_slots"` // 建筑槽位数量
	TaxRate       float32 `json:"tax_rate"`       // 公会税率
	SkillPoints   int32   `json:"skill_points"`   // 可用技能点
}
