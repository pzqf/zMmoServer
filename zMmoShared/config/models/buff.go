package models

// Buff buff配置模型
type Buff struct {
	BuffID      int32  // Buff ID
	Name        string // Buff名称
	Description string // Buff描述
	Type        string // Buff类型：增益、减益、中性
	Duration    int32  // 持续时间（秒）
	Value       int32  // Buff值
	Property    string // 影响的属性
	IsPermanent bool   // 是否永久
}
