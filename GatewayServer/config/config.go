package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zUtil/zConfig"
)

// Config 网关服务器配置
type Config struct {
	Server         ServerConfig         `ini:"Server"`
	Security       SecurityConfig       `ini:"Security"`
	DDoS           zNet.DDoSConfig      `ini:"ddos"`
	NetCompression NetCompressionConfig `ini:"net_compression"`
	GameServer     GameServerConfig     `ini:"GameServer"`
	Log            zLog.Config          `ini:"log"`
	Metrics        MetricsConfig        `ini:"Metrics"`
}

// ServerConfig 服务器基本配置
type ServerConfig struct {
	ServerName        string `ini:"ServerName"`
	ServerID          int    `ini:"ServerID"`
	ListenAddr        string `ini:"ListenAddr"`
	ExternalAddr      string `ini:"ExternalAddr"`
	MaxConnections    int    `ini:"MaxConnections"`
	ConnectionTimeout int    `ini:"ConnectionTimeout"`
	HeartbeatInterval int    `ini:"HeartbeatInterval"`
	JWTSecret         string `ini:"jwt_secret"`
	UseWorkerPool     bool   `ini:"UseWorkerPool"`
	WorkerPoolSize    int    `ini:"WorkerPoolSize"`
	WorkerQueueSize   int    `ini:"WorkerQueueSize"`
	ChanSize          int    `ini:"ChanSize"`
	MaxPacketDataSize int    `ini:"MaxPacketDataSize"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	TokenExpiry      int `ini:"TokenExpiry"`
	MaxLoginAttempts int `ini:"MaxLoginAttempts"`
	BanDuration      int `ini:"BanDuration"`
}

// NetCompressionConfig 网络压缩配置
type NetCompressionConfig struct {
	Enabled    bool
	Threshold  int
	Level      int
	MinQuality int
	MaxQuality int
}

// GameServerConfig 游戏服配置
type GameServerConfig struct {
	GameServerID             int    `ini:"GameServerID"`
	GameServerAddr           string `ini:"GameServerAddr"`
	GameServerConnectTimeout int    `ini:"GameServerConnectTimeout"`
}

// MetricsConfig 监控配置
type MetricsConfig struct {
	Enabled     bool   `ini:"Enabled"`
	MetricsAddr string `ini:"MetricsAddr"`
}

// LoadConfig 加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 使用zConfig加载配置文件
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	// 先获取 server_id，用于构建日志文件名
	serverID := getConfigInt(zcfg, "Server.ServerID", 1)

	// 创建配置实例
	config := &Config{}

	// 解析服务器配置
	listenAddr := getConfigString(zcfg, "Server.ListenAddr", GetEnv("LISTEN_ADDR", "0.0.0.0:10001"))
	externalAddr := getConfigString(zcfg, "Server.ExternalAddr", GetEnv("EXTERNAL_ADDR", ""))
	// 如果没有配置外网地址，默认使用监听地址
	if externalAddr == "" {
		externalAddr = listenAddr
	}

	config.Server = ServerConfig{
		ServerName:        getConfigString(zcfg, "Server.ServerName", GetEnv("SERVER_NAME", "GatewayServer")),
		ServerID:          serverID,
		ListenAddr:        listenAddr,
		ExternalAddr:      externalAddr,
		MaxConnections:    getConfigInt(zcfg, "Server.MaxConnections", 10000),
		ConnectionTimeout: getConfigInt(zcfg, "Server.ConnectionTimeout", 300),
		HeartbeatInterval: getConfigInt(zcfg, "Server.HeartbeatInterval", 30),
		JWTSecret:         getConfigString(zcfg, "Server.jwt_secret", GetEnv("JWT_SECRET", "zMmoServerSecretKey")),
		UseWorkerPool:     getConfigBool(zcfg, "Server.UseWorkerPool", true),
		WorkerPoolSize:    getConfigInt(zcfg, "Server.WorkerPoolSize", 100),
		WorkerQueueSize:   getConfigInt(zcfg, "Server.WorkerQueueSize", 10000),
		ChanSize:          getConfigInt(zcfg, "Server.ChanSize", 1024),
		MaxPacketDataSize: getConfigInt(zcfg, "Server.MaxPacketDataSize", 1024*1024),
	}

	// 解析安全配置
	config.Security = SecurityConfig{
		TokenExpiry:      getConfigInt(zcfg, "Security.TokenExpiry", 86400),
		MaxLoginAttempts: getConfigInt(zcfg, "Security.MaxLoginAttempts", 5),
		BanDuration:      getConfigInt(zcfg, "Security.BanDuration", 3600),
	}

	// 解析DDoS防护配置
	config.DDoS = zNet.DDoSConfig{
		MaxConnPerIP:      getConfigInt(zcfg, "ddos.max_conn_per_ip", 10),
		ConnTimeWindow:    getConfigInt(zcfg, "ddos.conn_time_window", 60),
		MaxPacketsPerIP:   getConfigInt(zcfg, "ddos.max_packets_per_ip", 100),
		PacketTimeWindow:  getConfigInt(zcfg, "ddos.packet_time_window", 1),
		MaxBytesPerIP:     int64(getConfigInt(zcfg, "ddos.max_bytes_per_ip", 10*1024*1024)),
		TrafficTimeWindow: getConfigInt(zcfg, "ddos.traffic_time_window", 3600),
		BanDuration:       getConfigInt(zcfg, "ddos.ban_duration", 24*3600),
	}

	// 解析网络压缩配置
	config.NetCompression = NetCompressionConfig{
		Enabled:    getConfigBool(zcfg, "net_compression.enabled", true),
		Threshold:  getConfigInt(zcfg, "net_compression.threshold", 1024),
		Level:      getConfigInt(zcfg, "net_compression.level", 1),
		MinQuality: getConfigInt(zcfg, "net_compression.min_quality", 0),
		MaxQuality: getConfigInt(zcfg, "net_compression.max_quality", 100),
	}

	// 解析游戏服配置
	config.GameServer = GameServerConfig{
		GameServerID:             getConfigInt(zcfg, "GameServer.GameServerID", 1),
		GameServerAddr:           getConfigString(zcfg, "GameServer.GameServerAddr", GetEnv("GAME_SERVER_ADDR", "game-service.game:9001")),
		GameServerConnectTimeout: getConfigInt(zcfg, "GameServer.GameServerConnectTimeout", 10),
	}

	// 解析日志配置
	config.Log = zLog.Config{
		Level:              getConfigInt(zcfg, "log.level", GetEnvAsInt("LOG_LEVEL", 0)),
		Console:            getConfigBool(zcfg, "log.console", true),
		ConsoleLevel:       getConfigInt(zcfg, "log.console_level", 0),
		Filename:           replacePlaceholder(getConfigString(zcfg, "log.filename", "./logs/gateway_server_{server_id}.log"), "{server_id}", serverID),
		MaxSize:            getConfigInt(zcfg, "log.max_size", 100),
		MaxDays:            getConfigInt(zcfg, "log.max_days", 15),
		MaxBackups:         getConfigInt(zcfg, "log.max_backups", 10),
		Compress:           getConfigBool(zcfg, "log.compress", true),
		ShowCaller:         getConfigBool(zcfg, "log.show_caller", true),
		Stacktrace:         getConfigInt(zcfg, "log.stacktrace", 3),
		Sampling:           getConfigBool(zcfg, "log.sampling", true),
		SamplingInitial:    getConfigInt(zcfg, "log.sampling_initial", 100),
		SamplingThereafter: getConfigInt(zcfg, "log.sampling_thereafter", 10),
		Async:              getConfigBool(zcfg, "log.async", true),
		AsyncBufferSize:    getConfigInt(zcfg, "log.async_buffer_size", 2048),
		AsyncFlushInterval: getConfigInt(zcfg, "log.async_flush_interval", 50),
	}

	// 解析监控配置
	config.Metrics = MetricsConfig{
		Enabled:     getConfigBool(zcfg, "Metrics.Enabled", true),
		MetricsAddr: getConfigString(zcfg, "Metrics.MetricsAddr", GetEnv("METRICS_ADDR", "0.0.0.0:9091")),
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return config, nil
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 验证服务器配置
	if c.Server.ListenAddr == "" {
		return fmt.Errorf("server listen address is required")
	}
	if c.Server.MaxConnections <= 0 {
		c.Server.MaxConnections = 10000
	}

	// 验证DDoS配置
	if c.DDoS.MaxConnPerIP <= 0 {
		c.DDoS.MaxConnPerIP = 10
	}
	if c.DDoS.ConnTimeWindow <= 0 {
		c.DDoS.ConnTimeWindow = 60
	}
	if c.DDoS.MaxPacketsPerIP <= 0 {
		c.DDoS.MaxPacketsPerIP = 100
	}
	if c.DDoS.PacketTimeWindow <= 0 {
		c.DDoS.PacketTimeWindow = 1
	}
	if c.DDoS.MaxBytesPerIP <= 0 {
		c.DDoS.MaxBytesPerIP = 10 * 1024 * 1024
	}
	if c.DDoS.TrafficTimeWindow <= 0 {
		c.DDoS.TrafficTimeWindow = 3600
	}
	if c.DDoS.BanDuration <= 0 {
		c.DDoS.BanDuration = 24 * 3600
	}

	// 验证压缩配置
	if c.NetCompression.Threshold <= 0 {
		c.NetCompression.Threshold = 1024
	}
	if c.NetCompression.Level < 1 || c.NetCompression.Level > 9 {
		c.NetCompression.Level = 1
	}
	if c.NetCompression.MinQuality < 0 || c.NetCompression.MinQuality > 100 {
		c.NetCompression.MinQuality = 0
	}
	if c.NetCompression.MaxQuality < 0 || c.NetCompression.MaxQuality > 100 {
		c.NetCompression.MaxQuality = 100
	}

	return nil
}

// GetDefaultConfigPath 获取默认配置文件路径
func GetDefaultConfigPath() string {
	return "config.ini"
}

// 辅助函数：获取字符串配置
func getConfigString(cfg *zConfig.Config, key string, defaultValue string) string {
	if value, err := cfg.GetString(key); err == nil {
		return value
	}
	return defaultValue
}

// 辅助函数：获取整数配置
func getConfigInt(cfg *zConfig.Config, key string, defaultValue int) int {
	if value, err := cfg.GetInt(key); err == nil {
		return value
	}
	return defaultValue
}

// 辅助函数：获取布尔配置
func getConfigBool(cfg *zConfig.Config, key string, defaultValue bool) bool {
	if value, err := cfg.GetBool(key); err == nil {
		return value
	}
	return defaultValue
}

// replacePlaceholder 替换占位符
func replacePlaceholder(s, placeholder string, value int) string {
	return strings.Replace(s, placeholder, fmt.Sprintf("%d", value), -1)
}

// GetEnv 获取环境变量
func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// GetEnvAsInt 获取整数类型的环境变量
func GetEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvAsBool 获取布尔类型的环境变量
func GetEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
