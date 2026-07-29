// Package loadtest 是**闭环**压测器：每个客户端发一条请求就等它的响应，拿到后再发下一条。
//
// 为什么要新写一个：原有的 concurrency 测试器把 `SendX() == nil` 当作"成功"，从不等响应
// ——它测的是客户端的发包速度，跟服务端处理能力无关，压出来的数字再大也说明不了任何事。
//
// 闭环（closed-loop）模型是量吞吐/延迟的标准做法：
//   - 并发数 = 在途请求数（每客户端恒 1 条在途）
//   - 吞吐 = 完成请求数 / 墙钟时间
//   - 延迟 = 单条请求从发出到收到响应
//
// 逐步加并发跑，看吞吐是否随并发上升：**不上升就说明服务端存在串行瓶颈**，
// 此时延迟会随并发线性增长（排队），这正是要测出来的东西。
package loadtest

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pzqf/zMmoServer/GameClient/internal/client"
)

// Op 压测的操作类型。不同 op 打到的服务端路径深度不同，用来定位瓶颈在哪一段。
type Op string

const (
	// OpMove 移动：客户端 → GameServer handler → 玩家 actor → 回包。
	// 到 MapServer 的转发是 fire-and-forget，**不含** MapServer 往返。
	OpMove Op = "move"
	// OpAttack 攻击：在 move 的基础上，actor 里还要**同步等 MapServer 的应答**。
	// 与 move 的差值 = MapServer 往返对吞吐的影响。
	OpAttack Op = "attack"
)

// Config 压测参数。
type Config struct {
	GlobalAddr  string
	GatewayAddr string
	Clients     int
	Duration    time.Duration
	Op          Op
	MapID       int32
	AccountPfx  string
}

// Result 一次压测的汇总结果。
type Result struct {
	Clients    int
	Duration   time.Duration
	Completed  int
	Failed     int
	Throughput float64
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
	Max        time.Duration
}

type clientOutcome struct {
	completed int
	failed    int
	latencies []time.Duration
	err       error
}

// Run 跑一轮压测。
func Run(cfg Config) (*Result, error) {
	if cfg.Clients <= 0 {
		cfg.Clients = 1
	}
	if cfg.Duration <= 0 {
		cfg.Duration = 10 * time.Second
	}
	if cfg.MapID == 0 {
		cfg.MapID = 1001
	}
	if cfg.AccountPfx == "" {
		cfg.AccountPfx = "load"
	}

	fmt.Printf("=== 闭环压测：op=%s 并发=%d 时长=%v ===\n", cfg.Op, cfg.Clients, cfg.Duration)

	// 阶段一：所有客户端先登录 + 进图（不计入压测窗口）。
	type readyClient struct {
		c        *client.Client
		playerID int64
	}
	ready := make([]readyClient, 0, cfg.Clients)
	var readyMu sync.Mutex
	var prepWg sync.WaitGroup

	for i := 0; i < cfg.Clients; i++ {
		prepWg.Add(1)
		// 错开登录：登录链路要打 GlobalServer 的 HTTP（有 20 次/秒限流）、建角色要写库，
		// 几十个客户端在同一毫秒涌进去测的是登录风暴而不是本次要量的请求吞吐。
		time.Sleep(80 * time.Millisecond)
		go func(idx int) {
			defer prepWg.Done()
			c, playerID, err := prepareClient(cfg, idx)
			if err != nil {
				fmt.Printf("[准备] 客户端 %d 失败: %v\n", idx, err)
				if c != nil {
					c.Disconnect()
				}
				return
			}
			readyMu.Lock()
			ready = append(ready, readyClient{c: c, playerID: playerID})
			readyMu.Unlock()
		}(i)
	}
	prepWg.Wait()

	if len(ready) == 0 {
		return nil, fmt.Errorf("没有任何客户端准备就绪")
	}
	if len(ready) < cfg.Clients {
		fmt.Printf("[准备] 就绪 %d/%d 个客户端（其余失败，按就绪数统计）\n", len(ready), cfg.Clients)
	}
	defer func() {
		for _, rc := range ready {
			rc.c.Disconnect()
		}
	}()

	// 阶段二：闭环压测窗口。
	outcomes := make([]clientOutcome, len(ready))
	var wg sync.WaitGroup
	deadline := time.Now().Add(cfg.Duration)
	start := time.Now()

	for i := range ready {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			outcomes[idx] = runLoop(cfg, ready[idx].c, ready[idx].playerID, deadline)
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	// 汇总。
	res := &Result{Clients: len(ready), Duration: elapsed}
	all := make([]time.Duration, 0, 1024)
	for _, o := range outcomes {
		res.Completed += o.completed
		res.Failed += o.failed
		all = append(all, o.latencies...)
	}
	if elapsed > 0 {
		res.Throughput = float64(res.Completed) / elapsed.Seconds()
	}
	if len(all) > 0 {
		sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
		res.P50 = all[len(all)*50/100]
		res.P95 = all[min(len(all)*95/100, len(all)-1)]
		res.P99 = all[min(len(all)*99/100, len(all)-1)]
		res.Max = all[len(all)-1]
	}
	return res, nil
}

// prepareClient 登录 + 建角色 + 进图，返回可直接压测的客户端。
func prepareClient(cfg Config, idx int) (*client.Client, int64, error) {
	account := fmt.Sprintf("%s_%d", cfg.AccountPfx, idx)

	c := client.NewClient(cfg.GlobalAddr)
	c.SetGatewayAddr(cfg.GatewayAddr)

	// 账号可能已存在（重复跑压测），注册失败不阻断，直接尝试登录。
	_, _ = c.Register(account, "123456", account+"@load.test")

	authResp, err := c.Login(account, "123456")
	if err != nil {
		return nil, 0, fmt.Errorf("登录: %w", err)
	}
	if authResp.Result != 0 {
		return nil, 0, fmt.Errorf("登录被拒: %s", authResp.ErrorMsg)
	}

	if err := c.Connect(); err != nil {
		return nil, 0, fmt.Errorf("连网关: %w", err)
	}
	if err := c.SendTokenVerify(c.GetToken()); err != nil {
		return c, 0, fmt.Errorf("发 token: %w", err)
	}
	if r, ok := c.WaitTokenVerify(10 * time.Second); !ok || r != 0 {
		return c, 0, fmt.Errorf("token 验证失败 (ok=%v result=%d)", ok, r)
	}

	// 角色名必须**全局唯一**：服务端建角色时按名字查重（GetPlayerByName），
	// 用 "L0/L1/..." 这种会跨档位、跨轮次重名，第二次起一律建号失败（表现为"未取到角色ID"）。
	// 账号名本身已带本轮时间戳，直接拿它当角色名最省事。
	if err := c.SendPlayerCreate(account, 1, 18); err != nil {
		return c, 0, fmt.Errorf("建角色: %w", err)
	}
	playerID := c.GetCreatedPlayerID()
	if playerID == 0 {
		return c, 0, fmt.Errorf("未取到角色ID（多为角色名重复/建号被拒）")
	}

	if err := c.SendPlayerLogin(playerID); err != nil {
		return c, playerID, fmt.Errorf("进游戏: %w", err)
	}
	if r, ok := c.WaitPlayerLogin(10 * time.Second); !ok || r != 0 {
		return c, playerID, fmt.Errorf("进游戏失败 (ok=%v result=%d)", ok, r)
	}

	if err := c.SendMapEnter(playerID, cfg.MapID); err != nil {
		return c, playerID, fmt.Errorf("进图: %w", err)
	}
	if r, ok := c.WaitMapEnter(10 * time.Second); !ok || r != 0 {
		return c, playerID, fmt.Errorf("进图失败 (ok=%v result=%d)", ok, r)
	}

	return c, playerID, nil
}

// runLoop 闭环发送：发一条 → 等响应 → 再发下一条，直到 deadline。
func runLoop(cfg Config, c *client.Client, playerID int64, deadline time.Time) clientOutcome {
	out := clientOutcome{latencies: make([]time.Duration, 0, 256)}

	var seq int
	for time.Now().Before(deadline) {
		seq++
		reqStart := time.Now()

		var sendErr error
		var wait func(time.Time, time.Duration) (int32, bool)

		// 清掉上一轮迟到/重复的响应信号（配合下面按到达时刻的严格关联，双保险）。
		c.DrainMapOpSignals()

		switch cfg.Op {
		case OpAttack:
			// 攻击自己：目标必在场，且不依赖怪物刷新；服务端仍走完整的
			// handler → actor → MapServer 往返 → 回包 链路。
			sendErr = c.SendMapAttack(playerID, cfg.MapID, playerID)
			wait = c.WaitMapAttackAfter
		default:
			x := float32(100 + seq%50)
			sendErr = c.SendMapMove(playerID, cfg.MapID, x, x, 0)
			wait = c.WaitMapMoveAfter
		}

		if sendErr != nil {
			out.failed++
			continue
		}
		// 只认**发出之后**才到达的响应：否则上一轮迟到的响应会被当成本轮的，
		// 计数与吞吐一起虚高（曾测出"2990 次/秒 而 p50=515µs"这种自相矛盾的数）。
		if _, ok := wait(reqStart, 5*time.Second); !ok {
			out.failed++
			continue
		}

		out.completed++
		out.latencies = append(out.latencies, time.Since(reqStart))
	}
	return out
}

// Print 打印一轮结果。
func (r *Result) Print() {
	fmt.Printf("并发=%-4d 完成=%-7d 失败=%-6d 吞吐=%8.1f 次/秒  p50=%-10v p95=%-10v p99=%-10v max=%v\n",
		r.Clients, r.Completed, r.Failed, r.Throughput, r.P50, r.P95, r.P99, r.Max)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
