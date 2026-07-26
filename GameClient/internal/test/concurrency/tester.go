package concurrency

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zMmoServer/GameClient/internal/test/common"
)

// ConcurrencyTestConfig 并发测试参数
type ConcurrencyTestConfig struct {
	GatewayAddr   string
	ClientCount   int
	MessageCount  int
	TestDuration  time.Duration
	PlayerIDStart int64
	MapID         int32
	TargetID      int64
}

// ClientTestResult 客户端测试结果
type ClientTestResult struct {
	ClientID           int
	ConnectSuccess     bool
	TokenVerifySuccess bool
	MessageSuccess     int
	MessageFailed      int
	Errors             []error
}

// ConcurrencyTester 并发测试工具
type ConcurrencyTester struct {
	config     ConcurrencyTestConfig
	results    []ClientTestResult
	resultsMu  sync.Mutex
	wg         sync.WaitGroup
	totalStart time.Time
}

// NewConcurrencyTester 创建新的并发测试工具
func NewConcurrencyTester(config ConcurrencyTestConfig) *ConcurrencyTester {
	return &ConcurrencyTester{
		config:     config,
		results:    make([]ClientTestResult, config.ClientCount),
		totalStart: time.Now(),
	}
}

// Run 运行并发测试
func (t *ConcurrencyTester) Run() {
	fmt.Printf("=== GatewayServer 并发测试 ===\n")
	fmt.Printf("客户端数量: %d\n", t.config.ClientCount)
	fmt.Printf("每个客户端消息数: %d\n", t.config.MessageCount)
	fmt.Printf("测试持续时间: %v\n", t.config.TestDuration)
	fmt.Printf("Gateway地址: %s\n", t.config.GatewayAddr)
	fmt.Println()

	// 启动所有客户端
	for i := 0; i < t.config.ClientCount; i++ {
		t.wg.Add(1)
		go t.runClient(i)
	}

	// 等待测试完成
	t.wg.Wait()

	// 统计结果
	t.printResults()
}

// runClient 运行单个客户端测试
func (t *ConcurrencyTester) runClient(clientID int) {
	defer t.wg.Done()

	result := ClientTestResult{
		ClientID: clientID,
		Errors:   make([]error, 0),
	}
	// defer 统一存结果（捕获最终值），免去每个 early-return 处重复的加锁写入。
	defer common.StoreResult(&t.resultsMu, t.results, clientID, &result)

	// 共享前奏：连接 + token 验证。
	c, connected, verified, err := common.ConnectAndVerify(t.config.GatewayAddr, clientID)
	result.ConnectSuccess = connected
	result.TokenVerifySuccess = verified
	if err != nil {
		result.Errors = append(result.Errors, err)
		if c != nil {
			c.Disconnect()
		}
		return
	}
	defer c.Disconnect()

	// 发送消息
	playerID := t.config.PlayerIDStart + int64(clientID)
	for i := 0; i < t.config.MessageCount; i++ {
		// 随机发送不同类型的消息
		switch i % 3 {
		case 0:
			// 进入地图
			if err := c.SendMapEnter(playerID, t.config.MapID); err != nil {
				result.MessageFailed++
				result.Errors = append(result.Errors, err)
			} else {
				result.MessageSuccess++
			}
		case 1:
			// 移动
			x := float32(100 + clientID*10 + i)
			y := float32(100 + clientID*10 + i)
			if err := c.SendMapMove(playerID, t.config.MapID, x, y, 0); err != nil {
				result.MessageFailed++
				result.Errors = append(result.Errors, err)
			} else {
				result.MessageSuccess++
			}
		case 2:
			// 攻击
			if err := c.SendMapAttack(playerID, t.config.MapID, t.config.TargetID); err != nil {
				result.MessageFailed++
				result.Errors = append(result.Errors, err)
			} else {
				result.MessageSuccess++
			}
		}

		// 随机间隔
		time.Sleep(time.Duration(clientID%10) * time.Millisecond)
	}

	// 等待消息处理完成
	time.Sleep(500 * time.Millisecond)
	// 断开连接与结果保存由 defer 统一处理。
}

// printResults 打印测试结果
func (t *ConcurrencyTester) printResults() {
	totalTime := time.Since(t.totalStart)

	// 统计汇总
	totalConnect := 0
	totalTokenVerify := 0
	totalMessageSuccess := 0
	totalMessageFailed := 0
	totalErrors := 0

	for _, result := range t.results {
		if result.ConnectSuccess {
			totalConnect++
		}
		if result.TokenVerifySuccess {
			totalTokenVerify++
		}
		totalMessageSuccess += result.MessageSuccess
		totalMessageFailed += result.MessageFailed
		totalErrors += len(result.Errors)
	}

	// 打印结果
	fmt.Printf("=== 测试结果 ===\n")
	fmt.Printf("总测试时间: %v\n", totalTime)
	fmt.Printf("连接成功率: %.2f%% (%d/%d)\n",
		float64(totalConnect)/float64(t.config.ClientCount)*100,
		totalConnect, t.config.ClientCount)
	fmt.Printf("Token验证成功率: %.2f%% (%d/%d)\n",
		float64(totalTokenVerify)/float64(t.config.ClientCount)*100,
		totalTokenVerify, t.config.ClientCount)
	fmt.Printf("消息成功率: %.2f%% (%d/%d)\n",
		float64(totalMessageSuccess)/float64(t.config.ClientCount*t.config.MessageCount)*100,
		totalMessageSuccess, t.config.ClientCount*t.config.MessageCount)
	fmt.Printf("总错误数: %d\n", totalErrors)
	fmt.Printf("消息处理速度: %.2f 消息/秒\n",
		float64(totalMessageSuccess+totalMessageFailed)/totalTime.Seconds())

	// 打印失败的客户端
	failedClients := 0
	for i, result := range t.results {
		if !result.ConnectSuccess || !result.TokenVerifySuccess || result.MessageFailed > 0 {
			if failedClients < 5 {
				fmt.Printf("客户端 %d: 连接=%v, Token验证=%v, 成功消息=%d, 失败消息=%d\n",
					i, result.ConnectSuccess, result.TokenVerifySuccess, result.MessageSuccess, result.MessageFailed)
			}
			failedClients++
		}
	}

	if failedClients > 5 {
		fmt.Printf("... 还有 %d 个失败的客户端\n", failedClients-5)
	}

	fmt.Println()
	fmt.Println("=== 并发测试完成 ===")
}

// RunConcurrencyTest 运行并发测试
func RunConcurrencyTest(gatewayAddr string, clientCount, messageCount int, playerIDStart int64, mapID int32, targetID int64) {
	// 创建测试配置
	config := ConcurrencyTestConfig{
		GatewayAddr:   gatewayAddr,
		ClientCount:   clientCount,
		MessageCount:  messageCount,
		TestDuration:  30 * time.Second,
		PlayerIDStart: playerIDStart,
		MapID:         mapID,
		TargetID:      targetID,
	}

	// 运行测试
	tester := NewConcurrencyTester(config)
	tester.Run()
}
