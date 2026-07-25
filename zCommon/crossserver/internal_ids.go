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
//
// 场景同步增强（业务层建设 2026-07-25）：在进入/离开/移动之外，再广播「对象状态变更」——
// Attr=EntityAttrNotify（战斗结算后的最新血量）、Death=EntityDeathNotify（死亡）。这两类
// 不由 AOI 网格事件驱动，而由战斗结算时主动枚举视野内玩家逐个 NotifyAOI。
const (
	MsgInternalAOIEnter uint32 = 500
	MsgInternalAOILeave uint32 = 501
	MsgInternalAOIMove  uint32 = 502
	MsgInternalAOIAttr  uint32 = 503
	MsgInternalAOIDeath uint32 = 504
	MsgInternalAOIBuff  uint32 = 505 // 视野内实体 buff 增删（EntityBuffNotify）
	MsgInternalItemGrant uint32 = 506 // 拾取授权：MapServer 判定拾取后通知 GameServer 发物品入背包（ItemGrantNotify）
)
