package protocol

// 消息ID常量
const (
	// 基础消息
	MsgIdAccountCreate = 1001 // 账号创建
	MsgIdAccountLogin  = 1002 // 账号登录
	MsgIdPlayerCreate  = 1003 // 角色创建
	MsgIdPlayerLogin   = 1004 // 角色登录
	MsgIdPlayerLogout  = 1005 // 角色登出
	MsgIdPlayerMove    = 1006 // 角色移动

	// 活动系统
	MsgIdActivityList     = 2001 // 活动列表
	MsgIdActivityJoin     = 2002 // 参与活动
	MsgIdActivityProgress = 2003 // 活动进度
	MsgIdActivityClaim    = 2004 // 领取奖励

	// 商城系统
	MsgIdShopItemList     = 3001 // 商品列表
	MsgIdShopCategoryList = 3002 // 分类列表
	MsgIdShopBuy          = 3003 // 购买商品
	MsgIdShopHotItems     = 3004 // 热销商品

	// 副本系统
	MsgIdDungeonList     = 4001 // 副本列表
	MsgIdDungeonEnter    = 4002 // 进入副本
	MsgIdDungeonComplete = 4003 // 完成副本
	MsgIdDungeonFail     = 4004 // 副本失败

	// Buff系统
	MsgIdBuffList   = 5001 // Buff列表
	MsgIdBuffApply  = 5002 // 施加Buff
	MsgIdBuffRemove = 5003 // 移除Buff
	MsgIdBuffDispel = 5004 // 驱散Buff

	// 掉落系统
	MsgIdDropPickup = 6001 // 拾取掉落
	MsgIdDropList   = 6002 // 掉落列表

	// 地图系统
	MsgIdMapEnter       = 4001 // 进入地图
	MsgIdMapLeave       = 4002 // 离开地图
	MsgIdMapMove        = 4003 // 地图移动
	MsgIdMapGetPath     = 4004 // 获取路径
	MsgIdMapGetObjects  = 4005 // 获取地图对象
	MsgIdMapSyncObjects = 4006 // 同步地图对象
	MsgIdAoiEnterNotify = 4007 // AOI进入通知
	MsgIdAoiLeaveNotify = 4008 // AOI离开通知
	MsgIdMapMoveSync    = 4009 // 地图移动同步

	// 聊天系统
	MsgIdChatSend    = 4101 // 发送聊天消息
	MsgIdChatReceive = 4102 // 接收聊天消息
	MsgIdChatNearby  = 4103 // 附近聊天
	MsgIdChatMap     = 4104 // 地图聊天
	MsgIdChatWorld   = 4105 // 世界聊天
	MsgIdChatGuild   = 4106 // 公会聊天
	MsgIdChatTeam    = 4107 // 队伍聊天
	MsgIdChatPrivate = 4108 // 私聊

	// 技能系统
	MsgIdSkillUse           = 7001 // 释放技能
	MsgIdSkillUseResult     = 7002 // 技能释放结果
	MsgIdSkillCooldown      = 7003 // 技能冷却状态
	MsgIdSkillList          = 7004 // 技能列表
	MsgIdSkillUpgrade       = 7005 // 技能升级
	MsgIdSkillUpgradeResult = 7006 // 技能升级结果
	MsgIdSkillUnlock        = 7007 // 技能解锁
	MsgIdSkillUnlockResult  = 7008 // 技能解锁结果
)
