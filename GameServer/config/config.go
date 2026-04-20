package config

import (
	"fmt"

	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zEngine/zConfig"
	"github.com/pzqf/zEngine/zLog"
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

	serverID := zConfig.GetIntWithDefault(zcfg, "Server.ServerID", 1)

	c := &Config{}

	c.Server = ServerConfig{
		ServerName:          zConfig.GetStringWithDefault(zcfg, "Server.ServerName", zConfig.GetEnv("SERVER_NAME", "GameServer")),
		ServerID:            serverID,
		GroupID:             zConfig.GetIntWithDefault(zcfg, "Server.GroupID", 1),
		ListenAddr:          zConfig.GetStringWithDefault(zcfg, "Server.ListenAddr", zConfig.GetEnv("LISTEN_ADDR", "0.0.0.0:9001")),
		ExternalAddr:        zConfig.GetStringWithDefault(zcfg, "Server.ExternalAddr", zConfig.GetEnv("GAME_EXTERNAL_ADDR", "")),
		MaxConnections:      zConfig.GetIntWithDefault(zcfg, "Server.MaxConnections", 10000),
		ConnectionTimeout:   zConfig.GetIntWithDefault(zcfg, "Server.ConnectionTimeout", 300),
		HeartbeatInterval:   zConfig.GetIntWithDefault(zcfg, "Server.HeartbeatInterval", 30),
		UseWorkerPool:       zConfig.GetBoolWithDefault(zcfg, "Server.UseWorkerPool", true),
		WorkerPoolSize:      zConfig.GetIntWithDefault(zcfg, "Server.WorkerPoolSize", 10),
		WorkerQueueSize:     zConfig.GetIntWithDefault(zcfg, "Server.WorkerQueueSize", 1000),
		ChanSize:            zConfig.GetIntWithDefault(zcfg, "Server.ChanSize", 1024),
		MaxPacketDataSize:   zConfig.GetIntWithDefault(zcfg, "Server.MaxPacketDataSize", 1024*1024),
		DisableEncryption:   zConfig.GetBoolWithDefault(zcfg, "Server.DisableEncryption", zConfig.GetEnvAsBool("DISABLE_ENCRYPTION", true)),
		EnableKeyRotation:   zConfig.GetBoolWithDefault(zcfg, "Server.EnableKeyRotation", false),
		KeyRotationInterval: zConfig.GetIntWithDefault(zcfg, "Server.KeyRotationInterval", 1800),
		MaxHistoryKeys:      zConfig.GetIntWithDefault(zcfg, "Server.MaxHistoryKeys", 3),
		EnableSequenceCheck: zConfig.GetBoolWithDefault(zcfg, "Server.EnableSequenceCheck", false),
		SequenceWindowSize:  uint64(zConfig.GetIntWithDefault(zcfg, "Server.SequenceWindowSize", 1000)),
		TimestampTolerance:  int64(zConfig.GetIntWithDefault(zcfg, "Server.TimestampTolerance", 30)),
	}

	c.Database = DatabaseConfig{
		DBType:          zConfig.GetStringWithDefault(zcfg, "Database.DBType", "mysql"),
		DBHost:          zConfig.GetStringWithDefault(zcfg, "Database.DBHost", zConfig.GetEnv("DB_HOST", "192.168.91.128")),
		DBPort:          zConfig.GetIntWithDefault(zcfg, "Database.DBPort", 30306),
		DBName:          zConfig.GetStringWithDefault(zcfg, "Database.DBName", zConfig.GetEnv("DB_NAME", "GameDB_000101")),
		DBUser:          zConfig.GetStringWithDefault(zcfg, "Database.DBUser", zConfig.GetEnv("DB_USER", "root")),
		DBPassword:      zConfig.GetStringWithDefault(zcfg, "Database.DBPassword", zConfig.GetEnv("DB_PASSWORD", "123456")),
		MaxOpenConns:    zConfig.GetIntWithDefault(zcfg, "Database.MaxOpenConns", 100),
		MaxIdleConns:    zConfig.GetIntWithDefault(zcfg, "Database.MaxIdleConns", 10),
		ConnMaxLifetime: zConfig.GetIntWithDefault(zcfg, "Database.ConnMaxLifetime", 3600),
	}

	c.Gateway = GatewayConfig{
		GatewayAddr:           zConfig.GetStringWithDefault(zcfg, "Gateway.GatewayAddr", zConfig.GetEnv("GATEWAY_ADDR", "gateway-service.game:8081")),
		GatewayConnectTimeout: zConfig.GetIntWithDefault(zcfg, "Gateway.GatewayConnectTimeout", 10),
	}

	c.GlobalServer = GlobalServerConfig{
		GlobalServerAddr: zConfig.GetStringWithDefault(zcfg, "GlobalServer.GlobalServerAddr", zConfig.GetEnv("GLOBAL_SERVER_ADDR", "global-service.game:8082")),
		RegisterInterval: zConfig.GetIntWithDefault(zcfg, "GlobalServer.RegisterInterval", 30),
	}

	c.MapServer = MapServerConfig{
		MapServerAddr: zConfig.GetStringWithDefault(zcfg, "MapServer.MapServerAddr", zConfig.GetEnv("MAP_SERVER_ADDR", "127.0.0.1:9002")),
	}

	c.Log = zLog.Config{
		Level:              zConfig.GetIntWithDefault(zcfg, "Log.Level", zConfig.GetEnvAsInt("LOG_LEVEL", 0)),
		Console:            zConfig.GetBoolWithDefault(zcfg, "Log.Console", true),
		ConsoleLevel:       zConfig.GetIntWithDefault(zcfg, "Log.ConsoleLevel", 0),
		Filename:           zConfig.ReplacePlaceholder(zConfig.GetStringWithDefault(zcfg, "Log.Filename", "./logs/game_server_{ServerID}.log"), "{ServerID}", serverID),
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
	}

	c.Metrics = MetricsConfig{
		Enabled:       zConfig.GetBoolWithDefault(zcfg, "Metrics.Enabled", true),
		ListenAddress: zConfig.GetStringWithDefault(zcfg, "Metrics.ListenAddress", zConfig.GetEnv("METRICS_ADDR", "0.0.0.0:9092")),
	}

	c.Pprof = PprofConfig{
		Enabled:       zConfig.GetBoolWithDefault(zcfg, "Pprof.Enabled", false),
		ListenAddress: zConfig.GetStringWithDefault(zcfg, "Pprof.ListenAddress", "localhost:6061"),
	}

	c.Etcd = discovery.EtcdConfig{
		Endpoints:      zConfig.GetStringWithDefault(zcfg, "Etcd.Endpoints", zConfig.GetEnv("ETCD_ENDPOINTS", "etcd-cluster.kube-system.svc.cluster.local:2379")),
		Username:       zConfig.GetStringWithDefault(zcfg, "Etcd.Username", ""),
		Password:       zConfig.GetStringWithDefault(zcfg, "Etcd.Password", ""),
		CACertPath:     zConfig.GetStringWithDefault(zcfg, "Etcd.CACertPath", "../resources/etcd/ca.crt"),
		ClientCertPath: zConfig.GetStringWithDefault(zcfg, "Etcd.ClientCertPath", "../resources/etcd/server.crt"),
		ClientKeyPath:  zConfig.GetStringWithDefault(zcfg, "Etcd.ClientKeyPath", "../resources/etcd/server.key"),
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
