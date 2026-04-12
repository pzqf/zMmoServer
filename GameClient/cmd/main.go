package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/pzqf/zMmoServer/GameClient/internal/client"
	"github.com/pzqf/zMmoServer/GameClient/internal/test/concurrency"
	"github.com/pzqf/zMmoServer/GameClient/internal/test/longtest"
)

func main() {
	// 命令行参数
	mode := flag.String("mode", "full", "测试模式: full, gateway-only, global-only, concurrency, long-test")
	globalServer := flag.String("global", "192.168.1.222:8080", "GlobalServer地址")
	gatewayServer := flag.String("gateway", "192.168.1.222:21001", "GatewayServer地址")
	account := flag.String("account", "tester1", "测试账号")
	password := flag.String("password", "123456", "测试密码")
	playerName := flag.String("name", "测试角色1", "角色名称")
	clientCount := flag.Int("clients", 10, "并发客户端数量")
	messageCount := flag.Int("messages", 100, "每个客户端发送的消息数")
	testDuration := flag.Int("duration", 30, "长时测试持续时间(秒)")
	flag.Parse()

	fmt.Println("=== GameClient 启动 ===")
	fmt.Printf("测试模式: %s\n", *mode)
	fmt.Printf("GlobalServer: %s\n", *globalServer)
	fmt.Printf("GatewayServer: %s\n", *gatewayServer)
	fmt.Printf("账号: %s\n", *account)
	fmt.Println()

	switch *mode {
	case "full":
		runFullTest(*globalServer, *gatewayServer, *account, *password, *playerName)
	case "gateway-only":
		runGatewayOnlyTest(*gatewayServer)
	case "global-only":
		runGlobalOnlyTest(*globalServer, *account, *password)
	case "concurrency":
		runConcurrencyTest(*gatewayServer, *clientCount, *messageCount)
	case "long-test":
		runLongTest(*gatewayServer, *clientCount, *testDuration)
	default:
		fmt.Printf("未知的测试模式: %s\n", *mode)
		os.Exit(1)
	}
}

// runFullTest 运行完整测试（全局服+网关服）
func runFullTest(globalServer, gatewayServer, account, password, playerName string) {
	fmt.Println("=== 完整测试模式 ===")

	// 1. 连接GlobalServer，获取token
	c := client.NewClient(globalServer)

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
		fmt.Println("未找到可用服务器")
		return
	}
	fmt.Printf("选择服务器: %s\n", c.SelectedServer().ServerName)

	// 3. 连接GatewayServer
	fmt.Println("3. 连接GatewayServer...")
	if err := c.Connect(); err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}
	fmt.Println("连接成功!")

	// 4. 验证token
	fmt.Println("4. 验证token...")
	if err := c.SendTokenVerify(c.GetToken()); err != nil {
		fmt.Printf("token验证失败: %v\n", err)
		c.Disconnect()
		return
	}
	fmt.Println("token验证成功!")

	// 5. 进入游戏（创建角色或登录已有角色）
	fmt.Println("5. 进入游戏...")
	// 这里简化处理，直接发送进入游戏请求
	// 实际情况可能需要先检查是否有角色，没有则创建
	if err := c.SendPlayerCreate(playerName, 1, 18); err != nil {
		fmt.Printf("创建角色失败: %v\n", err)
		c.Disconnect()
		return
	}

	// 6. 等待角色创建完成
	time.Sleep(2 * time.Second)

	// 7. 进入地图
	fmt.Println("6. 进入地图...")
	if err := c.SendMapEnter(1, 1001); err != nil {
		fmt.Printf("进入地图失败: %v\n", err)
		c.Disconnect()
		return
	}

	// 8. 移动
	fmt.Println("7. 移动...")
	if err := c.SendMapMove(1, 1001, 100.0, 100.0, 0.0); err != nil {
		fmt.Printf("移动失败: %v\n", err)
		c.Disconnect()
		return
	}

	// 9. 攻击
	fmt.Println("8. 攻击...")
	if err := c.SendMapAttack(1, 1001, 2); err != nil {
		fmt.Printf("攻击失败: %v\n", err)
		c.Disconnect()
		return
	}

	// 10. 等待响应
	time.Sleep(3 * time.Second)

	// 11. 登出
	fmt.Println("9. 登出...")
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

// runLongTest 运行长时测试
func runLongTest(gatewayServer string, clientCount, testDuration int) {
	fmt.Println("=== 长时测试模式 ===")
	longtest.RunLongTest(gatewayServer, clientCount, time.Duration(testDuration)*time.Second, 10*time.Second, 1, 1001, 2)
}
