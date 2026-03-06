package tables

import (
	"github.com/pzqf/zMmoShared/config/models"
)

// GuildTableLoader 公会表加载器
// 负责从Excel加载公会等级配置数据
type GuildTableLoader struct {
	guilds map[int32]*models.Guild // 公会配置映射（等级 -> 配置）
}

// NewGuildTableLoader 创建公会表加载器
// 返回: 初始化后的公会表加载器实例
func NewGuildTableLoader() *GuildTableLoader {
	return &GuildTableLoader{
		guilds: make(map[int32]*models.Guild),
	}
}

// Load 加载公会表数据
// 从guild.xlsx文件读取公会等级配置
// 参数:
//   - dir: Excel文件所在目录
//
// 返回: 加载错误
func (gtl *GuildTableLoader) Load(dir string) error {
	config := ExcelConfig{
		FileName:   "guild.xlsx",
		SheetName:  "Sheet1",
		MinColumns: 7,
		TableName:  "guilds",
	}

	// 使用临时map批量加载数据
	tempGuilds := make(map[int32]*models.Guild)

	err := ReadExcelFile(config, dir, func(row []string) error {
		guild := &models.Guild{
			GuildLevel:    StrToInt32(row[0]),
			RequiredExp:   StrToInt64(row[1]),
			MaxMembers:    StrToInt32(row[2]),
			BuildingSlots: StrToInt32(row[3]),
			TaxRate:       StrToFloat32(row[4]),
			SkillPoints:   StrToInt32(row[5]),
		}

		tempGuilds[guild.GuildLevel] = guild
		return nil
	})

	// 加载完成后一次性赋值
	if err == nil {
		gtl.guilds = tempGuilds
	}

	return err
}

// GetTableName 获取表格名称
// 返回: 表格名称"guilds"
func (gtl *GuildTableLoader) GetTableName() string {
	return "guilds"
}

// GetGuild 根据等级获取公会配置
// 参数:
//   - guildLevel: 公会等级
//
// 返回: 公会配置和是否存在
func (gtl *GuildTableLoader) GetGuild(guildLevel int32) (*models.Guild, bool) {
	guild, ok := gtl.guilds[guildLevel]
	return guild, ok
}

// GetAllGuilds 获取所有公会配置
// 返回配置的副本map，避免外部修改内部数据
// 返回: 公会配置映射副本
func (gtl *GuildTableLoader) GetAllGuilds() map[int32]*models.Guild {
	// 创建一个副本，避免外部修改内部数据
	guildsCopy := make(map[int32]*models.Guild, len(gtl.guilds))
	for id, guild := range gtl.guilds {
		guildsCopy[id] = guild
	}
	return guildsCopy
}
