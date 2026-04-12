package config

import (
	"fmt"
	"strings"

	"github.com/pzqf/zCommon/common/id"
	cfgutil "github.com/pzqf/zCommon/config"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zConfig"
)

const (
	MapModeSingleServer = "single_server"
	MapModeMirror       = "mirror"
	MapModeCrossGroup   = "cross_group"
)

type Config struct {
	Server     ServerConfig         `ini:"Server"`
	Database   DatabaseConfig       `ini:"Database"`
	GameServer GameServerConfig     `ini:"GameServer"`
	Etcd       discovery.EtcdConfig `ini:"Etcd"`
	Log        LogConfig            `ini:"Log"`
	Maps       MapsConfig           `ini:"Maps"`
}

type MapsConfig struct {
	Mode   string `ini:"Mode"`
	MapIDs []int  `ini:"MapIDs"`
}

type ServerConfig struct {
	ServerID          int    `ini:"ServerID"`
	ServerName        string `ini:"ServerName"`
	GroupID           int    `ini:"GroupID"`
	ListenAddr        string `ini:"ListenAddr"`
	MaxConnections    int    `ini:"MaxConnections"`
	HeartbeatInterval int    `ini:"HeartbeatInterval"`
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

type GameServerConfig struct {
	GameServerAddr           string `ini:"GameServerAddr"`
	GameServerConnectTimeout int    `ini:"GameServerConnectTimeout"`
}

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

func LoadConfig(configPath string) (*Config, error) {
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	c := &Config{
		Server: ServerConfig{
			ServerID:          cfgutil.GetConfigInt(zcfg, "Server.ServerID", 1),
			ServerName:        cfgutil.GetConfigString(zcfg, "Server.ServerName", "MapServer"),
			GroupID:           cfgutil.GetConfigInt(zcfg, "Server.GroupID", 1),
			ListenAddr:        cfgutil.GetConfigString(zcfg, "Server.ListenAddr", "0.0.0.0:9002"),
			MaxConnections:    cfgutil.GetConfigInt(zcfg, "Server.MaxConnections", 10000),
			HeartbeatInterval: cfgutil.GetConfigInt(zcfg, "Server.HeartbeatInterval", 30),
		},
		Database: DatabaseConfig{
			DBType:          cfgutil.GetConfigString(zcfg, "Database.DBType", "mysql"),
			DBHost:          cfgutil.GetConfigString(zcfg, "Database.DBHost", "127.0.0.1"),
			DBPort:          cfgutil.GetConfigInt(zcfg, "Database.DBPort", 3306),
			DBName:          cfgutil.GetConfigString(zcfg, "Database.DBName", "MapDB"),
			DBUser:          cfgutil.GetConfigString(zcfg, "Database.DBUser", "root"),
			DBPassword:      cfgutil.GetConfigString(zcfg, "Database.DBPassword", ""),
			MaxOpenConns:    cfgutil.GetConfigInt(zcfg, "Database.MaxOpenConns", 100),
			MaxIdleConns:    cfgutil.GetConfigInt(zcfg, "Database.MaxIdleConns", 10),
			ConnMaxLifetime: cfgutil.GetConfigInt(zcfg, "Database.ConnMaxLifetime", 3600),
		},
		GameServer: GameServerConfig{
			GameServerAddr:           cfgutil.GetConfigString(zcfg, "GameServer.GameServerAddr", "127.0.0.1:20002"),
			GameServerConnectTimeout: cfgutil.GetConfigInt(zcfg, "GameServer.GameServerConnectTimeout", 10),
		},
		Log: LogConfig{
			Level:              cfgutil.GetConfigInt(zcfg, "Log.Level", 0),
			Console:            cfgutil.GetConfigBool(zcfg, "Log.Console", true),
			Filename:           cfgutil.GetConfigString(zcfg, "Log.Filename", "./logs/server.log"),
			MaxSize:            cfgutil.GetConfigInt(zcfg, "Log.MaxSize", 100),
			MaxDays:            cfgutil.GetConfigInt(zcfg, "Log.MaxDays", 15),
			MaxBackups:         cfgutil.GetConfigInt(zcfg, "Log.MaxBackups", 10),
			Compress:           cfgutil.GetConfigBool(zcfg, "Log.Compress", true),
			ShowCaller:         cfgutil.GetConfigBool(zcfg, "Log.show-caller", true),
			Stacktrace:         cfgutil.GetConfigInt(zcfg, "Log.stacktrace", 3),
			Sampling:           cfgutil.GetConfigBool(zcfg, "Log.sampling", true),
			SamplingInitial:    cfgutil.GetConfigInt(zcfg, "Log.sampling-initial", 100),
			SamplingThereafter: cfgutil.GetConfigInt(zcfg, "Log.sampling-thereafter", 10),
			Async:              cfgutil.GetConfigBool(zcfg, "Log.async", true),
			AsyncBufferSize:    cfgutil.GetConfigInt(zcfg, "Log.async-buffer-size", 2048),
			AsyncFlushInterval: cfgutil.GetConfigInt(zcfg, "Log.async-flush-interval", 50),
		},
		Etcd: discovery.EtcdConfig{
			Endpoints:      cfgutil.GetConfigString(zcfg, "Etcd.Endpoints", "etcd-cluster.kube-system.svc.cluster.local:2379"),
			Username:       cfgutil.GetConfigString(zcfg, "Etcd.Username", ""),
			Password:       cfgutil.GetConfigString(zcfg, "Etcd.Password", ""),
			CACertPath:     cfgutil.GetConfigString(zcfg, "Etcd.CACertPath", "../resources/etcd/ca.crt"),
			ClientCertPath: cfgutil.GetConfigString(zcfg, "Etcd.ClientCertPath", "../resources/etcd/server.crt"),
			ClientKeyPath:  cfgutil.GetConfigString(zcfg, "Etcd.ClientKeyPath", "../resources/etcd/server.key"),
		},
		Maps: MapsConfig{
			Mode:   strings.ToLower(cfgutil.GetConfigString(zcfg, "Maps.Mode", MapModeSingleServer)),
			MapIDs: cfgutil.GetConfigIntSlice(zcfg, "Maps.MapIDs", []int{1001}),
		},
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) Validate() error {
	if _, err := id.ParseServerIDInt(int32(c.Server.ServerID)); err != nil {
		return fmt.Errorf("invalid ServerID %d: %w", c.Server.ServerID, err)
	}
	if c.Server.ListenAddr == "" {
		return fmt.Errorf("Server.ListenAddr is required")
	}
	switch strings.ToLower(c.Maps.Mode) {
	case MapModeSingleServer, MapModeMirror, MapModeCrossGroup:
	default:
		return fmt.Errorf(
			"invalid Maps.Mode %q, allowed values: %s, %s, %s",
			c.Maps.Mode,
			MapModeSingleServer,
			MapModeMirror,
			MapModeCrossGroup,
		)
	}
	c.Maps.Mode = strings.ToLower(c.Maps.Mode)

	if len(c.Maps.MapIDs) == 0 {
		return fmt.Errorf("Maps.MapIDs must not be empty")
	}
	return nil
}

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

func (c *Config) GetLogConfig() *zLog.Config {
	return c.Log.ToZLogConfig()
}
