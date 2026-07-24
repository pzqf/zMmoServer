package crossserver

// 跨服内部 AOI 视野事件 protoId（MapServer → GameServer，服务端主动推送）。
//
// 背景（成熟化改造 Phase 2.3 AOI 回程）：MapServer 是地图对象权威，其 AOI 计算出的
// "谁看见谁进入/离开/移动"视野事件需回传给拥有该观察者(watcher)的 GameServer，
// 再由 GameServer 复用既有推送链（player_aoi_handler → EntityXxxViewNotify → 客户端）。
//
// 取 500 段（internal.proto 的 InternalMsgId 用 400-499 表示地图请求/响应，600 段跨服）。
// 承载消息复用现有 protobuf：Enter=EntityEnterViewNotify、Leave=EntityLeaveViewNotify、
// Move=EntityMoveNotify；watcher 放 BaseMessage.PlayerId、mapID 放 BaseMessage.MapId。
const (
	MsgInternalAOIEnter uint32 = 500
	MsgInternalAOILeave uint32 = 501
	MsgInternalAOIMove  uint32 = 502
)
