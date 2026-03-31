package models

// Skill 技能配置结构
type Skill struct {
	SkillID              int32   `json:"skill_id"`
	Name                 string  `json:"name"`
	Type                 int32   `json:"type"`
	Level                int32   `json:"level"`
	ManaCost             int32   `json:"mana_cost"`
	Cooldown             float32 `json:"cooldown"`
	Damage               int32   `json:"damage"`
	Range                float32 `json:"range"`
	AreaRadius           float32 `json:"area_radius"`
	Description          string  `json:"description"`
	Effects              string  `json:"effects"`                // JSON格式的效果描述
	DamageType           string  `json:"damage_type"`            // 伤害类型：物理、魔法、真实
	EffectType           string  `json:"effect_type"`            // 效果类型：伤害、治疗、增益、减益
	CooldownGrowth       float32 `json:"cooldown_growth"`        // 冷却时间增长系数（每级）
	DamageGrowth         float32 `json:"damage_growth"`          // 伤害增长系数（每级）
	RangeGrowth          float32 `json:"range_growth"`           // 范围增长系数（每级）
	RequiredLevel        int32   `json:"required_level"`         // 技能所需等级
	AnimationID          int32   `json:"animation_id"`           // 技能动画ID
	SoundID              int32   `json:"sound_id"`               // 技能音效ID
	IconID               int32   `json:"icon_id"`                // 技能图标ID
	PreSkillID           int32   `json:"pre_skill_id"`           // 技能前置技能ID
	BuffID               int32   `json:"buff_id"`                // 技能附加的buff ID
	SkillCastTime        float32 `json:"skill_cast_time"`        // 技能施法时间
	SkillProjectileSpeed float32 `json:"skill_projectile_speed"` // 技能 projectile速度
}
