package service

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"go.uber.org/zap"
)

// —— 按客户端分道处理（吞吐模型修正，2026-07-29）——
//
// **问题**：Gateway↔GameServer 是**一条连接承载全服玩家**，而 zNet 的会话是
// `TcpServerSession.process` 一个 goroutine、**收发共用同一循环**（既消费 receiveChan 也消费
// sendChan）。原来的分发链 handleConnectionMessage → handleGatewayMessage → Router.Handle →
// handler 全程同步，handler 里还要阻塞等玩家 actor 回调（move 3s / attack 5s 超时）。于是：
//   - 吞吐：全服客户端请求被串成一条，实测并发 4 就开始明显掉边际（3017→5096→7806 次/秒）；
//   - 更糟的是**自锁**：`Session.Send` 是 `sendChan <- pkt` 的**阻塞**发送，而消费 sendChan 的
//     正是被 handler 堵住的那个 process 循环。负载一高，回包发不出去 → handler 等不到 →
//     队列填满 → 整条连接卡死，实测并发 8 时响应基本不回（每客户端 3 次 5 秒超时）。
//
// **修法**：把"路由 + handler 执行"从会话 goroutine 上挪走，按 **clientSessionID 分道**：
//   - 每个客户端一条 lane（带缓冲队列 + 自己的 goroutine）→ 玩家之间并行，一个玩家的慢请求
//     不再拖住别人，process 循环立刻返回去继续收发（sendChan 得以持续排空，自锁消失）；
//   - **同一客户端仍严格串行**（单 lane 单 goroutine 顺序执行）→ 保住"同一玩家消息按序处理"
//     这条语义，不会出现移动与攻击乱序。
//
// 队列满 = 该客户端自己在刷包，丢弃并记日志，不波及他人（这正是分道的意义）。
// 空闲 lane 由 reaper 回收，避免随登录过的客户端数无界增长。

const (
	// clientLaneQueueSize 单个客户端的待处理队列深度。够吸收正常突发，又不会让一个刷包的
	// 客户端占用过多内存；满了就丢它自己的包。
	clientLaneQueueSize = 256
	// clientLaneIdleTimeout 空闲多久后回收 lane（玩家下线/断线后不再有消息）。
	clientLaneIdleTimeout = 5 * time.Minute
	// clientLaneReapInterval 回收扫描周期。
	clientLaneReapInterval = time.Minute
)

// laneJob 一条待处理的客户端消息。
type laneJob struct {
	session zNet.Session
	protoID int32
	payload []byte
}

// clientLane 一个客户端的处理道：一条队列 + 一个 goroutine，顺序执行该客户端的消息。
type clientLane struct {
	jobs     chan laneJob
	lastUsed time.Time
	mu       sync.Mutex
}

func (l *clientLane) touch() {
	l.mu.Lock()
	l.lastUsed = time.Now()
	l.mu.Unlock()
}

func (l *clientLane) idleFor(now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	return now.Sub(l.lastUsed)
}

// dispatchToLane 把消息投给该客户端的处理道；lane 不存在则创建。
// 立刻返回——**绝不在会话 goroutine 上执行 handler**。
func (ts *TCPService) dispatchToLane(session zNet.Session, clientSessionID zNet.SessionIdType, protoID int32, payload []byte) {
	lane, loaded := ts.clientLanes.Load(clientSessionID)
	if !loaded {
		lane = &clientLane{jobs: make(chan laneJob, clientLaneQueueSize), lastUsed: time.Now()}
		if actual, existed := ts.clientLanes.LoadOrStore(clientSessionID, lane); existed {
			lane = actual
		} else {
			ts.wg.Add(1)
			go ts.runLane(clientSessionID, lane)
		}
	}
	lane.touch()

	select {
	case lane.jobs <- laneJob{session: session, protoID: protoID, payload: payload}:
	default:
		// 只丢这个客户端自己的包：他在刷包，不该影响别人。
		zLog.Warn("Client lane queue full, dropping packet",
			zap.Uint64("client_session_id", uint64(clientSessionID)),
			zap.Int32("proto_id", protoID))
	}
}

// runLane 单客户端处理循环：顺序执行，保住同一玩家的消息序。
func (ts *TCPService) runLane(clientSessionID zNet.SessionIdType, lane *clientLane) {
	defer ts.wg.Done()
	for {
		select {
		case job := <-lane.jobs:
			ts.runLaneJob(job)
		case <-ts.laneStop:
			return
		}
	}
}

// runLaneJob 执行一条消息。就地 recover：单条消息 panic 只丢这条，不带走整条 lane。
func (ts *TCPService) runLaneJob(job laneJob) {
	defer func() {
		if r := recover(); r != nil {
			zLog.Error("Panic while handling client message",
				zap.Int32("proto_id", job.protoID), zap.Any("panic", r))
		}
	}()

	if ts.messageRouter == nil {
		return
	}
	if err := ts.messageRouter.Handle(job.session, job.protoID, job.payload); err != nil {
		zLog.Error("Failed to handle message via router",
			zap.Int32("proto_id", job.protoID), zap.Error(err))
	}
}

// laneReapLoop 回收空闲 lane（客户端早已下线），防其随历史客户端数无界增长。
func (ts *TCPService) laneReapLoop() {
	defer ts.wg.Done()
	ticker := time.NewTicker(clientLaneReapInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ts.laneStop:
			return
		case <-ticker.C:
			now := time.Now()
			var dead []zNet.SessionIdType
			ts.clientLanes.Range(func(sid zNet.SessionIdType, lane *clientLane) bool {
				if lane.idleFor(now) > clientLaneIdleTimeout && len(lane.jobs) == 0 {
					dead = append(dead, sid)
				}
				return true
			})
			for _, sid := range dead {
				ts.clientLanes.Delete(sid)
			}
			if len(dead) > 0 {
				zLog.Debug("Reaped idle client lanes", zap.Int("count", len(dead)))
			}
		}
	}
}
