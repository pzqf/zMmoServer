package common

import (
	"context"

	"github.com/pzqf/zEngine/zNet"
)

// ClientServiceInterface 客户端服务接口
type ClientServiceInterface interface {
	Start() error
	Stop() error
	SendToClient(sessionID zNet.SessionIdType, msgID uint32, data []byte) error
	GetSessionCount() int
}

// GameServerProxyInterface GameServer代理接口
type GameServerProxyInterface interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SendToGameServer(sessionID zNet.SessionIdType, protoId int32, data []byte) error
}

// SecurityManagerInterface 安全管理器接口
type SecurityManagerInterface interface {
	CheckIPAllowed(ip string) bool
	BanIP(ip string, duration int64) error
	UnbanIP(ip string) error
	StartCleanupTask()
	AddConnection(ip string)
	RemoveConnection(ip string)
}

// AntiCheatManagerInterface 防作弊管理器接口。
//
// subject = 统计**主体**，由调用方决定（见 client/message_handler.go 的 cheatSubject）：
// 已认证会话用 `acct:<账号ID>`、未认证阶段回退 `ip:<地址>`。
// ⚠ 别传裸 IP：NAT/CGNAT 下大量玩家共用出口 IP，会一人超标、全楼连坐。
type AntiCheatManagerInterface interface {
	RecordClientAction(subject string, packetSize int)
	RecordError(subject string, errorType string)
	CheckClientStatus(subject string) (bool, string)
	StartCleanupTask(ctx context.Context)
	Stop()
}
