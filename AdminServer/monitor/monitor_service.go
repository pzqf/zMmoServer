package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/AdminServer/config"
	"go.uber.org/zap"
)

// MonitorService 监控服务
type MonitorService struct {
	mu           sync.RWMutex
	serverStatus map[string]*ServerStatus // 服务器状态
	alerts       []Alert                  // 告警列表
	alertRules   []AlertRule              // 告警规则
	isRunning    bool
	ctx          context.Context
	cancel       context.CancelFunc
}

// ServerStatus 服务器状态
type ServerStatus struct {
	ServerID      string    `json:"server_id"`
	ServerType    string    `json:"server_type"`
	ServerName    string    `json:"server_name"`
	Status        string    `json:"status"` // online, offline, warning
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   float64   `json:"memory_usage"`
	ConnectionCount int     `json:"connection_count"`
	Uptime        int64     `json:"uptime"`
	Version       string    `json:"version"`
}

// Alert 告警信息
type Alert struct {
	ID          string    `json:"id"`
	ServerID    string    `json:"server_id"`
	AlertType   string    `json:"alert_type"`
	Severity    string    `json:"severity"` // critical, warning, info
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	IsResolved  bool      `json:"is_resolved"`
	ResolvedAt  time.Time `json:"resolved_at"`
}

// AlertRule 告警规则
type AlertRule struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ServerType  string  `json:"server_type"`
	Metric      string  `json:"metric"`
	Operator    string  `json:"operator"` // >, <, ==, >=, <=
	Threshold   float64 `json:"threshold"`
	Severity    string  `json:"severity"`
	Enabled     bool    `json:"enabled"`
}

// NewMonitorService 创建监控服务
func NewMonitorService(cfg *config.Config) *MonitorService {
	ctx, cancel := context.WithCancel(context.Background())
	return &MonitorService{
		serverStatus: make(map[string]*ServerStatus),
		alerts:       make([]Alert, 0),
		alertRules:   make([]AlertRule, 0),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start 启动监控服务
func (ms *MonitorService) Start() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.isRunning {
		return fmt.Errorf("monitor service already running")
	}

	ms.isRunning = true

	// 注册HTTP路由
	http.HandleFunc("/api/monitor/status", ms.handleGetStatus)
	http.HandleFunc("/api/monitor/servers", ms.handleGetServers)
	http.HandleFunc("/api/monitor/alerts", ms.handleGetAlerts)
	http.HandleFunc("/api/monitor/heartbeat", ms.handleHeartbeat)
	http.HandleFunc("/api/monitor/rules", ms.handleAlertRules)

	// 启动HTTP服务
	go func() {
		zLog.Info("Monitor service started")
		if err := http.ListenAndServe(":8083", nil); err != nil {
			zLog.Error("Monitor service failed", zap.Error(err))
		}
	}()

	// 启动监控任务
	go ms.monitorLoop()

	return nil
}

// Stop 停止监控服务
func (ms *MonitorService) Stop() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if !ms.isRunning {
		return
	}

	ms.isRunning = false
	ms.cancel()

	zLog.Info("Monitor service stopped")
}

// monitorLoop 监控循环
func (ms *MonitorService) monitorLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ms.ctx.Done():
			return
		case <-ticker.C:
			ms.checkServerHealth()
			ms.checkAlertRules()
		}
	}
}

// checkServerHealth 检查服务器健康状态
func (ms *MonitorService) checkServerHealth() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	for serverID, status := range ms.serverStatus {
		// 检查心跳超时（5分钟）
		if now.Sub(status.LastHeartbeat) > 5*time.Minute {
			if status.Status != "offline" {
				status.Status = "offline"
				ms.addAlert(serverID, "ServerOffline", "critical",
					fmt.Sprintf("Server %s is offline", serverID))
			}
		}
	}
}

// checkAlertRules 检查告警规则
func (ms *MonitorService) checkAlertRules() {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, rule := range ms.alertRules {
		if !rule.Enabled {
			continue
		}

		for serverID, status := range ms.serverStatus {
			if status.Status == "offline" {
				continue
			}

			var value float64
			switch rule.Metric {
			case "cpu_usage":
				value = status.CPUUsage
			case "memory_usage":
				value = status.MemoryUsage
			case "connection_count":
				value = float64(status.ConnectionCount)
			default:
				continue
			}

			// 检查是否触发告警
			if ms.checkThreshold(value, rule.Operator, rule.Threshold) {
				ms.addAlert(serverID, rule.Name, rule.Severity,
					fmt.Sprintf("%s: %.2f %s %.2f", rule.Metric, value, rule.Operator, rule.Threshold))
			}
		}
	}
}

// checkThreshold 检查阈值
func (ms *MonitorService) checkThreshold(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case "==":
		return value == threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	default:
		return false
	}
}

// addAlert 添加告警
func (ms *MonitorService) addAlert(serverID, alertType, severity, message string) {
	alert := Alert{
		ID:        fmt.Sprintf("%s-%d", serverID, time.Now().Unix()),
		ServerID:  serverID,
		AlertType: alertType,
		Severity:  severity,
		Message:   message,
		Timestamp: time.Now(),
	}

	ms.alerts = append(ms.alerts, alert)

	zLog.Warn("Alert triggered",
		zap.String("server_id", serverID),
		zap.String("type", alertType),
		zap.String("severity", severity),
		zap.String("message", message))
}

// UpdateServerStatus 更新服务器状态
func (ms *MonitorService) UpdateServerStatus(status *ServerStatus) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	status.LastHeartbeat = time.Now()
	ms.serverStatus[status.ServerID] = status

	// 如果服务器恢复在线，标记相关告警为已解决
	if status.Status == "online" {
		ms.resolveAlerts(status.ServerID, "ServerOffline")
	}
}

// resolveAlerts 解决告警
func (ms *MonitorService) resolveAlerts(serverID, alertType string) {
	for i := range ms.alerts {
		if ms.alerts[i].ServerID == serverID &&
			ms.alerts[i].AlertType == alertType &&
			!ms.alerts[i].IsResolved {
			ms.alerts[i].IsResolved = true
			ms.alerts[i].ResolvedAt = time.Now()
		}
	}
}

// AddAlertRule 添加告警规则
func (ms *MonitorService) AddAlertRule(rule AlertRule) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.alertRules = append(ms.alertRules, rule)
}

// RemoveAlertRule 移除告警规则
func (ms *MonitorService) RemoveAlertRule(ruleID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for i, rule := range ms.alertRules {
		if rule.ID == ruleID {
			ms.alertRules = append(ms.alertRules[:i], ms.alertRules[i+1:]...)
			break
		}
	}
}

// HTTP处理函数

func (ms *MonitorService) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now(),
		"servers":   len(ms.serverStatus),
		"alerts":    len(ms.alerts),
	})
}

func (ms *MonitorService) handleGetServers(w http.ResponseWriter, r *http.Request) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ms.serverStatus)
}

func (ms *MonitorService) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ms.alerts)
}

func (ms *MonitorService) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var status ServerStatus
	if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ms.UpdateServerStatus(&status)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (ms *MonitorService) handleAlertRules(w http.ResponseWriter, r *http.Request) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ms.alertRules)
}
