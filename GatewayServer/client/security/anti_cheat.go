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

// ClientStats 客户端行为统计。
//
// 统计**主体**（Subject）由调用方决定，见 GatewayServer/client/message_handler.go 的 cheatSubject：
// 已认证会话用 `acct:<账号ID>`，未认证阶段才回退 `ip:<地址>`。
// ⚠ 别改回按 IP 聚合：CGNAT/网吧下大量玩家共用一个出口 IP，动作数叠加会一人超标、全楼连坐；
// 也别改回按 sessionID：重连即换 ID、统计清零，检测会被绕过（SEC-4 的原始教训）。
type ClientStats struct {
	Subject         string
	LoginTime       time.Time
	LastActionTime  time.Time
	WindowStart     time.Time // SEC-4: 当前速率统计窗口起点，每分钟滚动重置 ActionCount/ErrorCount
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
func (acm *AntiCheatManager) RecordClientAction(subject string, packetSize int) {
	now := time.Now()
	stats, exists := acm.clientStats.Load(subject)
	if !exists {
		stats = &ClientStats{
			Subject:        subject,
			LoginTime:      now,
			LastActionTime: now,
			WindowStart:    now,
		}
		acm.clientStats.Store(subject, stats)
	}

	// SEC-4: 每分钟滚动重置速率窗口。此前用 `time.Since(LoginTime) < 1min` 判定→登录一分钟后
	// 速率检测永久失效、ActionCount 无界累积。改为真实滚动窗口，使"每分钟动作数上限"持续生效。
	// AbnormalActions（累积违规数）不随窗口清零，由 CheckClientStatus 据其封禁。
	if now.Sub(stats.WindowStart) >= time.Minute {
		stats.WindowStart = now
		stats.ActionCount = 0
		stats.ErrorCount = 0
	}

	stats.LastActionTime = now
	stats.ActionCount++
	stats.PacketCount++

	acm.checkAbnormalBehavior(stats)
}

// RecordError 记录错误
func (acm *AntiCheatManager) RecordError(subject string, errorType string) {
	stats, exists := acm.clientStats.Load(subject)
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

	reports, _ := acm.cheatReports.Load(subject)
	reports = append(reports, report)
	acm.cheatReports.Store(subject, reports)
}

// checkAbnormalBehavior 检查异常行为
func (acm *AntiCheatManager) checkAbnormalBehavior(stats *ClientStats) {
	ac := acm.config.AntiCheat

	// SEC-4: 窗口内动作数超上限即判违规（窗口每分钟由 RecordClientAction 滚动重置）。
	if ac.MaxActionsPerMinute > 0 && stats.ActionCount > ac.MaxActionsPerMinute {
		acm.reportCheat(stats.Subject, "HighActionRate", "Too many actions in short time", 3)
		stats.AbnormalActions++
	}

	if ac.MaxErrorRatio > 0 && stats.ActionCount > 0 {
		errorRatio := float64(stats.ErrorCount) / float64(stats.ActionCount)
		if errorRatio > ac.MaxErrorRatio {
			acm.reportCheat(stats.Subject, "HighErrorRate", "Too many errors", 2)
			stats.AbnormalActions++
		}
	}
}

// reportCheat 报告作弊行为
func (acm *AntiCheatManager) reportCheat(subject, cheatType, description string, severity int) {
	report := &CheatReport{
		Time:        time.Now(),
		Type:        cheatType,
		Description: description,
		Severity:    severity,
	}

	reports, _ := acm.cheatReports.Load(subject)
	reports = append(reports, report)
	acm.cheatReports.Store(subject, reports)

	zLog.Warn("Cheat detected",
		zap.String("subject", subject),
		zap.String("type", cheatType),
		zap.String("description", description),
		zap.Int("severity", severity))
}

// CheckClientStatus 检查客户端状态
func (acm *AntiCheatManager) CheckClientStatus(subject string) (bool, string) {
	ac := acm.config.AntiCheat

	stats, exists := acm.clientStats.Load(subject)
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

	reports, _ := acm.cheatReports.Load(subject)
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

	acm.clientStats.Range(func(subject string, stats *ClientStats) bool {
		if now.Sub(stats.LastActionTime) > inactiveTimeout {
			toDelete = append(toDelete, subject)
		}
		return true
	})

	for _, subject := range toDelete {
		acm.clientStats.Delete(subject)
		acm.cheatReports.Delete(subject)
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
