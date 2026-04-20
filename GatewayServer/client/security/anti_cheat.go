package security

import (
	"context"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zUtil/zMap"
	"go.uber.org/zap"
)

// AntiCheatManager 防作弊管理器
type AntiCheatManager struct {
	config       *config.Config
	clientStats  *zMap.TypedMap[string, *ClientStats]
	cheatReports *zMap.TypedMap[string, []*CheatReport]
	ctx          context.Context
	cancel       context.CancelFunc
}

// ClientStats 客户端行为统计
type ClientStats struct {
	IP              string
	LoginTime       time.Time
	LastActionTime  time.Time
	ActionCount     int
	PacketCount     int
	ErrorCount      int
	AbnormalActions int
}

// CheatReport 作弊报告
type CheatReport struct {
	Time        time.Time
	Type        string
	Description string
	Severity    int
}

// NewAntiCheatManager 创建防作弊管理器
func NewAntiCheatManager(cfg *config.Config) *AntiCheatManager {
	return &AntiCheatManager{
		config:       cfg,
		clientStats:  zMap.NewTypedMap[string, *ClientStats](),
		cheatReports: zMap.NewTypedMap[string, []*CheatReport](),
	}
}

// RecordClientAction 记录客户端行为
func (acm *AntiCheatManager) RecordClientAction(ip string, packetSize int) {
	stats, exists := acm.clientStats.Load(ip)
	if !exists {
		stats = &ClientStats{
			IP:             ip,
			LoginTime:      time.Now(),
			LastActionTime: time.Now(),
		}
		acm.clientStats.Store(ip, stats)
	}

	stats.LastActionTime = time.Now()
	stats.ActionCount++
	stats.PacketCount++

	acm.checkAbnormalBehavior(stats)
}

// RecordError 记录错误
func (acm *AntiCheatManager) RecordError(ip string, errorType string) {
	stats, exists := acm.clientStats.Load(ip)
	if !exists {
		return
	}

	stats.ErrorCount++

	report := &CheatReport{
		Time:        time.Now(),
		Type:        errorType,
		Description: "Client error detected",
		Severity:    1,
	}

	reports, _ := acm.cheatReports.Load(ip)
	reports = append(reports, report)
	acm.cheatReports.Store(ip, reports)
}

// checkAbnormalBehavior 检查异常行为
func (acm *AntiCheatManager) checkAbnormalBehavior(stats *ClientStats) {
	ac := acm.config.AntiCheat

	if ac.MaxActionsPerMinute > 0 && stats.ActionCount > ac.MaxActionsPerMinute && time.Since(stats.LoginTime) < time.Minute {
		acm.reportCheat(stats.IP, "HighActionRate", "Too many actions in short time", 3)
		stats.AbnormalActions++
	}

	if ac.MaxErrorRatio > 0 && stats.ActionCount > 0 {
		errorRatio := float64(stats.ErrorCount) / float64(stats.ActionCount)
		if errorRatio > ac.MaxErrorRatio {
			acm.reportCheat(stats.IP, "HighErrorRate", "Too many errors", 2)
			stats.AbnormalActions++
		}
	}
}

// reportCheat 报告作弊行为
func (acm *AntiCheatManager) reportCheat(ip, cheatType, description string, severity int) {
	report := &CheatReport{
		Time:        time.Now(),
		Type:        cheatType,
		Description: description,
		Severity:    severity,
	}

	reports, _ := acm.cheatReports.Load(ip)
	reports = append(reports, report)
	acm.cheatReports.Store(ip, reports)

	zLog.Warn("Cheat detected",
		zap.String("ip", ip),
		zap.String("type", cheatType),
		zap.String("description", description),
		zap.Int("severity", severity))
}

// CheckClientStatus 检查客户端状态
func (acm *AntiCheatManager) CheckClientStatus(ip string) (bool, string) {
	ac := acm.config.AntiCheat

	stats, exists := acm.clientStats.Load(ip)
	if !exists {
		return true, ""
	}

	maxAbnormal := ac.MaxAbnormalActions
	if maxAbnormal <= 0 {
		maxAbnormal = 5
	}
	if stats.AbnormalActions > maxAbnormal {
		return false, "Too many abnormal actions"
	}

	reports, _ := acm.cheatReports.Load(ip)
	maxHighSeverity := ac.MaxHighSeverityReports
	if maxHighSeverity <= 0 {
		maxHighSeverity = 3
	}

	highSeverityCount := 0
	for _, report := range reports {
		if report.Severity >= 3 {
			highSeverityCount++
		}
	}

	if highSeverityCount >= maxHighSeverity {
		return false, "Multiple high severity cheat attempts"
	}

	return true, ""
}

// CleanupInactiveClients 清理不活跃的客户端
func (acm *AntiCheatManager) CleanupInactiveClients() {
	ac := acm.config.AntiCheat
	inactiveTimeout := time.Duration(ac.InactiveTimeoutMinutes) * time.Minute
	if inactiveTimeout <= 0 {
		inactiveTimeout = 30 * time.Minute
	}

	now := time.Now()
	var toDelete []string

	acm.clientStats.Range(func(ip string, stats *ClientStats) bool {
		if now.Sub(stats.LastActionTime) > inactiveTimeout {
			toDelete = append(toDelete, ip)
		}
		return true
	})

	for _, ip := range toDelete {
		acm.clientStats.Delete(ip)
		acm.cheatReports.Delete(ip)
	}
}

// StartCleanupTask 启动清理任务
func (acm *AntiCheatManager) StartCleanupTask(ctx context.Context) {
	acm.ctx, acm.cancel = context.WithCancel(ctx)

	go func() {
		ac := acm.config.AntiCheat
		cleanupInterval := time.Duration(ac.CleanupIntervalMinutes) * time.Minute
		if cleanupInterval <= 0 {
			cleanupInterval = 10 * time.Minute
		}

		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-acm.ctx.Done():
				return
			case <-ticker.C:
				acm.CleanupInactiveClients()
			}
		}
	}()
}

// Stop 停止防作弊管理器
func (acm *AntiCheatManager) Stop() {
	if acm.cancel != nil {
		acm.cancel()
	}
	acm.clientStats.Clear()
	acm.cheatReports.Clear()
	zLog.Info("AntiCheatManager stopped")
}
