package config

import (
	"fmt"

	"github.com/pzqf/zCommon/db"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zCommon/redis"
	"github.com/pzqf/zEngine/zConfig"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
)

type Config struct {
	Server   ServerConfig         `ini:"Server"`
	HTTP     HTTPConfig           `ini:"HTTP"`
	Etcd     discovery.EtcdConfig `ini:"Etcd"`
	Log      zLog.Config          `ini:"Log"`
	Database db.DBConfig          `ini:"Database"`
	Redis    RedisConfig          `ini:"Redis"`
	Pprof    PprofConfig          `ini:"Pprof"`
	Metrics  MetricsConfig        `ini:"Metrics"`
}

type ServerConfig struct {
	ServerID         int32  `ini:"ServerID"`
	ServerName       string `ini:"ServerName"`
	GroupID          string `ini:"GroupID"`
	WorkerID         int64  `ini:"WorkerID"`
	DatacenterID     int64  `ini:"DatacenterID"`
	JWTSecret        string `ini:"JWTSecret"`
	TokenExpiryHours int    `ini:"TokenExpiryHours"`
}

type HTTPConfig struct {
	ListenAddress     string `ini:"ListenAddress"`
	MaxClientCount    int    `ini:"MaxClientCount"`
	MaxPacketDataSize int32  `ini:"MaxPacketDataSize"`
	Enabled           bool   `ini:"Enabled"`
}

type PprofConfig struct {
	Enabled       bool   `ini:"Enabled"`
	ListenAddress string `ini:"ListenAddress"`
}

type MetricsConfig metrics.MetricsConfig

type RedisConfig redis.RedisConfig

func LoadConfig(filePath string) (*Config, error) {
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(filePath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	serverID := zConfig.GetIntWithDefault(zcfg, "Server.ServerID", 1)

	cfg := &Config{
		Server: ServerConfig{
			ServerID:         int32(serverID),
			ServerName:       zConfig.GetStringWithDefault(zcfg, "Server.ServerName", "GlobalServer"),
			GroupID:          zConfig.GetStringWithDefault(zcfg, "Server.GroupID", "default"),
			WorkerID:         int64(zConfig.GetIntWithDefault(zcfg, "Server.WorkerID", 1)),
			DatacenterID:     int64(zConfig.GetIntWithDefault(zcfg, "Server.DatacenterID", 1)),
			JWTSecret:        zConfig.GetStringWithDefault(zcfg, "Server.JWTSecret", zConfig.GetEnv("JWT_SECRET", "")),
			TokenExpiryHours: zConfig.GetIntWithDefault(zcfg, "Server.TokenExpiryHours", 1),
		},
		HTTP: HTTPConfig{
			ListenAddress:     zConfig.GetStringWithDefault(zcfg, "HTTP.ListenAddress", "0.0.0.0:8888"),
			MaxClientCount:    zConfig.GetIntWithDefault(zcfg, "HTTP.MaxClientCount", 10000),
			MaxPacketDataSize: int32(zConfig.GetIntWithDefault(zcfg, "HTTP.MaxPacketDataSize", 1048576)),
			Enabled:           zConfig.GetBoolWithDefault(zcfg, "HTTP.Enabled", true),
		},
		Log: zLog.Config{
			Level:              zConfig.GetIntWithDefault(zcfg, "Log.Level", 0),
			Console:            zConfig.GetBoolWithDefault(zcfg, "Log.Console", true),
			ConsoleLevel:       zConfig.GetIntWithDefault(zcfg, "Log.ConsoleLevel", 0),
			Filename:           zConfig.ReplacePlaceholder(zConfig.GetStringWithDefault(zcfg, "Log.Filename", "./logs/global_server_{ServerID}.log"), "{ServerID}", serverID),
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
		},
		Database: db.DBConfig{
			Host:           zConfig.GetStringWithDefault(zcfg, "Database.Host", "localhost"),
			Port:           zConfig.GetIntWithDefault(zcfg, "Database.Port", 3306),
			User:           zConfig.GetStringWithDefault(zcfg, "Database.User", "root"),
			Password:       zConfig.GetStringWithDefault(zcfg, "Database.Password", "123456"),
			DBName:         zConfig.GetStringWithDefault(zcfg, "Database.DBName", "global"),
			Charset:        zConfig.GetStringWithDefault(zcfg, "Database.Charset", "utf8mb4"),
			MaxIdle:        zConfig.GetIntWithDefault(zcfg, "Database.MaxIdle", 10),
			MaxOpen:        zConfig.GetIntWithDefault(zcfg, "Database.MaxOpen", 100),
			Driver:         zConfig.GetStringWithDefault(zcfg, "Database.Driver", "mysql"),
			MaxPoolSize:    zConfig.GetIntWithDefault(zcfg, "Database.MaxPoolSize", 100),
			MinPoolSize:    zConfig.GetIntWithDefault(zcfg, "Database.MinPoolSize", 10),
			ConnectTimeout: zConfig.GetIntWithDefault(zcfg, "Database.ConnectTimeout", 30),
		},
		Redis: RedisConfig{
			Host:     zConfig.GetStringWithDefault(zcfg, "Redis.Host", "192.168.91.128"),
			Port:     zConfig.GetIntWithDefault(zcfg, "Redis.Port", 6379),
			Password: zConfig.GetStringWithDefault(zcfg, "Redis.Password", ""),
			DB:       zConfig.GetIntWithDefault(zcfg, "Redis.DB", 0),
			PoolSize: zConfig.GetIntWithDefault(zcfg, "Redis.PoolSize", 10),
		},
		Pprof: PprofConfig{
			Enabled:       zConfig.GetBoolWithDefault(zcfg, "Pprof.Enabled", false),
			ListenAddress: zConfig.GetStringWithDefault(zcfg, "Pprof.ListenAddress", "localhost:6060"),
		},
		Metrics: MetricsConfig{
			Enabled:       zConfig.GetBoolWithDefault(zcfg, "Metrics.Enabled", true),
			ListenAddress: zConfig.GetStringWithDefault(zcfg, "Metrics.ListenAddress", "0.0.0.0:8889"),
		},
		Etcd: discovery.EtcdConfig{
			Endpoints:      zConfig.GetStringWithDefault(zcfg, "Etcd.Endpoints", "etcd-cluster.kube-system.svc.cluster.local:2379"),
			Username:       zConfig.GetStringWithDefault(zcfg, "Etcd.Username", ""),
			Password:       zConfig.GetStringWithDefault(zcfg, "Etcd.Password", ""),
			CACertPath:     zConfig.GetStringWithDefault(zcfg, "Etcd.CACertPath", "resources/etcd/ca.crt"),
			ClientCertPath: zConfig.GetStringWithDefault(zcfg, "Etcd.ClientCertPath", "resources/etcd/server.crt"),
			ClientKeyPath:  zConfig.GetStringWithDefault(zcfg, "Etcd.ClientKeyPath", "resources/etcd/server.key"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.ServerID <= 0 {
		return fmt.Errorf("ServerID must be greater than 0")
	}
	if c.Server.JWTSecret == "" {
		return fmt.Errorf("Server.JWTSecret is required, please set in config.ini or JWT_SECRET env")
	}
	if c.Server.TokenExpiryHours <= 0 {
		return fmt.Errorf("Server.TokenExpiryHours must be greater than 0")
	}
	if c.HTTP.ListenAddress == "" {
		return fmt.Errorf("HTTP.ListenAddress is required")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("Database.Host is required")
	}
	if c.Redis.Host == "" {
		return fmt.Errorf("Redis.Host is required")
	}
	return nil
}

func (c *RedisConfig) ToRedisConfig() redis.RedisConfig {
	return redis.RedisConfig(*c)
}

func (c *HTTPConfig) ToZNetHTTPConfig() *zNet.HttpConfig {
	return &zNet.HttpConfig{
		ListenAddress:     c.ListenAddress,
		MaxClientCount:    c.MaxClientCount,
		MaxPacketDataSize: c.MaxPacketDataSize,
	}
}

func (c *Config) GetLogConfig() *zLog.Config {
	return &c.Log
}
