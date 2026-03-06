package config

import (
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zUtil/zConfig"
)

// Config GlobalServer 配置
type Config struct {
	Server   ServerConfig   `ini:"server"`
	HTTP     HTTPConfig     `ini:"http"`
	Log      LogConfig      `ini:"log"`
	Database DatabaseConfig `ini:"database.global"`
	Pprof    PprofConfig    `ini:"pprof"`
}

// ServerConfig 服务器基本配置
type ServerConfig struct {
	ServerID     int32  `ini:"server_id"`
	ServerName   string `ini:"server_name"`
	WorkerID     int64  `ini:"worker_id"`
	DatacenterID int64  `ini:"datacenter_id"`
}

// HTTPConfig HTTP服务配置
type HTTPConfig struct {
	ListenAddress     string `ini:"listen_address"`
	MaxClientCount    int    `ini:"max_client_count"`
	MaxPacketDataSize int32  `ini:"max_packet_data_size"`
	Enabled           bool   `ini:"enabled"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level              int    `ini:"level"`
	Console            bool   `ini:"console"`
	Filename           string `ini:"filename"`
	MaxSize            int    `ini:"max-size"`
	MaxDays            int    `ini:"max-days"`
	MaxBackups         int    `ini:"max-backups"`
	Compress           bool   `ini:"compress"`
	ShowCaller         bool   `ini:"show-caller"`
	Stacktrace         int    `ini:"stacktrace"`
	Sampling           bool   `ini:"sampling"`
	SamplingInitial    int    `ini:"sampling-initial"`
	SamplingThereafter int    `ini:"sampling-thereafter"`
	Async              bool   `ini:"async"`
	AsyncBufferSize    int    `ini:"async-buffer-size"`
	AsyncFlushInterval int    `ini:"async-flush-interval"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host           string `ini:"host"`
	Port           int    `ini:"port"`
	User           string `ini:"user"`
	Password       string `ini:"password"`
	DBName         string `ini:"dbname"`
	Driver         string `ini:"driver"`
	MaxPoolSize    int    `ini:"max_pool_size"`
	MinPoolSize    int    `ini:"min_pool_size"`
	ConnectTimeout int    `ini:"connect_timeout"`
}

// PprofConfig pprof性能分析配置
type PprofConfig struct {
	Enabled       bool   `ini:"enabled"`
	ListenAddress string `ini:"listen_address"`
}

// LoadConfig 加载配置文件
func LoadConfig(filePath string) (*Config, error) {
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(filePath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			ServerID:     int32(getConfigInt(zcfg, "server.server_id", 1)),
			ServerName:   getConfigString(zcfg, "server.server_name", "GlobalServer"),
			WorkerID:     int64(getConfigInt(zcfg, "server.worker_id", 1)),
			DatacenterID: int64(getConfigInt(zcfg, "server.datacenter_id", 1)),
		},
		HTTP: HTTPConfig{
			ListenAddress:     getConfigString(zcfg, "http.listen_address", "0.0.0.0:8082"),
			MaxClientCount:    getConfigInt(zcfg, "http.max_client_count", 10000),
			MaxPacketDataSize: int32(getConfigInt(zcfg, "http.max_packet_data_size", 1048576)),
			Enabled:           getConfigBool(zcfg, "http.enabled", true),
		},
		Log: LogConfig{
			Level:              getConfigInt(zcfg, "log.level", 0),
			Console:            getConfigBool(zcfg, "log.console", true),
			Filename:           getConfigString(zcfg, "log.filename", "./logs/server.log"),
			MaxSize:            getConfigInt(zcfg, "log.max-size", 100),
			MaxDays:            getConfigInt(zcfg, "log.max-days", 15),
			MaxBackups:         getConfigInt(zcfg, "log.max-backups", 10),
			Compress:           getConfigBool(zcfg, "log.compress", true),
			ShowCaller:         getConfigBool(zcfg, "log.show-caller", true),
			Stacktrace:         getConfigInt(zcfg, "log.stacktrace", 3),
			Sampling:           getConfigBool(zcfg, "log.sampling", true),
			SamplingInitial:    getConfigInt(zcfg, "log.sampling-initial", 100),
			SamplingThereafter: getConfigInt(zcfg, "log.sampling-thereafter", 10),
			Async:              getConfigBool(zcfg, "log.async", true),
			AsyncBufferSize:    getConfigInt(zcfg, "log.async-buffer-size", 2048),
			AsyncFlushInterval: getConfigInt(zcfg, "log.async-flush-interval", 50),
		},
		Database: DatabaseConfig{
			Host:           getConfigString(zcfg, "database.global.host", "localhost"),
			Port:           getConfigInt(zcfg, "database.global.port", 3306),
			User:           getConfigString(zcfg, "database.global.user", "root"),
			Password:       getConfigString(zcfg, "database.global.password", ""),
			DBName:         getConfigString(zcfg, "database.global.dbname", "global"),
			Driver:         getConfigString(zcfg, "database.global.driver", "mysql"),
			MaxPoolSize:    getConfigInt(zcfg, "database.global.max_pool_size", 100),
			MinPoolSize:    getConfigInt(zcfg, "database.global.min_pool_size", 10),
			ConnectTimeout: getConfigInt(zcfg, "database.global.connect_timeout", 30),
		},
		Pprof: PprofConfig{
			Enabled:       getConfigBool(zcfg, "pprof.enabled", false),
			ListenAddress: getConfigString(zcfg, "pprof.listen_address", "localhost:6060"),
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
	return nil
}

// ToZLogConfig 转换为zLog配置
func (c *LogConfig) ToZLogConfig() *zLog.Config {
	return &zLog.Config{
		Level:              c.Level,
		Console:            c.Console,
		Filename:           c.Filename,
		MaxSize:            c.MaxSize,
		MaxDays:            c.MaxDays,
		MaxBackups:         c.MaxBackups,
		Compress:           c.Compress,
		ShowCaller:         c.ShowCaller,
		Stacktrace:         c.Stacktrace,
		Sampling:           c.Sampling,
		SamplingInitial:    c.SamplingInitial,
		SamplingThereafter: c.SamplingThereafter,
		Async:              c.Async,
		AsyncBufferSize:    c.AsyncBufferSize,
		AsyncFlushInterval: c.AsyncFlushInterval,
	}
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
	return c.Log.ToZLogConfig()
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
