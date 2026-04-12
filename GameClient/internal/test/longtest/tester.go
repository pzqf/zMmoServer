package longtest

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zMmoServer/GameClient/internal/client"
	"github.com/pzqf/zMmoServer/GameClient/internal/utils"
)

// LongTestConfig 长时测试参数
type LongTestConfig struct {
	GatewayAddr       string
	ClientCount       int
	TestDuration      time.Duration
	HeartbeatInterval time.Duration
	PlayerIDStart     int64
	MapID             int32
	TargetID          int64
}

// ClientLongTestResult 客户端长时测试结果
type ClientLongTestResult struct {
	ClientID           int
	ConnectSuccess     bool
	TokenVerifySuccess bool
	HeartbeatSuccess   int
	HeartbeatFailed    int
	SessionDuration    time.Duration
	Errors             []error
}

// LongTester 长时测试工具
type LongTester struct {
	config     LongTestConfig
	results    []ClientLongTestResult
	resultsMu  sync.Mutex
	wg         sync.WaitGroup
	totalStart time.Time
}

// NewLongTester 创建新的长时测试工具
func NewLongTester(config LongTestConfig) *LongTester {
	return &LongTester{
		config:     config,
		results:    make([]ClientLongTestResult, config.ClientCount),
		totalStart: time.Now(),
	}
}

// Run 运行长时测试
func (t *LongTester) Run() {
	fmt.Printf("=== GatewayServer 长时测试 ===\n")
	fmt.Printf("客户端数量: %d\n", t.config.ClientCount)
	fmt.Printf("测试持续时间: %v\n", t.config.TestDuration)
	fmt.Printf("心跳间隔: %v\n", t.config.HeartbeatInterval)
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

// runClient 运行单个客户端长时测试
func (t *LongTester) runClient(clientID int) {
	defer t.wg.Done()

	result := ClientLongTestResult{
		ClientID: clientID,
		Errors:   make([]error, 0),
	}

	// 创建客户端
	c := client.NewClient("")
	c.SetGatewayAddr(t.config.GatewayAddr)

	// 连接服务器
	if err := c.Connect(); err != nil {
		result.ConnectSuccess = false
		result.Errors = append(result.Errors, err)
		t.resultsMu.Lock()
		t.results[clientID] = result
		t.resultsMu.Unlock()
		return
	}
	result.ConnectSuccess = true

	// 生成token
	token, err := utils.GenerateToken(int64(clientID+1), fmt.Sprintf("test_account_%d", clientID), "zMmoServerSecretKey")
	if err != nil {
		result.Errors = append(result.Errors, err)
		c.Disconnect()
		t.resultsMu.Lock()
		t.results[clientID] = result
		t.resultsMu.Unlock()
		return
	}

	// 验证token
	if err := c.SendTokenVerify(token); err != nil {
		result.Errors = append(result.Errors, err)
		c.Disconnect()
		t.resultsMu.Lock()
		t.results[clientID] = result
		t.resultsMu.Unlock()
		return
	}
	result.TokenVerifySuccess = true

	// 等待token验证完成
	time.Sleep(100 * time.Millisecond)

	// 开始长时测试
	startTime := time.Now()
	heartbeatTicker := time.NewTicker(t.config.HeartbeatInterval)
	defer heartbeatTicker.Stop()

	endTime := startTime.Add(t.config.TestDuration)
	for time.Now().Before(endTime) {
		select {
		case <-heartbeatTicker.C:
			// 发送心跳
			if err := c.SendHeartbeat(); err != nil {
				result.HeartbeatFailed++
				result.Errors = append(result.Errors, err)
			} else {
				result.HeartbeatSuccess++
			}

			// 随机发送一些业务消息
			if clientID%3 == 0 {
				playerID := t.config.PlayerIDStart + int64(clientID)
				// 移动
				x := float32(100 + clientID*10 + int(time.Now().Unix()%100))
				y := float32(100 + clientID*10 + int(time.Now().Unix()%100))
				c.SendMapMove(playerID, t.config.MapID, x, y, 0)
			}
		}
	}

	// 计算会话持续时间
	result.SessionDuration = time.Since(startTime)

	// 断开连接
	c.Disconnect()

	// 保存结果
	t.resultsMu.Lock()
	t.results[clientID] = result
	t.resultsMu.Unlock()
}

// printResults 打印测试结果
func (t *LongTester) printResults() {
	totalTime := time.Since(t.totalStart)

	// 统计汇总
	totalConnect := 0
	totalTokenVerify := 0
	totalHeartbeatSuccess := 0
	totalHeartbeatFailed := 0
	totalErrors := 0
	totalSessionDuration := time.Duration(0)

	for _, result := range t.results {
		if result.ConnectSuccess {
			totalConnect++
		}
		if result.TokenVerifySuccess {
			totalTokenVerify++
		}
		totalHeartbeatSuccess += result.HeartbeatSuccess
		totalHeartbeatFailed += result.HeartbeatFailed
		totalErrors += len(result.Errors)
		if result.ConnectSuccess {
			totalSessionDuration += result.SessionDuration
		}
	}

	// 计算平均会话持续时间
	var avgSessionDuration time.Duration
	if totalConnect > 0 {
		avgSessionDuration = totalSessionDuration / time.Duration(totalConnect)
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
	fmt.Printf("心跳成功率: %.2f%% (%d/%d)\n",
		float64(totalHeartbeatSuccess)/float64(totalHeartbeatSuccess+totalHeartbeatFailed)*100,
		totalHeartbeatSuccess, totalHeartbeatSuccess+totalHeartbeatFailed)
	fmt.Printf("总错误数: %d\n", totalErrors)
	fmt.Printf("平均会话持续时间: %v\n", avgSessionDuration)

	// 打印失败的客户端
	failedClients := 0
	for i, result := range t.results {
		if !result.ConnectSuccess || !result.TokenVerifySuccess || result.HeartbeatFailed > 0 {
			if failedClients < 5 {
				fmt.Printf("客户端 %d: 连接=%v, Token验证=%v, 成功心跳=%d, 失败心跳=%d, 会话持续时间=%v\n",
					i, result.ConnectSuccess, result.TokenVerifySuccess, result.HeartbeatSuccess, result.HeartbeatFailed, result.SessionDuration)
			}
			failedClients++
		}
	}

	if failedClients > 5 {
		fmt.Printf("... 还有 %d 个失败的客户端\n", failedClients-5)
	}

	fmt.Println()
	fmt.Println("=== 长时测试完成 ===")
}

// RunLongTest 运行长时测试
func RunLongTest(gatewayAddr string, clientCount int, testDuration, heartbeatInterval time.Duration, playerIDStart int64, mapID int32, targetID int64) {
	// 创建测试配置
	config := LongTestConfig{
		GatewayAddr:       gatewayAddr,
		ClientCount:       clientCount,
		TestDuration:      testDuration,
		HeartbeatInterval: heartbeatInterval,
		PlayerIDStart:     playerIDStart,
		MapID:             mapID,
		TargetID:          targetID,
	}

	// 运行测试
	tester := NewLongTester(config)
	tester.Run()
}
