package config

import (
	"fmt"
	"strings"

	"github.com/pzqf/zCommon/db"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zCommon/redis"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zUtil/zConfig"
)

// Config GlobalServer 配置
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

// ServerConfig 服务器基本配置
type ServerConfig struct {
	ServerID     int32  `ini:"server_id"`
	ServerName   string `ini:"server_name"`
	WorkerID     int64  `ini:"worker_id"`
	DatacenterID int64  `ini:"datacenter_id"`
	JWTSecret    string `ini:"jwt_secret"`
}

// HTTPConfig HTTP服务配置
type HTTPConfig struct {
	ListenAddress     string `ini:"listen_address"`
	MaxClientCount    int    `ini:"max_client_count"`
	MaxPacketDataSize int32  `ini:"max_packet_data_size"`
	Enabled           bool   `ini:"enabled"`
}

// PprofConfig pprof性能分析配置
type PprofConfig struct {
	Enabled       bool   `ini:"enabled"`
	ListenAddress string `ini:"listen_address"`
}

// MetricsConfig 监控指标配置
type MetricsConfig metrics.MetricsConfig

// RedisConfig Redis配置 - 直接使用zCommon/redis.RedisConfig
type RedisConfig redis.RedisConfig

// LoadConfig 加载配置文件
func LoadConfig(filePath string) (*Config, error) {
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(filePath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	// 先获取server_id，用于构建日志文件名
	serverID := getConfigInt(zcfg, "server.server_id", 1)

	cfg := &Config{
		Server: ServerConfig{
			ServerID:     int32(serverID),
			ServerName:   getConfigString(zcfg, "server.server_name", "GlobalServer"),
			WorkerID:     int64(getConfigInt(zcfg, "server.worker_id", 1)),
			DatacenterID: int64(getConfigInt(zcfg, "server.datacenter_id", 1)),
			JWTSecret:    getConfigString(zcfg, "server.jwt_secret", "zMmoServerSecretKey"),
		},
		HTTP: HTTPConfig{
			ListenAddress:     getConfigString(zcfg, "http.listen_address", "0.0.0.0:8888"),
			MaxClientCount:    getConfigInt(zcfg, "http.max_client_count", 10000),
			MaxPacketDataSize: int32(getConfigInt(zcfg, "http.max_packet_data_size", 1048576)),
			Enabled:           getConfigBool(zcfg, "http.enabled", true),
		},
		Log: zLog.Config{
			Level:              getConfigInt(zcfg, "log.level", 0),
			Console:            getConfigBool(zcfg, "log.console", true),
			ConsoleLevel:       getConfigInt(zcfg, "log.console_level", 0),
			Filename:           replacePlaceholder(getConfigString(zcfg, "log.filename", "./logs/global_server_{server_id}.log"), "{server_id}", serverID),
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
		},
		Database: db.DBConfig{
			Host:           getConfigString(zcfg, "database.global.host", "localhost"),
			Port:           getConfigInt(zcfg, "database.global.port", 3306),
			User:           getConfigString(zcfg, "database.global.user", "root"),
			Password:       getConfigString(zcfg, "database.global.password", "123456"),
			DBName:         getConfigString(zcfg, "database.global.dbname", "global"),
			Charset:        getConfigString(zcfg, "database.global.charset", "utf8mb4"),
			MaxIdle:        getConfigInt(zcfg, "database.global.max_idle", 10),
			MaxOpen:        getConfigInt(zcfg, "database.global.max_open", 100),
			Driver:         getConfigString(zcfg, "database.global.driver", "mysql"),
			MaxPoolSize:    getConfigInt(zcfg, "database.global.max_pool_size", 100),
			MinPoolSize:    getConfigInt(zcfg, "database.global.min_pool_size", 10),
			ConnectTimeout: getConfigInt(zcfg, "database.global.connect_timeout", 30),
		},
		Redis: RedisConfig{
			Host:     getConfigString(zcfg, "redis.host", "192.168.91.128"),
			Port:     getConfigInt(zcfg, "redis.port", 6379),
			Password: getConfigString(zcfg, "redis.password", ""),
			DB:       getConfigInt(zcfg, "redis.db", 0),
			PoolSize: getConfigInt(zcfg, "redis.pool_size", 10),
		},
		Pprof: PprofConfig{
			Enabled:       getConfigBool(zcfg, "pprof.enabled", false),
			ListenAddress: getConfigString(zcfg, "pprof.listen_address", "localhost:6060"),
		},
		Metrics: MetricsConfig{
			Enabled:       getConfigBool(zcfg, "metrics.enabled", true),
			ListenAddress: getConfigString(zcfg, "metrics.listen_address", "0.0.0.0:8889"),
		},
		Etcd: discovery.EtcdConfig{
			Endpoints:      getConfigString(zcfg, "Etcd.Endpoints", "etcd-cluster.kube-system.svc.cluster.local:2379"),
			Username:       getConfigString(zcfg, "Etcd.Username", ""),
			Password:       getConfigString(zcfg, "Etcd.Password", ""),
			CACertPath:     getConfigString(zcfg, "Etcd.CACertPath", "resources/etcd/ca.crt"),
			ClientCertPath: getConfigString(zcfg, "Etcd.ClientCertPath", "resources/etcd/server.crt"),
			ClientKeyPath:  getConfigString(zcfg, "Etcd.ClientKeyPath", "resources/etcd/server.key"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 验证配置
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

// ToRedisConfig 转换为Redis配置
func (c *RedisConfig) ToRedisConfig() redis.RedisConfig {
	return redis.RedisConfig(*c)
}

// ToZNetHTTPConfig 转换为zNet HTTP配置
func (c *HTTPConfig) ToZNetHTTPConfig() *zNet.HttpConfig {
	return &zNet.HttpConfig{
		ListenAddress:     c.ListenAddress,
		MaxClientCount:    c.MaxClientCount,
		MaxPacketDataSize: c.MaxPacketDataSize,
	}
}

// GetLogConfig 获取日志配置（实现LogConfigurable接口）
func (c *Config) GetLogConfig() *zLog.Config {
	return &c.Log
}

// 辅助函数
func getConfigString(cfg *zConfig.Config, key string, defaultValue string) string {
	if value, err := cfg.GetString(key); err == nil {
		return value
	}
	return defaultValue
}

func getConfigInt(cfg *zConfig.Config, key string, defaultValue int) int {
	if value, err := cfg.GetInt(key); err == nil {
		return value
	}
	return defaultValue
}

func getConfigBool(cfg *zConfig.Config, key string, defaultValue bool) bool {
	if value, err := cfg.GetBool(key); err == nil {
		return value
	}
	return defaultValue
}

func replacePlaceholder(s, placeholder string, value int) string {
	return strings.Replace(s, placeholder, fmt.Sprintf("%d", value), -1)
}
