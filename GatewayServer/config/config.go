package config

import (
	"fmt"

	cfgutil "github.com/pzqf/zCommon/config"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zUtil/zConfig"
)

type Config struct {
	Server      ServerConfig           `ini:"Server"`
	Security    SecurityConfig         `ini:"Security"`
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

	serverID := cfgutil.GetConfigInt(zcfg, "Server.ServerID", 1)

	c := &Config{}

	listenAddr := cfgutil.GetConfigString(zcfg, "Server.ListenAddr", cfgutil.GetEnv("LISTEN_ADDR", "0.0.0.0:10001"))
	externalAddr := cfgutil.GetConfigString(zcfg, "Server.ExternalAddr", cfgutil.GetEnv("EXTERNAL_ADDR", ""))
	if externalAddr == "" {
		externalAddr = listenAddr
	}

	c.Server = ServerConfig{
		ServerName:          cfgutil.GetConfigString(zcfg, "Server.ServerName", cfgutil.GetEnv("SERVER_NAME", "GatewayServer")),
		ServerID:            serverID,
		GroupID:             cfgutil.GetConfigInt(zcfg, "Server.GroupID", 1),
		ListenAddr:          listenAddr,
		ExternalAddr:        externalAddr,
		MaxConnections:      cfgutil.GetConfigInt(zcfg, "Server.MaxConnections", 10000),
		ConnectionTimeout:   cfgutil.GetConfigInt(zcfg, "Server.ConnectionTimeout", 300),
		HeartbeatInterval:   cfgutil.GetConfigInt(zcfg, "Server.HeartbeatInterval", 30),
		JWTSecret:           cfgutil.GetConfigString(zcfg, "Server.JWTSecret", cfgutil.GetEnv("JWT_SECRET", "zMmoServerSecretKey")),
		UseWorkerPool:       cfgutil.GetConfigBool(zcfg, "Server.UseWorkerPool", true),
		WorkerPoolSize:      cfgutil.GetConfigInt(zcfg, "Server.WorkerPoolSize", 100),
		WorkerQueueSize:     cfgutil.GetConfigInt(zcfg, "Server.WorkerQueueSize", 10000),
		ChanSize:            cfgutil.GetConfigInt(zcfg, "Server.ChanSize", 1024),
		MaxPacketDataSize:   cfgutil.GetConfigInt(zcfg, "Server.MaxPacketDataSize", 1024*1024),
		DisableEncryption:   cfgutil.GetConfigBool(zcfg, "Server.DisableEncryption", cfgutil.GetEnvAsBool("DISABLE_ENCRYPTION", false)),
		EnableKeyRotation:   cfgutil.GetConfigBool(zcfg, "Server.EnableKeyRotation", false),
		KeyRotationInterval: cfgutil.GetConfigInt(zcfg, "Server.KeyRotationInterval", 1800),
		MaxHistoryKeys:      cfgutil.GetConfigInt(zcfg, "Server.MaxHistoryKeys", 3),
		EnableSequenceCheck: cfgutil.GetConfigBool(zcfg, "Server.EnableSequenceCheck", false),
		SequenceWindowSize:  uint64(cfgutil.GetConfigInt(zcfg, "Server.SequenceWindowSize", 1000)),
		TimestampTolerance:  int64(cfgutil.GetConfigInt(zcfg, "Server.TimestampTolerance", 30)),
	}

	c.Security = SecurityConfig{
		TokenExpiry:      cfgutil.GetConfigInt(zcfg, "Security.TokenExpiry", 86400),
		MaxLoginAttempts: cfgutil.GetConfigInt(zcfg, "Security.MaxLoginAttempts", 5),
		BanDuration:      cfgutil.GetConfigInt(zcfg, "Security.BanDuration", 3600),
	}

	c.DDoS = zNet.DDoSConfig{
		MaxConnPerIP:      cfgutil.GetConfigInt(zcfg, "DDoS.MaxConnPerIP", 10),
		ConnTimeWindow:    cfgutil.GetConfigInt(zcfg, "DDoS.ConnTimeWindow", 60),
		MaxPacketsPerIP:   cfgutil.GetConfigInt(zcfg, "DDoS.MaxPacketsPerIP", 100),
		PacketTimeWindow:  cfgutil.GetConfigInt(zcfg, "DDoS.PacketTimeWindow", 1),
		MaxBytesPerIP:     int64(cfgutil.GetConfigInt(zcfg, "DDoS.MaxBytesPerIP", 10*1024*1024)),
		TrafficTimeWindow: cfgutil.GetConfigInt(zcfg, "DDoS.TrafficTimeWindow", 3600),
		BanDuration:       cfgutil.GetConfigInt(zcfg, "DDoS.BanDuration", 24*3600),
	}

	c.Compression = zNet.CompressionConfig{
		Enabled:              cfgutil.GetConfigBool(zcfg, "Compression.Enabled", false),
		CompressionThreshold: cfgutil.GetConfigInt(zcfg, "Compression.CompressionThreshold", 1024),
		MaxCompressSize:      cfgutil.GetConfigInt(zcfg, "Compression.MaxCompressSize", 1024*1024),
	}

	c.GameServer = GameServerConfig{
		GameServerID:             cfgutil.GetConfigInt(zcfg, "GameServer.GameServerID", 1),
		GameServerAddr:           cfgutil.GetConfigString(zcfg, "GameServer.GameServerAddr", cfgutil.GetEnv("GAME_SERVER_ADDR", "game-service.game:9001")),
		GameServerConnectTimeout: cfgutil.GetConfigInt(zcfg, "GameServer.GameServerConnectTimeout", 10),
	}

	c.Log = zLog.Config{
		Level:              cfgutil.GetConfigInt(zcfg, "Log.Level", cfgutil.GetEnvAsInt("LOG_LEVEL", 0)),
		Console:            cfgutil.GetConfigBool(zcfg, "Log.Console", true),
		ConsoleLevel:       cfgutil.GetConfigInt(zcfg, "Log.ConsoleLevel", 0),
		Filename:           cfgutil.ReplacePlaceholder(cfgutil.GetConfigString(zcfg, "Log.Filename", "./logs/gateway_server_{ServerID}.log"), "{ServerID}", serverID),
		MaxSize:            cfgutil.GetConfigInt(zcfg, "Log.MaxSize", 100),
		MaxDays:            cfgutil.GetConfigInt(zcfg, "Log.MaxDays", 15),
		MaxBackups:         cfgutil.GetConfigInt(zcfg, "Log.MaxBackups", 10),
		Compress:           cfgutil.GetConfigBool(zcfg, "Log.Compress", true),
		ShowCaller:         cfgutil.GetConfigBool(zcfg, "Log.ShowCaller", true),
		Stacktrace:         cfgutil.GetConfigInt(zcfg, "Log.Stacktrace", 3),
		Sampling:           cfgutil.GetConfigBool(zcfg, "Log.Sampling", true),
		SamplingInitial:    cfgutil.GetConfigInt(zcfg, "Log.SamplingInitial", 100),
		SamplingThereafter: cfgutil.GetConfigInt(zcfg, "Log.SamplingThereafter", 10),
		Async:              cfgutil.GetConfigBool(zcfg, "Log.Async", true),
		AsyncBufferSize:    cfgutil.GetConfigInt(zcfg, "Log.AsyncBufferSize", 2048),
		AsyncFlushInterval: cfgutil.GetConfigInt(zcfg, "Log.AsyncFlushInterval", 50),
	}

	c.Metrics = MetricsConfig{
		Enabled:       cfgutil.GetConfigBool(zcfg, "Metrics.Enabled", true),
		ListenAddress: cfgutil.GetConfigString(zcfg, "Metrics.ListenAddress", cfgutil.GetEnv("METRICS_ADDR", "0.0.0.0:9091")),
	}

	c.Pprof = PprofConfig{
		Enabled:       cfgutil.GetConfigBool(zcfg, "Pprof.Enabled", false),
		ListenAddress: cfgutil.GetConfigString(zcfg, "Pprof.ListenAddress", "localhost:6062"),
	}

	c.Etcd = discovery.EtcdConfig{
		Endpoints:      cfgutil.GetConfigString(zcfg, "Etcd.Endpoints", cfgutil.GetEnv("ETCD_ENDPOINTS", "etcd-cluster.kube-system.svc.cluster.local:2379")),
		Username:       cfgutil.GetConfigString(zcfg, "Etcd.Username", ""),
		Password:       cfgutil.GetConfigString(zcfg, "Etcd.Password", ""),
		CACertPath:     cfgutil.GetConfigString(zcfg, "Etcd.CACertPath", "../resources/etcd/ca.crt"),
		ClientCertPath: cfgutil.GetConfigString(zcfg, "Etcd.ClientCertPath", "../resources/etcd/server.crt"),
		ClientKeyPath:  cfgutil.GetConfigString(zcfg, "Etcd.ClientKeyPath", "../resources/etcd/server.key"),
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
