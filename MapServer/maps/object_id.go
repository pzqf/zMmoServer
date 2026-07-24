package maps

import (
	"sync/atomic"

	"github.com/pzqf/zCommon/common/id"
)

// 非玩家对象（怪物/掉落物等）的全局唯一 ID 生成器（MAP-5）。
//
// 此前各处用 `time.Now().UnixNano() % 1e9` 生成对象 ID —— 同一纳秒（或同一 %1e9 窗口）内
// 生成的多个对象会得到**相同 ID**，在 map.objects 里互相覆盖（一个 tick 刷多只怪、一次死亡掉多件
// 物品时极易发生）。改为进程内原子自增，保证唯一。
//
// 基数取较高值以与顺序玩家对象 ID（玩家用 playerID 作对象 ID）分离；对象均为单个 MapServer
// 进程内本地存活，跨进程无需协调。
var nonPlayerObjectIDSeq atomic.Int64

func init() {
	nonPlayerObjectIDSeq.Store(3_000_000_000)
}

// nextMapObjectID 返回一个进程内唯一的非玩家对象 ID。
func nextMapObjectID() id.ObjectIdType {
	return id.ObjectIdType(nonPlayerObjectIDSeq.Add(1))
}
