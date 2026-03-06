package models

// Pet 宠物配置结构
type Pet struct {
	PetID        int32   `json:"pet_id"`        // 宠物ID
	Name         string  `json:"name"`          // 宠物名称
	Type         int32   `json:"type"`          // 宠物类型（1:攻击型, 2:防御型, 3:辅助型）
	BaseHP       int32   `json:"base_hp"`       // 基础生命值
	BaseAttack   int32   `json:"base_attack"`   // 基础攻击力
	BaseDefense  int32   `json:"base_defense"`  // 基础防御力
	GrowthRate   float32 `json:"growth_rate"`   // 成长率
	SkillID      int32   `json:"skill_id"`      // 技能ID
	ObtainMethod string  `json:"obtain_method"` // 获得方法
	Rarity       int32   `json:"rarity"`        // 稀有度（1-5）
}
