package models

// PlayerLevel 人物等级配置结构
type PlayerLevel struct {
	LevelID      int32   `json:"level_id"`      // 等级ID
	RequiredExp  int64   `json:"required_exp"`  // 升级所需经验
	HP           int32   `json:"hp"`            // 生命值上限
	MP           int32   `json:"mp"`            // 魔法值上限
	Attack       int32   `json:"attack"`        // 攻击力
	Defense      int32   `json:"defense"`       // 防御力
	CriticalRate float32 `json:"critical_rate"` // 暴击率
	SkillPoints  int32   `json:"skill_points"`  // 获得的技能点
}
