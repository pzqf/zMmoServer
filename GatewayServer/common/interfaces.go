package common

import (
	"context"

	"github.com/pzqf/zEngine/zNet"
)

// ClientServiceInterface 客户端服务接口
type ClientServiceInterface interface {
	Start() error
	Stop() error
	SendToClient(sessionID zNet.SessionIdType, data []byte) error
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

// AntiCheatManagerInterface 防作弊管理器接口
type AntiCheatManagerInterface interface {
	RecordClientAction(ip string, packetSize int)
	RecordError(ip string, errorType string)
	CheckClientStatus(ip string) (bool, string)
	StartCleanupTask()
}
