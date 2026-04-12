package config

import (
	"fmt"
	"strings"

	cfgutil "github.com/pzqf/zCommon/config"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zConfig"
)

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

type ServerConfig struct {
	ServerName          string `ini:"ServerName"`
	ServerID            int    `ini:"ServerID"`
	GroupID             int    `ini:"GroupID"`
	ListenAddr          string `ini:"ListenAddr"`
	ExternalAddr        string `ini:"ExternalAddr"`
	MaxConnections      int    `ini:"MaxConnections"`
	ConnectionTimeout   int    `ini:"ConnectionTimeout"`
	HeartbeatInterval   int    `ini:"HeartbeatInterval"`
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

type GatewayConfig struct {
	GatewayAddr           string `ini:"GatewayAddr"`
	GatewayConnectTimeout int    `ini:"GatewayConnectTimeout"`
}

type MapServerConfig struct {
	MapServerAddr string `ini:"MapServerAddr"`
}

type GlobalServerConfig struct {
	GlobalServerAddr string `ini:"GlobalServerAddr"`
	RegisterInterval int    `ini:"RegisterInterval"`
}

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

type MetricsConfig struct {
	Enabled     bool   `ini:"Enabled"`
	MetricsAddr string `ini:"MetricsAddr"`
}

var globalConfig *Config

func LoadConfig(configPath string) (*Config, error) {
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	c := &Config{}

	c.Server = ServerConfig{
		ServerName:          cfgutil.GetConfigString(zcfg, "Server.ServerName", cfgutil.GetEnv("SERVER_NAME", "GameServer")),
		ServerID:            cfgutil.GetConfigInt(zcfg, "Server.ServerID", cfgutil.GetEnvAsInt("SERVER_ID", 1)),
		GroupID:             cfgutil.GetConfigInt(zcfg, "Server.GroupID", 1),
		ListenAddr:          cfgutil.GetConfigString(zcfg, "Server.ListenAddr", cfgutil.GetEnv("LISTEN_ADDR", "0.0.0.0:9001")),
		ExternalAddr:        cfgutil.GetConfigString(zcfg, "Server.ExternalAddr", cfgutil.GetEnv("GAME_EXTERNAL_ADDR", "")),
		MaxConnections:      cfgutil.GetConfigInt(zcfg, "Server.MaxConnections", 10000),
		ConnectionTimeout:   cfgutil.GetConfigInt(zcfg, "Server.ConnectionTimeout", 300),
		HeartbeatInterval:   cfgutil.GetConfigInt(zcfg, "Server.HeartbeatInterval", 30),
		UseWorkerPool:       cfgutil.GetConfigBool(zcfg, "Server.UseWorkerPool", true),
		WorkerPoolSize:      cfgutil.GetConfigInt(zcfg, "Server.WorkerPoolSize", 10),
		WorkerQueueSize:     cfgutil.GetConfigInt(zcfg, "Server.WorkerQueueSize", 1000),
		ChanSize:            cfgutil.GetConfigInt(zcfg, "Server.ChanSize", 1024),
		MaxPacketDataSize:   cfgutil.GetConfigInt(zcfg, "Server.MaxPacketDataSize", 1024*1024),
		DisableEncryption:   cfgutil.GetConfigBool(zcfg, "Server.DisableEncryption", cfgutil.GetEnvAsBool("DISABLE_ENCRYPTION", true)),
		EnableKeyRotation:   cfgutil.GetConfigBool(zcfg, "Server.EnableKeyRotation", false),
		KeyRotationInterval: cfgutil.GetConfigInt(zcfg, "Server.KeyRotationInterval", 1800),
		MaxHistoryKeys:      cfgutil.GetConfigInt(zcfg, "Server.MaxHistoryKeys", 3),
		EnableSequenceCheck: cfgutil.GetConfigBool(zcfg, "Server.EnableSequenceCheck", false),
		SequenceWindowSize:  uint64(cfgutil.GetConfigInt(zcfg, "Server.SequenceWindowSize", 1000)),
		TimestampTolerance:  int64(cfgutil.GetConfigInt(zcfg, "Server.TimestampTolerance", 30)),
	}

	c.Database = DatabaseConfig{
		DBType:          cfgutil.GetConfigString(zcfg, "Database.DBType", "mysql"),
		DBHost:          cfgutil.GetConfigString(zcfg, "Database.DBHost", cfgutil.GetEnv("DB_HOST", "192.168.91.128")),
		DBPort:          cfgutil.GetConfigInt(zcfg, "Database.DBPort", 30306),
		DBName:          cfgutil.GetConfigString(zcfg, "Database.DBName", cfgutil.GetEnv("DB_NAME", "GameDB_000101")),
		DBUser:          cfgutil.GetConfigString(zcfg, "Database.DBUser", cfgutil.GetEnv("DB_USER", "root")),
		DBPassword:      cfgutil.GetConfigString(zcfg, "Database.DBPassword", cfgutil.GetEnv("DB_PASSWORD", "123456")),
		MaxOpenConns:    cfgutil.GetConfigInt(zcfg, "Database.MaxOpenConns", 100),
		MaxIdleConns:    cfgutil.GetConfigInt(zcfg, "Database.MaxIdleConns", 10),
		ConnMaxLifetime: cfgutil.GetConfigInt(zcfg, "Database.ConnMaxLifetime", 3600),
	}

	c.Gateway = GatewayConfig{
		GatewayAddr:           cfgutil.GetConfigString(zcfg, "Gateway.GatewayAddr", cfgutil.GetEnv("GATEWAY_ADDR", "gateway-service.game:8081")),
		GatewayConnectTimeout: cfgutil.GetConfigInt(zcfg, "Gateway.GatewayConnectTimeout", 10),
	}

	c.GlobalServer = GlobalServerConfig{
		GlobalServerAddr: cfgutil.GetConfigString(zcfg, "GlobalServer.GlobalServerAddr", cfgutil.GetEnv("GLOBAL_SERVER_ADDR", "global-service.game:8082")),
		RegisterInterval: cfgutil.GetConfigInt(zcfg, "GlobalServer.RegisterInterval", 30),
	}

	c.MapServer = MapServerConfig{
		MapServerAddr: cfgutil.GetConfigString(zcfg, "MapServer.MapServerAddr", cfgutil.GetEnv("MAP_SERVER_ADDR", "127.0.0.1:9002")),
	}

	c.Logging = LoggingConfig{
		LogLevel:           cfgutil.GetConfigInt(zcfg, "Logging.LogLevel", cfgutil.GetEnvAsInt("LOG_LEVEL", 0)),
		Console:            cfgutil.GetConfigBool(zcfg, "Logging.console", true),
		LogFile:            cfgutil.GetConfigString(zcfg, "Logging.LogFile", "logs/server.log"),
		LogMaxSize:         cfgutil.GetConfigInt(zcfg, "Logging.LogMaxSize", 100),
		LogMaxBackups:      cfgutil.GetConfigInt(zcfg, "Logging.LogMaxBackups", 10),
		LogMaxAge:          cfgutil.GetConfigInt(zcfg, "Logging.LogMaxAge", 15),
		Compress:           cfgutil.GetConfigBool(zcfg, "Logging.compress", true),
		ShowCaller:         cfgutil.GetConfigBool(zcfg, "Logging.show-caller", true),
		Stacktrace:         cfgutil.GetConfigInt(zcfg, "Logging.stacktrace", 3),
		Sampling:           cfgutil.GetConfigBool(zcfg, "Logging.sampling", true),
		SamplingInitial:    cfgutil.GetConfigInt(zcfg, "Logging.sampling-initial", 100),
		SamplingThereafter: cfgutil.GetConfigInt(zcfg, "Logging.sampling-thereafter", 10),
		Async:              cfgutil.GetConfigBool(zcfg, "Logging.async", true),
		AsyncBufferSize:    cfgutil.GetConfigInt(zcfg, "Logging.async-buffer-size", 2048),
		AsyncFlushInterval: cfgutil.GetConfigInt(zcfg, "Logging.async-flush-interval", 50),
	}

	c.Metrics = MetricsConfig{
		Enabled:     cfgutil.GetConfigBool(zcfg, "Metrics.Enabled", true),
		MetricsAddr: cfgutil.GetConfigString(zcfg, "Metrics.MetricsAddr", cfgutil.GetEnv("METRICS_ADDR", "0.0.0.0:9092")),
	}

	c.Etcd = discovery.EtcdConfig{
		Endpoints:      cfgutil.GetConfigString(zcfg, "Etcd.Endpoints", cfgutil.GetEnv("ETCD_ENDPOINTS", "etcd-cluster.kube-system.svc.cluster.local:2379")),
		Username:       cfgutil.GetConfigString(zcfg, "Etcd.Username", ""),
		Password:       cfgutil.GetConfigString(zcfg, "Etcd.Password", ""),
		CACertPath:     cfgutil.GetConfigString(zcfg, "Etcd.CACertPath", "../resources/etcd/ca.crt"),
		ClientCertPath: cfgutil.GetConfigString(zcfg, "Etcd.ClientCertPath", "../resources/etcd/server.crt"),
		ClientKeyPath:  cfgutil.GetConfigString(zcfg, "Etcd.ClientKeyPath", "../resources/etcd/server.key"),
	}

	globalConfig = c
	return c, nil
}

func GetServerConfig() *Config {
	return globalConfig
}

func (c *Config) GetLogConfig() *zLog.Config {
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
