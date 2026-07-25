package maps

import (
	"sync"
	"testing"
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zMmoServer/MapServer/common"
	"github.com/pzqf/zMmoServer/MapServer/maps/object"
)

// TestMapActor_ConcurrentMutation_NoRace 验证单写者 / Actor 模型（MAP-2）：
// 帧更新（AI/战斗/移动）与网络命令（攻击/移动/进出地图）由多条 goroutine 并发发起时，
// 全部经 m.Do / m.postTick 串行到该地图独占的一条 goroutine 执行，因此在 `go test -race`
// 下应当零数据竞争。此前对象字段无同步、帧更新 goroutine 与网络 goroutine 直接并发改动
// 同一批对象 → -race 必报。
//
// 本用例同时是一道回归护栏：若日后有人把网络入口改回绕过 m.Do 直接改对象，-race 会重新报错。
func TestMapActor_ConcurrentMutation_NoRace(t *testing.T) {
	m := NewMap(id.MapIdType(1), 1, "RaceMap", 1000, 1000)
	defer m.StopActor()

	const playerID = id.PlayerIdType(100)
	const playerObjID = id.ObjectIdType(100)

	m.AddPlayer(playerID, playerObjID, 100, 0, 100)

	monsterIDs := make([]id.ObjectIdType, 0, 8)
	for i := 0; i < 8; i++ {
		mid := id.ObjectIdType(500 + i)
		monster := object.NewMonster(mid, 1, "M", common.Vector3{X: 101, Y: 0, Z: 101}, 1)
		m.Do(func() { m.AddObject(monster) })
		monsterIDs = append(monsterIDs, mid)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// 帧更新驱动（模拟游戏主循环，向 actor 投递 tick）
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				m.postTick(50 * time.Millisecond)
				time.Sleep(time.Millisecond)
			}
		}
	}()

	// 多条 goroutine 并发发起 攻击 / 移动 / 补活目标
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			i := seed
			for {
				select {
				case <-stop:
					return
				default:
					tgt := monsterIDs[i%len(monsterIDs)]
					m.AttackTarget(playerID, playerObjID, tgt)
					m.MovePlayer(playerID, playerObjID, float32(100+i%50), 0, float32(100+i%50))
					// 被打死的怪补加回来，保证攻击路径持续命中活目标
					m.Do(func() {
						if m.GetObject(tgt) == nil {
							mon := object.NewMonster(tgt, 1, "M", common.Vector3{X: 101, Y: 0, Z: 101}, 1)
							m.AddObject(mon)
						}
					})
					i++
				}
			}
		}(w)
	}

	time.Sleep(700 * time.Millisecond)
	close(stop)
	wg.Wait()
}

// TestSpawnRemove_NoDeadlockNoRace 压测 MAP-8 的锁序安全：刷怪循环
// （checkSpawnPoints：持 sm.mu 建对象→释放→经 Do 加入地图）与地图 goroutine 上的
// 对象移除（RemoveObject：持 m.mu→取 sm.mu 回退计数）并发。若二者锁序成环会死锁 → 本用例超时失败；
// 同时在 -race 下验证 spawnedBy/currentCount 访问无竞争。
func TestSpawnRemove_NoDeadlockNoRace(t *testing.T) {
	m := NewMap(id.MapIdType(2), 2, "SpawnMap", 1000, 1000)
	defer m.StopActor()

	// 注入一个高频刷怪点（respawnTime=0，maxCount 较大），令 checkSpawnPoints 真正刷怪。
	m.spawnManager.mu.Lock()
	m.spawnManager.spawnPoints = append(m.spawnManager.spawnPoints, &SpawnPoint{
		spawnID:     1,
		spawnType:   1, // monster
		objectID:    1,
		position:    common.Vector3{X: 50, Y: 0, Z: 50},
		respawnTime: 0,
		maxCount:    20,
		active:      true,
		lastSpawn:   time.Now().Add(-time.Hour),
	})
	m.spawnManager.mu.Unlock()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// 刷怪驱动：高频调用 checkSpawnPoints。
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				m.spawnManager.checkSpawnPoints()
				time.Sleep(time.Millisecond)
			}
		}
	}()

	// 帧更新驱动（AI 会移动怪，触发 MAP-6 的 AOI 同步路径）。
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				m.postTick(50 * time.Millisecond)
				time.Sleep(time.Millisecond)
			}
		}
	}()

	// 移除驱动：不断把已刷出的怪移除（经地图 goroutine），触发 RemoveObject→spawnManager.RemoveObject。
	for w := 0; w < 3; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					for _, obj := range m.GetObjects() {
						if obj.GetType() == common.GameObjectTypeMonster {
							oid := obj.GetID()
							m.Do(func() { m.RemoveObject(oid) })
						}
					}
				}
			}
		}()
	}

	time.Sleep(700 * time.Millisecond)
	close(stop)
	wg.Wait()
}
