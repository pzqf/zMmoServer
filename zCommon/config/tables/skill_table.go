package tables

import (
	"github.com/pzqf/zCommon/config/models"
)

// SkillTableLoader 技能表加载器
// 从Excel表加载技能配置数据
type SkillTableLoader struct {
	skills map[int32]*models.Skill // 技能配置映射（技能ID -> 配置）
}

// NewSkillTableLoader 创建技能表加载器
// 功能: 初始化技能表加载器实例
func NewSkillTableLoader() *SkillTableLoader {
	return &SkillTableLoader{
		skills: make(map[int32]*models.Skill),
	}
}

// Load ���ؼ��ܱ�����
// ��skill.xlsx�ļ���ȡ��������
// ����:
//   - dir: Excel�ļ�����Ŀ¼
//
// ����: ���ش���
func (stl *SkillTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "skill.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 10,
		TableName:  "skills",
	}

	// ʹ����ʱmap������������
	tempSkills := make(map[int32]*models.Skill)

	err := ReadExcelFile(config, dir, func(row []string) error {
		// ȷ���������㹻��
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

	// ������ɺ�һ���Ը�ֵ
	if err == nil {
		stl.skills = tempSkills
	}

	return err
}

// GetTableName ��ȡ��������
// ����: ��������"skills"
func (stl *SkillTableLoader) GetTableName() string {
	return "skills"
}

// GetSkill ����ID��ȡ����
// ����:
//   - skillID: ����ID
//
// ����: �������ú��Ƿ����
func (stl *SkillTableLoader) GetSkill(skillID int32) (*models.Skill, bool) {
	skill, ok := stl.skills[skillID]
	return skill, ok
}

// GetAllSkills ��ȡ���м���
// �������õĸ���map�������ⲿ�޸��ڲ�����
// ����: ��������ӳ�丱��
func (stl *SkillTableLoader) GetAllSkills() map[int32]*models.Skill {
	// ����һ�������������ⲿ�޸��ڲ�����
	skillsCopy := make(map[int32]*models.Skill, len(stl.skills))
	for id, skill := range stl.skills {
		skillsCopy[id] = skill
	}
	return skillsCopy
}

