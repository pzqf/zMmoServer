package common

import (
	"fmt"
	"sync"
	"time"

	"github.com/pzqf/zMmoServer/GameClient/internal/client"
	"github.com/pzqf/zMmoServer/GameClient/internal/utils"
)

// ConnectAndVerify 是 concurrency / longtest 两个测试器共享的前奏：
// 新建客户端 → 连接网关 → 生成 token → 发送 token 验证 → 短暂等待验证到达。
// 返回 connected / verified 两个阶段标志，供调用方回填各自 result；
// err 非 nil 表示前奏在某阶段失败（此时 c 可能已连接，需由调用方 Disconnect）。
func ConnectAndVerify(gatewayAddr string, clientID int) (c *client.Client, connected, verified bool, err error) {
	c = client.NewClient("")
	c.SetGatewayAddr(gatewayAddr)

	if err = c.Connect(); err != nil {
		return c, false, false, err
	}
	connected = true

	var token string
	token, err = utils.GenerateToken(int64(clientID+1), fmt.Sprintf("test_account_%d", clientID), "zMmoServerSecretKey")
	if err != nil {
		return c, connected, false, err
	}

	if err = c.SendTokenVerify(token); err != nil {
		return c, connected, false, err
	}
	verified = true

	// 等 token 验证响应到达（保持与原两测试器一致的固定等待）。
	time.Sleep(100 * time.Millisecond)
	return c, connected, verified, nil
}

// StoreResult 在互斥锁下把 result 写入 results[idx]。
// 配合 defer 使用（defer StoreResult(&mu, results, id, &result)）以捕获最终值，
// 免去每个 early-return 处重复的 Lock/写/Unlock 三连。
func StoreResult[T any](mu *sync.Mutex, results []T, idx int, result *T) {
	mu.Lock()
	results[idx] = *result
	mu.Unlock()
}
