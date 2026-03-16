package db

// RepoType Repository类型
type RepoType string

const (
	// RepoTypeAccount 账号Repository
	RepoTypeAccount RepoType = "account"
	// RepoTypePlayer 玩家Repository
	RepoTypePlayer RepoType = "player"
	// RepoTypePlayerItem 玩家物品Repository
	RepoTypePlayerItem RepoType = "player_item"
	// RepoTypePlayerSkill 玩家技能Repository
	RepoTypePlayerSkill RepoType = "player_skill"
	// RepoTypePlayerMail 玩家邮件Repository
	RepoTypePlayerMail RepoType = "player_mail"
	// RepoTypePlayerQuest 玩家任务Repository
	RepoTypePlayerQuest RepoType = "player_quest"
	// RepoTypePlayerPet 玩家宠物Repository
	RepoTypePlayerPet RepoType = "player_pet"
	// RepoTypePlayerBuff 玩家BuffRepository
	RepoTypePlayerBuff RepoType = "player_buff"
	// RepoTypeGuild 公会Repository
	RepoTypeGuild RepoType = "guild"
	// RepoTypeGuildMember 公会成员Repository
	RepoTypeGuildMember RepoType = "guild_member"
	// RepoTypeAuction 拍卖行Repository
	RepoTypeAuction RepoType = "auction"
	// RepoTypeLoginLog 登录日志Repository
	RepoTypeLoginLog RepoType = "login_log"
	// RepoTypeMailLog 邮件日志Repository
	RepoTypeMailLog RepoType = "mail_log"
	// RepoTypeQuestLog 任务日志Repository
	RepoTypeQuestLog RepoType = "quest_log"
	// RepoTypeAuctionLog 拍卖日志Repository
	RepoTypeAuctionLog RepoType = "auction_log"
	// RepoTypeGameServer 游戏服务器Repository
	RepoTypeGameServer RepoType = "game_server"
)

// RepoTypeAll 所有Repository类型
var RepoTypeAll = []RepoType{
	RepoTypeAccount,
	RepoTypePlayer,
	RepoTypePlayerItem,
	RepoTypePlayerSkill,
	RepoTypePlayerMail,
	RepoTypePlayerQuest,
	RepoTypePlayerPet,
	RepoTypePlayerBuff,
	RepoTypeGuild,
	RepoTypeGuildMember,
	RepoTypeAuction,
	RepoTypeLoginLog,
	RepoTypeMailLog,
	RepoTypeQuestLog,
	RepoTypeAuctionLog,
	RepoTypeGameServer,
}

// RepoTypeGlobalServer GlobalServer需要的Repository类型
var RepoTypeGlobalServer = []RepoType{
	RepoTypeAccount,
	RepoTypeGameServer,
	RepoTypeLoginLog,
}
