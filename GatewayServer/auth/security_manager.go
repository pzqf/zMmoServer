package auth

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"go.uber.org/zap"
)

// SecurityManager 安全管理器
type SecurityManager struct {
	config            *config.Config
	loginAttempts     map[string]int           // IP -> 尝试次数
	banList           map[string]time.Time     // IP -> 封禁时间
	loginAttemptsMutex sync.RWMutex
	banListMutex     sync.RWMutex
}

// NewSecurityManager 创建安全管理器
func NewSecurityManager(cfg *config.Config) *SecurityManager {
	return &SecurityManager{
		config:        cfg,
		loginAttempts: make(map[string]int),
		banList:       make(map[string]time.Time),
	}
}

// CheckIPAllowed 检查IP是否被允许
func (sm *SecurityManager) CheckIPAllowed(ip string) bool {
	sm.banListMutex.RLock()
	defer sm.banListMutex.RUnlock()

	if banTime, exists := sm.banList[ip]; exists {
		if time.Now().Before(banTime) {
			zLog.Warn("IP is banned", zap.String("ip", ip), zap.Time("ban_until", banTime))
			return false
		}
		// 封禁时间已过，移除封禁
		delete(sm.banList, ip)
	}

	return true
}

// RecordLoginAttempt 记录登录尝试
func (sm *SecurityManager) RecordLoginAttempt(ip string, success bool) {
	if success {
		// 登录成功，重置尝试次数
		sm.loginAttemptsMutex.Lock()
		delete(sm.loginAttempts, ip)
		sm.loginAttemptsMutex.Unlock()
		return
	}

	// 登录失败，增加尝试次数
	sm.loginAttemptsMutex.Lock()
	defer sm.loginAttemptsMutex.Unlock()

	sm.loginAttempts[ip]++
	attempts := sm.loginAttempts[ip]

	if attempts >= sm.config.Security.MaxLoginAttempts {
		// 达到最大尝试次数，封禁IP
		sm.banListMutex.Lock()
		banDuration := time.Duration(sm.config.Security.BanDuration) * time.Second
		sm.banList[ip] = time.Now().Add(banDuration)
		sm.banListMutex.Unlock()

		// 重置尝试次数
		delete(sm.loginAttempts, ip)

		zLog.Warn("IP banned due to too many login attempts",
			zap.String("ip", ip),
			zap.Duration("ban_duration", banDuration))
	}

	zLog.Warn("Login attempt failed",
		zap.String("ip", ip),
		zap.Int("attempts", attempts),
		zap.Int("max_attempts", sm.config.Security.MaxLoginAttempts))
}

// GetLoginAttempts 获取登录尝试次数
func (sm *SecurityManager) GetLoginAttempts(ip string) int {
	sm.loginAttemptsMutex.RLock()
	defer sm.loginAttemptsMutex.RUnlock()

	return sm.loginAttempts[ip]
}

// GetBanStatus 获取IP封禁状态
func (sm *SecurityManager) GetBanStatus(ip string) (bool, time.Time) {
	sm.banListMutex.RLock()
	defer sm.banListMutex.RUnlock()

	if banTime, exists := sm.banList[ip]; exists {
		if time.Now().Before(banTime) {
			return true, banTime
		}
		// 封禁时间已过
		return false, time.Time{}
	}

	return false, time.Time{}
}

// UnbanIP 解除IP封禁
func (sm *SecurityManager) UnbanIP(ip string) {
	sm.banListMutex.Lock()
	defer sm.banListMutex.Unlock()

	delete(sm.banList, ip)
	zLog.Info("IP unbanned", zap.String("ip", ip))
}

// CleanupExpiredBans 清理过期的封禁
func (sm *SecurityManager) CleanupExpiredBans() {
	sm.banListMutex.Lock()
	defer sm.banListMutex.Unlock()

	now := time.Now()
	for ip, banTime := range sm.banList {
		if now.After(banTime) {
			delete(sm.banList, ip)
			zLog.Info("Expired ban removed", zap.String("ip", ip))
		}
	}
}

// StartCleanupTask 启动清理任务
func (sm *SecurityManager) StartCleanupTask() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			<-ticker.C
			sm.CleanupExpiredBans()
		}
	}()
}
