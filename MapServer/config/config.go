package config

import (
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zConfig"
)

// Config MapServer 配置
type Config struct {
	Server     ServerConfig     `ini:"Server"`
	Database   DatabaseConfig   `ini:"Database"`
	GameServer GameServerConfig `ini:"GameServer"`
	Log        LogConfig        `ini:"Log"`
}

// ServerConfig 服务器基本配置
type ServerConfig struct {
	ServerID          int    `ini:"ServerID"`
	ServerName        string `ini:"ServerName"`
	ListenAddr        string `ini:"ListenAddr"`
	MaxConnections    int    `ini:"MaxConnections"`
	HeartbeatInterval int    `ini:"HeartbeatInterval"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	DBType          string `ini:"DBType"`
	DBHost          string `ini:"DBHost"`
	DBPort          int    `ini:"DBPort"`
	DBName          string `ini:"DBName"`
	DBUser          string `ini:"DBUser"`
	DBPassword      string `ini:"DBPassword"`
	MaxOpenConns    int    `ini:"MaxOpenConns"`
	MaxIdleConns    int    `ini:"MaxIdleConns"`
	ConnMaxLifetime int    `ini:"ConnMaxLifetime"`
}

// GameServerConfig GameServer连接配置
type GameServerConfig struct {
	GameServerAddr           string `ini:"GameServerAddr"`
	GameServerConnectTimeout int    `ini:"GameServerConnectTimeout"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level              int    `ini:"Level"`
	Console            bool   `ini:"Console"`
	Filename           string `ini:"Filename"`
	MaxSize            int    `ini:"MaxSize"`
	MaxDays            int    `ini:"MaxDays"`
	MaxBackups         int    `ini:"MaxBackups"`
	Compress           bool   `ini:"Compress"`
	ShowCaller         bool   `ini:"show-caller"`
	Stacktrace         int    `ini:"stacktrace"`
	Sampling           bool   `ini:"sampling"`
	SamplingInitial    int    `ini:"sampling-initial"`
	SamplingThereafter int    `ini:"sampling-thereafter"`
	Async              bool   `ini:"async"`
	AsyncBufferSize    int    `ini:"async-buffer-size"`
	AsyncFlushInterval int    `ini:"async-flush-interval"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			ServerID:          getConfigInt(zcfg, "Server.ServerID", 1),
			ServerName:        getConfigString(zcfg, "Server.ServerName", "MapServer"),
			ListenAddr:        getConfigString(zcfg, "Server.ListenAddr", "0.0.0.0:9002"),
			MaxConnections:    getConfigInt(zcfg, "Server.MaxConnections", 10000),
			HeartbeatInterval: getConfigInt(zcfg, "Server.HeartbeatInterval", 30),
		},
		Database: DatabaseConfig{
			DBType:          getConfigString(zcfg, "Database.DBType", "mysql"),
			DBHost:          getConfigString(zcfg, "Database.DBHost", "127.0.0.1"),
			DBPort:          getConfigInt(zcfg, "Database.DBPort", 3306),
			DBName:          getConfigString(zcfg, "Database.DBName", "MapDB"),
			DBUser:          getConfigString(zcfg, "Database.DBUser", "root"),
			DBPassword:      getConfigString(zcfg, "Database.DBPassword", ""),
			MaxOpenConns:    getConfigInt(zcfg, "Database.MaxOpenConns", 100),
			MaxIdleConns:    getConfigInt(zcfg, "Database.MaxIdleConns", 10),
			ConnMaxLifetime: getConfigInt(zcfg, "Database.ConnMaxLifetime", 3600),
		},
		GameServer: GameServerConfig{
			GameServerAddr:           getConfigString(zcfg, "GameServer.GameServerAddr", "127.0.0.1:20002"),
			GameServerConnectTimeout: getConfigInt(zcfg, "GameServer.GameServerConnectTimeout", 10),
		},
		Log: LogConfig{
			Level:              getConfigInt(zcfg, "Log.Level", 0),
			Console:            getConfigBool(zcfg, "Log.Console", true),
			Filename:           getConfigString(zcfg, "Log.Filename", "./logs/server.log"),
			MaxSize:            getConfigInt(zcfg, "Log.MaxSize", 100),
			MaxDays:            getConfigInt(zcfg, "Log.MaxDays", 15),
			MaxBackups:         getConfigInt(zcfg, "Log.MaxBackups", 10),
			Compress:           getConfigBool(zcfg, "Log.Compress", true),
			ShowCaller:         getConfigBool(zcfg, "Log.show-caller", true),
			Stacktrace:         getConfigInt(zcfg, "Log.stacktrace", 3),
			Sampling:           getConfigBool(zcfg, "Log.sampling", true),
			SamplingInitial:    getConfigInt(zcfg, "Log.sampling-initial", 100),
			SamplingThereafter: getConfigInt(zcfg, "Log.sampling-thereafter", 10),
			Async:              getConfigBool(zcfg, "Log.async", true),
			AsyncBufferSize:    getConfigInt(zcfg, "Log.async-buffer-size", 2048),
			AsyncFlushInterval: getConfigInt(zcfg, "Log.async-flush-interval", 50),
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
		return fmt.Errorf("ServerID must be greater than 0")
	}
	if c.Server.ListenAddr == "" {
		return fmt.Errorf("Server.ListenAddr is required")
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
