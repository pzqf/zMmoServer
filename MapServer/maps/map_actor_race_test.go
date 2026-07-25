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
