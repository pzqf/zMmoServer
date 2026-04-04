package crossserver

import (
	"encoding/json"
	"testing"
)

func TestBaseMessageSerialization(t *testing.T) {
	// 创建基础消息
	baseMsg := BaseMessage{
		MsgID:     1001,
		SessionID: 123456789,
		PlayerID:  987654321,
		ServerID:  1,
		Timestamp: 1640995200,
		Data:      []byte("test data"),
		MapID:     100,
		MapServerID: 200,
	}

	// 序列化
	data, err := json.Marshal(baseMsg)
	if err != nil {
		t.Fatalf("Failed to marshal base message: %v", err)
	}

	// 反序列化
	var decodedMsg BaseMessage
	if err := json.Unmarshal(data, &decodedMsg); err != nil {
		t.Fatalf("Failed to unmarshal base message: %v", err)
	}

	// 验证字段
	if decodedMsg.MsgID != baseMsg.MsgID {
		t.Errorf("MsgID mismatch: got %d, want %d", decodedMsg.MsgID, baseMsg.MsgID)
	}
	if decodedMsg.SessionID != baseMsg.SessionID {
		t.Errorf("SessionID mismatch: got %d, want %d", decodedMsg.SessionID, baseMsg.SessionID)
	}
	if decodedMsg.PlayerID != baseMsg.PlayerID {
		t.Errorf("PlayerID mismatch: got %d, want %d", decodedMsg.PlayerID, baseMsg.PlayerID)
	}
	if string(decodedMsg.Data) != string(baseMsg.Data) {
		t.Errorf("Data mismatch: got %s, want %s", string(decodedMsg.Data), string(baseMsg.Data))
	}
}

func TestCrossServerMessageSerialization(t *testing.T) {
	// 创建基础消息
	baseMsg := BaseMessage{
		MsgID:     1001,
		SessionID: 123456789,
		PlayerID:  987654321,
		ServerID:  1,
		Timestamp: 1640995200,
		Data:      []byte("test data"),
	}

	// 创建跨服务器消息
	crossMsg := CrossServerMessage{
		TraceID:      1234567890,
		FromService:  ServiceTypeGateway,
		ToService:    ServiceTypeGame,
		FromServerID: 1,
		ToServerID:   2,
		Message:      baseMsg,
	}

	// 序列化
	data, err := json.Marshal(crossMsg)
	if err != nil {
		t.Fatalf("Failed to marshal cross server message: %v", err)
	}

	// 反序列化
	var decodedMsg CrossServerMessage
	if err := json.Unmarshal(data, &decodedMsg); err != nil {
		t.Fatalf("Failed to unmarshal cross server message: %v", err)
	}

	// 验证字段
	if decodedMsg.TraceID != crossMsg.TraceID {
		t.Errorf("TraceID mismatch: got %d, want %d", decodedMsg.TraceID, crossMsg.TraceID)
	}
	if decodedMsg.FromService != crossMsg.FromService {
		t.Errorf("FromService mismatch: got %d, want %d", decodedMsg.FromService, crossMsg.FromService)
	}
	if decodedMsg.ToService != crossMsg.ToService {
		t.Errorf("ToService mismatch: got %d, want %d", decodedMsg.ToService, crossMsg.ToService)
	}
	if decodedMsg.Message.MsgID != baseMsg.MsgID {
		t.Errorf("Message.MsgID mismatch: got %d, want %d", decodedMsg.Message.MsgID, baseMsg.MsgID)
	}
}
