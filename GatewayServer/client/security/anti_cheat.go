package security

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"go.uber.org/zap"
)

// AntiCheatManager 防作弊管理器
type AntiCheatManager struct {
	config            *config.Config
	clientStats       map[string]*ClientStats
	cheatReports      map[string][]CheatReport
	clientStatsMutex  sync.RWMutex
	cheatReportsMutex sync.RWMutex
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
		clientStats:  make(map[string]*ClientStats),
		cheatReports: make(map[string][]CheatReport),
	}
}

// RecordClientAction 记录客户端行为
func (acm *AntiCheatManager) RecordClientAction(ip string, packetSize int) {
	acm.clientStatsMutex.Lock()
	defer acm.clientStatsMutex.Unlock()

	stats, exists := acm.clientStats[ip]
	if !exists {
		stats = &ClientStats{
			IP:             ip,
			LoginTime:      time.Now(),
			LastActionTime: time.Now(),
		}
		acm.clientStats[ip] = stats
	}

	stats.LastActionTime = time.Now()
	stats.ActionCount++
	stats.PacketCount++

	// 检查异常行为
	acm.checkAbnormalBehavior(stats)
}

// RecordError 记录错误
func (acm *AntiCheatManager) RecordError(ip string, errorType string) {
	acm.clientStatsMutex.Lock()
	defer acm.clientStatsMutex.Unlock()

	stats, exists := acm.clientStats[ip]
	if !exists {
		return
	}

	stats.ErrorCount++

	// 记录作弊报告
	report := CheatReport{
		Time:        time.Now(),
		Type:        errorType,
		Description: "Client error detected",
		Severity:    1,
	}

	acm.cheatReportsMutex.Lock()
	acm.cheatReports[ip] = append(acm.cheatReports[ip], report)
	acm.cheatReportsMutex.Unlock()
}

// checkAbnormalBehavior 检查异常行为
func (acm *AntiCheatManager) checkAbnormalBehavior(stats *ClientStats) {
	// 检查操作频率
	if stats.ActionCount > 1000 && time.Since(stats.LoginTime) < time.Minute {
		acm.reportCheat(stats.IP, "HighActionRate", "Too many actions in short time", 3)
		stats.AbnormalActions++
	}

	// 检查错误率
	if stats.ErrorCount > stats.ActionCount/2 {
		acm.reportCheat(stats.IP, "HighErrorRate", "Too many errors", 2)
		stats.AbnormalActions++
	}
}

// reportCheat 报告作弊行为
func (acm *AntiCheatManager) reportCheat(ip, cheatType, description string, severity int) {
	acm.cheatReportsMutex.Lock()
	defer acm.cheatReportsMutex.Unlock()

	report := CheatReport{
		Time:        time.Now(),
		Type:        cheatType,
		Description: description,
		Severity:    severity,
	}

	acm.cheatReports[ip] = append(acm.cheatReports[ip], report)

	zLog.Warn("Cheat detected",
		zap.String("ip", ip),
		zap.String("type", cheatType),
		zap.String("description", description),
		zap.Int("severity", severity))
}

// CheckClientStatus 检查客户端状态
func (acm *AntiCheatManager) CheckClientStatus(ip string) (bool, string) {
	acm.clientStatsMutex.RLock()
	stats, exists := acm.clientStats[ip]
	acm.clientStatsMutex.RUnlock()

	if !exists {
		return true, ""
	}

	// 检查异常行为次数
	if stats.AbnormalActions > 5 {
		return false, "Too many abnormal actions"
	}

	// 检查作弊报告
	acm.cheatReportsMutex.RLock()
	reports := acm.cheatReports[ip]
	acm.cheatReportsMutex.RUnlock()

	highSeverityCount := 0
	for _, report := range reports {
		if report.Severity >= 3 {
			highSeverityCount++
		}
	}

	if highSeverityCount >= 3 {
		return false, "Multiple high severity cheat attempts"
	}

	return true, ""
}

// CleanupInactiveClients 清理不活跃的客户端
func (acm *AntiCheatManager) CleanupInactiveClients() {
	acm.clientStatsMutex.Lock()
	defer acm.clientStatsMutex.Unlock()

	now := time.Now()
	for ip, stats := range acm.clientStats {
		if now.Sub(stats.LastActionTime) > 30*time.Minute {
			delete(acm.clientStats, ip)

			// 清理对应的作弊报告
			acm.cheatReportsMutex.Lock()
			delete(acm.cheatReports, ip)
			acm.cheatReportsMutex.Unlock()
		}
	}
}

// StartCleanupTask 启动清理任务
func (acm *AntiCheatManager) StartCleanupTask() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			<-ticker.C
			acm.CleanupInactiveClients()
		}
	}()
}
