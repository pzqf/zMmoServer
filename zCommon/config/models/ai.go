package models

// AI AI配置模型
type AI struct {
	AIID           int32   // AI ID
	Type           string  // AI类型：怪物、NPC
	DetectionRange float32 // 检测范围
	AttackRange    float32 // 攻击范围
	ChaseRange     float32 // 追击范围
	FleeHealth     float32 // 逃跑生命值阈值
	PatrolPoints   string  // 巡逻点，格式：x1,y1,z1;x2,y2,z2;...
	Behavior       string  // 行为模式：被动、主动、混合
	SkillIDs       string  // 技能ID列表，格式：id1,id2,id3
}
