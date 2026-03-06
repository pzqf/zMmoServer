package auth

import (
	"testing"
	"time"

	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/stretchr/testify/assert"
)

func TestNewSecurityManager(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			MaxLoginAttempts: 3,
			BanDuration:      3600,
		},
	}

	sm := NewSecurityManager(cfg)
	assert.NotNil(t, sm)
}

func TestSecurityManager_CheckIPAllowed(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			MaxLoginAttempts: 3,
			BanDuration:      3600,
		},
	}

	sm := NewSecurityManager(cfg)

	// 测试未被封禁的IP
	assert.True(t, sm.CheckIPAllowed("192.168.1.1"))

	// 测试被封禁的IP
	sm.RecordLoginAttempt("192.168.1.2", false)
	sm.RecordLoginAttempt("192.168.1.2", false)
	sm.RecordLoginAttempt("192.168.1.2", false)

	assert.False(t, sm.CheckIPAllowed("192.168.1.2"))
}

func TestSecurityManager_RecordLoginAttempt(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			MaxLoginAttempts: 3,
			BanDuration:      3600,
		},
	}

	sm := NewSecurityManager(cfg)

	// 测试登录失败
	sm.RecordLoginAttempt("192.168.1.1", false)
	sm.RecordLoginAttempt("192.168.1.1", false)
	assert.Equal(t, 2, sm.GetLoginAttempts("192.168.1.1"))

	// 测试登录成功，应该重置尝试次数
	sm.RecordLoginAttempt("192.168.1.1", true)
	assert.Equal(t, 0, sm.GetLoginAttempts("192.168.1.1"))

	// 测试达到最大尝试次数，应该被封禁
	sm.RecordLoginAttempt("192.168.1.2", false)
	sm.RecordLoginAttempt("192.168.1.2", false)
	sm.RecordLoginAttempt("192.168.1.2", false)

	banned, _ := sm.GetBanStatus("192.168.1.2")
	assert.True(t, banned)
}

func TestSecurityManager_GetBanStatus(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			MaxLoginAttempts: 3,
			BanDuration:      1, // 1秒封禁
		},
	}

	sm := NewSecurityManager(cfg)

	// 测试未被封禁的IP
	banned, _ := sm.GetBanStatus("192.168.1.1")
	assert.False(t, banned)

	// 测试被封禁的IP
	sm.RecordLoginAttempt("192.168.1.2", false)
	sm.RecordLoginAttempt("192.168.1.2", false)
	sm.RecordLoginAttempt("192.168.1.2", false)

	//banned, banTime := sm.GetBanStatus("192.168.1.2")
	//assert.True(t, banned)
	//assert.After(time.Now(), banTime)

	// 等待封禁时间过期
	time.Sleep(1 * time.Second)

	banned, _ = sm.GetBanStatus("192.168.1.2")
	assert.False(t, banned)
}

func TestSecurityManager_UnbanIP(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			MaxLoginAttempts: 3,
			BanDuration:      3600,
		},
	}

	sm := NewSecurityManager(cfg)

	// 封禁IP
	sm.RecordLoginAttempt("192.168.1.1", false)
	sm.RecordLoginAttempt("192.168.1.1", false)
	sm.RecordLoginAttempt("192.168.1.1", false)

	banned, _ := sm.GetBanStatus("192.168.1.1")
	assert.True(t, banned)

	// 解除封禁
	sm.UnbanIP("192.168.1.1")

	banned, _ = sm.GetBanStatus("192.168.1.1")
	assert.False(t, banned)
}

func TestSecurityManager_CleanupExpiredBans(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			MaxLoginAttempts: 3,
			BanDuration:      1, // 1秒封禁
		},
	}

	sm := NewSecurityManager(cfg)

	// 封禁IP
	sm.RecordLoginAttempt("192.168.1.1", false)
	sm.RecordLoginAttempt("192.168.1.1", false)
	sm.RecordLoginAttempt("192.168.1.1", false)

	banned, _ := sm.GetBanStatus("192.168.1.1")
	assert.True(t, banned)

	// 等待封禁时间过期
	time.Sleep(1 * time.Second)

	// 清理过期封禁
	sm.CleanupExpiredBans()

	banned, _ = sm.GetBanStatus("192.168.1.1")
	assert.False(t, banned)
}
