package config

import (
	"fmt"

	cfgutil "github.com/pzqf/zCommon/config"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/metrics"
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
	Log          zLog.Config          `ini:"Log"`
	Metrics      MetricsConfig        `ini:"Metrics"`
	Pprof        PprofConfig          `ini:"Pprof"`
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

type MetricsConfig metrics.MetricsConfig

type PprofConfig struct {
	Enabled       bool   `ini:"Enabled"`
	ListenAddress string `ini:"ListenAddress"`
}

var globalConfig *Config

func LoadConfig(configPath string) (*Config, error) {
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	serverID := cfgutil.GetConfigInt(zcfg, "Server.ServerID", 1)

	c := &Config{}

	c.Server = ServerConfig{
		ServerName:          cfgutil.GetConfigString(zcfg, "Server.ServerName", cfgutil.GetEnv("SERVER_NAME", "GameServer")),
		ServerID:            serverID,
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

	c.Log = zLog.Config{
		Level:              cfgutil.GetConfigInt(zcfg, "Log.Level", cfgutil.GetEnvAsInt("LOG_LEVEL", 0)),
		Console:            cfgutil.GetConfigBool(zcfg, "Log.Console", true),
		ConsoleLevel:       cfgutil.GetConfigInt(zcfg, "Log.ConsoleLevel", 0),
		Filename:           cfgutil.ReplacePlaceholder(cfgutil.GetConfigString(zcfg, "Log.Filename", "./logs/game_server_{ServerID}.log"), "{ServerID}", serverID),
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
		ListenAddress: cfgutil.GetConfigString(zcfg, "Metrics.ListenAddress", cfgutil.GetEnv("METRICS_ADDR", "0.0.0.0:9092")),
	}

	c.Pprof = PprofConfig{
		Enabled:       cfgutil.GetConfigBool(zcfg, "Pprof.Enabled", false),
		ListenAddress: cfgutil.GetConfigString(zcfg, "Pprof.ListenAddress", "localhost:6061"),
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
	return &c.Log
}
