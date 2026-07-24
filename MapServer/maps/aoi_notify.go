package maps

import (
	"github.com/pzqf/zMmoServer/MapServer/common"
)

// AOINotifier 把 MapServer 的 AOI 视野事件回传给拥有 watcher 的 GameServer。
// 由网络层（net/service.TCPService）实现（它持有 GameServer 会话与 watcher→GameServer 映射），
// 经 SetAOINotifier 注入 Map，避免 maps 包依赖 net 层（依赖倒置）。
type AOINotifier interface {
	// NotifyAOI 推送一条视野事件。eventType 为 crossserver.MsgInternalAOIEnter/Leave/Move。
	NotifyAOI(watcherPlayerID int64, mapID int32, eventType uint32, targetID int64, pos common.Vector3)
}

// SetAOINotifier 注入 AOI 回程通知器。
func (m *Map) SetAOINotifier(n AOINotifier) {
	m.aoiNotifier = n
}
