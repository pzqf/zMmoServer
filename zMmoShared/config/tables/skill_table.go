package tables

import (
	"github.com/pzqf/zMmoShared/config/models"
)

// SkillTableLoader 技能表格加载器
// 负责从Excel加载技能配置数据
type SkillTableLoader struct {
	skills map[int32]*models.Skill // 技能配置映射（技能ID -> 配置）
}

// NewSkillTableLoader 创建技能表格加载器
// 返回: 初始化后的技能表加载器实例
func NewSkillTableLoader() *SkillTableLoader {
	return &SkillTableLoader{
		skills: make(map[int32]*models.Skill),
	}
}

// Load 加载技能表数据
// 从skill.xlsx文件读取技能配置
// 参数:
//   - dir: Excel文件所在目录
//
// 返回: 加载错误
func (stl *SkillTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "skill.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 25,
		TableName:  "skills",
	}

	// 使用临时map批量加载数据
	tempSkills := make(map[int32]*models.Skill)

	err := ReadExcelFile(config, dir, func(row []string) error {
		// 确保行数据足够长
		for len(row) < 25 {
			row = append(row, "")
		}

		skill := &models.Skill{
			SkillID:              StrToInt32(row[0]),
			Name:                 row[1],
			Type:                 StrToInt32(row[2]),
			Level:                StrToInt32(row[3]),
			ManaCost:             StrToInt32(row[4]),
			Cooldown:             StrToFloat32(row[5]),
			Damage:               StrToInt32(row[6]),
			Range:                StrToFloat32(row[7]),
			AreaRadius:           StrToFloat32(row[8]),
			Description:          row[9],
			Effects:              row[10],
			DamageType:           row[11],
			EffectType:           row[12],
			CooldownGrowth:       StrToFloat32(row[13]),
			DamageGrowth:         StrToFloat32(row[14]),
			RangeGrowth:          StrToFloat32(row[15]),
			RequiredLevel:        StrToInt32(row[16]),
			AnimationID:          StrToInt32(row[17]),
			SoundID:              StrToInt32(row[18]),
			IconID:               StrToInt32(row[19]),
			PreSkillID:           StrToInt32(row[20]),
			BuffID:               StrToInt32(row[21]),
			SkillCastTime:        StrToFloat32(row[22]),
			SkillProjectileSpeed: StrToFloat32(row[23]),
		}

		tempSkills[skill.SkillID] = skill
		return nil
	})

	// 加载完成后一次性赋值
	if err == nil {
		stl.skills = tempSkills
	}

	return err
}

// GetTableName 获取表格名称
// 返回: 表格名称"skills"
func (stl *SkillTableLoader) GetTableName() string {
	return "skills"
}

// GetSkill 根据ID获取技能
// 参数:
//   - skillID: 技能ID
//
// 返回: 技能配置和是否存在
func (stl *SkillTableLoader) GetSkill(skillID int32) (*models.Skill, bool) {
	skill, ok := stl.skills[skillID]
	return skill, ok
}

// GetAllSkills 获取所有技能
// 返回配置的副本map，避免外部修改内部数据
// 返回: 技能配置映射副本
func (stl *SkillTableLoader) GetAllSkills() map[int32]*models.Skill {
	// 创建一个副本，避免外部修改内部数据
	skillsCopy := make(map[int32]*models.Skill, len(stl.skills))
	for id, skill := range stl.skills {
		skillsCopy[id] = skill
	}
	return skillsCopy
}
