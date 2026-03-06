package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/connection"
	"go.uber.org/zap"
)

// HeartbeatReporter 心跳上报器
type HeartbeatReporter struct {
	config        *config.Config
	connManager   *connection.ConnectionManager
	adminAddr     string
	serverID      string
	serverType    string
	serverName    string
	version       string
	startTime     time.Time
	isRunning     bool
	stopChan      chan struct{}
}

// ServerStatus 服务器状态
type ServerStatus struct {
	ServerID        string    `json:"server_id"`
	ServerType      string    `json:"server_type"`
	ServerName      string    `json:"server_name"`
	Status          string    `json:"status"`
	LastHeartbeat   time.Time `json:"last_heartbeat"`
	CPUUsage        float64   `json:"cpu_usage"`
	MemoryUsage     float64   `json:"memory_usage"`
	ConnectionCount int       `json:"connection_count"`
	Uptime          int64     `json:"uptime"`
	Version         string    `json:"version"`
}

// NewHeartbeatReporter 创建心跳上报器
func NewHeartbeatReporter(cfg *config.Config, connManager *connection.ConnectionManager, adminAddr string) *HeartbeatReporter {
	return &HeartbeatReporter{
		config:      cfg,
		connManager: connManager,
		adminAddr:   adminAddr,
		serverID:    fmt.Sprintf("gateway-%d", cfg.Server.ServerID),
		serverType:  "gateway",
		serverName:  cfg.Server.ServerName,
		version:     "1.0.0",
		startTime:   time.Now(),
		stopChan:    make(chan struct{}),
	}
}

// Start 启动心跳上报
func (hr *HeartbeatReporter) Start() {
	if hr.isRunning {
		return
	}

	hr.isRunning = true

	go hr.reportLoop()

	zLog.Info("Heartbeat reporter started",
		zap.String("server_id", hr.serverID),
		zap.String("admin_addr", hr.adminAddr))
}

// Stop 停止心跳上报
func (hr *HeartbeatReporter) Stop() {
	if !hr.isRunning {
		return
	}

	hr.isRunning = false
	close(hr.stopChan)

	zLog.Info("Heartbeat reporter stopped")
}

// reportLoop 上报循环
func (hr *HeartbeatReporter) reportLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 立即上报一次
	hr.report()

	for {
		select {
		case <-hr.stopChan:
			return
		case <-ticker.C:
			hr.report()
		}
	}
}

// report 上报状态
func (hr *HeartbeatReporter) report() {
	status := hr.collectStatus()

	data, err := json.Marshal(status)
	if err != nil {
		zLog.Error("Failed to marshal status", zap.Error(err))
		return
	}

	url := fmt.Sprintf("http://%s/api/monitor/heartbeat", hr.adminAddr)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		zLog.Warn("Failed to report heartbeat", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		zLog.Warn("Heartbeat report failed", zap.Int("status_code", resp.StatusCode))
		return
	}

	zLog.Debug("Heartbeat reported successfully")
}

// collectStatus 收集服务器状态
func (hr *HeartbeatReporter) collectStatus() *ServerStatus {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 计算内存使用率（简化计算，实际应该使用系统API）
	memoryUsage := float64(m.Alloc) / float64(m.Sys) * 100

	return &ServerStatus{
		ServerID:        hr.serverID,
		ServerType:      hr.serverType,
		ServerName:      hr.serverName,
		Status:          "online",
		LastHeartbeat:   time.Now(),
		CPUUsage:        0, // 需要系统API获取
		MemoryUsage:     memoryUsage,
		ConnectionCount: hr.connManager.GetConnectionCount(),
		Uptime:          int64(time.Since(hr.startTime).Seconds()),
		Version:         hr.version,
	}
}
