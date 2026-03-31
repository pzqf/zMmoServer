package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/connection"
	"github.com/pzqf/zMmoServer/GatewayServer/version"
	"github.com/pzqf/zCommon/protocol"
	"go.uber.org/zap"
)

// HeartbeatReporter 心跳上报器
type HeartbeatReporter struct {
	config           *config.Config
	connManager      *connection.ConnectionManager
	globalServerAddr string
	serverID         int32
	serverType       string
	serverName       string
	version          string
	startTime        time.Time
	isRunning        bool
	stopChan         chan struct{}
}

// ServerStatus 服务器状态
type ServerStatus struct {
	ServerID        int32     `json:"server_id"`
	ServerType      string    `json:"server_type"`
	ServerName      string    `json:"server_name"`
	Status          int32     `json:"status"`
	OnlineCount     int32     `json:"online_count"`
	ExternalIP      string    `json:"external_ip"`
	ExternalPort    int32     `json:"external_port"`
	Version         string    `json:"version"`
	LastHeartbeat   time.Time `json:"last_heartbeat"`
	CPUUsage        float64   `json:"cpu_usage"`
	MemoryUsage     float64   `json:"memory_usage"`
	ConnectionCount int       `json:"connection_count"`
	Uptime          int64     `json:"uptime"`
}

// NewHeartbeatReporter 创建心跳上报器
func NewHeartbeatReporter(cfg *config.Config, connManager *connection.ConnectionManager, globalServerAddr string) *HeartbeatReporter {
	return &HeartbeatReporter{
		config:           cfg,
		connManager:      connManager,
		globalServerAddr: globalServerAddr,
		serverID:         int32(cfg.Server.ServerID),
		serverType:       "gateway",
		serverName:       cfg.Server.ServerName,
		version:          version.Version,
		startTime:        time.Now(),
		stopChan:         make(chan struct{}),
	}
}

// Start 启动心跳上报
func (hr *HeartbeatReporter) Start() {
	if hr.isRunning {
		return
	}

	// 先尝试注册服务器
	if err := hr.register(); err != nil {
		zLog.Warn("Failed to register server, will retry in heartbeat", zap.Error(err))
	}

	hr.isRunning = true
	go hr.reportLoop()

	zLog.Info("Heartbeat reporter started",
		zap.Int32("server_id", hr.serverID),
		zap.String("global_server_addr", hr.globalServerAddr))
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
	// 收集状态信息
	status := hr.collectStatus()
	// 创建符合 GlobalServer 要求的心跳请求
	// 心跳上报时只需要上报在线人数和状态，不需要上报地址
	heartbeatReq := &protocol.ServerHeartbeatRequest{
		ServerId:    status.ServerID,
		OnlineCount: status.OnlineCount,
		Status:      status.Status,
		Version:     status.Version,
		Load:        0, // 暂时设为0
	}

	data, err := json.Marshal(heartbeatReq)
	if err != nil {
		zLog.Error("Failed to marshal heartbeat request", zap.Error(err))
		return
	}

	// 使用正确的URL路径
	url := fmt.Sprintf("http://%s/api/v1/server/heartbeat", hr.globalServerAddr)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		zLog.Warn("Failed to report heartbeat", zap.Error(err))
		// 如果心跳失败，尝试重新注册
		if registerErr := hr.register(); registerErr != nil {
			zLog.Warn("Failed to register server during heartbeat retry", zap.Error(registerErr))
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		zLog.Warn("Heartbeat report failed", zap.Int("status_code", resp.StatusCode))
		// 如果返回404或其他错误，尝试重新注册
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
			if registerErr := hr.register(); registerErr != nil {
				zLog.Warn("Failed to register server during retry", zap.Error(registerErr))
			}
		}
		return
	}

	// 解析响应
	var heartbeatResp protocol.ServerHeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err != nil {
		zLog.Warn("Failed to decode heartbeat response", zap.Error(err))
		return
	}

	if heartbeatResp.Result != int32(protocol.ErrorCode_ERR_SUCCESS) {
		zLog.Warn("Heartbeat report failed",
			zap.Int32("result", heartbeatResp.Result),
			zap.String("error_msg", heartbeatResp.ErrorMsg))
		// 如果是服务器未注册错误，尝试重新注册
		if heartbeatResp.Result == int32(protocol.ErrorCode_ERR_NOT_FOUND) {
			if registerErr := hr.register(); registerErr != nil {
				zLog.Warn("Failed to register server during retry", zap.Error(registerErr))
			}
		}
		return
	}

	zLog.Info("Heartbeat reported successfully")
}

// register 注册服务器到 GlobalServer
func (hr *HeartbeatReporter) register() error {
	// 收集状态信息
	status := hr.collectStatus()

	// 创建注册请求
	registerReq := &protocol.ServerRegisterRequest{
		ServerId:       status.ServerID,
		ServerName:     hr.serverName,
		ServerType:     hr.serverType,
		GroupId:        1, // 默认分组ID
		Address:        status.ExternalIP,
		Port:           status.ExternalPort,
		MaxOnlineCount: int32(hr.config.Server.MaxConnections),
		Region:         "default", // 默认区域
		Version:        status.Version,
	}

	data, err := json.Marshal(registerReq)
	if err != nil {
		return fmt.Errorf("marshal register request: %w", err)
	}

	// 发送注册请求
	url := fmt.Sprintf("http://%s/api/v1/server/register", hr.globalServerAddr)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("send register request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register failed with status code: %d", resp.StatusCode)
	}

	// 解析响应
	var registerResp protocol.ServerRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		return fmt.Errorf("decode register response: %w", err)
	}

	if registerResp.Result != int32(protocol.ErrorCode_ERR_SUCCESS) {
		return fmt.Errorf("register failed: %s", registerResp.ErrorMsg)
	}

	zLog.Info("Server registered successfully", zap.Int32("server_id", hr.serverID))
	return nil
}

// collectStatus 收集服务器状态
func (hr *HeartbeatReporter) collectStatus() *ServerStatus {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 计算内存使用率（简化计算，实际应该使用系统API）
	memoryUsage := float64(m.Alloc) / float64(m.Sys) * 100

	// 解析对外服务地址（客户端实际连接的地址）
	externalIP := "0.0.0.0"
	externalPort := int32(0)

	// 优先从环境变量获取对外地址
	addr := os.Getenv("GATEWAY_EXTERNAL_ADDR")
	// 其次使用配置的外网地址
	if addr == "" {
		addr = hr.config.Server.ExternalAddr
	}
	// 最后使用监听地址
	if addr == "" {
		addr = hr.config.Server.ListenAddr
	}

	if addr != "" {
		// 提取端口
		for i := len(addr) - 1; i >= 0; i-- {
			if addr[i] == ':' {
				externalIP = addr[:i]
				portStr := addr[i+1:]
				fmt.Sscanf(portStr, "%d", &externalPort)
				break
			}
		}
	}

	connectionCount := hr.connManager.GetConnectionCount()

	return &ServerStatus{
		ServerID:        hr.serverID,
		ServerType:      hr.serverType,
		ServerName:      hr.serverName,
		Status:          1, // 1: 在线
		OnlineCount:     int32(connectionCount),
		ExternalIP:      externalIP,
		ExternalPort:    externalPort,
		Version:         hr.version,
		LastHeartbeat:   time.Now(),
		CPUUsage:        0, // 需要系统API获取
		MemoryUsage:     memoryUsage,
		ConnectionCount: connectionCount,
		Uptime:          int64(time.Since(hr.startTime).Seconds()),
	}
}
