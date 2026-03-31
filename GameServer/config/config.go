package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zConfig"
	"github.com/pzqf/zCommon/discovery"
)

// Config 游戏服务器配置
type Config struct {
	Server       ServerConfig         `ini:"Server"`
	Database     DatabaseConfig       `ini:"Database"`
	Gateway      GatewayConfig        `ini:"Gateway"`
	MapServer    MapServerConfig      `ini:"MapServer"`
	GlobalServer GlobalServerConfig   `ini:"GlobalServer"`
	Etcd         discovery.EtcdConfig `ini:"Etcd"`
	Logging      LoggingConfig        `ini:"Logging"`
	Metrics      MetricsConfig        `ini:"Metrics"`
}

// ServerConfig 服务器基本配置
type ServerConfig struct {
	ServerName        string `ini:"ServerName"`
	ServerID          int    `ini:"ServerID"`
	GroupID           int    `ini:"GroupID"`
	ListenAddr        string `ini:"ListenAddr"`
	ExternalAddr      string `ini:"ExternalAddr"`
	MaxConnections    int    `ini:"MaxConnections"`
	ConnectionTimeout int    `ini:"ConnectionTimeout"`
	HeartbeatInterval int    `ini:"HeartbeatInterval"`
	UseWorkerPool     bool   `ini:"UseWorkerPool"`
	WorkerPoolSize    int    `ini:"WorkerPoolSize"`
	WorkerQueueSize   int    `ini:"WorkerQueueSize"`
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

// GatewayConfig Gateway配置
type GatewayConfig struct {
	GatewayAddr           string `ini:"GatewayAddr"`
	GatewayConnectTimeout int    `ini:"GatewayConnectTimeout"`
}

// MapServerConfig MapServer配置
type MapServerConfig struct {
	MapServerAddr string `ini:"MapServerAddr"`
}

// GlobalServerConfig GlobalServer配置
type GlobalServerConfig struct {
	GlobalServerAddr string `ini:"GlobalServerAddr"`
	RegisterInterval int    `ini:"RegisterInterval"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	LogLevel           int    `ini:"LogLevel"`
	Console            bool   `ini:"console"`
	LogFile            string `ini:"LogFile"`
	LogMaxSize         int    `ini:"LogMaxSize"`
	LogMaxBackups      int    `ini:"LogMaxBackups"`
	LogMaxAge          int    `ini:"LogMaxAge"`
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

// MetricsConfig 监控配置
type MetricsConfig struct {
	Enabled     bool   `ini:"Enabled"`
	MetricsAddr string `ini:"MetricsAddr"`
}



var globalConfig *Config

// LoadConfig 加载配置
func LoadConfig(configPath string) (*Config, error) {
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	config := &Config{}

	config.Server = ServerConfig{
		ServerName:        getConfigString(zcfg, "Server.ServerName", getEnv("SERVER_NAME", "GameServer")),
		ServerID:          getConfigInt(zcfg, "Server.ServerID", getEnvAsInt("SERVER_ID", 1)),
		GroupID:           getConfigInt(zcfg, "Server.GroupID", 1),
		ListenAddr:        getConfigString(zcfg, "Server.ListenAddr", getEnv("LISTEN_ADDR", "0.0.0.0:9001")),
		ExternalAddr:      getConfigString(zcfg, "Server.ExternalAddr", getEnv("GAME_EXTERNAL_ADDR", "")),
		MaxConnections:    getConfigInt(zcfg, "Server.MaxConnections", 10000),
		ConnectionTimeout: getConfigInt(zcfg, "Server.ConnectionTimeout", 300),
		HeartbeatInterval: getConfigInt(zcfg, "Server.HeartbeatInterval", 30),
		UseWorkerPool:     getConfigBool(zcfg, "Server.UseWorkerPool", true),
		WorkerPoolSize:    getConfigInt(zcfg, "Server.WorkerPoolSize", 10),
		WorkerQueueSize:   getConfigInt(zcfg, "Server.WorkerQueueSize", 1000),
	}

	config.Database = DatabaseConfig{
		DBType:          getConfigString(zcfg, "Database.DBType", "mysql"),
		DBHost:          getConfigString(zcfg, "Database.DBHost", getEnv("DB_HOST", "192.168.91.128")),
		DBPort:          getConfigInt(zcfg, "Database.DBPort", 30306),
		DBName:          getConfigString(zcfg, "Database.DBName", getEnv("DB_NAME", "GameDB_000101")),
		DBUser:          getConfigString(zcfg, "Database.DBUser", getEnv("DB_USER", "root")),
		DBPassword:      getConfigString(zcfg, "Database.DBPassword", getEnv("DB_PASSWORD", "123456")),
		MaxOpenConns:    getConfigInt(zcfg, "Database.MaxOpenConns", 100),
		MaxIdleConns:    getConfigInt(zcfg, "Database.MaxIdleConns", 10),
		ConnMaxLifetime: getConfigInt(zcfg, "Database.ConnMaxLifetime", 3600),
	}

	config.Gateway = GatewayConfig{
		GatewayAddr:           getConfigString(zcfg, "Gateway.GatewayAddr", getEnv("GATEWAY_ADDR", "gateway-service.game:8081")),
		GatewayConnectTimeout: getConfigInt(zcfg, "Gateway.GatewayConnectTimeout", 10),
	}

	config.GlobalServer = GlobalServerConfig{
		GlobalServerAddr: getConfigString(zcfg, "GlobalServer.GlobalServerAddr", getEnv("GLOBAL_SERVER_ADDR", "global-service.game:8082")),
		RegisterInterval: getConfigInt(zcfg, "GlobalServer.RegisterInterval", 30),
	}

	config.MapServer = MapServerConfig{
		MapServerAddr: getConfigString(zcfg, "MapServer.MapServerAddr", getEnv("MAP_SERVER_ADDR", "127.0.0.1:9002")),
	}

	config.Logging = LoggingConfig{
		LogLevel:           getConfigInt(zcfg, "Logging.LogLevel", getEnvAsInt("LOG_LEVEL", 0)),
		Console:            getConfigBool(zcfg, "Logging.console", true),
		LogFile:            getConfigString(zcfg, "Logging.LogFile", "logs/server.log"),
		LogMaxSize:         getConfigInt(zcfg, "Logging.LogMaxSize", 100),
		LogMaxBackups:      getConfigInt(zcfg, "Logging.LogMaxBackups", 10),
		LogMaxAge:          getConfigInt(zcfg, "Logging.LogMaxAge", 15),
		Compress:           getConfigBool(zcfg, "Logging.compress", true),
		ShowCaller:         getConfigBool(zcfg, "Logging.show-caller", true),
		Stacktrace:         getConfigInt(zcfg, "Logging.stacktrace", 3),
		Sampling:           getConfigBool(zcfg, "Logging.sampling", true),
		SamplingInitial:    getConfigInt(zcfg, "Logging.sampling-initial", 100),
		SamplingThereafter: getConfigInt(zcfg, "Logging.sampling-thereafter", 10),
		Async:              getConfigBool(zcfg, "Logging.async", true),
		AsyncBufferSize:    getConfigInt(zcfg, "Logging.async-buffer-size", 2048),
		AsyncFlushInterval: getConfigInt(zcfg, "Logging.async-flush-interval", 50),
	}

	config.Metrics = MetricsConfig{
		Enabled:     getConfigBool(zcfg, "Metrics.Enabled", true),
		MetricsAddr: getConfigString(zcfg, "Metrics.MetricsAddr", getEnv("METRICS_ADDR", "0.0.0.0:9092")),
	}

	// 解析etcd配置
	config.Etcd = discovery.EtcdConfig{
		Endpoints:      getConfigString(zcfg, "Etcd.Endpoints", getEnv("ETCD_ENDPOINTS", "etcd-cluster.kube-system.svc.cluster.local:2379")),
		Username:       getConfigString(zcfg, "Etcd.Username", ""),
		Password:       getConfigString(zcfg, "Etcd.Password", ""),
		CACertPath:     getConfigString(zcfg, "Etcd.CACertPath", "../resources/etcd/ca.crt"),
		ClientCertPath: getConfigString(zcfg, "Etcd.ClientCertPath", "../resources/etcd/server.crt"),
		ClientKeyPath:  getConfigString(zcfg, "Etcd.ClientKeyPath", "../resources/etcd/server.key"),
	}

	globalConfig = config
	return config, nil
}

// GetServerConfig 获取服务器配置
func GetServerConfig() *Config {
	return globalConfig
}

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

// GetLogConfig 获取日志配置（实现LogConfigurable接口）
func (c *Config) GetLogConfig() *zLog.Config {
	// 处理日志文件名中的{server_id}占位符
	logFile := c.Logging.LogFile
	if strings.Contains(logFile, "{server_id}") {
		logFile = strings.ReplaceAll(logFile, "{server_id}", fmt.Sprintf("%06d", c.Server.ServerID))
	}

	return &zLog.Config{
		Level:              c.Logging.LogLevel,
		Console:            c.Logging.Console,
		Filename:           logFile,
		MaxSize:            c.Logging.LogMaxSize,
		MaxDays:            c.Logging.LogMaxAge,
		MaxBackups:         c.Logging.LogMaxBackups,
		Compress:           c.Logging.Compress,
		ShowCaller:         c.Logging.ShowCaller,
		Stacktrace:         c.Logging.Stacktrace,
		Sampling:           c.Logging.Sampling,
		SamplingInitial:    c.Logging.SamplingInitial,
		SamplingThereafter: c.Logging.SamplingThereafter,
		Async:              c.Logging.Async,
		AsyncBufferSize:    c.Logging.AsyncBufferSize,
		AsyncFlushInterval: c.Logging.AsyncFlushInterval,
	}
}

// 辅助函数：获取环境变量
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// 辅助函数：获取整数类型的环境变量
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// 辅助函数：获取布尔类型的环境变量
func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
