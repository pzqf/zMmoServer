package crossserver

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/pzqf/zCommon/protocol"
)

// TestCodec_EnterRoundTrip 锁定 GameServer 发送端与 MapServer 接收端的统一契约：
// 进入地图请求经 PackMessage 打包后，能被 UnpackMessage 完整还原，且内层
// MapEnterRequest 的坐标不丢失、外层/内层 protoId 与 InternalMsgId 一致。
// 这条测试正是修复 Game↔Map 断链（protoId 300≠400、protobuf/JSON 混用、坐标丢失）的靶子。
func TestCodec_EnterRoundTrip(t *testing.T) {
	const (
		gameServerID uint32 = 101
		mapServerID  uint32 = 301
		playerID     uint64 = 100001
		mapID        uint32 = 1001
	)

	// —— GameServer 发送端 ——
	inner := &protocol.MapEnterRequest{
		PlayerId: int64(playerID),
		MapId:    int64(mapID),
		X:        12.5,
		Y:        0,
		Z:        -7.5,
	}
	innerData, err := proto.Marshal(inner)
	if err != nil {
		t.Fatalf("marshal inner: %v", err)
	}

	base := BuildBaseMessage(uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_REQUEST),
		playerID, gameServerID, mapID, innerData)

	meta := NewRequestMeta(ServiceTypeGame, int32(gameServerID))
	wire, err := PackMessage(meta, ServiceTypeGame, ServiceTypeMap, gameServerID, mapServerID, base)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}

	// 外层 zNet ProtoId 约定 == 内层 MsgId == InternalMsgId 请求值（400）
	if base.MsgId != uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_REQUEST) {
		t.Fatalf("outer protoId contract broken: got %d want 400", base.MsgId)
	}

	// —— MapServer 接收端 ——
	gotMeta, cross, gotBase, err := UnpackMessage(wire)
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	if gotMeta.TraceID != meta.TraceID || gotMeta.RequestID != meta.RequestID {
		t.Fatalf("meta mismatch: got trace=%d req=%d", gotMeta.TraceID, gotMeta.RequestID)
	}
	if cross.FromService != uint32(ServiceTypeGame) || cross.ToService != uint32(ServiceTypeMap) {
		t.Fatalf("service routing mismatch: from=%d to=%d", cross.FromService, cross.ToService)
	}
	if gotBase == nil {
		t.Fatal("inner base message is nil")
	}
	if gotBase.MsgId != uint32(protocol.InternalMsgId_MSG_INTERNAL_MAP_ENTER_REQUEST) {
		t.Fatalf("inner MsgId mismatch: got %d want 400", gotBase.MsgId)
	}

	var gotReq protocol.MapEnterRequest
	if err := proto.Unmarshal(gotBase.Data, &gotReq); err != nil {
		t.Fatalf("unmarshal inner MapEnterRequest (encoding contract broken): %v", err)
	}
	// 坐标必须完整往返（旧代码用无坐标的 ClientMapEnterRequest，会丢 X/Z）
	if gotReq.X != inner.X || gotReq.Z != inner.Z || gotReq.PlayerId != inner.PlayerId {
		t.Fatalf("coord/field loss: got x=%v z=%v pid=%d", gotReq.X, gotReq.Z, gotReq.PlayerId)
	}
}

// TestCodec_AttackResponseRoundTrip 锁定回程（MapServer -> GameServer）契约，
// 保证攻击结果（damage/target_hp）能被 GameServer 正确还原以回填等待中的 channel。
func TestCodec_AttackResponseRoundTrip(t *testing.T) {
	req := NewRequestMeta(ServiceTypeGame, 101)
	respMeta := NewResponseMetaFromRequest(req, ServiceTypeMap, 301)

	resp := &protocol.MapAttackResponse{
		Success:  true,
		PlayerId: 100001,
		TargetId: 200002,
		Damage:   37,
		TargetHp: 63,
	}
	respData, _ := proto.Marshal(resp)
	base := BuildBaseMessage(uint32(protocol.InternalMsgId_MSG_INTERNAL_COMBAT_ACTION), 100001, 301, 1001, respData)

	wire, err := PackMessage(respMeta, ServiceTypeMap, ServiceTypeGame, 301, 101, base)
	if err != nil {
		t.Fatalf("pack resp: %v", err)
	}

	gotMeta, _, gotBase, err := UnpackMessage(wire)
	if err != nil {
		t.Fatalf("unpack resp: %v", err)
	}
	if gotMeta.MessageType != MessageTypeResponse {
		t.Fatalf("message type not response: %d", gotMeta.MessageType)
	}
	if gotMeta.RequestID != req.RequestID {
		t.Fatalf("response requestID must match request: got %d want %d", gotMeta.RequestID, req.RequestID)
	}
	if gotBase.MsgId != uint32(protocol.InternalMsgId_MSG_INTERNAL_COMBAT_ACTION) {
		t.Fatalf("resp MsgId mismatch: %d", gotBase.MsgId)
	}
	var gotResp protocol.MapAttackResponse
	if err := proto.Unmarshal(gotBase.Data, &gotResp); err != nil {
		t.Fatalf("unmarshal MapAttackResponse: %v", err)
	}
	if gotResp.Damage != 37 || gotResp.TargetHp != 63 {
		t.Fatalf("attack result lost: dmg=%d hp=%d", gotResp.Damage, gotResp.TargetHp)
	}
}

// TestCodec_AOIEnterViewRoundTrip 锁定 AOI 回程契约（MapServer→GameServer，Phase 2.3）：
// 视野进入事件用内部 protoId MsgInternalAOIEnter(500) 承载 EntityEnterViewNotify，
// watcher 放 BaseMessage.PlayerId、mapID 放 BaseMessage.MapId；GameServer 端据此还原并
// 路由到 watcher 玩家推送客户端。守护该 wire 契约不回归。
func TestCodec_AOIEnterViewRoundTrip(t *testing.T) {
	const (
		mapServerID uint32 = 301
		gameServerID uint32 = 101
		watcher     uint64 = 100001
		mapID       uint32 = 1001
		targetID    int64  = 2
	)

	inner := &protocol.EntityEnterViewNotify{
		EntityId: targetID,
		Pos:      &protocol.Position{X: 100, Y: 100, Z: 0},
	}
	innerData, err := proto.Marshal(inner)
	if err != nil {
		t.Fatalf("marshal inner: %v", err)
	}

	base := BuildBaseMessage(MsgInternalAOIEnter, watcher, mapServerID, mapID, innerData)
	meta := NewRequestMeta(ServiceTypeMap, int32(mapServerID))
	wire, err := PackMessage(meta, ServiceTypeMap, ServiceTypeGame, mapServerID, gameServerID, base)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}

	_, cross, gotBase, err := UnpackMessage(wire)
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	if cross.FromService != uint32(ServiceTypeMap) || cross.ToService != uint32(ServiceTypeGame) {
		t.Fatalf("service routing mismatch: from=%d to=%d", cross.FromService, cross.ToService)
	}
	if gotBase == nil {
		t.Fatal("inner base is nil")
	}
	// 事件类型经 MsgId 区分；watcher/mapID 经 BaseMessage 承载
	if gotBase.MsgId != MsgInternalAOIEnter {
		t.Fatalf("event type lost: got %d want %d", gotBase.MsgId, MsgInternalAOIEnter)
	}
	if gotBase.PlayerId != watcher {
		t.Fatalf("watcher lost: got %d want %d", gotBase.PlayerId, watcher)
	}
	if gotBase.MapId != mapID {
		t.Fatalf("mapID lost: got %d want %d", gotBase.MapId, mapID)
	}
	var gotNotify protocol.EntityEnterViewNotify
	if err := proto.Unmarshal(gotBase.Data, &gotNotify); err != nil {
		t.Fatalf("unmarshal EntityEnterViewNotify: %v", err)
	}
	if gotNotify.EntityId != targetID || gotNotify.Pos == nil || gotNotify.Pos.X != 100 {
		t.Fatalf("notify payload lost: id=%d pos=%v", gotNotify.EntityId, gotNotify.Pos)
	}
}

// TestCodec_RejectsLegacyJSON 守护回归：旧链路把 JSON 字节直接当负载发送，
// 新契约要求 UnpackMessage 拒绝非 Envelope+protobuf 的数据，而非静默吞掉。
func TestCodec_RejectsLegacyJSON(t *testing.T) {
	// 旧 MapServer 发的是不带 Envelope 的裸 JSON
	legacy, _ := json.Marshal(map[string]any{"trace_id": 1, "message": map[string]any{"msg_id": 401}})
	if _, _, _, err := UnpackMessage(legacy); err == nil {
		t.Fatal("expected UnpackMessage to reject legacy non-enveloped JSON payload")
	}
}
