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
	Server   ServerConfig         `ini:"server"`
	HTTP     HTTPConfig           `ini:"http"`
	Etcd     discovery.EtcdConfig `ini:"etcd"`
	Log      zLog.Config          `ini:"log"`
	Database db.DBConfig          `ini:"database.global"`
	Redis    RedisConfig          `ini:"redis"`
	Pprof    PprofConfig          `ini:"pprof"`
	Metrics  MetricsConfig        `ini:"metrics"`
}

type ServerConfig struct {
	ServerID     int32  `ini:"server_id"`
	ServerName   string `ini:"server_name"`
	WorkerID     int64  `ini:"worker_id"`
	DatacenterID int64  `ini:"datacenter_id"`
	JWTSecret    string `ini:"jwt_secret"`
}

type HTTPConfig struct {
	ListenAddress     string `ini:"listen_address"`
	MaxClientCount    int    `ini:"max_client_count"`
	MaxPacketDataSize int32  `ini:"max_packet_data_size"`
	Enabled           bool   `ini:"enabled"`
}

type PprofConfig struct {
	Enabled       bool   `ini:"enabled"`
	ListenAddress string `ini:"listen_address"`
}

type MetricsConfig metrics.MetricsConfig

type RedisConfig redis.RedisConfig

func LoadConfig(filePath string) (*Config, error) {
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(filePath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	serverID := cfgutil.GetConfigInt(zcfg, "server.server_id", 1)

	cfg := &Config{
		Server: ServerConfig{
			ServerID:     int32(serverID),
			ServerName:   cfgutil.GetConfigString(zcfg, "server.server_name", "GlobalServer"),
			WorkerID:     int64(cfgutil.GetConfigInt(zcfg, "server.worker_id", 1)),
			DatacenterID: int64(cfgutil.GetConfigInt(zcfg, "server.datacenter_id", 1)),
			JWTSecret:    cfgutil.GetConfigString(zcfg, "server.jwt_secret", "zMmoServerSecretKey"),
		},
		HTTP: HTTPConfig{
			ListenAddress:     cfgutil.GetConfigString(zcfg, "http.listen_address", "0.0.0.0:8888"),
			MaxClientCount:    cfgutil.GetConfigInt(zcfg, "http.max_client_count", 10000),
			MaxPacketDataSize: int32(cfgutil.GetConfigInt(zcfg, "http.max_packet_data_size", 1048576)),
			Enabled:           cfgutil.GetConfigBool(zcfg, "http.enabled", true),
		},
		Log: zLog.Config{
			Level:              cfgutil.GetConfigInt(zcfg, "log.level", 0),
			Console:            cfgutil.GetConfigBool(zcfg, "log.console", true),
			ConsoleLevel:       cfgutil.GetConfigInt(zcfg, "log.console_level", 0),
			Filename:           cfgutil.ReplacePlaceholder(cfgutil.GetConfigString(zcfg, "log.filename", "./logs/global_server_{server_id}.log"), "{server_id}", serverID),
			MaxSize:            cfgutil.GetConfigInt(zcfg, "log.max_size", 100),
			MaxDays:            cfgutil.GetConfigInt(zcfg, "log.max_days", 15),
			MaxBackups:         cfgutil.GetConfigInt(zcfg, "log.max_backups", 10),
			Compress:           cfgutil.GetConfigBool(zcfg, "log.compress", true),
			ShowCaller:         cfgutil.GetConfigBool(zcfg, "log.show_caller", true),
			Stacktrace:         cfgutil.GetConfigInt(zcfg, "log.stacktrace", 3),
			Sampling:           cfgutil.GetConfigBool(zcfg, "log.sampling", true),
			SamplingInitial:    cfgutil.GetConfigInt(zcfg, "log.sampling_initial", 100),
			SamplingThereafter: cfgutil.GetConfigInt(zcfg, "log.sampling_thereafter", 10),
			Async:              cfgutil.GetConfigBool(zcfg, "log.async", true),
			AsyncBufferSize:    cfgutil.GetConfigInt(zcfg, "log.async_buffer_size", 2048),
			AsyncFlushInterval: cfgutil.GetConfigInt(zcfg, "log.async_flush_interval", 50),
		},
		Database: db.DBConfig{
			Host:           cfgutil.GetConfigString(zcfg, "database.global.host", "localhost"),
			Port:           cfgutil.GetConfigInt(zcfg, "database.global.port", 3306),
			User:           cfgutil.GetConfigString(zcfg, "database.global.user", "root"),
			Password:       cfgutil.GetConfigString(zcfg, "database.global.password", "123456"),
			DBName:         cfgutil.GetConfigString(zcfg, "database.global.dbname", "global"),
			Charset:        cfgutil.GetConfigString(zcfg, "database.global.charset", "utf8mb4"),
			MaxIdle:        cfgutil.GetConfigInt(zcfg, "database.global.max_idle", 10),
			MaxOpen:        cfgutil.GetConfigInt(zcfg, "database.global.max_open", 100),
			Driver:         cfgutil.GetConfigString(zcfg, "database.global.driver", "mysql"),
			MaxPoolSize:    cfgutil.GetConfigInt(zcfg, "database.global.max_pool_size", 100),
			MinPoolSize:    cfgutil.GetConfigInt(zcfg, "database.global.min_pool_size", 10),
			ConnectTimeout: cfgutil.GetConfigInt(zcfg, "database.global.connect_timeout", 30),
		},
		Redis: RedisConfig{
			Host:     cfgutil.GetConfigString(zcfg, "redis.host", "192.168.91.128"),
			Port:     cfgutil.GetConfigInt(zcfg, "redis.port", 6379),
			Password: cfgutil.GetConfigString(zcfg, "redis.password", ""),
			DB:       cfgutil.GetConfigInt(zcfg, "redis.db", 0),
			PoolSize: cfgutil.GetConfigInt(zcfg, "redis.pool_size", 10),
		},
		Pprof: PprofConfig{
			Enabled:       cfgutil.GetConfigBool(zcfg, "pprof.enabled", false),
			ListenAddress: cfgutil.GetConfigString(zcfg, "pprof.listen_address", "localhost:6060"),
		},
		Metrics: MetricsConfig{
			Enabled:       cfgutil.GetConfigBool(zcfg, "metrics.enabled", true),
			ListenAddress: cfgutil.GetConfigString(zcfg, "metrics.listen_address", "0.0.0.0:8889"),
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
		return fmt.Errorf("server_id must be greater than 0")
	}
	if c.HTTP.ListenAddress == "" {
		return fmt.Errorf("http.listen_address is required")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("database.global.host is required")
	}
	if c.Redis.Host == "" {
		return fmt.Errorf("redis.host is required")
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
