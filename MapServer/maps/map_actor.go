package maps

import (
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zEngine/zActor"
)

// 单写者 / Actor 模型（MAP-2、并连带缓解 MAP-6/7/8）
//
// 此前游戏对象（Player/Monster）的内部字段（位置/血量/目标/状态）没有任何同步，却同时被
// 两类 goroutine 读写：
//   - 100ms 游戏主循环（AI/战斗/Buff/技能帧更新）
//   - zNet 网络分发 goroutine（玩家 进入/离开/移动/攻击 请求）
// 外加 刷怪 goroutine 与 死亡/重生 定时 goroutine。地图的 m.mu 只保护"对象容器"的增删查，
// 取出对象指针后对其字段的读写是裸的 → 真实数据竞争（-race 可直接抓到）。
//
// 本模型让"每张地图独占一条 goroutine"串行处理所有会改动对象状态的工作：帧更新（postTick）
// 与网络命令（Do）都投递到同一条命令队列，逐个执行——从根上消除竞争。承载由 zEngine 的
// zActor.Runner（单写者闭包串行器）提供：同步 Do、合帧 PostTick、每命令 panic 隔离。
// 此前本文件自造的等价实现已下沉为引擎能力（见 zEngine/zActor/runner.go）。
// 内部辅助方法（AddObject/RemoveObject/MoveObject/attack*Locked 等）只在这条 goroutine 上被
// 调用，绝不能再自行投递 Do（会造成自投递死锁）。

const mapCmdChanSize = 1024

// startActor 启动地图的单写者 Runner。由 NewMap 调用一次。
func (m *Map) startActor() {
	m.runner = zActor.NewRunner(int64(m.mapID), mapCmdChanSize)
	m.runner.Start()
}

// Do 把 fn 投递到地图 goroutine 上同步执行并等待其完成（请求-响应语义，供网络入口拿返回值）。
// 若 actor 未启动（部分单元测试直接构造 Map 且不经 NewMap），就地执行以保持行为不变。
// 注意：绝不能在地图 goroutine 内部再调用 Do（自投递死锁）——内部逻辑一律直接调用 *Locked 方法。
func (m *Map) Do(fn func()) {
	if m.runner == nil {
		fn()
		return
	}
	m.runner.Do(fn)
}

// postTick 把一帧更新合帧投递到地图 goroutine。若队列已满（上一帧尚未处理完）则丢弃本帧，
// 避免慢帧堆积拖垮内存/时序。由全局游戏主循环按固定节拍对每张地图调用。
func (m *Map) postTick(dt time.Duration) {
	if m.runner == nil {
		return
	}
	m.runner.PostTick(func() { m.tick(dt) })
}

// tick 在地图 goroutine 上执行一帧的全部子系统更新（AI/Buff/玩家/技能/事件）。
func (m *Map) tick(dt time.Duration) {
	if m.aiManager != nil {
		m.aiManager.Update(dt)
	}
	if m.buffManager != nil {
		// 到期 buff 经 AOI 广播移除给视野内玩家（场景同步增强）。本调用在地图 actor
		// goroutine 内，broadcastEntityBuff 只入队 NotifyAOI，不阻塞。
		for _, e := range m.buffManager.Update(dt) {
			if obj := m.GetObject(id.ObjectIdType(e.PlayerID)); obj != nil {
				m.broadcastEntityBuff(obj, e.BuffID, false, 0)
			}
		}
	}
	m.UpdatePlayers()
	if m.skillManager != nil {
		m.skillManager.Update()
	}
	if m.eventManager != nil {
		m.eventManager.UpdateEvents()
	}
}

// StopActor 停止地图 goroutine（幂等）。
func (m *Map) StopActor() {
	if m.runner != nil {
		m.runner.Stop()
	}
}
