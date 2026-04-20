package config

import (
	"fmt"

	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zEngine/zConfig"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
)

type Config struct {
	Server      ServerConfig           `ini:"Server"`
	Security    SecurityConfig         `ini:"Security"`
	AntiCheat   AntiCheatConfig        `ini:"AntiCheat"`
	DDoS        zNet.DDoSConfig        `ini:"DDoS"`
	Compression zNet.CompressionConfig `ini:"Compression"`
	GameServer  GameServerConfig       `ini:"GameServer"`
	Etcd        discovery.EtcdConfig   `ini:"Etcd"`
	Log         zLog.Config            `ini:"Log"`
	Metrics     MetricsConfig          `ini:"Metrics"`
	Pprof       PprofConfig            `ini:"Pprof"`
}

type ServerConfig struct {
	ServerName          string `ini:"ServerName"`
	ServerID            int    `ini:"ServerID"`
	GroupID             int    `ini:"GroupID"`
	ListenAddr          string `ini:"ListenAddr"`
	ExternalAddr        string `ini:"ExternalAddr"`
	MaxConnections      int    `ini:"MaxConnections"`
	ConnectionTimeout   int    `ini:"ConnectionTimeout"`
	HeartbeatInterval   int    `ini:"HeartbeatInterval"`
	JWTSecret           string `ini:"JWTSecret"`
	UseWorkerPool       bool   `ini:"UseWorkerPool"`
	WorkerPoolSize      int    `ini:"WorkerPoolSize"`
	WorkerQueueSize     int    `ini:"WorkerQueueSize"`
	ChanSize            int    `ini:"ChanSize"`
	MaxPacketDataSize   int    `ini:"MaxPacketDataSize"`
	DisableEncryption   bool   `ini:"DisableEncryption"`
	EnableKeyRotation   bool   `ini:"EnableKeyRotation"`
	KeyRotationInterval int    `ini:"KeyRotationInterval"`
	MaxHistoryKeys      int    `ini:"MaxHistoryKeys"`
	EnableSequenceCheck bool   `ini:"EnableSequenceCheck"`
	SequenceWindowSize  uint64 `ini:"SequenceWindowSize"`
	TimestampTolerance  int64  `ini:"TimestampTolerance"`
}

type SecurityConfig struct {
	TokenExpiry      int `ini:"TokenExpiry"`
	MaxLoginAttempts int `ini:"MaxLoginAttempts"`
	BanDuration      int `ini:"BanDuration"`
	MaxIPConnections int `ini:"MaxIPConnections"`
}

type AntiCheatConfig struct {
	MaxActionsPerMinute    int     `ini:"MaxActionsPerMinute"`
	MaxErrorRatio          float64 `ini:"MaxErrorRatio"`
	MaxAbnormalActions     int     `ini:"MaxAbnormalActions"`
	MaxHighSeverityReports int     `ini:"MaxHighSeverityReports"`
	InactiveTimeoutMinutes int     `ini:"InactiveTimeoutMinutes"`
	CleanupIntervalMinutes int     `ini:"CleanupIntervalMinutes"`
}

type GameServerConfig struct {
	GameServerID             int    `ini:"GameServerID"`
	GameServerAddr           string `ini:"GameServerAddr"`
	GameServerConnectTimeout int    `ini:"GameServerConnectTimeout"`
}

type MetricsConfig metrics.MetricsConfig

type PprofConfig struct {
	Enabled       bool   `ini:"Enabled"`
	ListenAddress string `ini:"ListenAddress"`
}

func LoadConfig(configPath string) (*Config, error) {
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	serverID := zConfig.GetIntWithDefault(zcfg, "Server.ServerID", 1)

	c := &Config{}

	listenAddr := zConfig.GetStringWithDefault(zcfg, "Server.ListenAddr", zConfig.GetEnv("LISTEN_ADDR", "0.0.0.0:10001"))
	externalAddr := zConfig.GetStringWithDefault(zcfg, "Server.ExternalAddr", zConfig.GetEnv("EXTERNAL_ADDR", ""))
	if externalAddr == "" {
		externalAddr = listenAddr
	}

	c.Server = ServerConfig{
		ServerName:          zConfig.GetStringWithDefault(zcfg, "Server.ServerName", zConfig.GetEnv("SERVER_NAME", "GatewayServer")),
		ServerID:            serverID,
		GroupID:             zConfig.GetIntWithDefault(zcfg, "Server.GroupID", 1),
		ListenAddr:          listenAddr,
		ExternalAddr:        externalAddr,
		MaxConnections:      zConfig.GetIntWithDefault(zcfg, "Server.MaxConnections", 10000),
		ConnectionTimeout:   zConfig.GetIntWithDefault(zcfg, "Server.ConnectionTimeout", 300),
		HeartbeatInterval:   zConfig.GetIntWithDefault(zcfg, "Server.HeartbeatInterval", 30),
		JWTSecret:           zConfig.GetStringWithDefault(zcfg, "Server.JWTSecret", zConfig.GetEnv("JWT_SECRET", "zMmoServerSecretKey")),
		UseWorkerPool:       zConfig.GetBoolWithDefault(zcfg, "Server.UseWorkerPool", true),
		WorkerPoolSize:      zConfig.GetIntWithDefault(zcfg, "Server.WorkerPoolSize", 100),
		WorkerQueueSize:     zConfig.GetIntWithDefault(zcfg, "Server.WorkerQueueSize", 10000),
		ChanSize:            zConfig.GetIntWithDefault(zcfg, "Server.ChanSize", 1024),
		MaxPacketDataSize:   zConfig.GetIntWithDefault(zcfg, "Server.MaxPacketDataSize", 1024*1024),
		DisableEncryption:   zConfig.GetBoolWithDefault(zcfg, "Server.DisableEncryption", zConfig.GetEnvAsBool("DISABLE_ENCRYPTION", false)),
		EnableKeyRotation:   zConfig.GetBoolWithDefault(zcfg, "Server.EnableKeyRotation", false),
		KeyRotationInterval: zConfig.GetIntWithDefault(zcfg, "Server.KeyRotationInterval", 1800),
		MaxHistoryKeys:      zConfig.GetIntWithDefault(zcfg, "Server.MaxHistoryKeys", 3),
		EnableSequenceCheck: zConfig.GetBoolWithDefault(zcfg, "Server.EnableSequenceCheck", false),
		SequenceWindowSize:  uint64(zConfig.GetIntWithDefault(zcfg, "Server.SequenceWindowSize", 1000)),
		TimestampTolerance:  int64(zConfig.GetIntWithDefault(zcfg, "Server.TimestampTolerance", 30)),
	}

	c.Security = SecurityConfig{
		TokenExpiry:      zConfig.GetIntWithDefault(zcfg, "Security.TokenExpiry", 86400),
		MaxLoginAttempts: zConfig.GetIntWithDefault(zcfg, "Security.MaxLoginAttempts", 5),
		BanDuration:      zConfig.GetIntWithDefault(zcfg, "Security.BanDuration", 3600),
		MaxIPConnections: zConfig.GetIntWithDefault(zcfg, "Security.MaxIPConnections", 5000),
	}

	c.AntiCheat = AntiCheatConfig{
		MaxActionsPerMinute:    zConfig.GetIntWithDefault(zcfg, "AntiCheat.MaxActionsPerMinute", 1000),
		MaxErrorRatio:          zConfig.GetFloatWithDefault(zcfg, "AntiCheat.MaxErrorRatio", 0.5),
		MaxAbnormalActions:     zConfig.GetIntWithDefault(zcfg, "AntiCheat.MaxAbnormalActions", 5),
		MaxHighSeverityReports: zConfig.GetIntWithDefault(zcfg, "AntiCheat.MaxHighSeverityReports", 3),
		InactiveTimeoutMinutes: zConfig.GetIntWithDefault(zcfg, "AntiCheat.InactiveTimeoutMinutes", 30),
		CleanupIntervalMinutes: zConfig.GetIntWithDefault(zcfg, "AntiCheat.CleanupIntervalMinutes", 10),
	}

	c.DDoS = zNet.DDoSConfig{
		MaxConnPerIP:      zConfig.GetIntWithDefault(zcfg, "DDoS.MaxConnPerIP", 10),
		ConnTimeWindow:    zConfig.GetIntWithDefault(zcfg, "DDoS.ConnTimeWindow", 60),
		MaxPacketsPerIP:   zConfig.GetIntWithDefault(zcfg, "DDoS.MaxPacketsPerIP", 100),
		PacketTimeWindow:  zConfig.GetIntWithDefault(zcfg, "DDoS.PacketTimeWindow", 1),
		MaxBytesPerIP:     int64(zConfig.GetIntWithDefault(zcfg, "DDoS.MaxBytesPerIP", 10*1024*1024)),
		TrafficTimeWindow: zConfig.GetIntWithDefault(zcfg, "DDoS.TrafficTimeWindow", 3600),
		BanDuration:       zConfig.GetIntWithDefault(zcfg, "DDoS.BanDuration", 24*3600),
	}

	c.Compression = zNet.CompressionConfig{
		Enabled:              zConfig.GetBoolWithDefault(zcfg, "Compression.Enabled", false),
		CompressionThreshold: zConfig.GetIntWithDefault(zcfg, "Compression.CompressionThreshold", 1024),
		MaxCompressSize:      zConfig.GetIntWithDefault(zcfg, "Compression.MaxCompressSize", 1024*1024),
	}

	c.GameServer = GameServerConfig{
		GameServerID:             zConfig.GetIntWithDefault(zcfg, "GameServer.GameServerID", 1),
		GameServerAddr:           zConfig.GetStringWithDefault(zcfg, "GameServer.GameServerAddr", zConfig.GetEnv("GAME_SERVER_ADDR", "game-service.game:9001")),
		GameServerConnectTimeout: zConfig.GetIntWithDefault(zcfg, "GameServer.GameServerConnectTimeout", 10),
	}

	c.Log = zLog.Config{
		Level:              zConfig.GetIntWithDefault(zcfg, "Log.Level", zConfig.GetEnvAsInt("LOG_LEVEL", 0)),
		Console:            zConfig.GetBoolWithDefault(zcfg, "Log.Console", true),
		ConsoleLevel:       zConfig.GetIntWithDefault(zcfg, "Log.ConsoleLevel", 0),
		Filename:           zConfig.ReplacePlaceholder(zConfig.GetStringWithDefault(zcfg, "Log.Filename", "./logs/gateway_server_{ServerID}.log"), "{ServerID}", serverID),
		MaxSize:            zConfig.GetIntWithDefault(zcfg, "Log.MaxSize", 100),
		MaxDays:            zConfig.GetIntWithDefault(zcfg, "Log.MaxDays", 15),
		MaxBackups:         zConfig.GetIntWithDefault(zcfg, "Log.MaxBackups", 10),
		Compress:           zConfig.GetBoolWithDefault(zcfg, "Log.Compress", true),
		ShowCaller:         zConfig.GetBoolWithDefault(zcfg, "Log.ShowCaller", true),
		Stacktrace:         zConfig.GetIntWithDefault(zcfg, "Log.Stacktrace", 3),
		Sampling:           zConfig.GetBoolWithDefault(zcfg, "Log.Sampling", true),
		SamplingInitial:    zConfig.GetIntWithDefault(zcfg, "Log.SamplingInitial", 100),
		SamplingThereafter: zConfig.GetIntWithDefault(zcfg, "Log.SamplingThereafter", 10),
		Async:              zConfig.GetBoolWithDefault(zcfg, "Log.Async", true),
		AsyncBufferSize:    zConfig.GetIntWithDefault(zcfg, "Log.AsyncBufferSize", 2048),
		AsyncFlushInterval: zConfig.GetIntWithDefault(zcfg, "Log.AsyncFlushInterval", 50),
	}

	c.Metrics = MetricsConfig{
		Enabled:       zConfig.GetBoolWithDefault(zcfg, "Metrics.Enabled", true),
		ListenAddress: zConfig.GetStringWithDefault(zcfg, "Metrics.ListenAddress", zConfig.GetEnv("METRICS_ADDR", "0.0.0.0:9091")),
	}

	c.Pprof = PprofConfig{
		Enabled:       zConfig.GetBoolWithDefault(zcfg, "Pprof.Enabled", false),
		ListenAddress: zConfig.GetStringWithDefault(zcfg, "Pprof.ListenAddress", "localhost:6062"),
	}

	c.Etcd = discovery.EtcdConfig{
		Endpoints:      zConfig.GetStringWithDefault(zcfg, "Etcd.Endpoints", zConfig.GetEnv("ETCD_ENDPOINTS", "etcd-cluster.kube-system.svc.cluster.local:2379")),
		Username:       zConfig.GetStringWithDefault(zcfg, "Etcd.Username", ""),
		Password:       zConfig.GetStringWithDefault(zcfg, "Etcd.Password", ""),
		CACertPath:     zConfig.GetStringWithDefault(zcfg, "Etcd.CACertPath", "../resources/etcd/ca.crt"),
		ClientCertPath: zConfig.GetStringWithDefault(zcfg, "Etcd.ClientCertPath", "../resources/etcd/server.crt"),
		ClientKeyPath:  zConfig.GetStringWithDefault(zcfg, "Etcd.ClientKeyPath", "../resources/etcd/server.key"),
	}

	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return c, nil
}

func (c *Config) Validate() error {
	if c.Server.ListenAddr == "" {
		return fmt.Errorf("server listen address is required")
	}
	if c.Server.MaxConnections <= 0 {
		c.Server.MaxConnections = 10000
	}

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

	if c.Compression.CompressionThreshold <= 0 {
		c.Compression.CompressionThreshold = 1024
	}
	if c.Compression.MaxCompressSize <= 0 {
		c.Compression.MaxCompressSize = 1024 * 1024
	}

	return nil
}

func GetDefaultConfigPath() string {
	return "config.ini"
}

func (c *Config) GetLogConfig() *zLog.Config {
	return &c.Log
}
