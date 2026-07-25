package gameserverlist

import (
	"sync"
	"testing"
	"time"

	"github.com/pzqf/zCommon/db/models"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zUtil/zMap"
)

// TestGetAllServerFullInfos_ConcurrentNoRace 验证 GW-4：多 goroutine（模拟并发 HTTP 请求）
// 同时调用 GetAllServerFullInfos 时，缓存 slice/时间戳的读检查与重建不再有数据竞争。
// 用短 TTL 强制频繁走重建（写）路径，最大化与读路径的竞争窗口。`go test -race` 下应零竞争。
func TestGetAllServerFullInfos_ConcurrentNoRace(t *testing.T) {
	m := &ServerListManager{
		staticServers:      zMap.NewTypedMap[int32, *models.GameServer](),
		serviceCache:       zMap.NewTypedMap[string, *discovery.ServerInfo](),
		serverListCacheTTL: 5 * time.Millisecond,
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 2000; j++ {
				_ = m.GetAllServerFullInfos()
			}
		}()
	}
	wg.Wait()
}
