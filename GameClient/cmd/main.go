package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pzqf/zMmoServer/GameClient/internal/client"
	"github.com/pzqf/zMmoServer/GameClient/internal/msg/handler"
	"github.com/pzqf/zMmoServer/GameClient/internal/test/concurrency"
	"github.com/pzqf/zMmoServer/GameClient/internal/test/loadtest"
	"github.com/pzqf/zMmoServer/GameClient/internal/test/longtest"
)

func main() {
	// 命令行参数
	mode := flag.String("mode", "full", "测试模式: full, gateway-only, global-only, concurrency, long-test")
	globalServer := flag.String("global", "127.0.0.1:8888", "GlobalServer地址")
	gatewayServer := flag.String("gateway", "127.0.0.1:10001", "GatewayServer地址")
	account := flag.String("account", "tester1", "测试账号")
	password := flag.String("password", "123456", "测试密码")
	playerName := flag.String("name", "测试角色1", "角色名称")
	clientCount := flag.Int("clients", 10, "并发客户端数量")
	messageCount := flag.Int("messages", 100, "每个客户端发送的消息数")
	testDuration := flag.Int("duration", 30, "长时测试持续时间(秒)")
	// 场景同步增强 E2E：指定攻击目标(默认 2)、攻击次数、以及流程结束后驻留秒数(供 PvP 受害端在场接收广播)。
	attackTarget := flag.Int64("attackTarget", 2, "攻击目标对象ID(玩家 objectID==playerID)")
	attackCount := flag.Int("attackCount", 1, "攻击次数(击杀需多次)")
	idleSeconds := flag.Int("idle", 0, "流程末尾驻留秒数(保持在场接收 AOI 广播)")
	teamCreate := flag.Bool("teamCreate", false, "进图后创建队伍(队长)")
	teamJoin := flag.Int("teamJoin", 0, "进图后加入指定队伍ID(>0 生效)")
	tradeTarget := flag.Int64("tradeTarget", 0, "向该玩家发起交易(>0 生效,本方为发起者)")
	tradeGold := flag.Int64("tradeGold", 0, "交易愿出金币(>0 则开启自动应价,双方均需设置)")
	loginPlayerID := flag.Int64("loginPlayerID", 0, "复用已有角色ID登录(>0 跳过创建,用于持久化重登验证)")
	mailTo := flag.Int64("mailTo", 0, "发邮件给该玩家(>0 生效)")
	mailGold := flag.Int64("mailGold", 0, "邮件金币附件")
	mailClaim := flag.Bool("mailClaim", false, "登录后拉取邮件并自动领取")
	// 跨服活动（realm §5.1）：mode=cross 时进入指定活动的跨服实例（可能落到别的 realm 的 MapServer）。
	crossActivity := flag.Int64("crossActivity", 0, "跨服活动ID(mode=cross 必填)")
	crossMapConfig := flag.Int("crossMapConfig", 5001, "跨服活动使用的地图配置ID")
	// 闭环压测（mode=loadtest）：每客户端"发一条等一条"，量真实的服务端吞吐与延迟分位。
	loadOp := flag.String("loadOp", "move", "压测操作: move(不含MapServer往返) / attack(含MapServer往返)")
	loadClients := flag.String("loadClients", "1,2,4,8,16,32", "压测并发梯度(逗号分隔)")
	loadDuration := flag.Int("loadDuration", 10, "每档并发的压测时长(秒)")
	flag.Parse()

	fmt.Println("=== GameClient 启动 ===")
	fmt.Printf("测试模式: %s\n", *mode)
	fmt.Printf("GlobalServer: %s\n", *globalServer)
	fmt.Printf("GatewayServer: %s\n", *gatewayServer)
	fmt.Printf("账号: %s\n", *account)
	fmt.Println()

	switch *mode {
	case "full":
		runFullTest(*globalServer, *gatewayServer, *account, *password, *playerName, *attackTarget, *attackCount, *idleSeconds, *teamCreate, *teamJoin, *tradeTarget, *tradeGold, *loginPlayerID, *mailTo, *mailGold, *mailClaim)
	case "cross":
		runCrossTest(*globalServer, *gatewayServer, *account, *password, *playerName,
			*crossActivity, int32(*crossMapConfig), *idleSeconds)
	case "gateway-only":
		runGatewayOnlyTest(*gatewayServer)
	case "global-only":
		runGlobalOnlyTest(*globalServer, *account, *password)
	case "loadtest":
		runLoadTest(*globalServer, *gatewayServer, *loadOp, *loadClients, *loadDuration)
	case "concurrency":
		runConcurrencyTest(*gatewayServer, *clientCount, *messageCount)
	case "long-test":
		runLongTest(*gatewayServer, *clientCount, *testDuration)
	default:
		fmt.Printf("未知的测试模式: %s\n", *mode)
		os.Exit(1)
	}
}

// runCrossTest 跨服活动 E2E（realm §5.1）：登录本 realm → 进跨服活动实例 → 驻留观察 AOI。
//
// 两个不同 realm 各跑一个本进程，用同一个 -crossActivity，即可验证跨服物理路由：
// 两边打印的 MapServerID/MapID 应完全一致（= 落到同一台 MapServer 的同一张实例图），
// 且各自 [AOI] 视野里应出现对方的玩家 ID（= 真在同一张地图上，能互相看见）。
func runCrossTest(globalServer, gatewayServer, account, password, playerName string,
	activityID int64, mapConfigID int32, idleSeconds int) {
	fmt.Println("=== 跨服活动测试模式 ===")
	if activityID <= 0 {
		fmt.Println("必须指定 -crossActivity（跨服活动ID）")
		os.Exit(2)
	}

	c := client.NewClient(globalServer)

	// 注册（跨服 E2E 每次用全新账号：复用旧账号会因已有角色而取不到新角色ID）。
	fmt.Println("0. 注册账号...")
	if regResp, err := c.Register(account, password, account+"@test.local"); err != nil {
		fmt.Printf("注册请求失败: %v\n", err)
		os.Exit(1)
	} else if regResp.Result != 0 {
		fmt.Printf("注册返回: Result=%d ErrorMsg=%s（已存在则继续）\n", regResp.Result, regResp.ErrorMsg)
	}

	fmt.Println("1. 登录账号...")
	authResp, err := c.Login(account, password)
	if err != nil || authResp.Result != 0 {
		fmt.Printf("登录失败: %v %+v\n", err, authResp)
		os.Exit(1)
	}
	// 必须在 Login **之后**再定 Gateway：Login 会用服务器列表里的地址覆盖它，
	// 而跨 realm 测试要求两个客户端各自连**指定 realm** 的 Gateway，不能被全局列表带跑。
	c.SetGatewayAddr(gatewayServer)

	fmt.Println("2. 连接GatewayServer...")
	if err := c.Connect(); err != nil {
		fmt.Printf("连接失败: %v\n", err)
		os.Exit(1)
	}
	defer c.Disconnect()

	fmt.Println("3. 验证token...")
	if err := c.SendTokenVerify(c.GetToken()); err != nil {
		fmt.Printf("token验证发送失败: %v\n", err)
		os.Exit(1)
	}
	if result, ok := c.WaitTokenVerify(5 * time.Second); !ok || result != 0 {
		fmt.Printf("token验证失败: result=%d ok=%v\n", result, ok)
		os.Exit(1)
	}

	fmt.Println("4. 创建角色...")
	if err := c.SendPlayerCreate(playerName, 1, 18); err != nil {
		fmt.Printf("创建角色失败: %v\n", err)
		os.Exit(1)
	}
	playerID := c.GetCreatedPlayerID()
	if playerID == 0 {
		fmt.Println("未获取到角色ID")
		os.Exit(1)
	}
	fmt.Printf("角色ID: %d\n", playerID)

	fmt.Println("5. 进入游戏...")
	if err := c.SendPlayerLogin(playerID); err != nil {
		fmt.Printf("进入游戏发送失败: %v\n", err)
		os.Exit(1)
	}
	if result, ok := c.WaitPlayerLogin(5 * time.Second); !ok || result != 0 {
		fmt.Printf("进入游戏失败: result=%d ok=%v\n", result, ok)
		os.Exit(1)
	}

	fmt.Printf("6. 进入跨服活动实例 activity=%d mapConfig=%d ...\n", activityID, mapConfigID)
	if err := c.SendCrossEnter(playerID, activityID, mapConfigID); err != nil {
		fmt.Printf("跨服进图发送失败: %v\n", err)
		os.Exit(1)
	}
	// 服务端要跑 分配→连外域→建实例→进图，故等待放宽到 10s。
	result, ok := c.WaitCrossEnter(10 * time.Second)
	if !ok {
		fmt.Println("CROSS_ENTER_FAILED: 超时未收到跨服进图响应")
		os.Exit(1)
	}
	if result != 0 {
		fmt.Printf("CROSS_ENTER_FAILED: Result=%d\n", result)
		os.Exit(1)
	}
	mapServerID, mapID := c.CrossEnterInfo()
	// 这一行是 E2E 断言的锚点：两个 realm 的输出必须完全一致。
	fmt.Printf("CROSS_ENTER_OK playerID=%d mapServerID=%d mapID=%d\n", playerID, mapServerID, mapID)

	if idleSeconds > 0 {
		fmt.Printf("驻留 %ds 观察跨服实例内的 AOI 视野...\n", idleSeconds)
		time.Sleep(time.Duration(idleSeconds) * time.Second)
	}

	fmt.Println("7. 登出...")
	_ = c.SendPlayerLogout()
	time.Sleep(500 * time.Millisecond)
	fmt.Println("=== 跨服活动测试结束 ===")
}

// runFullTest 运行完整测试（全局服+网关服）
func runFullTest(globalServer, gatewayServer, account, password, playerName string, attackTarget int64, attackCount, idleSeconds int, teamCreate bool, teamJoin int, tradeTarget, tradeGold, loginPlayerID, mailTo, mailGold int64, mailClaim bool) {
	fmt.Println("=== 完整测试模式 ===")

	// 1. 连接GlobalServer，获取token
	c := client.NewClient(globalServer)
	c.SetGatewayAddr(gatewayServer)

	// 登录
	fmt.Println("1. 登录账号...")
	authResp, err := c.Login(account, password)
	if err != nil {
		fmt.Printf("登录失败: %v\n", err)
		return
	}
	if authResp.Result != 0 {
		fmt.Printf("登录失败: %s\n", authResp.ErrorMsg)
		return
	}
	fmt.Println("登录成功!")

	// 2. 选择服务器
	fmt.Println("2. 选择服务器...")
	if c.SelectedServer() == nil {
		fmt.Println("未找到可用服务器，使用命令行指定的Gateway地址")
	} else {
		fmt.Printf("选择服务器: %s\n", c.SelectedServer().ServerName)
	}

	// 3. 连接GatewayServer
	fmt.Println("3. 连接GatewayServer...")
	if err := c.Connect(); err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}
	fmt.Println("连接成功!")

	// 4. 验证token（关键屏障：发送后等 TOKEN_VERIFY_RESPONSE 且判 Result，取代盲等）
	fmt.Println("4. 验证token...")
	if err := c.SendTokenVerify(c.GetToken()); err != nil {
		fmt.Printf("token验证发送失败: %v\n", err)
		c.Disconnect()
		return
	}
	if result, ok := c.WaitTokenVerify(5 * time.Second); !ok {
		fmt.Println("token验证超时: 未收到 TOKEN_VERIFY_RESPONSE")
		c.Disconnect()
		return
	} else if result != 0 {
		fmt.Printf("token验证被拒绝: Result=%d\n", result)
		c.Disconnect()
		return
	}
	fmt.Println("token验证成功!")

	// 5. 创建 或 复用已有角色（-loginPlayerID>0 时跳过创建，直接登录该角色——用于持久化重登验证）
	var playerID int64
	if loginPlayerID > 0 {
		playerID = loginPlayerID
		// 复用分支跳过创建，须显式把 playerID 透传给 sender，
		// 否则后续 SendPlayerLogout(读 sender.playerID) 会发出 PlayerId=0，登出玩家 0。
		c.SetPlayerID(playerID)
		fmt.Printf("5. 复用已有角色ID: %d（跳过创建）\n", playerID)
	} else {
		fmt.Println("5. 创建角色...")
		if err := c.SendPlayerCreate(playerName, 1, 18); err != nil {
			fmt.Printf("创建角色失败: %v\n", err)
			c.Disconnect()
			return
		}
		// GetCreatedPlayerID 内部经 WaitForPlayerID 在 channel 上阻塞最多 5s 直到创建响应到达，
		// 并在拿到非 0 ID 时同步透传给 sender；无需再叠加轮询。
		playerID = c.GetCreatedPlayerID()
		if playerID == 0 {
			fmt.Println("警告: 未获取到创建的角色ID，使用默认值1")
			playerID = 1
		}
		fmt.Printf("使用角色ID: %d\n", playerID)
	}

	// 交易自动应价：若本方愿出金币，则一收到交易 OPEN 通知就设价+确认（事件驱动，双方各自开启即可成交）
	if tradeGold > 0 {
		c.EnableTradeAuto(playerID, tradeGold)
	}
	// 邮件自动领取：登录后拉取邮件列表，对未领邮件逐封领取
	if mailClaim {
		c.EnableMailAutoClaim(playerID)
	}

	// 7. 进入游戏（关键屏障：发送后等 PlayerLoginResponse 且判 Result）
	fmt.Println("6. 进入游戏...")
	if err := c.SendPlayerLogin(playerID); err != nil {
		fmt.Printf("进入游戏发送失败: %v\n", err)
		c.Disconnect()
		return
	}
	if result, ok := c.WaitPlayerLogin(5 * time.Second); !ok {
		fmt.Println("进入游戏超时: 未收到 PlayerLoginResponse")
		c.Disconnect()
		return
	} else if result != 0 {
		fmt.Printf("进入游戏被拒绝: Result=%d\n", result)
		c.Disconnect()
		return
	}

	// 9. 进入地图（关键屏障：发送后等 ClientMapEnterResponse 且判 Result）
	fmt.Println("7. 进入地图...")
	if err := c.SendMapEnter(playerID, 1001); err != nil {
		fmt.Printf("进入地图发送失败: %v\n", err)
		c.Disconnect()
		return
	}
	if result, ok := c.WaitMapEnter(5 * time.Second); !ok {
		fmt.Println("进入地图超时: 未收到 ClientMapEnterResponse")
		c.Disconnect()
		return
	} else if result != 0 {
		fmt.Printf("进入地图被拒绝: Result=%d\n", result)
		c.Disconnect()
		return
	}

	// 7.5 就近拾取掉落物（ZMMO_TEST_LOOT=1 时服务端在进图点旁种一件；趁玩家还在进图点、掉落物在拾取半径内）
	time.Sleep(600 * time.Millisecond) // 等进图落地 + 掉落种子生成
	fmt.Println("7.5 拾取附近掉落物...")
	_ = c.SendItemPickup(playerID, 1001)
	time.Sleep(800 * time.Millisecond)

	// 10. 移动
	fmt.Println("8. 移动...")
	if err := c.SendMapMove(playerID, 1001, 100.0, 100.0, 0.0); err != nil {
		fmt.Printf("移动失败: %v\n", err)
		c.Disconnect()
		return
	}

	// 8.5 世界聊天（业务层建设）：发一条世界频道，全服在线（含自己 + 其它在场客户端）都会收到 [聊天] 广播
	fmt.Println("8.5 发送世界聊天...")
	_ = c.SendChat(playerID, 0 /*CHAT_WORLD*/, "Hello 世界, from "+playerName)
	time.Sleep(600 * time.Millisecond)

	// 8.6 组队（业务层建设）：队长创建、队员加入；花名册变更经 UPDATE_NOTIFY 推给全体成员
	if teamCreate {
		fmt.Println("8.6 创建队伍...")
		_ = c.SendTeamCreate(playerID)
		time.Sleep(600 * time.Millisecond)
	}
	if teamJoin > 0 {
		fmt.Printf("8.6 加入队伍 team=%d...\n", teamJoin)
		_ = c.SendTeamJoin(playerID, int32(teamJoin))
		time.Sleep(600 * time.Millisecond)
	}

	// 8.7 交易（业务层建设）：发起者向目标发起交易；双方 tradeGold 已开启自动应价，事件驱动收敛到成交
	if tradeTarget > 0 {
		fmt.Printf("8.7 向 %d 发起交易(本方出 %d 金币)...\n", tradeTarget, tradeGold)
		_ = c.SendTradeStart(playerID, tradeTarget)
		time.Sleep(2500 * time.Millisecond) // 等自动应价来回收敛到成交
	}

	// 8.8 邮件（业务层建设）：发件（可发给离线玩家）/ 拉取并自动领取
	if mailTo > 0 {
		fmt.Printf("8.8 发邮件给 %d(附 %d 金币)...\n", mailTo, mailGold)
		_ = c.SendMail(playerID, mailTo, "测试邮件", "领点金币", mailGold)
		time.Sleep(500 * time.Millisecond)
	}
	if mailClaim {
		fmt.Println("8.8 拉取邮件并自动领取...")
		_ = c.SendMailList(playerID)
		time.Sleep(1500 * time.Millisecond) // 等列表→逐封领取往返
	}

	// 11. 攻击（attackTarget 默认 2；PvP 场景传对方 playerID。attackCount>1 用于打到击杀触发死亡广播）
	fmt.Printf("9. 攻击 target=%d x%d...\n", attackTarget, attackCount)
	for i := 0; i < attackCount; i++ {
		if err := c.SendMapAttack(playerID, 1001, attackTarget); err != nil {
			fmt.Printf("攻击失败: %v\n", err)
			c.Disconnect()
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	// 12. 等待响应
	time.Sleep(3 * time.Second)

	// —— 物品 / 仓库（业务层建设 2026-07-25，服务端 ZMMO_TEST_ITEMS 门控在进入游戏时种子几件物品）——
	fmt.Println("11. 查背包...")
	_ = c.SendItemList(playerID)
	time.Sleep(500 * time.Millisecond)

	fmt.Println("12. 使用背包槽0一件...")
	_ = c.SendItemUse(playerID, 0, 1)
	time.Sleep(500 * time.Millisecond)

	fmt.Println("13. 背包槽1整格存入仓库...")
	_ = c.SendWarehouseStore(playerID, 1, 5)
	time.Sleep(500 * time.Millisecond)

	fmt.Println("14. 查仓库...")
	_ = c.SendWarehouseList(playerID)
	time.Sleep(500 * time.Millisecond)

	fmt.Println("15. 从仓库槽0取回2件到背包...")
	_ = c.SendWarehouseRetrieve(playerID, 0, 2)
	time.Sleep(500 * time.Millisecond)

	fmt.Println("16. 再查背包...")
	_ = c.SendItemList(playerID)
	time.Sleep(800 * time.Millisecond)

	// —— 技能（业务层建设 2026-07-25）——
	fmt.Println("17. 学习技能 2001...")
	_ = c.SendSkillLearn(playerID, 2001)
	time.Sleep(400 * time.Millisecond)

	fmt.Println("18. 查技能...")
	_ = c.SendSkillList(playerID)
	time.Sleep(400 * time.Millisecond)

	fmt.Println("19. 升级技能 2001...")
	_ = c.SendSkillUpgrade(playerID, 2001)
	time.Sleep(400 * time.Millisecond)

	fmt.Println("20. 释放技能 2001(目标2)...")
	_ = c.SendSkillCast(playerID, 2001, 2)
	time.Sleep(400 * time.Millisecond)

	fmt.Println("21. 再释放一次(应在冷却)...")
	_ = c.SendSkillCast(playerID, 2001, 2)
	time.Sleep(600 * time.Millisecond)

	// 驻留：保持在场以接收其他玩家攻击我方时的 AOI 血量/死亡广播
	if idleSeconds > 0 {
		fmt.Printf("驻留 %ds 接收 AOI 广播...\n", idleSeconds)
		time.Sleep(time.Duration(idleSeconds) * time.Second)
	}

	// 13. 登出
	fmt.Println("10. 登出...")
	if err := c.SendPlayerLogout(); err != nil {
		fmt.Printf("登出失败: %v\n", err)
		c.Disconnect()
		return
	}

	// 12. 断开连接
	time.Sleep(1 * time.Second)
	c.Disconnect()

	fmt.Println()
	fmt.Println("=== 完整测试完成 ===")
}

// runGatewayOnlyTest 运行仅网关服测试
func runGatewayOnlyTest(gatewayServer string) {
	fmt.Println("=== 仅网关服测试模式 ===")

	// 1. 连接GatewayServer
	c := client.NewClient("")
	c.SetGatewayAddr(gatewayServer)

	fmt.Println("1. 连接GatewayServer...")
	if err := c.Connect(); err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}
	fmt.Println("连接成功!")

	// 2. 验证token（使用测试token）
	fmt.Println("2. 验证token...")
	testToken := "test_token_123456"
	if err := c.SendTokenVerify(testToken); err != nil {
		fmt.Printf("token验证失败: %v\n", err)
		c.Disconnect()
		return
	}
	fmt.Println("token验证成功!")

	// 3. 发送心跳
	fmt.Println("3. 发送心跳...")
	if err := c.SendHeartbeat(); err != nil {
		fmt.Printf("发送心跳失败: %v\n", err)
		c.Disconnect()
		return
	}
	fmt.Println("心跳发送成功!")

	// 4. 等待响应
	time.Sleep(2 * time.Second)

	// 5. 断开连接
	c.Disconnect()

	fmt.Println()
	fmt.Println("=== 仅网关服测试完成 ===")
}

// runGlobalOnlyTest 运行仅全局服测试
func runGlobalOnlyTest(globalServer, account, password string) {
	fmt.Println("=== 仅全局服测试模式 ===")

	// 1. 连接GlobalServer
	c := client.NewClient(globalServer)

	// 2. 登录
	fmt.Println("1. 登录账号...")
	authResp, err := c.Login(account, password)
	if err != nil {
		fmt.Printf("登录失败: %v\n", err)
		return
	}
	if authResp.Result != 0 {
		fmt.Printf("登录失败: %s\n", authResp.ErrorMsg)
		return
	}
	fmt.Println("登录成功!")

	// 3. 获取服务器列表
	fmt.Println("2. 获取服务器列表...")
	serverList, err := c.GetServerList()
	if err != nil {
		fmt.Printf("获取服务器列表失败: %v\n", err)
		return
	}
	if serverList.Result != 0 {
		fmt.Printf("获取服务器列表失败: %s\n", serverList.ErrorMsg)
		return
	}
	fmt.Printf("服务器数量: %d\n", len(serverList.Servers))
	for i, server := range serverList.Servers {
		fmt.Printf("  %d. %s (ID: %d, 状态: %d)\n", i+1, server.ServerName, server.ServerId, server.Status)
	}

	// 4. 注册新账号（可选）
	fmt.Println("3. 注册新账号...")
	registerResp, err := c.Register("new_test_account", "123456", "test@example.com")
	if err != nil {
		fmt.Printf("注册失败: %v\n", err)
	} else {
		if registerResp.Result == 0 {
			fmt.Println("注册成功!")
		} else {
			fmt.Printf("注册失败: %s\n", registerResp.ErrorMsg)
		}
	}

	fmt.Println()
	fmt.Println("=== 仅全局服测试完成 ===")
}

// runConcurrencyTest 运行并发测试
func runConcurrencyTest(gatewayServer string, clientCount, messageCount int) {
	fmt.Println("=== 并发测试模式 ===")
	concurrency.RunConcurrencyTest(gatewayServer, clientCount, messageCount, 1, 1001, 2)
}

// runLoadTest 闭环压测：按并发梯度逐档跑，输出吞吐与延迟分位。
//
// 看什么：**吞吐是否随并发上升**。若并发翻倍而吞吐不动、延迟等比例变长，
// 说明服务端存在串行瓶颈（请求在某处排队），这正是要量出来的东西。
func runLoadTest(globalServer, gatewayServer, op, clientsSpec string, durationSec int) {
	fmt.Println("=== 闭环压测模式 ===")
	// 压测期间关掉客户端的人类可读输出，否则量的是终端刷屏速度。
	handler.SetQuiet(true)

	var grades []int
	for _, s := range strings.Split(clientsSpec, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n <= 0 {
			fmt.Printf("无效的并发档位: %q\n", s)
			return
		}
		grades = append(grades, n)
	}

	runStamp := time.Now().Unix()
	fmt.Printf("操作=%s 并发梯度=%v 每档时长=%ds 账号批次=%d\n\n", op, grades, durationSec, runStamp)

	var results []*loadtest.Result
	for i, n := range grades {
		res, err := loadtest.Run(loadtest.Config{
			GlobalAddr:  globalServer,
			GatewayAddr: gatewayServer,
			Clients:     n,
			Duration:    time.Duration(durationSec) * time.Second,
			Op:          loadtest.Op(op),
			MapID:       1001,
			// 账号前缀带**本次运行的时间戳**：复用上一轮的账号会因该账号已有角色而建号失败
			// （拿不到 playerID），压测直接跑不起来。每轮全新账号最省事。
			AccountPfx: fmt.Sprintf("l%d%s%d", runStamp, op, i),
		})
		if err != nil {
			fmt.Printf("并发=%d 档位失败: %v\n", n, err)
			continue
		}
		results = append(results, res)
		res.Print()
	}

	fmt.Println("\n=== 汇总 ===")
	fmt.Printf("%-8s %-12s %-12s %-12s %-12s\n", "并发", "吞吐(次/秒)", "p50", "p95", "p99")
	for _, r := range results {
		fmt.Printf("%-8d %-12.1f %-12v %-12v %-12v\n", r.Clients, r.Throughput, r.P50, r.P95, r.P99)
	}
	if len(results) >= 2 {
		first, last := results[0], results[len(results)-1]
		scale := last.Throughput / first.Throughput
		concurrencyScale := float64(last.Clients) / float64(first.Clients)
		fmt.Printf("\n并发放大 %.0f 倍 → 吞吐放大 %.2f 倍（理想为接近并发放大倍数；接近 1 = 服务端串行）\n",
			concurrencyScale, scale)
	}
}

// runLongTest 运行长时测试
func runLongTest(gatewayServer string, clientCount, testDuration int) {
	fmt.Println("=== 长时测试模式 ===")
	longtest.RunLongTest(gatewayServer, clientCount, time.Duration(testDuration)*time.Second, 10*time.Second, 1, 1001, 2)
}
