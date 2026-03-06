package auth

import (
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoShared/common/id"
	"go.uber.org/zap"
)

// AntiCheatManager 防作弊管理器
type AntiCheatManager struct {
	config           *config.Config
	playerStats      map[id.PlayerIdType]*PlayerStats // 玩家统计
	cheatReports     map[id.PlayerIdType][]CheatReport // 作弊报告
	playerStatsMutex sync.RWMutex
	cheatReportsMutex sync.RWMutex
}

// PlayerStats 玩家行为统计
type PlayerStats struct {
	PlayerID          id.PlayerIdType
	LastMoveTime      time.Time
	LastAttackTime    time.Time
	LastSkillTime     time.Time
	MoveCount         int
	AttackCount       int
	SkillCount        int
	MoveSpeed         float64
	AttackFrequency   float64
	SkillFrequency    float64
	AbnormalActions   int
}

// CheatReport 作弊报告
type CheatReport struct {
	Time        time.Time
	Type        string
	Description string
	Severity    int // 1-低, 2-中, 3-高
}

// NewAntiCheatManager 创建防作弊管理器
func NewAntiCheatManager(cfg *config.Config) *AntiCheatManager {
	return &AntiCheatManager{
		config:       cfg,
		playerStats:  make(map[id.PlayerIdType]*PlayerStats),
		cheatReports: make(map[id.PlayerIdType][]CheatReport),
	}
}

// RecordPlayerMove 记录玩家移动
func (acm *AntiCheatManager) RecordPlayerMove(playerID id.PlayerIdType, speed float64) {
	acm.playerStatsMutex.Lock()
	defer acm.playerStatsMutex.Unlock()

	stats, exists := acm.playerStats[playerID]
	if !exists {
		stats = &PlayerStats{
			PlayerID:     playerID,
			LastMoveTime: time.Now(),
		}
		acm.playerStats[playerID] = stats
	}

	// 计算移动速度
	now := time.Now()
	timeDiff := now.Sub(stats.LastMoveTime).Seconds()
	if timeDiff > 0 {
		stats.MoveSpeed = speed
	}
	stats.LastMoveTime = now
	stats.MoveCount++

	// 检查异常移动
	acm.checkAbnormalMove(stats)
}

// RecordPlayerAttack 记录玩家攻击
func (acm *AntiCheatManager) RecordPlayerAttack(playerID id.PlayerIdType) {
	acm.playerStatsMutex.Lock()
	defer acm.playerStatsMutex.Unlock()

	stats, exists := acm.playerStats[playerID]
	if !exists {
		stats = &PlayerStats{
			PlayerID:       playerID,
			LastAttackTime: time.Now(),
		}
		acm.playerStats[playerID] = stats
	}

	// 计算攻击频率
	now := time.Now()
	timeDiff := now.Sub(stats.LastAttackTime).Seconds()
	if timeDiff > 0 {
		stats.AttackFrequency = 1.0 / timeDiff
	}
	stats.LastAttackTime = now
	stats.AttackCount++

	// 检查异常攻击
	acm.checkAbnormalAttack(stats)
}

// RecordPlayerSkill 记录玩家技能使用
func (acm *AntiCheatManager) RecordPlayerSkill(playerID id.PlayerIdType) {
	acm.playerStatsMutex.Lock()
	defer acm.playerStatsMutex.Unlock()

	stats, exists := acm.playerStats[playerID]
	if !exists {
		stats = &PlayerStats{
			PlayerID:      playerID,
			LastSkillTime: time.Now(),
		}
		acm.playerStats[playerID] = stats
	}

	// 计算技能使用频率
	now := time.Now()
	timeDiff := now.Sub(stats.LastSkillTime).Seconds()
	if timeDiff > 0 {
		stats.SkillFrequency = 1.0 / timeDiff
	}
	stats.LastSkillTime = now
	stats.SkillCount++

	// 检查异常技能使用
	acm.checkAbnormalSkill(stats)
}

// checkAbnormalMove 检查异常移动
func (acm *AntiCheatManager) checkAbnormalMove(stats *PlayerStats) {
	// 检查移动速度
	if stats.MoveSpeed > 10.0 { // 假设正常最大速度为10
		acm.reportCheat(stats.PlayerID, "HighMoveSpeed", "Abnormal movement speed", 3)
		stats.AbnormalActions++
	}

	// 检查移动频率
	if stats.MoveCount > 100 && time.Since(stats.LastMoveTime) < time.Minute {
		acm.reportCheat(stats.PlayerID, "HighMoveFrequency", "Too many moves in short time", 2)
		stats.AbnormalActions++
	}
}

// checkAbnormalAttack 检查异常攻击
func (acm *AntiCheatManager) checkAbnormalAttack(stats *PlayerStats) {
	// 检查攻击频率
	if stats.AttackFrequency > 10.0 { // 假设正常最大攻击频率为10次/秒
		acm.reportCheat(stats.PlayerID, "HighAttackFrequency", "Abnormal attack frequency", 3)
		stats.AbnormalActions++
	}

	// 检查攻击次数
	if stats.AttackCount > 500 && time.Since(stats.LastAttackTime) < time.Minute {
		acm.reportCheat(stats.PlayerID, "HighAttackCount", "Too many attacks in short time", 2)
		stats.AbnormalActions++
	}
}

// checkAbnormalSkill 检查异常技能使用
func (acm *AntiCheatManager) checkAbnormalSkill(stats *PlayerStats) {
	// 检查技能使用频率
	if stats.SkillFrequency > 5.0 { // 假设正常最大技能频率为5次/秒
		acm.reportCheat(stats.PlayerID, "HighSkillFrequency", "Abnormal skill usage frequency", 3)
		stats.AbnormalActions++
	}

	// 检查技能使用次数
	if stats.SkillCount > 200 && time.Since(stats.LastSkillTime) < time.Minute {
		acm.reportCheat(stats.PlayerID, "HighSkillCount", "Too many skills in short time", 2)
		stats.AbnormalActions++
	}
}

// reportCheat 报告作弊行为
func (acm *AntiCheatManager) reportCheat(playerID id.PlayerIdType, cheatType, description string, severity int) {
	acm.cheatReportsMutex.Lock()
	defer acm.cheatReportsMutex.Unlock()

	report := CheatReport{
		Time:        time.Now(),
		Type:        cheatType,
		Description: description,
		Severity:    severity,
	}

	acm.cheatReports[playerID] = append(acm.cheatReports[playerID], report)

	zLog.Warn("Cheat detected",
		zap.Uint64("player_id", uint64(playerID)),
		zap.String("type", cheatType),
		zap.String("description", description),
		zap.Int("severity", severity))
}

// CheckPlayerStatus 检查玩家状态
func (acm *AntiCheatManager) CheckPlayerStatus(playerID id.PlayerIdType) (bool, string) {
	acm.playerStatsMutex.RLock()
	stats, exists := acm.playerStats[playerID]
	acm.playerStatsMutex.RUnlock()

	if !exists {
		return true, ""
	}

	// 检查异常行为次数
	if stats.AbnormalActions > 5 {
		return false, "Too many abnormal actions"
	}

	// 检查作弊报告
	acm.cheatReportsMutex.RLock()
	reports := acm.cheatReports[playerID]
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

// GetPlayerStats 获取玩家统计信息
func (acm *AntiCheatManager) GetPlayerStats(playerID id.PlayerIdType) *PlayerStats {
	acm.playerStatsMutex.RLock()
	defer acm.playerStatsMutex.RUnlock()

	return acm.playerStats[playerID]
}

// GetCheatReports 获取作弊报告
func (acm *AntiCheatManager) GetCheatReports(playerID id.PlayerIdType) []CheatReport {
	acm.cheatReportsMutex.RLock()
	defer acm.cheatReportsMutex.RUnlock()

	return acm.cheatReports[playerID]
}

// CleanupInactivePlayers 清理不活跃的玩家
func (acm *AntiCheatManager) CleanupInactivePlayers() {
	acm.playerStatsMutex.Lock()
	defer acm.playerStatsMutex.Unlock()

	now := time.Now()
	for playerID, stats := range acm.playerStats {
		// 检查最后活动时间
		lastActivity := stats.LastMoveTime
		if stats.LastAttackTime.After(lastActivity) {
			lastActivity = stats.LastAttackTime
		}
		if stats.LastSkillTime.After(lastActivity) {
			lastActivity = stats.LastSkillTime
		}

		if now.Sub(lastActivity) > 30*time.Minute {
			delete(acm.playerStats, playerID)
			
			// 清理对应的作弊报告
			acm.cheatReportsMutex.Lock()
			delete(acm.cheatReports, playerID)
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
			acm.CleanupInactivePlayers()
		}
	}()
}
