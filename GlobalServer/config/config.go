package config

import (
	"fmt"

	cfgutil "github.com/pzqf/zCommon/config"
	"github.com/pzqf/zCommon/db"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zCommon/redis"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zUtil/zConfig"
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
	ServerID     int32  `ini:"ServerID"`
	ServerName   string `ini:"ServerName"`
	GroupID      string `ini:"GroupID"`
	WorkerID     int64  `ini:"WorkerID"`
	DatacenterID int64  `ini:"DatacenterID"`
	JWTSecret    string `ini:"JWTSecret"`
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

	serverID := cfgutil.GetConfigInt(zcfg, "Server.ServerID", 1)

	cfg := &Config{
		Server: ServerConfig{
			ServerID:     int32(serverID),
			ServerName:   cfgutil.GetConfigString(zcfg, "Server.ServerName", "GlobalServer"),
			GroupID:      cfgutil.GetConfigString(zcfg, "Server.GroupID", "default"),
			WorkerID:     int64(cfgutil.GetConfigInt(zcfg, "Server.WorkerID", 1)),
			DatacenterID: int64(cfgutil.GetConfigInt(zcfg, "Server.DatacenterID", 1)),
			JWTSecret:    cfgutil.GetConfigString(zcfg, "Server.JWTSecret", "zMmoServerSecretKey"),
		},
		HTTP: HTTPConfig{
			ListenAddress:     cfgutil.GetConfigString(zcfg, "HTTP.ListenAddress", "0.0.0.0:8888"),
			MaxClientCount:    cfgutil.GetConfigInt(zcfg, "HTTP.MaxClientCount", 10000),
			MaxPacketDataSize: int32(cfgutil.GetConfigInt(zcfg, "HTTP.MaxPacketDataSize", 1048576)),
			Enabled:           cfgutil.GetConfigBool(zcfg, "HTTP.Enabled", true),
		},
		Log: zLog.Config{
			Level:              cfgutil.GetConfigInt(zcfg, "Log.Level", 0),
			Console:            cfgutil.GetConfigBool(zcfg, "Log.Console", true),
			ConsoleLevel:       cfgutil.GetConfigInt(zcfg, "Log.ConsoleLevel", 0),
			Filename:           cfgutil.ReplacePlaceholder(cfgutil.GetConfigString(zcfg, "Log.Filename", "./logs/global_server_{ServerID}.log"), "{ServerID}", serverID),
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
		},
		Database: db.DBConfig{
			Host:           cfgutil.GetConfigString(zcfg, "Database.Host", "localhost"),
			Port:           cfgutil.GetConfigInt(zcfg, "Database.Port", 3306),
			User:           cfgutil.GetConfigString(zcfg, "Database.User", "root"),
			Password:       cfgutil.GetConfigString(zcfg, "Database.Password", "123456"),
			DBName:         cfgutil.GetConfigString(zcfg, "Database.DBName", "global"),
			Charset:        cfgutil.GetConfigString(zcfg, "Database.Charset", "utf8mb4"),
			MaxIdle:        cfgutil.GetConfigInt(zcfg, "Database.MaxIdle", 10),
			MaxOpen:        cfgutil.GetConfigInt(zcfg, "Database.MaxOpen", 100),
			Driver:         cfgutil.GetConfigString(zcfg, "Database.Driver", "mysql"),
			MaxPoolSize:    cfgutil.GetConfigInt(zcfg, "Database.MaxPoolSize", 100),
			MinPoolSize:    cfgutil.GetConfigInt(zcfg, "Database.MinPoolSize", 10),
			ConnectTimeout: cfgutil.GetConfigInt(zcfg, "Database.ConnectTimeout", 30),
		},
		Redis: RedisConfig{
			Host:     cfgutil.GetConfigString(zcfg, "Redis.Host", "192.168.91.128"),
			Port:     cfgutil.GetConfigInt(zcfg, "Redis.Port", 6379),
			Password: cfgutil.GetConfigString(zcfg, "Redis.Password", ""),
			DB:       cfgutil.GetConfigInt(zcfg, "Redis.DB", 0),
			PoolSize: cfgutil.GetConfigInt(zcfg, "Redis.PoolSize", 10),
		},
		Pprof: PprofConfig{
			Enabled:       cfgutil.GetConfigBool(zcfg, "Pprof.Enabled", false),
			ListenAddress: cfgutil.GetConfigString(zcfg, "Pprof.ListenAddress", "localhost:6060"),
		},
		Metrics: MetricsConfig{
			Enabled:       cfgutil.GetConfigBool(zcfg, "Metrics.Enabled", true),
			ListenAddress: cfgutil.GetConfigString(zcfg, "Metrics.ListenAddress", "0.0.0.0:8889"),
		},
		Etcd: discovery.EtcdConfig{
			Endpoints:      cfgutil.GetConfigString(zcfg, "Etcd.Endpoints", "etcd-cluster.kube-system.svc.cluster.local:2379"),
			Username:       cfgutil.GetConfigString(zcfg, "Etcd.Username", ""),
			Password:       cfgutil.GetConfigString(zcfg, "Etcd.Password", ""),
			CACertPath:     cfgutil.GetConfigString(zcfg, "Etcd.CACertPath", "resources/etcd/ca.crt"),
			ClientCertPath: cfgutil.GetConfigString(zcfg, "Etcd.ClientCertPath", "resources/etcd/server.crt"),
			ClientKeyPath:  cfgutil.GetConfigString(zcfg, "Etcd.ClientKeyPath", "resources/etcd/server.key"),
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
